package user

import "time"

type UserType string

const (
	TypeGuest      UserType = "guest"
	TypeRegistered UserType = "registered"
)

type UserStatus string

const (
	StatusActive   UserStatus = "active"
	StatusInactive UserStatus = "inactive"
)

type Gender string

const (
	GenderMale      Gender = "male"
	GenderFemale    Gender = "female"
	GenderNonBinary Gender = "non-binary"
	GenderFemboy    Gender = "femboy"
	GenderOther     Gender = "other"
	GenderCouple    Gender = "couple"
)

type User struct {
	ID           uint64
	UserType     UserType
	Status       UserStatus
	Username     string
	Gender       Gender
	Age          uint8
	About        string
	Email        *string
	PasswordHash *string
	LastSeenAt   *time.Time
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func (u *User) IsGuest() bool {
	return u.UserType == TypeGuest
}

func (u *User) IsActive() bool {
	return u.Status == StatusActive || u.Status == ""
}
