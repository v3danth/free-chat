package websocket

import (
	"context"
	"encoding/json"
	"log"
	"math"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/v3danth/free-chat/internal/apperr"
	"github.com/v3danth/free-chat/internal/filter"
	"github.com/v3danth/free-chat/internal/id"
	"github.com/v3danth/free-chat/internal/letters"
	"github.com/v3danth/free-chat/internal/media"
	"github.com/v3danth/free-chat/internal/message"
	"github.com/v3danth/free-chat/internal/ratelimit"
	"github.com/v3danth/free-chat/internal/user"
)

const dbTimeout = 5 * time.Second

var (
	errNoRoom       = apperr.New(apperr.NotFound, "that room does not exist")
	errNotInRoom    = apperr.New(apperr.Forbidden, "join the room before sending to it")
	errRateLimited  = apperr.New(apperr.RateLimited, "slow down, you are sending too fast")
	errMuted        = apperr.New(apperr.Forbidden, "you are muted for now")
	errBlockedWords = apperr.New(apperr.Invalid, "your message contains words that are not allowed")
	errUnavailable  = apperr.New(apperr.NotFound, "this person is not available")
	errSelf         = apperr.New(apperr.Invalid, "you cannot do that to yourself")
	errWaitReply    = apperr.New(apperr.Forbidden, "wait for them to reply before sending more")
	errKnockImage   = apperr.New(apperr.Forbidden, "you can send photos once they reply")
	errPhotoLater   = apperr.New(apperr.Forbidden, "photos open up once you both choose to get closer")
	errNoKnocks     = apperr.New(apperr.Forbidden, "they are not taking knocks right now")
	errMembersOnly  = apperr.New(apperr.Forbidden, "they only take knocks from members")
	errTooManyKnock = apperr.New(apperr.RateLimited, "you have knocked on a lot of people; give them time to answer")
	errNotOpen      = apperr.New(apperr.Forbidden, "you need an open chat with them first")
	errNoKnock      = apperr.New(apperr.NotFound, "there is no knock from them to decline")
	errDeclined     = apperr.New(apperr.Forbidden, "they passed on this chat")
	errLetterKinds  = apperr.New(apperr.Forbidden, "letters are for members writing to members")
	// errReconsider asks the sender whether a flagged message is what they
	// meant; sending it again with confirm goes through.
	errReconsider = apperr.New(apperr.Invalid, "this might land badly. Send it anyway?")
)

type Writer interface {
	Enqueue(message.Message) bool
}

type Attacher interface {
	Attach(ctx context.Context, mediaID, owner uint64) (media.Attachment, error)
}

type Presence interface {
	Touch(ctx context.Context, id uint64) error
}

type Blocks interface {
	Between(ctx context.Context, user uint64) (map[uint64]struct{}, error)
	Add(ctx context.Context, blocker, blocked uint64) error
	Remove(ctx context.Context, blocker, blocked uint64) error
}

// Letterbox stores letters until they are due.
type Letterbox interface {
	Send(ctx context.Context, l letters.Letter) error
}

type Deps struct {
	Writer  Writer
	IDs     *id.Generator
	Media   Attacher
	Users   Presence
	Blocks  Blocks
	Limiter *ratelimit.Limiter
	// KnockLimiter caps new chats per sender, so one person cannot open on
	// everyone in the room at once.
	KnockLimiter *ratelimit.Limiter
	Filter       *filter.Live
	HistoryLimit int
	Letters      Letterbox
	// LetterDelay is how long a letter takes to arrive.
	LetterDelay time.Duration
}

type room struct {
	id      uint64
	name    string
	members map[*Client]struct{}
	recent  []chatEvent // oldest first, at most HistoryLimit
}

// pair is an unordered pair of users; a door is the private chat between them.
type pair struct{ lo, hi uint64 }

// other is the second person in p, if id is one of them.
func (p pair) other(id uint64) (uint64, bool) {
	switch id {
	case p.lo:
		return p.hi, true
	case p.hi:
		return p.lo, true
	}
	return 0, false
}

func pairOf(a, b uint64) pair {
	if a > b {
		a, b = b, a
	}
	return pair{a, b}
}

