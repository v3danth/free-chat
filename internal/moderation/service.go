package moderation

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/v3danth/free-chat/internal/auth"
	"github.com/v3danth/free-chat/internal/filter"
	"github.com/v3danth/free-chat/internal/media"
	"github.com/v3danth/free-chat/internal/mediapath"
	"github.com/v3danth/free-chat/internal/message"
	"github.com/v3danth/free-chat/internal/ratelimit"
	"github.com/v3danth/free-chat/internal/user"
)

const (
	contextMessages = 10
	// IP bans are capped: Indian mobile carriers share one IP across many
	// people (CGNAT), so a long IP ban punishes strangers.
	maxIPBan = 7 * 24 * time.Hour
	maxNote  = 500
	maxWord  = 64
)

// forever is the largest DATETIME MySQL stores, used for permanent bans.
var forever = time.Date(9999, 12, 31, 0, 0, 0, 0, time.UTC)

type Store interface {
	ListWords(ctx context.Context) ([]Word, error)
	AddWord(ctx context.Context, word string, action filter.Action, actor uint64) error
	RemoveWord(ctx context.Context, id uint64) (string, error)
	BanIP(ctx context.Context, ipHash []byte, until time.Time, reason string, actor uint64) error
	CreateReport(ctx context.Context, r Report) error
	CountOpen(ctx context.Context, t TargetType, targetID uint64) (int, error)
	ListReports(ctx context.Context, status Status, limit int) ([]Report, error)
	SetReportStatus(ctx context.Context, id uint64, status Status, actor uint64) error
	ResolveTarget(ctx context.Context, t TargetType, targetID uint64, status Status, actor uint64) error
	Log(ctx context.Context, a Action) error
	ListActions(ctx context.Context, limit int) ([]Action, error)
}

type Users interface {
	GetByID(ctx context.Context, id uint64) (user.User, error)
	SetBan(ctx context.Context, id uint64, until *time.Time) error
	SetMute(ctx context.Context, id uint64, until *time.Time) error
	SetRole(ctx context.Context, id uint64, role user.Role) error
}

type Messages interface {
	Get(ctx context.Context, id uint64) (message.Message, error)
	Before(ctx context.Context, m message.Message, n int) ([]message.Message, error)
	BySender(ctx context.Context, senderID uint64, limit int) ([]message.Message, error)
	Hide(ctx context.Context, id uint64) error
}

type Images interface {
	Get(ctx context.Context, id uint64) (media.Media, error)
	Remove(ctx context.Context, id, actorID uint64, banHash bool) (media.Media, error)
}

// Flusher makes sure queued messages are in MySQL before they are looked up.
type Flusher interface {
	Flush(ctx context.Context) error
}

// Live applies decisions to connected users immediately.
type Live interface {
	Kick(userID uint64, code, text string, until time.Time)
	Mute(userID uint64, until time.Time)
	RemoveMessage(m message.Message)
	RemoveMedia(mediaID uint64)
	UpdateUser(u user.User)
}

type Deps struct {
	Store     Store
	Users     Users
	Messages  Messages
	Writer    Flusher
	Images    Images
	Live      Live
	Filter    *filter.Live
	MaxLength int
	// AutoHide is how many open reports hide a message or image pending review.
	AutoHide int
}

type Service struct {
	Deps
	reports *ratelimit.Limiter
}

func NewService(d Deps) *Service {
	return &Service{Deps: d, reports: ratelimit.New(ratelimit.Config{Rate: 10, Window: 10 * time.Minute})}
}

// ---------------------------------------------------------------------------
// Reporting (any user)
// ---------------------------------------------------------------------------

type ReportInput struct {
	TargetType TargetType
	TargetID   uint64
	Reason     Reason
	Note       string
}

func ParseReport(targetType, reason string, targetID uint64, note string) (ReportInput, error) {
	t := TargetType(targetType)
	if (t != TargetMessage && t != TargetMedia && t != TargetUser) || targetID == 0 || !reasons[Reason(reason)] {
		return ReportInput{}, ErrInvalidReport
	}
	note = strings.TrimSpace(note)
	if utf8.RuneCountInString(note) > maxNote {
		return ReportInput{}, ErrInvalidReport
	}
	return ReportInput{TargetType: t, TargetID: targetID, Reason: Reason(reason), Note: note}, nil
}

