package user

import (
	"regexp"
	"slices"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/v3danth/free-chat/internal/apperr"
)

type Kind string

const (
	KindGuest  Kind = "guest"
	KindMember Kind = "member"
)

type Role string

const (
	RoleUser      Role = "user"
	RoleModerator Role = "moderator"
	RoleAdmin     Role = "admin"
)

func (r Role) rank() int {
	switch r {
	case RoleAdmin:
		return 2
	case RoleModerator:
		return 1
	}
	return 0
}

// AtLeast reports whether r has min's privileges.
func (r Role) AtLeast(min Role) bool { return r.rank() >= min.rank() }

// Outranks reports whether r may moderate someone holding other.
func (r Role) Outranks(other Role) bool { return r.rank() > other.rank() }

func ParseRole(s string) (Role, bool) {
	switch r := Role(s); r {
	case RoleUser, RoleModerator, RoleAdmin:
		return r, true
	}
	return "", false
}

type Gender string

const (
	GenderMale      Gender = "male"
	GenderFemale    Gender = "female"
	GenderNonBinary Gender = "non-binary"
	GenderFemboy    Gender = "femboy"
	GenderOther     Gender = "other"
	GenderCouple    Gender = "couple"
)

// Profile is the public card. A Profile value only exists once ParseProfile
// has accepted it.
type Profile struct {
	Name     string
	Gender   Gender
	Age      uint8
	Tags     []string // what they are here for, in their own words
	Color    string   // a NameColors key: how their name is shown
	About    string
	Location string
}

type User struct {
	ID      uint64
	Kind    Kind
	Role    Role
	Profile Profile
	// Country is set server-side from GeoIP; "" when unknown.
	Country string
	PhotoID *uint64
	// PhotoKey is the file key of PhotoID while that photo is not removed.
	PhotoKey     *string
	Email        *string
	PasswordHash *string
	TokenVersion uint32
	BannedUntil  *time.Time
	MutedUntil   *time.Time
	IPHash       []byte
	LastSeenAt   time.Time
	CreatedAt    time.Time
}

func (u User) BannedAt(now time.Time) bool { return u.BannedUntil != nil && now.Before(*u.BannedUntil) }

// Limits mirror the column sizes in migrations/001_schema.sql.
const (
	minAge      = 18
	maxAge      = 99
	maxName     = 32
	maxAbout    = 140
	maxLocation = 40
	maxTags     = 3
	maxTag      = 20
)

// Names allow any script's letters and combining marks (Devanagari matras
// are marks, not letters), digits, underscores and single inner spaces.
var namePattern = regexp.MustCompile(`^[\p{L}\p{M}\p{N}_]+( [\p{L}\p{M}\p{N}_]+)*$`)

// Locations also allow common punctuation: "Delhi NCR", "St. John's, NL".
var locationPattern = regexp.MustCompile(`^[\p{L}\p{M}\p{N} .,'()-]*$`)

// reserved stops anyone but staff from looking like staff.
var reserved = []string{"admin", "moderator", "official", "staff", "support", "system"}

var (
	errName     = apperr.New(apperr.Invalid, "name must be 2-32 letters or numbers")
	errReserved = apperr.New(apperr.Invalid, "that name is reserved")
	errGender   = apperr.New(apperr.Invalid, "gender must be one of male, female, non-binary, femboy, other, couple")
	errAge      = apperr.New(apperr.Invalid, "you must be 18 or older")
	errTags     = apperr.New(apperr.Invalid, "add up to 3 tags of 1-20 letters or numbers each")
	errColor    = apperr.New(apperr.Invalid, "pick one of the name colours")
	errAbout    = apperr.New(apperr.Invalid, "about must be at most 140 characters")
	errLocation = apperr.New(apperr.Invalid, "location must be at most 40 letters")
)

// ParseName validates a display name; collapses runs of spaces first.
func ParseName(raw string) (string, error) {
	name := strings.Join(strings.Fields(raw), " ")
	if n := utf8.RuneCountInString(name); n < 2 || n > maxName || !namePattern.MatchString(name) {
		return "", errName
	}
	folded := strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) {
			return unicode.ToLower(r)
		}
		return -1
	}, name)
	for _, word := range reserved {
		if strings.Contains(folded, word) {
			return "", errReserved
		}
	}
	return name, nil
}

func ParseGender(s string) (Gender, error) {
	switch g := Gender(s); g {
	case GenderMale, GenderFemale, GenderNonBinary, GenderFemboy, GenderOther, GenderCouple:
		return g, nil
	}
	return "", errGender
}