// door: the knocker's first message is a knock; the other side replying
// opens it. Doors live only while both people are online.
type door struct {
	knocker uint64
	open    bool
	// knock is the knocker's message while the door is shut, replayed in
	// hello so a reload shows what is waiting.
	knock *knockView
	// closer holds who asked to get closer. Photos are shared only once both
	// have, so neither side is ever shown more than they agreed to.
	closer map[uint64]bool
}

func (d *door) isClose() bool { return len(d.closer) == 2 }

// ladder is the door's reveal state as seen by viewer, who is talking to other.
func (d *door) ladder(viewer, other uint64) ladderView {
	return ladderView{AskedByMe: d.closer[viewer], AskedByThem: d.closer[other], Close: d.isClose()}
}

// Hub is the only mutable shared state for live connections. Everything
// below mu, including each Client's guarded fields, is accessed with it held.
type Hub struct {
	Deps

	mu      sync.RWMutex
	clients map[uint64]*Client
	rooms   map[uint64]*room
	order   []uint64
	doors   map[pair]*door
	// declined maps a pair to whoever declined the other's knock, so the
	// knocker cannot knock again until one of them leaves.
	declined map[pair]uint64
}

// NewHub starts with the live rooms and their recent messages (newest
// first, as the repository returns them), so joining never reads MySQL.
func NewHub(d Deps, rooms []message.Room, recent map[uint64][]message.Entry) *Hub {
	h := &Hub{
		Deps:     d,
		clients:  make(map[uint64]*Client),
		rooms:    make(map[uint64]*room),
		doors:    make(map[pair]*door),
		declined: make(map[pair]uint64),
	}
	for _, r := range rooms {
		rm := &room{id: r.ID, name: r.Name, members: make(map[*Client]struct{})}
		entries := recent[r.ID]
		for i := len(entries) - 1; i >= 0; i-- {
			rm.push(entryEvent(entries[i]), d.HistoryLimit)
		}
		h.rooms[r.ID] = rm
		h.order = append(h.order, r.ID)
	}
	return h
}

func (r *room) push(ev chatEvent, limit int) {
	if limit <= 0 {
		return
	}
	if len(r.recent) == limit {
		r.recent = slices.Delete(r.recent, 0, 1)
	}
	r.recent = append(r.recent, ev)
}

// ---------------------------------------------------------------------------
// Connection lifecycle
// ---------------------------------------------------------------------------

func (h *Hub) register(c *Client) {
	h.mu.Lock()
	if old, ok := h.clients[c.id()]; ok {
		h.detachLocked(old) // a reconnect replaces the old socket; doors survive
	}
	h.clients[c.id()] = c

	online := make([]json.RawMessage, 0, len(h.clients)-1)
	for _, other := range h.clients {
		if other != c && !blockedLocked(c, other) {
			online = append(online, other.card)
		}
	}
	rooms := make([]roomView, 0, len(h.order))
	for _, rid := range h.order {
		rooms = append(rooms, roomView{ID: rid, Name: h.rooms[rid].name})
	}
	doors := []doorView{}
	for p, d := range h.doors {
		if other, ok := p.other(c.id()); ok {
			v := doorView{With: other, Open: d.open, KnockedByMe: !d.open && d.knocker == c.id(),
				Knock: d.knock, ladderView: d.ladder(c.id(), other)}
			if peer, online := h.clients[other]; online && d.isClose() {
				revealed := user.ToRevealed(peer.user, peer.since)
				v.Peer = &revealed
			}
			doors = append(doors, v)
		}
	}
	card := c.card
	// hello goes out before the lock is released, so nothing sent to this
	// client (a DM, someone else's join) can arrive ahead of it.
	c.deliver(helloFrame(helloEvent{
		Type:  "hello",
		You:   user.ToSelf(c.user),
		Rooms: rooms,
		RateLimit: rateLimitView{
			Remaining: h.Limiter.Remaining(c.id()),
			ResetIn:   int64(math.Ceil(h.Limiter.ResetIn(c.id()).Seconds())),
		},
		Doors: doors,
	}, online))
	h.mu.Unlock()

	h.broadcastPresence(c, presenceFrame("join", c.id(), card))
	h.touch(c.id())
}

