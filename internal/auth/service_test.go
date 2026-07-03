package auth_test

import (
	"context"
	"testing"

	"github.com/v3danth/free-chat/internal/auth"
	"github.com/v3danth/free-chat/internal/user"
)

type fakeUserRepo struct {
	users  map[string]*user.User
	byID   map[uint64]*user.User
	nextID uint64
}

func newFakeRepo() *fakeUserRepo {
	return &fakeUserRepo{
		users:  make(map[string]*user.User),
		byID:   make(map[uint64]*user.User),
		nextID: 1,
	}
}

func (f *fakeUserRepo) Create(ctx context.Context, u *user.User) error {
	u.ID = f.nextID
	f.nextID++
	u.Status = user.StatusActive
	f.users[u.Username] = u
	f.byID[u.ID] = u
	return nil
}

func (f *fakeUserRepo) GetByUsername(ctx context.Context, username string) (*user.User, error) {
	return f.users[username], nil
}

func (f *fakeUserRepo) GetByEmail(ctx context.Context, email string) (*user.User, error) {
	for _, u := range f.users {
		if u.Email != nil && *u.Email == email {
			return u, nil
		}
	}
	return nil, nil
}

func (f *fakeUserRepo) GetByID(ctx context.Context, id uint64) (*user.User, error) {
	return f.byID[id], nil
}

func (f *fakeUserRepo) IsGuestUsernameAvailable(ctx context.Context, username string) (bool, error) {
	for _, u := range f.users {
		if u.Username == username && u.Status == user.StatusActive {
			if u.UserType == user.TypeGuest {
				return false, nil
			}
			if u.UserType == user.TypeRegistered {
				return false, nil
			}
		}
	}
	return true, nil
}

func (f *fakeUserRepo) SetInactive(ctx context.Context, id uint64) error {
	if u, ok := f.byID[id]; ok {
		u.Status = user.StatusInactive
	}
	return nil
}

func (f *fakeUserRepo) DeleteInactiveGuests(ctx context.Context, _ interface{}) (int64, error) {
	// Simplified for testing
	return 0, nil
}

func TestCreateGuestUser(t *testing.T) {
	repo := newFakeRepo()
	svc := auth.NewService(repo, "test-secret")

	token, u, err := svc.CreateGuestUser(
		context.Background(),
		"guest1",
		user.GenderMale,
		20,
		"hello",
	)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if token == "" {
		t.Fatal("expected non-empty token")
	}

	if u.UserType != user.TypeGuest {
		t.Fatal("expected guest user type")
	}

	if u.Username != "guest1" {
		t.Fatalf("username mismatch: got %s, want guest1", u.Username)
	}

	if u.Status != user.StatusActive {
		t.Fatal("expected active status")
	}
}

func TestCreateGuestUserDirect(t *testing.T) {
	repo := newFakeRepo()
	svc := auth.NewService(repo, "test-secret")

	token, u, err := svc.CreateGuestUserDirect(context.Background(), "quickguest")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if token == "" {
		t.Fatal("expected non-empty token")
	}

	if u.UserType != user.TypeGuest {
		t.Fatal("expected guest user type")
	}

	if u.Username != "quickguest" {
		t.Fatalf("username mismatch: got %s", u.Username)
	}
}

func TestDuplicateGuestUsername(t *testing.T) {
	repo := newFakeRepo()
	svc := auth.NewService(repo, "test-secret")

	_, _, err := svc.CreateGuestUserDirect(context.Background(), "sameuser")
	if err != nil {
		t.Fatalf("first creation failed: %v", err)
	}

	_, _, err = svc.CreateGuestUserDirect(context.Background(), "sameuser")
	if err == nil {
		t.Fatal("expected error for duplicate username")
	}
}