// NameColors are the colours a name can be shown in. The UI defines the
// matching hex values (web/style.css); each is at least 8:1 on the dark
// background, so any choice stays readable.
var NameColors = []string{"sky", "rose", "lavender", "orchid", "mint", "lime", "lemon", "peach", "coral", "ice", "stone"}

// DefaultColor is used when no colour is picked.
const DefaultColor = "stone"

// ParseColor accepts a NameColors key; empty means the default.
func ParseColor(s string) (string, error) {
	if s == "" {
		return DefaultColor, nil
	}
	if !slices.Contains(NameColors, s) {
		return "", errColor
	}
	return s, nil
}

var tagPattern = regexp.MustCompile(`^[\p{L}\p{M}\p{N}_ -]+$`)

// ParseTags cleans free-form "here to" tags: spaces collapsed, empty ones
// dropped, duplicates (ignoring case) removed, at most maxTags kept.
func ParseTags(raw []string) ([]string, error) {
	tags := []string{}
	seen := map[string]bool{}
	for _, r := range raw {
		tag := strings.Join(strings.Fields(r), " ")
		if tag == "" || seen[strings.ToLower(tag)] {
			continue
		}
		if utf8.RuneCountInString(tag) > maxTag || !tagPattern.MatchString(tag) || len(tags) == maxTags {
			return nil, errTags
		}
		seen[strings.ToLower(tag)] = true
		tags = append(tags, tag)
	}
	return tags, nil
}

func ParseAbout(s string) (string, error) {
	about := strings.TrimSpace(s)
	if utf8.RuneCountInString(about) > maxAbout {
		return "", errAbout
	}
	return about, nil
}

func ParseLocation(s string) (string, error) {
	loc := strings.Join(strings.Fields(s), " ")
	if utf8.RuneCountInString(loc) > maxLocation || !locationPattern.MatchString(loc) {
		return "", errLocation
	}
	return loc, nil
}

// Screen checks one piece of user text against the word filter: the
// cleaned text, whether anything was masked, and whether it must be rejected.
type Screen func(string) (cleaned string, masked, blocked bool)

var (
	errNameWords = apperr.New(apperr.Invalid, "that name is not allowed")
	errTextWords = apperr.New(apperr.Invalid, "your profile contains words that are not allowed")
	errTagWords  = apperr.New(apperr.Invalid, "a tag contains words that are not allowed")
)

// Screened applies the word filter to everything others will read. A name
// or tag is rejected outright if it trips the filter; about and location
// keep the masked text unless a word is blocked.
func (p Profile) Screened(screen Screen) (Profile, error) {
	if _, masked, blocked := screen(p.Name); masked || blocked {
		return Profile{}, errNameWords
	}
	for _, tag := range p.Tags {
		if _, masked, blocked := screen(tag); masked || blocked {
			return Profile{}, errTagWords
		}
	}
	about, _, aboutBlocked := screen(p.About)
	location, _, locBlocked := screen(p.Location)
	if aboutBlocked || locBlocked {
		return Profile{}, errTextWords
	}
	p.About, p.Location = about, location
	return p, nil
}

// ProfileInput is the untrusted shape every boundary decodes into.
type ProfileInput struct {
	Name     string   `json:"name"`
	Gender   string   `json:"gender"`
	Age      int      `json:"age"`
	Tags     []string `json:"tags"`
	Color    string   `json:"color"`
	About    string   `json:"about"`
	Location string   `json:"location"`
}

func (in ProfileInput) Parse() (Profile, error) {
	name, err := ParseName(in.Name)
	if err != nil {
		return Profile{}, err
	}
	gender, err := ParseGender(in.Gender)
	if err != nil {
		return Profile{}, err
	}
	if in.Age < minAge || in.Age > maxAge {
		return Profile{}, errAge
	}
	tags, err := ParseTags(in.Tags)
	if err != nil {
		return Profile{}, err
	}
	color, err := ParseColor(in.Color)
	if err != nil {
		return Profile{}, err
	}
	about, err := ParseAbout(in.About)
	if err != nil {
		return Profile{}, err
	}
	location, err := ParseLocation(in.Location)
	if err != nil {
		return Profile{}, err
	}
	return Profile{Name: name, Gender: gender, Age: uint8(in.Age), Tags: tags, Color: color, About: about, Location: location}, nil
}