func (h *Hub) unregister(c *Client) {
	h.mu.Lock()
	h.detachLocked(c)
	current := h.clients[c.id()] == c
	if current {
		delete(h.clients, c.id())
		for p := range h.doors {
			if p.lo == c.id() || p.hi == c.id() {
				delete(h.doors, p)
			}
		}
		for p := range h.declined {
			if p.lo == c.id() || p.hi == c.id() {
				delete(h.declined, p)
			}
		}
	}
	h.mu.Unlock()

	if current {
		h.broadcastPresence(c, presenceFrame("leave", c.id(), nil))
		h.touch(c.id())
	}
}

// OnlineIDs lists everyone connected right now.
func (h *Hub) OnlineIDs() []uint64 {
	h.mu.RLock()
	defer h.mu.RUnlock()
	ids := make([]uint64, 0, len(h.clients))
	for uid := range h.clients {
		ids = append(ids, uid)
	}
	return ids
}

// IsNameOnline reports whether someone connected answers to name. Sign-up
// asks before handing out a name; it compares case-insensitively, like the
// users table does.
func (h *Hub) IsNameOnline(name string) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, c := range h.clients {
		if strings.EqualFold(c.user.Profile.Name, name) {
			return true
		}
	}
	return false
}

// Forget drops room history the database no longer has: messages older
// than beforeID (retention) and anything sent by the given deleted users.
func (h *Hub) Forget(beforeID uint64, senders []uint64) {
	gone := make(map[uint64]bool, len(senders))
	for _, id := range senders {
		gone[id] = true
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	for _, r := range h.rooms {
		r.recent = slices.DeleteFunc(r.recent, func(ev chatEvent) bool { return ev.ID < beforeID || gone[ev.SenderID] })
	}
}

// PhotosShared reports whether a and b both chose to get closer, so each
// sees the other's profile photo.
func (h *Hub) PhotosShared(a, b uint64) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	d, ok := h.doors[pairOf(a, b)]
	return ok && d.isClose()
}

// Snapshot counts who is connected right now, for the control panel.
type Snapshot struct {
	Online    int            `json:"online"`
	Guests    int            `json:"guests"`
	Members   int            `json:"members"`
	Staff     int            `json:"staff"`
	Countries map[string]int `json:"countries"`
	Tags      map[string]int `json:"tags"` // lowercased, so "Music" and "music" count together
	Knocks    int            `json:"knocks_waiting"`
	OpenDoors int            `json:"open_chats"`
}

func (h *Hub) Snapshot() Snapshot {
	h.mu.RLock()
	defer h.mu.RUnlock()
	s := Snapshot{Online: len(h.clients), Countries: map[string]int{}, Tags: map[string]int{}}
	for _, c := range h.clients {
		u := c.user
		if u.Kind == user.KindGuest {
			s.Guests++
		} else {
			s.Members++
		}
		if u.Role.AtLeast(user.RoleModerator) {
			s.Staff++
		}
		country := u.Country
		if country == "" {
			country = "unknown"
		}
		s.Countries[country]++
		for _, tag := range u.Profile.Tags {
			s.Tags[strings.ToLower(tag)]++
		}
	}
	for _, d := range h.doors {
		if d.open {
			s.OpenDoors++
		} else {
			s.Knocks++
		}
	}
	return s
}

// Close disconnects everyone; used on server shutdown.
func (h *Hub) Close() {
	h.mu.Lock()
	defer h.mu.Unlock()
	for _, c := range h.clients {
		h.detachLocked(c)
	}
}

func (h *Hub) detachLocked(c *Client) {
	for rid := range c.rooms {
		h.leaveLocked(c, rid)
	}
	c.close()
}

func (h *Hub) leaveLocked(c *Client, rid uint64) {
	delete(c.rooms, rid)
	if r, ok := h.rooms[rid]; ok {
		delete(r.members, c)
	}
}

// ---------------------------------------------------------------------------
// Commands
// ---------------------------------------------------------------------------

func (h *Hub) dispatch(c *Client, cmd command) error {
	switch cmd := cmd.(type) {
	case joinCmd:
		return h.join(c, cmd.room)
	case leaveCmd:
		h.mu.Lock()
		h.leaveLocked(c, cmd.room)
		h.mu.Unlock()
		return nil
	case roomSendCmd:
		return h.sendRoom(c, cmd)
	case dmCmd:
		return h.sendDM(c, cmd)
	case blockCmd:
		return h.block(c, cmd.user)
	case unblockCmd:
		return h.unblock(c, cmd.user)
	case closerCmd:
		return h.askCloser(c, cmd.user)
	case tomorrowCmd:
		return h.promiseTomorrow(c, cmd.user)
	case declineCmd:
		return h.decline(c, cmd.user)
	case letterCmd:
		return h.sendLetter(c, cmd)
	}
	return errUnknown
}

