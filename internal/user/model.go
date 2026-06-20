package user

import "time"

type Type string

const (
	TypeGuest      Type = "guest"
	TypeRegistered Type = "registered"
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
	ID uint64

	UserType Type

	Username string

	Gender Gender

	Age uint8

	About string

	Email *string

	PasswordHash *string

	CreatedAt time.Time
	UpdatedAt time.Time
}
