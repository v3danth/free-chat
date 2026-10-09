package websocket

import (
	"context"
	"log"
	"math"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/v3danth/free-chat/internal/apperr"
	"github.com/v3danth/free-chat/internal/filter"
	"github.com/v3danth/free-chat/internal/id"
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

type Deps struct {
	Writer       Writer
	IDs          *id.Generator
	Media        Attacher
	Users        Presence
	Blocks       Blocks
	Limiter      *ratelimit.Limiter
	Filter       *filter.Live
	HistoryLimit int
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
}

// NewHub starts with the live rooms and their recent messages (newest
// first, as the repository returns them), so joining never reads MySQL.
func NewHub(d Deps, rooms []message.Room, recent map[uint64][]message.Entry) *Hub {
	h := &Hub{
		Deps:    d,
		clients: make(map[uint64]*Client),
		rooms:   make(map[uint64]*room),
		doors:   make(map[pair]*door),
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

	online := make([]user.Card, 0, len(h.clients)-1)
	for _, other := range h.clients {
		if other != c && !blockedLocked(c, other) {
			online = append(online, user.ToCard(other.user, other.since))
		}
	}
	rooms := make([]roomView, 0, len(h.order))
	for _, rid := range h.order {
		rooms = append(rooms, roomView{ID: rid, Name: h.rooms[rid].name})
	}
	doors := []doorView{}
	for p, d := range h.doors {
		if other, ok := p.other(c.id()); ok {
			doors = append(doors, doorView{With: other, Open: d.open, KnockedByMe: !d.open && d.knocker == c.id()})
		}
	}
	card := user.ToCard(c.user, c.since)
	// hello goes out before the lock is released, so nothing sent to this
	// client (a DM, someone else's join) can arrive ahead of it.
	c.deliver(encode(helloEvent{
		Type:   "hello",
		You:    user.ToSelf(c.user),
		Online: online,
		Rooms:  rooms,
		RateLimit: rateLimitView{
			Remaining: h.Limiter.Remaining(c.id()),
			ResetIn:   int64(math.Ceil(h.Limiter.ResetIn(c.id()).Seconds())),
		},
		Doors: doors,
	}))
	h.mu.Unlock()

	h.broadcastPresence(c, presenceFrame("join", c.id(), &card))
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

// DoorOpen reports whether a and b have an open private chat right now.
func (h *Hub) DoorOpen(a, b uint64) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	d, ok := h.doors[pairOf(a, b)]
	return ok && d.open
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

	out, err := h.prepare(c, cmd.content, cmd.mediaID)
	if err != nil {
		return err
	}
	m := message.NewRoom(h.IDs.Next(), cmd.room, c.id(), out.name, out.body, out.att.ID)
	ev := roomEvent(m, out.color, out.att, out.filtered)
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
	unavailable := !online || blockedLocked(c, peer)
	_, _, doorErr := h.knockLocked(c.id(), cmd.to, cmd.mediaID != 0, false)
	h.mu.RUnlock()
	if unavailable {
		return errUnavailable
	}
	if doorErr != nil {
		return doorErr
	}

	out, err := h.prepare(c, cmd.content, cmd.mediaID)
	if err != nil {
		return err
	}
	m := message.NewDirect(h.IDs.Next(), cmd.to, c.id(), out.name, out.body, out.att.ID)

	h.mu.Lock()
	peer, online = h.clients[cmd.to]
	if !online || blockedLocked(c, peer) {
		h.mu.Unlock()
		return errUnavailable
	}
	knock, opened, err := h.knockLocked(c.id(), cmd.to, cmd.mediaID != 0, true)
	if err != nil {
		h.mu.Unlock()
		return err
	}
	frame := encode(directEvent(m, out.att, knock, out.filtered))
	peer.deliver(frame)
	c.deliver(frame) // echo: the sender learns the id and the filtered text
	if opened {
		c.deliver(encode(doorEvent{Type: "door", With: user.ToRevealed(peer.user, peer.since)}))
		peer.deliver(encode(doorEvent{Type: "door", With: user.ToRevealed(c.user, c.since)}))
	}
	h.mu.Unlock()

	h.Writer.Enqueue(m)
	return nil
}

// knockLocked decides what a private message from -> to means. Without
// commit it only checks; with commit it records the transition. Knocks are
// text only, so nobody receives an unsolicited photo.
func (h *Hub) knockLocked(from, to uint64, hasImage, commit bool) (knock, opened bool, err error) {
	k := pairOf(from, to)
	d, ok := h.doors[k]
	switch {
	case !ok:
		if hasImage {
			return false, false, errKnockImage
		}
		if commit {
			h.doors[k] = &door{knocker: from}
		}
		return true, false, nil
	case d.open:
		return false, false, nil
	case d.knocker == from:
		return false, false, errWaitReply
	default:
		if commit {
			d.open = true
		}
		return false, true, nil
	}
}

type prepared struct {
	name     string
	color    string
	body     string
	filtered bool
	att      media.Attachment
}

// prepare runs every check a message needs before it can be sent: mute,
// rate limit, word filter, and image ownership.
func (h *Hub) prepare(c *Client, content string, mediaID uint64) (prepared, error) {
	h.mu.RLock()
	name, color, muted := c.user.Profile.Name, c.user.Profile.Color, c.user.MutedUntil
	h.mu.RUnlock()

	if muted != nil && time.Now().Before(*muted) {
		return prepared{}, errMuted
	}
	if !h.Limiter.Allow(c.id()) {
		return prepared{}, errRateLimited
	}
	res := h.Filter.Load().Apply(content)
	if res.Blocked {
		return prepared{}, errBlockedWords
	}
	out := prepared{name: name, color: color, body: res.Content, filtered: res.Filtered}
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
	var cCard, pCard user.Card
	if peer != nil {
		peer.blocked = theirs
		visible = !blockedLocked(c, peer)
		cCard, pCard = user.ToCard(c.user, c.since), user.ToCard(peer.user, peer.since)
	}
	h.mu.Unlock()

	if visible {
		c.deliver(presenceFrame("join", target, &pCard))
		peer.deliver(presenceFrame("join", c.id(), &cCard))
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
	}
	h.mu.Unlock()
	if c != nil {
		h.publishCard(c)
	}
}

func (h *Hub) publishCard(c *Client) {
	h.mu.RLock()
	card := user.ToCard(c.user, c.since)
	h.mu.RUnlock()
	h.broadcastPresence(c, presenceFrame("update", c.id(), &card))
	c.deliver(presenceFrame("update", c.id(), &card))
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
