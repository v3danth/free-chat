package auth

import (
	"context"
	"errors"

	"golang.org/x/crypto/bcrypt"

	"github.com/v3danth/free-chat/internal/user"
)

type Service struct {
	userRepo user.Repository
	jwt      *jwt.JWTManager
}

func NewService(repo user.Repository) *Service {
	return &Service{
		userRepo: repo,
		jwt:      NewJWTManager(jwtSecret),
	}
}

// CreateGuestUser
func (s *Service) CreateGuestUser(
	ctx context.Context,
	username string,
	gender user.Gender,
	age uint8,
	about string,
) (*user.User, error) {

	// check if username exists
	existing, _ := s.userRepo.GetByUsername(ctx, username)
	if existing != nil {
		return nil, errors.New("username already taken")
	}

	u := &user.User{
		UserType: user.TypeGuest,
		Username: username,
		Gender:   gender,
		Age:      age,
		About:    about,
	}

	err := s.userRepo.Create(ctx, u)
	if err != nil {
		return nil, err
	}

	return u, nil
}

// RegisterUser
func (s *Service) RegisterUser(
	ctx context.Context,
	username string,
	email string,
	password string,
	gender user.Gender,
	age uint8,
	about string,
) (*user.User, error) {

	// check username
	existing, _ := s.userRepo.GetByUsername(ctx, username)
	if existing != nil {
		return nil, errors.New("username already taken")
	}

	// check email
	existingEmail, _ := s.userRepo.GetByEmail(ctx, email)
	if existingEmail != nil {
		return nil, errors.New("email already registered")
	}

	// hash password
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	u := &user.User{
		UserType:     user.TypeRegistered,
		Username:     username,
		Gender:       gender,
		Age:          age,
		About:        about,
		Email:        &email,
		PasswordHash: ptr(string(hashed)),
	}

	err = s.userRepo.Create(ctx, u)
	if err != nil {
		return nil, err
	}

	return u, nil
}

// Login User
func (s *Service) Login(ctx context.Context, email, password string) (string, *user.User, error) {

	u, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil || u == nil {
		return "", nil, errors.New("invalid credentials")
	}

	if u.PasswordHash == nil {
		return "", nil, errors.New("invalid credentials")
	}

	err = bcrypt.CompareHashAndPassword(
		[]byte(*u.PasswordHash),
		[]byte(password),
	)

	if err != nil {
		return "", nil, errors.New("invalid credentials")
	}

	token, err := s.jwt.Generate(u.ID)
	if err != nil {
		return "", nil, err
	}

	return token, u, nil
}

// Helper function
func ptr(s string) *string {
	return &s
}