func (s *Service) Report(ctx context.Context, reporter auth.Identity, in ReportInput) error {
	if !s.reports.Allow(reporter.UserID) {
		return ErrReportRate
	}
	evidence, targetUser, err := s.evidence(ctx, reporter.UserID, in)
	if err != nil {
		return err
	}
	if targetUser == reporter.UserID {
		return ErrReportSelf
	}
	raw, err := json.Marshal(evidence)
	if err != nil {
		return err
	}
	reporterID := reporter.UserID
	if err := s.Store.CreateReport(ctx, Report{
		ReporterID: &reporterID, TargetType: in.TargetType, TargetID: in.TargetID,
		TargetUserID: &targetUser, Reason: in.Reason, Note: in.Note, Evidence: raw,
	}); err != nil {
		return err
	}
	return s.maybeAutoHide(ctx, in, evidence)
}

// evidence snapshots the target and returns whose it is. A private message
// can only be reported by someone in that conversation.
func (s *Service) evidence(ctx context.Context, reporter uint64, in ReportInput) (Evidence, uint64, error) {
	switch in.TargetType {
	case TargetMessage:
		if err := s.Writer.Flush(ctx); err != nil {
			return Evidence{}, 0, err
		}
		m, err := s.Messages.Get(ctx, in.TargetID)
		if err != nil {
			return Evidence{}, 0, err
		}
		if !m.Involves(reporter) {
			return Evidence{}, 0, ErrNotVisible
		}
		before, err := s.Messages.Before(ctx, m, contextMessages)
		if err != nil {
			return Evidence{}, 0, err
		}
		ev := Evidence{Message: ptr(evidenceMessage(m))}
		for _, b := range before {
			ev.Context = append(ev.Context, evidenceMessage(b))
		}
		return ev, m.SenderID, nil
	case TargetMedia:
		m, err := s.Images.Get(ctx, in.TargetID)
		if err != nil {
			return Evidence{}, 0, err
		}
		return Evidence{Image: &EvidenceImage{ID: m.ID, OwnerID: m.OwnerID, URL: mediapath.URL(mediapath.Full, m.Key)}}, m.OwnerID, nil
	default:
		u, err := s.Users.GetByID(ctx, in.TargetID)
		if err != nil {
			return Evidence{}, 0, err
		}
		return Evidence{User: &EvidenceUser{
			ID: u.ID, Kind: string(u.Kind), Name: u.Profile.Name, Age: u.Profile.Age,
			About: u.Profile.About, Location: u.Profile.Location, Country: u.Country,
		}}, u.ID, nil
	}
}

// maybeAutoHide takes reported content down pending review once enough
// people report it. Users are never auto-actioned; that needs a human.
func (s *Service) maybeAutoHide(ctx context.Context, in ReportInput, ev Evidence) error {
	if in.TargetType == TargetUser {
		return nil
	}
	n, err := s.Store.CountOpen(ctx, in.TargetType, in.TargetID)
	if err != nil || n < s.AutoHide {
		return err
	}
	if in.TargetType == TargetMessage {
		if err := s.Messages.Hide(ctx, in.TargetID); err != nil {
			return err
		}
		s.Live.RemoveMessage(toMessage(*ev.Message))
	} else {
		if _, err := s.Images.Remove(ctx, in.TargetID, 0, false); err != nil {
			return err
		}
		s.Live.RemoveMedia(in.TargetID)
	}
	return s.Store.Log(ctx, Action{Action: ActAutoHide, TargetID: in.TargetID,
		Detail: fmt.Sprintf("%s hidden after %d reports", in.TargetType, n)})
}

// ---------------------------------------------------------------------------
// Moderator actions
// ---------------------------------------------------------------------------

func (s *Service) RemoveMessage(ctx context.Context, actor auth.Identity, id uint64) error {
	if err := s.Writer.Flush(ctx); err != nil {
		return err
	}
	m, err := s.Messages.Get(ctx, id)
	if err != nil {
		return err
	}
	if err := s.Messages.Hide(ctx, id); err != nil {
		return err
	}
	s.Live.RemoveMessage(m)
	return s.closeOut(ctx, actor, Action{Action: ActRemoveMessage, TargetUserID: m.SenderID, TargetName: m.SenderName, TargetID: id},
		TargetMessage, id)
}