func (h *Hub) join(c *Client, rid uint64) error {
	h.mu.Lock()
	r, ok := h.rooms[rid]
	if !ok {
		h.mu.Unlock()
		return errNoRoom
	}
	r.members[c] = struct{}{}
	c.rooms[rid] = struct{}{}
	history := make([]chatEvent, 0, len(r.recent))
	for _, ev := range r.recent {
		if _, hidden := c.blocked[ev.SenderID]; !hidden {
			history = append(history, ev)
		}
	}
	h.mu.Unlock()

	c.deliver(historyFrame(rid, history))
	return nil
}

func (h *Hub) sendRoom(c *Client, cmd roomSendCmd) error {
	h.mu.RLock()
	_, member := c.rooms[cmd.room]
	h.mu.RUnlock()
	if !member {
		return errNotInRoom
	}

	out, err := h.prepare(c, cmd.content, cmd.mediaID, cmd.confirm)
	if err != nil {
		return err
	}
	m := message.NewRoom(h.IDs.Next(), cmd.room, c.id(), out.name, out.body, out.att.ID)
	ev := roomEvent(m, out.att, out.filtered)
	frame := encode(ev)

	h.mu.Lock()
	if r, ok := h.rooms[cmd.room]; ok {
		r.push(ev, h.HistoryLimit)
		for member := range r.members {
			if !blockedLocked(member, c) {
				member.deliver(frame)
			}
		}
	}
	h.mu.Unlock()

	h.Writer.Enqueue(m)
	return nil
}

func (h *Hub) sendDM(c *Client, cmd dmCmd) error {
	if cmd.to == c.id() {
		return errSelf
	}
	// Cheap checks first, so a refused knock does not spend rate limit.
	h.mu.RLock()
	peer, online := h.clients[cmd.to]
	if !online || blockedLocked(c, peer) {
		h.mu.RUnlock()
		return errUnavailable
	}
	isKnock, _, doorErr := h.knockLocked(c, peer, cmd.mediaID != 0, false)
	h.mu.RUnlock()
	if doorErr != nil {
		return doorErr
	}

	out, err := h.prepare(c, cmd.content, cmd.mediaID, cmd.confirm)
	if err != nil {
		return err
	}
	if isKnock && !h.KnockLimiter.Allow(c.id()) {
		return errTooManyKnock
	}
	m := message.NewDirect(h.IDs.Next(), cmd.to, c.id(), out.name, out.body, out.att.ID)

	h.mu.Lock()
	peer, online = h.clients[cmd.to]
	if !online || blockedLocked(c, peer) {
		h.mu.Unlock()
		return errUnavailable
	}
	knock, opened, err := h.knockLocked(c, peer, cmd.mediaID != 0, true)
	if err != nil {
		h.mu.Unlock()
		return err
	}
	ev := directEvent(m, out.att, knock, out.filtered)
	if knock {
		h.doors[pairOf(c.id(), cmd.to)].knock = &knockView{ID: ev.ID, From: ev.From, Content: ev.Content, Timestamp: ev.Timestamp}
	}
	frame := encode(ev)
	peer.deliver(frame)
	c.deliver(frame) // echo: the sender learns the id and the filtered text
	if opened {
		c.deliver(encode(doorEvent{Type: "door", With: user.ToKnown(peer.user, peer.since)}))
		peer.deliver(encode(doorEvent{Type: "door", With: user.ToKnown(c.user, c.since)}))
	}
	h.mu.Unlock()

	h.Writer.Enqueue(m)
	return nil
}

// knockRefusal tells a sender why policy turns them away, so a guest knows
// that becoming a member would let them through. Nil means admitted.
func knockRefusal(policy user.KnockPolicy, sender user.Kind) error {
	switch {
	case policy.Admits(sender):
		return nil
	case policy == user.KnocksMembers:
		return errMembersOnly
	}
	return errNoKnocks
}

