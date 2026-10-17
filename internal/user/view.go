package user

import (
	"time"

	"github.com/v3danth/free-chat/internal/avatar"
	"github.com/v3danth/free-chat/internal/mediapath"
)

// Card is what everyone online sees: the pixel face, never the photo. A
// member's photo is only sent to people they have grown close to;
// HasPhoto lets the other side know there is one to look forward to.
type Card struct {
	ID   uint64 `json:"id"`
	Kind Kind   `json:"kind"`
	Role Role   `json:"role"`
	Name string `json:"name"`
	// Gender is empty when the person hides it from strangers.
	Gender      Gender   `json:"gender,omitempty"`
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
	Aura       string `json:"aura,omitempty"`
}

// Revealed is a Card plus the sharp photo.
type Revealed struct {
	Card
	Photo string `json:"photo_url,omitempty"`
}

// Self is what a user sees about themselves, settings included.
type Self struct {
	Revealed
	Email      *string     `json:"email,omitempty"`
	KnocksFrom KnockPolicy `json:"knocks_from"`
	HideGender bool        `json:"hide_gender"`
}

func ToCard(u User, onlineSince time.Time) Card {
	c := Card{
		ID:       u.ID,
		Kind:     u.Kind,
		Role:     u.Role,
		Name:     u.Profile.Name,
		Age:      u.Profile.Age,
		Tags:     u.Profile.Tags,
		About:    u.Profile.About,
		Location: u.Profile.Location,
		Country:  u.Country,
		Aura:     u.Settings.Aura,
	}
	if !u.Settings.HideGender {
		c.Gender = u.Profile.Gender
	}
	mark := avatar.For2(u.Profile.Name)
	c.Avatar, c.AvatarTier = avatar.URLV2(u.Profile.Name), mark.Rank
	c.HasPhoto = u.PhotoKey != nil
	if !onlineSince.IsZero() {
		c.OnlineSince = onlineSince.Unix()
	}
	return c
}

// ToKnown is the card someone sees once a private chat opens: a person who
// hides their gender from strangers shows it to people they talk to.
func ToKnown(u User, onlineSince time.Time) Revealed {
	c := ToCard(u, onlineSince)
	c.Gender = u.Profile.Gender
	return Revealed{Card: c}
}

// ToRevealed adds the photo, for people who have both chosen to get closer.
func ToRevealed(u User, onlineSince time.Time) Revealed {
	r := ToKnown(u, onlineSince)
	if u.PhotoKey != nil {
		r.Photo = mediapath.URL(mediapath.Full, *u.PhotoKey)
	}
	return r
}

func ToSelf(u User) Self {
	return Self{Revealed: ToRevealed(u, time.Time{}), Email: u.Email,
		KnocksFrom: u.Settings.KnocksFrom, HideGender: u.Settings.HideGender}
}
