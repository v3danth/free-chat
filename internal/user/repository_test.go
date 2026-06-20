package user_test

import (
	"context"
	"testing"

	"github.com/v3danth/free-chat/internal/config"
	"github.com/v3danth/free-chat/internal/database"
	"github.com/v3danth/free-chat/internal/user"
)

func setupRepo(t *testing.T) user.Repository {
	cfg := config.LoadTestConfig()

	// override DB name for tests
	cfg.MySQLDatabase = "chat_db_test"

	db, err := database.New(cfg)
	if err != nil {
		t.Fatal(err)
	}

	return user.NewRepository(db)
}

func TestCreateAndGetUser(t *testing.T) {
	repo := setupRepo(t)
	ctx := context.Background()

	u := &user.User{
		UserType: user.TypeGuest,
		Username: "testuser",
		Gender:   user.GenderMale,
		Age:      25,
		About:    "test",
	}

	err := repo.Create(ctx, u)
	if err != nil {
		t.Fatal(err)
	}

	if u.ID == 0 {
		t.Fatal("expected user id to be set")
	}

	fetched, err := repo.GetByID(ctx, u.ID)
	if err != nil {
		t.Fatal(err)
	}

	if fetched.Username != "testuser" {
		t.Fatal("username mismatch")
	}
}
