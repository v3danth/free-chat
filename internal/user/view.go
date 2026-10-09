package user

import (
	"time"

	"github.com/v3danth/free-chat/internal/avatar"
	"github.com/v3danth/free-chat/internal/mediapath"
)

// Card is what everyone online sees: the pixel face, never the photo. A
// member's photo is only sent to people they have opened a door with;
// HasPhoto lets the other side know there is one to look forward to.
type Card struct {
	ID          uint64   `json:"id"`
	Kind        Kind     `json:"kind"`
	Role        Role     `json:"role"`
	Name        string   `json:"name"`
	Gender      Gender   `json:"gender"`
	Age         uint8    `json:"age"`
	Tags        []string `json:"tags"`
	About       string   `json:"about,omitempty"`
	Location    string   `json:"location,omitempty"`
	Country     string   `json:"country,omitempty"`
	HasPhoto    bool     `json:"has_photo,omitempty"`
	OnlineSince int64    `json:"online_since,omitempty"`
	// Avatar is the pixel face drawn from the name; everyone has one.
	Avatar     string `json:"avatar_url"`
	AvatarTier string `json:"avatar_tier"`
}

// Revealed is a Card plus the sharp photo.
type Revealed struct {
	Card
	Photo string `json:"photo_url,omitempty"`
}

// Self is what a user sees about themselves.
type Self struct {
	Revealed
	Email *string `json:"email,omitempty"`
}

func ToCard(u User, onlineSince time.Time) Card {
	c := Card{
		ID:       u.ID,
		Kind:     u.Kind,
		Role:     u.Role,
		Name:     u.Profile.Name,
		Gender:   u.Profile.Gender,
		Age:      u.Profile.Age,
		Tags:     u.Profile.Tags,
		About:    u.Profile.About,
		Location: u.Profile.Location,
		Country:  u.Country,
	}
	face := avatar.For(u.Profile.Name)
	c.Avatar, c.AvatarTier = avatar.URL(u.Profile.Name), face.Tier
	c.HasPhoto = u.PhotoKey != nil
	if !onlineSince.IsZero() {
		c.OnlineSince = onlineSince.Unix()
	}
	return c
}

func ToRevealed(u User, onlineSince time.Time) Revealed {
	r := Revealed{Card: ToCard(u, onlineSince)}
	if u.PhotoKey != nil {
		r.Photo = mediapath.URL(mediapath.Full, *u.PhotoKey)
	}
	return r
}

func ToSelf(u User) Self {
	return Self{Revealed: ToRevealed(u, time.Time{}), Email: u.Email}
}