// knockLocked decides what a private message from -> to means. Without
// commit it only checks; with commit it records the transition. Knocks are
// text only and photos wait until both choose to get closer, so nobody
// receives a photo they did not ask for.
func (h *Hub) knockLocked(from, to *Client, hasImage, commit bool) (knock, opened bool, err error) {
	k := pairOf(from.id(), to.id())
	d, ok := h.doors[k]
	switch {
	case !ok:
		if hasImage {
			return false, false, errKnockImage
		}
		if h.declined[k] == to.id() {
			return false, false, errDeclined
		}
		if err := knockRefusal(to.user.Settings.KnocksFrom, from.user.Kind); err != nil {
			return false, false, err
		}
		if commit {
			h.doors[k] = &door{knocker: from.id()}
		}
		return true, false, nil
	case d.open:
		if hasImage && !d.isClose() {
			return false, false, errPhotoLater
		}
		return false, false, nil
	case d.knocker == from.id():
		return false, false, errWaitReply
	default:
		if commit {
			d.open = true
			d.knock = nil
		}
		return false, true, nil
	}
}

type prepared struct {
	name     string
	body     string
	filtered bool
	att      media.Attachment
}

// prepare runs every check a message needs before it can be sent: mute,
// word filter, rate limit, and image ownership. The filter runs before the
// rate limit so being asked to reconsider never costs a message.
func (h *Hub) prepare(c *Client, content string, mediaID uint64, confirm bool) (prepared, error) {
	h.mu.RLock()
	name, muted := c.user.Profile.Name, c.user.MutedUntil
	h.mu.RUnlock()

	if muted != nil && time.Now().Before(*muted) {
		return prepared{}, errMuted
	}
	res := h.Filter.Load().Apply(content)
	switch {
	case res.Blocked:
		return prepared{}, errBlockedWords
	case !confirm && slices.Contains(res.Violations, filter.ViolationProhibited):
		return prepared{}, errReconsider
	}
	if !h.Limiter.Allow(c.id()) {
		return prepared{}, errRateLimited
	}
	out := prepared{name: name, body: res.Content, filtered: res.Filtered}
	if mediaID != 0 {
		err := h.withDB(func(ctx context.Context) (err error) {
			out.att, err = h.Media.Attach(ctx, mediaID, c.id())
			return err
		})
		if err != nil {
			return prepared{}, err
		}
	}
	return out, nil
}

// openPeerLocked is the precondition shared by every command that only an
// open chat may send, so the rule lives in one place. The caller holds h.mu.
func (h *Hub) openPeerLocked(c *Client, target uint64) (*Client, *door, error) {
	peer, online := h.clients[target]
	d, ok := h.doors[pairOf(c.id(), target)]
	if !online || !ok || !d.open {
		return nil, nil, errNotOpen
	}
	return peer, d, nil
}

// askCloser records that c wants to climb the reveal ladder with target.
// Once both have asked, each receives the other's photo.
func (h *Hub) askCloser(c *Client, target uint64) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	peer, d, err := h.openPeerLocked(c, target)
	if err != nil {
		return err
	}
	if d.closer == nil {
		d.closer = make(map[uint64]bool, 2)
	}
	d.closer[c.id()] = true

	mine := closerEvent{Type: "closer", With: target, ladderView: d.ladder(c.id(), target)}
	theirs := closerEvent{Type: "closer", With: c.id(), ladderView: d.ladder(target, c.id())}
	if d.isClose() {
		peerPhoto, myPhoto := user.ToRevealed(peer.user, peer.since), user.ToRevealed(c.user, c.since)
		mine.Peer, theirs.Peer = &peerPhoto, &myPhoto
	}
	c.deliver(encode(mine))
	peer.deliver(encode(theirs))
	return nil
}

// promiseTomorrow tells target that c would like to pick the chat up
// again. The client keeps the promise; the server only passes it on.
func (h *Hub) promiseTomorrow(c *Client, target uint64) error {
	h.mu.RLock()
	defer h.mu.RUnlock()
	peer, _, err := h.openPeerLocked(c, target)
	if err != nil {
		return err
	}
	peer.deliver(encode(tomorrowEvent{Type: "tomorrow", From: c.id()}))
	return nil
}

