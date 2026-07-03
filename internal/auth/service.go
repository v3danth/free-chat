package auth

import (
	"context"
	"errors"

	"golang.org/x/crypto/bcrypt"

	"github.com/v3danth/free-chat/internal/user"
)

type Service struct {
	userRepo user.Repository
	jwt      *JWTManager
}

func NewService(repo user.Repository, jwtSecret string) *Service {
	return &Service{
		userRepo: repo,
		jwt:      NewJWTManager(jwtSecret),
	}
}

func (s *Service) CreateGuestUser(
	ctx context.Context,
	username string,
	gender user.Gender,
	age uint8,
	about string,
) (string, *user.User, error) {
	available, err := s.userRepo.IsGuestUsernameAvailable(ctx, username)
	if err != nil {
		return "", nil, err
	}
	if !available {
		return "", nil, errors.New("username already taken")
	}

	u := &user.User{
		UserType: user.TypeGuest,
		Status:   user.StatusActive,
		Username: username,
		Gender:   gender,
		Age:      age,
		About:    about,
	}

	if err := s.userRepo.Create(ctx, u); err != nil {
		return "", nil, err
	}

	token, err := s.jwt.Generate(u.ID, u.UserType, u.Username)
	if err != nil {
		return "", nil, err
	}

	return token, u, nil
}

func (s *Service) CreateGuestUserDirect(
	ctx context.Context,
	username string,
) (string, *user.User, error) {
	available, err := s.userRepo.IsGuestUsernameAvailable(ctx, username)
	if err != nil {
		return "", nil, err
	}
	if !available {
		return "", nil, errors.New("username already taken")
	}

	u := &user.User{
		UserType: user.TypeGuest,
		Status:   user.StatusActive,
		Username: username,
		Gender:   user.GenderOther,
		Age:      0,
		About:    "",
	}

	if err := s.userRepo.Create(ctx, u); err != nil {
		return "", nil, err
	}

	token, err := s.jwt.Generate(u.ID, u.UserType, u.Username)
	if err != nil {
		return "", nil, err
	}

	return token, u, nil
}

func (s *Service) RegisterUser(
	ctx context.Context,
	username string,
	email string,
	password string,
	gender user.Gender,
	age uint8,
	about string,
) (*user.User, error) {
	existing, _ := s.userRepo.GetByUsername(ctx, username)
	if existing != nil {
		return nil, errors.New("username already taken")
	}

	existingEmail, _ := s.userRepo.GetByEmail(ctx, email)
	if existingEmail != nil {
		return nil, errors.New("email already registered")
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	u := &user.User{
		UserType:     user.TypeRegistered,
		Status:       user.StatusActive,
		Username:     username,
		Gender:       gender,
		Age:          age,
		About:        about,
		Email:        &email,
		PasswordHash: ptr(string(hashed)),
	}

	if err := s.userRepo.Create(ctx, u); err != nil {
		return nil, err
	}

	return u, nil
}

func (s *Service) Login(ctx context.Context, email, password string) (string, *user.User, error) {
	u, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil || u == nil {
		return "", nil, errors.New("invalid credentials")
	}

	if u.PasswordHash == nil {
		return "", nil, errors.New("invalid credentials")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(*u.PasswordHash), []byte(password)); err != nil {
		return "", nil, errors.New("invalid credentials")
	}

	token, err := s.jwt.Generate(u.ID, u.UserType, u.Username)
	if err != nil {
		return "", nil, err
	}

	return token, u, nil
}

func ptr(s string) *string {
	return &s
}