func (s *Service) RemoveMedia(ctx context.Context, actor auth.Identity, id uint64) error {
	m, err := s.Images.Remove(ctx, id, actor.UserID, true)
	if err != nil {
		return err
	}
	s.Live.RemoveMedia(id)
	return s.closeOut(ctx, actor, Action{Action: ActRemoveMedia, TargetUserID: m.OwnerID, TargetID: id,
		Detail: "image removed and its hash banned"}, TargetMedia, id)
}

func (s *Service) Kick(ctx context.Context, actor auth.Identity, userID uint64) error {
	u, err := s.target(ctx, actor, userID)
	if err != nil {
		return err
	}
	s.Live.Kick(u.ID, "kicked", "A moderator removed you from the chat.", time.Time{})
	return s.log(ctx, actor, Action{Action: ActKick, TargetUserID: u.ID, TargetName: u.Profile.Name})
}

func (s *Service) Mute(ctx context.Context, actor auth.Identity, userID uint64, d time.Duration) error {
	u, err := s.target(ctx, actor, userID)
	if err != nil {
		return err
	}
	until := time.Now().Add(d)
	if err := s.Users.SetMute(ctx, u.ID, &until); err != nil {
		return err
	}
	s.Live.Mute(u.ID, until)
	return s.log(ctx, actor, Action{Action: ActMute, TargetUserID: u.ID, TargetName: u.Profile.Name, Detail: d.String()})
}

// Ban blocks the account (d == 0 is permanent), revokes its sessions, and
// with ipToo also bans its network for at most maxIPBan.
func (s *Service) Ban(ctx context.Context, actor auth.Identity, userID uint64, d time.Duration, ipToo bool, reason string) error {
	u, err := s.target(ctx, actor, userID)
	if err != nil {
		return err
	}
	until := forever
	if d > 0 {
		until = time.Now().Add(d)
	}
	if err := s.Users.SetBan(ctx, u.ID, &until); err != nil {
		return err
	}
	detail := "until " + until.Format(time.DateTime)
	if ipToo && len(u.IPHash) > 0 {
		ipUntil := time.Now().Add(min(maxIPBan, d))
		if d == 0 {
			ipUntil = time.Now().Add(maxIPBan)
		}
		if err := s.Store.BanIP(ctx, u.IPHash, ipUntil, reason, actor.UserID); err != nil {
			return err
		}
		detail += "; network until " + ipUntil.Format(time.DateTime)
	}
	s.Live.Kick(u.ID, "banned", "You have been banned.", until)
	if reason != "" {
		detail += "; " + reason
	}
	return s.closeOut(ctx, actor, Action{Action: ActBan, TargetUserID: u.ID, TargetName: u.Profile.Name, Detail: detail},
		TargetUser, u.ID)
}

func (s *Service) Unban(ctx context.Context, actor auth.Identity, userID uint64) error {
	u, err := s.target(ctx, actor, userID)
	if err != nil {
		return err
	}
	if err := s.Users.SetBan(ctx, u.ID, nil); err != nil {
		return err
	}
	return s.log(ctx, actor, Action{Action: ActUnban, TargetUserID: u.ID, TargetName: u.Profile.Name})
}

// SetRole is admin-only (enforced by the route). Staff must be members.
func (s *Service) SetRole(ctx context.Context, actor auth.Identity, userID uint64, role user.Role) error {
	u, err := s.target(ctx, actor, userID)
	if err != nil {
		return err
	}
	if u.Kind != user.KindMember {
		return ErrMembersOnly
	}
	if err := s.Users.SetRole(ctx, u.ID, role); err != nil {
		return err
	}
	if fresh, err := s.Users.GetByID(ctx, u.ID); err == nil {
		s.Live.UpdateUser(fresh)
	}
	return s.log(ctx, actor, Action{Action: ActSetRole, TargetUserID: u.ID, TargetName: u.Profile.Name, Detail: string(role)})
}

func (s *Service) Dismiss(ctx context.Context, actor auth.Identity, reportID uint64) error {
	if err := s.Store.SetReportStatus(ctx, reportID, StatusDismissed, actor.UserID); err != nil {
		return err
	}
	return s.log(ctx, actor, Action{Action: ActDismiss, TargetID: reportID})
}