// decline turns down a knock c received. Only the person knocked may
// decline, and only while the door is still shut; an open chat ends by
// leaving or blocking.
func (h *Hub) decline(c *Client, knocker uint64) error {
	k := pairOf(c.id(), knocker)
	h.mu.Lock()
	defer h.mu.Unlock()
	d, ok := h.doors[k]
	if !ok || d.open || d.knocker != knocker {
		return errNoKnock
	}
	delete(h.doors, k)
	h.declined[k] = c.id()
	c.deliver(encode(doorClosedEvent{Type: "door_closed", With: knocker, Reason: "declined", ByMe: true}))
	if peer, online := h.clients[knocker]; online {
		peer.deliver(encode(doorClosedEvent{Type: "door_closed", With: c.id(), Reason: "declined"}))
	}
	return nil
}

// sendLetter stores a letter that arrives LetterDelay later. Letters need
// an open chat, so nobody can be written to by a stranger, and both sides
// must be members, whose ids outlive a visit.
func (h *Hub) sendLetter(c *Client, cmd letterCmd) error {
	h.mu.RLock()
	peer, _, err := h.openPeerLocked(c, cmd.to)
	members := err == nil && c.user.Kind == user.KindMember && peer.user.Kind == user.KindMember
	h.mu.RUnlock()
	switch {
	case err != nil:
		return err
	case !members:
		return errLetterKinds
	}

	out, err := h.prepare(c, cmd.content, 0, cmd.confirm)
	if err != nil {
		return err
	}
	l := letters.Letter{ID: h.IDs.Next(), SenderID: c.id(), RecipientID: cmd.to, SenderName: out.name,
		Body: out.body, DeliverAt: time.Now().Add(h.LetterDelay)}
	if err := h.withDB(func(ctx context.Context) error { return h.Letters.Send(ctx, l) }); err != nil {
		return err
	}
	c.deliver(encode(letterSentEvent{Type: "letter_sent", To: cmd.to, DeliverAt: l.DeliverAt.Unix()}))
	return nil
}

// DeliverLetter hands a due letter to its recipient if they are online. A
// letter between people who have since blocked each other is dropped.
func (h *Hub) DeliverLetter(l letters.Letter) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	c, online := h.clients[l.RecipientID]
	if !online {
		return false
	}
	if _, blocked := c.blocked[l.SenderID]; !blocked {
		c.deliver(encode(letterEvent{Type: "letter", ID: l.ID, From: l.SenderID, Name: l.SenderName,
			Content: l.Body, SentAt: l.SentAt().Unix()}))
	}
	return true
}

// block is mutual and silent: each side simply disappears for the other.
func (h *Hub) block(c *Client, target uint64) error {
	if target == c.id() {
		return errSelf
	}
	if err := h.withDB(func(ctx context.Context) error { return h.Blocks.Add(ctx, c.id(), target) }); err != nil {
		return err
	}

	h.mu.Lock()
	c.blocked[target] = struct{}{}
	peer := h.clients[target]
	if peer != nil {
		peer.blocked[c.id()] = struct{}{}
	}
	delete(h.doors, pairOf(c.id(), target))
	h.mu.Unlock()

	c.deliver(presenceFrame("leave", target, nil))
	if peer != nil {
		peer.deliver(presenceFrame("leave", c.id(), nil))
	}
	return nil
}

func (h *Hub) unblock(c *Client, target uint64) error {
	var mine, theirs map[uint64]struct{}
	err := h.withDB(func(ctx context.Context) (err error) {
		if err = h.Blocks.Remove(ctx, c.id(), target); err != nil {
			return err
		}
		if mine, err = h.Blocks.Between(ctx, c.id()); err != nil {
			return err
		}
		theirs, err = h.Blocks.Between(ctx, target)
		return err
	})
	if err != nil {
		return err
	}

	h.mu.Lock()
	c.blocked = mine
	peer := h.clients[target]
	var visible bool
	var cCard, pCard json.RawMessage
	if peer != nil {
		peer.blocked = theirs
		visible = !blockedLocked(c, peer)
		cCard, pCard = c.card, peer.card
	}
	h.mu.Unlock()

	if visible {
		c.deliver(presenceFrame("join", target, pCard))
		peer.deliver(presenceFrame("join", c.id(), cCard))
	}
	return nil
}

// ---------------------------------------------------------------------------
// Moderation and profile hooks (called from HTTP handlers)
// ---------------------------------------------------------------------------

