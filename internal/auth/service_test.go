package auth_test

import (
	"context"
	"testing"

	"github.com/v3danth/free-chat/internal/auth"
	"github.com/v3danth/free-chat/internal/user"
)

type fakeUserRepo struct {
	users map[string]*user.User
}

func newFakeRepo() *fakeUserRepo {
	return &fakeUserRepo{
		users: make(map[string]*user.User),
	}
}

func (f *fakeUserRepo) Create(ctx context.Context, u *user.User) error {
	u.ID = uint64(len(f.users) + 1)
	f.users[u.Username] = u
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
	for _, u := range f.users {
		if u.ID == id {
			return u, nil
		}
	}
	return nil, nil
}

func TestCreateGuestUser(t *testing.T) {
	repo := newFakeRepo()
	svc := auth.NewService(repo)

	u, err := svc.CreateGuestUser(
		context.Background(),
		"guest1",
		user.GenderMale,
		20,
		"hello",
	)

	if err != nil {
		t.Fatal(err)
	}

	if u.UserType != user.TypeGuest {
		t.Fatal("expected guest user")
	}

	if u.Username != "guest1" {
		t.Fatal("username mismatch")
	}
}