// ---------------------------------------------------------------------------
// Banned words
// ---------------------------------------------------------------------------

func (s *Service) AddWord(ctx context.Context, actor auth.Identity, word string, action filter.Action) error {
	norm := filter.Normalize(word)
	isPhrase := strings.Contains(norm, " ")
	if norm == "" || utf8.RuneCountInString(norm) > maxWord ||
		(action != filter.Mask && action != filter.Block) || (isPhrase && action != filter.Block) {
		return ErrInvalidWord
	}
	if err := s.Store.AddWord(ctx, norm, action, actor.UserID); err != nil {
		return err
	}
	if err := s.ReloadWords(ctx); err != nil {
		return err
	}
	return s.log(ctx, actor, Action{Action: ActAddWord, Detail: string(action) + ": " + norm})
}

func (s *Service) RemoveWord(ctx context.Context, actor auth.Identity, id uint64) error {
	word, err := s.Store.RemoveWord(ctx, id)
	if err != nil {
		return err
	}
	if err := s.ReloadWords(ctx); err != nil {
		return err
	}
	return s.log(ctx, actor, Action{Action: ActRemoveWord, Detail: word})
}

// ReloadWords rebuilds the filter from the database and swaps it in; every
// message filtered after this sees the new list.
func (s *Service) ReloadWords(ctx context.Context) error {
	words, err := s.Store.ListWords(ctx)
	if err != nil {
		return err
	}
	rules := make([]filter.Rule, len(words))
	for i, w := range words {
		rules[i] = filter.Rule{Word: w.Word, Action: w.Action}
	}
	s.Filter.Store(filter.New(s.MaxLength, rules))
	return nil
}

// ---------------------------------------------------------------------------
// Reads
// ---------------------------------------------------------------------------

func (s *Service) Reports(ctx context.Context, status Status, limit int) ([]Report, error) {
	return s.Store.ListReports(ctx, status, limit)
}

func (s *Service) Actions(ctx context.Context, limit int) ([]Action, error) {
	return s.Store.ListActions(ctx, limit)
}

func (s *Service) Words(ctx context.Context) ([]Word, error) { return s.Store.ListWords(ctx) }

func (s *Service) UserMessages(ctx context.Context, userID uint64, limit int) ([]EvidenceMessage, error) {
	if err := s.Writer.Flush(ctx); err != nil {
		return nil, err
	}
	msgs, err := s.Messages.BySender(ctx, userID, limit)
	if err != nil {
		return nil, err
	}
	out := make([]EvidenceMessage, len(msgs))
	for i, m := range msgs {
		out[i] = evidenceMessage(m)
	}
	return out, nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// target loads the user an action is about and enforces the role ladder:
// nobody moderates themselves or anyone of equal or higher rank.
func (s *Service) target(ctx context.Context, actor auth.Identity, userID uint64) (user.User, error) {
	if userID == actor.UserID {
		return user.User{}, ErrSelf
	}
	u, err := s.Users.GetByID(ctx, userID)
	if err != nil {
		return user.User{}, err
	}
	if !actor.Role.Outranks(u.Role) {
		return user.User{}, ErrOutranked
	}
	return u, nil
}

// closeOut marks every open report on the target as actioned and logs.
func (s *Service) closeOut(ctx context.Context, actor auth.Identity, a Action, t TargetType, targetID uint64) error {
	if err := s.Store.ResolveTarget(ctx, t, targetID, StatusActioned, actor.UserID); err != nil {
		return err
	}
	return s.log(ctx, actor, a)
}

func (s *Service) log(ctx context.Context, actor auth.Identity, a Action) error {
	a.ActorID = actor.UserID
	return s.Store.Log(ctx, a)
}

func evidenceMessage(m message.Message) EvidenceMessage {
	return EvidenceMessage{ID: m.ID, RoomID: m.RoomID, RecipientID: m.RecipientID, SenderID: m.SenderID,
		SenderName: m.SenderName, Body: m.Body, MediaID: m.MediaID}
}

func toMessage(e EvidenceMessage) message.Message {
	return message.Message{ID: e.ID, RoomID: e.RoomID, RecipientID: e.RecipientID, SenderID: e.SenderID,
		SenderName: e.SenderName, Body: e.Body, MediaID: e.MediaID}
}

func ptr[T any](v T) *T { return &v }