// Kick shows a notice and disconnects the user if online.
func (h *Hub) Kick(userID uint64, code, text string, until time.Time) {
	h.mu.Lock()
	c := h.clients[userID]
	h.mu.Unlock()
	if c != nil {
		c.deliver(noticeFrame(code, text, until))
		c.close()
	}
}

// Mute takes effect immediately for a live connection.
func (h *Hub) Mute(userID uint64, until time.Time) {
	h.mu.Lock()
	c := h.clients[userID]
	if c != nil {
		c.user.MutedUntil = &until
	}
	h.mu.Unlock()
	if c != nil {
		c.deliver(noticeFrame("muted", "A moderator muted you.", until))
	}
}

// ThankReporters tells everyone online whose report led to action that it
// did. People keep reporting, and keep coming back, when they see reports
// work.
func (h *Hub) ThankReporters(ids []uint64) {
	frame := noticeFrame("report_actioned", "A moderator acted on your report. Thank you for flagging it.", time.Time{})
	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, uid := range ids {
		if c := h.clients[uid]; c != nil {
			c.deliver(frame)
		}
	}
}

// RemoveMessage pulls a message out of room history and off every screen
// that may be showing it.
func (h *Hub) RemoveMessage(m message.Message) {
	frame := encode(removedEvent{Type: "removed", MessageID: m.ID})
	h.mu.Lock()
	defer h.mu.Unlock()
	if !m.IsDirect() {
		if r, ok := h.rooms[m.RoomID]; ok {
			r.recent = slices.DeleteFunc(r.recent, func(ev chatEvent) bool { return ev.ID == m.ID })
			for member := range r.members {
				member.deliver(frame)
			}
		}
		return
	}
	for _, uid := range []uint64{m.SenderID, m.RecipientID} {
		if c := h.clients[uid]; c != nil {
			c.deliver(frame)
		}
	}
}

// RemoveMedia removes every room message showing the image and clears it
// from any live profile card.
func (h *Hub) RemoveMedia(mediaID uint64) {
	h.mu.Lock()
	for _, r := range h.rooms {
		var gone []uint64
		r.recent = slices.DeleteFunc(r.recent, func(ev chatEvent) bool {
			if ev.mediaID == mediaID {
				gone = append(gone, ev.ID)
				return true
			}
			return false
		})
		for _, mid := range gone {
			frame := encode(removedEvent{Type: "removed", MessageID: mid})
			for member := range r.members {
				member.deliver(frame)
			}
		}
	}
	var changed []*Client
	for _, c := range h.clients {
		if c.user.PhotoID != nil && *c.user.PhotoID == mediaID {
			c.user.PhotoID, c.user.PhotoKey = nil, nil
			c.refreshCardLocked()
			changed = append(changed, c)
		}
	}
	h.mu.Unlock()

	for _, c := range changed {
		h.publishCard(c)
	}
}

// UpdateUser refreshes a live connection from a freshly read user row,
// after a profile edit or role change.
func (h *Hub) UpdateUser(u user.User) {
	h.mu.Lock()
	c := h.clients[u.ID]
	if c != nil {
		c.user = u
		c.refreshCardLocked()
	}
	h.mu.Unlock()
	if c != nil {
		h.publishCard(c)
	}
}

func (h *Hub) publishCard(c *Client) {
	h.mu.RLock()
	card := c.card
	h.mu.RUnlock()
	frame := presenceFrame("update", c.id(), card)
	h.broadcastPresence(c, frame)
	c.deliver(frame)
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// broadcastPresence sends frame to everyone except about, skipping anyone
// on either side of a block with them.
func (h *Hub) broadcastPresence(about *Client, frame []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, c := range h.clients {
		if c != about && !blockedLocked(c, about) {
			c.deliver(frame)
		}
	}
}

func blockedLocked(a, b *Client) bool {
	_, ab := a.blocked[b.id()]
	_, ba := b.blocked[a.id()]
	return ab || ba
}

func (h *Hub) touch(userID uint64) {
	if err := h.withDB(func(ctx context.Context) error { return h.Users.Touch(ctx, userID) }); err != nil {
		log.Printf("websocket: touch user %d: %v", userID, err)
	}
}

func (h *Hub) withDB(fn func(context.Context) error) error {
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()
	return fn(ctx)
}
