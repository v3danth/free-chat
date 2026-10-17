// Package usertest provides an in-memory user.Repository that enforces the
// same constraints as the MySQL schema, for use in other packages' tests.
package usertest

import (
	"cmp"
	"context"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/v3danth/free-chat/internal/user"
)

type Memory struct {
	mu     sync.Mutex
	byID   map[uint64]user.User
	nextID uint64
}

func NewMemory() *Memory {
	return &Memory{byID: map[uint64]user.User{}, nextID: 1}
}

func (m *Memory) Create(_ context.Context, u user.User) (user.User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, existing := range m.byID {
		if u.Kind == user.KindMember && existing.Kind == user.KindMember &&
			strings.EqualFold(u.Profile.Name, existing.Profile.Name) {
			return user.User{}, user.ErrNameTaken
		}
		if u.Email != nil && existing.Email != nil && *u.Email == *existing.Email {
			return user.User{}, user.ErrEmailTaken
		}
	}
	u.ID, u.CreatedAt, u.LastSeenAt = m.nextID, time.Now(), time.Now()
	if u.Settings.KnocksFrom == "" {
		u.Settings.KnocksFrom = user.KnocksAnyone // the column default
	}
	m.nextID++
	m.byID[u.ID] = u
	return u, nil
}

func (m *Memory) GetByID(_ context.Context, id uint64) (user.User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if u, ok := m.byID[id]; ok {
		return u, nil
	}
	return user.User{}, user.ErrNotFound
}

func (m *Memory) GetByEmail(_ context.Context, email string) (user.User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, u := range m.byID {
		if u.Email != nil && *u.Email == email {
			return u, nil
		}
	}
	return user.User{}, user.ErrNotFound
}

func (m *Memory) UpdateCard(_ context.Context, id uint64, tags []string, about, location string, photoID *uint64) error {
	return m.update(id, func(u *user.User) {
		u.Profile.Tags, u.Profile.About, u.Profile.Location, u.PhotoID = tags, about, location, photoID
	})
}

func (m *Memory) UpdateSettings(_ context.Context, id uint64, s user.Settings) error {
	return m.update(id, func(u *user.User) { u.Settings = s })
}

func (m *Memory) Touch(_ context.Context, id uint64) error {
	return m.update(id, func(u *user.User) { u.LastSeenAt = time.Now() })
}

func (m *Memory) SetRole(_ context.Context, id uint64, role user.Role) error {
	return m.update(id, func(u *user.User) { u.Role = role })
}

func (m *Memory) SetBan(_ context.Context, id uint64, until *time.Time) error {
	return m.update(id, func(u *user.User) {
		u.BannedUntil = until
		if until != nil {
			u.TokenVersion++
		}
	})
}

func (m *Memory) RevokeGuestsNamed(_ context.Context, name string, keep uint64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for id, u := range m.byID {
		if isGuestNamed(u, name) && id != keep {
			u.TokenVersion++
			m.byID[id] = u
		}
	}
	return nil
}

func (m *Memory) IsGuestNameClaimedSince(_ context.Context, name string, id uint64) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for other, u := range m.byID {
		if isGuestNamed(u, name) && other > id {
			return true, nil
		}
	}
	return false, nil
}

// isGuestNamed matches the users table collation, which ignores case.
func isGuestNamed(u user.User, name string) bool {
	return u.Kind == user.KindGuest && strings.EqualFold(u.Profile.Name, name)
}

func (m *Memory) SetMute(_ context.Context, id uint64, until *time.Time) error {
	return m.update(id, func(u *user.User) { u.MutedUntil = until })
}

// Delete simulates retention removing an account.
func (m *Memory) ListStaff(context.Context) ([]user.User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var staff []user.User
	for _, u := range m.byID {
		if u.Role.AtLeast(user.RoleModerator) {
			staff = append(staff, u)
		}
	}
	slices.SortFunc(staff, func(a, b user.User) int {
		if (a.Role == user.RoleAdmin) != (b.Role == user.RoleAdmin) {
			if a.Role == user.RoleAdmin {
				return -1
			}
			return 1
		}
		return cmp.Compare(a.ID, b.ID)
	})
	return staff, nil
}

func (m *Memory) Delete(id uint64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.byID, id)
}

func (m *Memory) update(id uint64, fn func(*user.User)) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	u, ok := m.byID[id]
	if !ok {
		return user.ErrNotFound
	}
	fn(&u)
	m.byID[id] = u
	return nil
}
