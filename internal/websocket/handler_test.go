package websocket

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/v3danth/free-chat/internal/user"
)

func TestCardUpdateApply(t *testing.T) {
	photo, key := uint64(5), "key"
	current := user.User{
		Kind:     user.KindMember,
		Profile:  user.Profile{Tags: []string{"night owl"}, About: "cannot sleep", Location: "Pune"},
		PhotoID:  &photo,
		PhotoKey: &key,
	}
	removed := current
	removed.PhotoKey = nil
	guest := current
	guest.Kind = user.KindGuest

	tests := []struct {
		name      string
		body      string
		from      user.User
		wantP     user.Profile
		wantPhoto *uint64
		wantErr   bool
	}{
		{"photo only keeps the card", `{"photo_id": 9}`, current,
			user.Profile{Tags: []string{"night owl"}, About: "cannot sleep", Location: "Pune"}, ptr(9), false},
		{"tags only keeps the photo", `{"tags": ["flirt", "music"]}`, current,
			user.Profile{Tags: []string{"flirt", "music"}, About: "cannot sleep", Location: "Pune"}, ptr(5), false},
		{"null clears the photo", `{"photo_id": null}`, current, current.Profile, nil, false},
		{"empty strings clear text", `{"about": "", "location": ""}`, current,
			user.Profile{Tags: []string{"night owl"}}, ptr(5), false},
		{"a removed photo is dropped", `{}`, removed, current.Profile, nil, false},
		{"bad photo id", `{"photo_id": "x"}`, current, user.Profile{}, nil, true},
		{"colour only keeps the rest", `{"color": "coral"}`, current,
			user.Profile{Tags: []string{"night owl"}, Color: "coral", About: "cannot sleep", Location: "Pune"}, ptr(5), false},
		{"bad colour", `{"color": "black"}`, current, user.Profile{}, nil, true},
		{"bad tags", `{"tags": ["a", "b", "c", "d"]}`, current, user.Profile{}, nil, true},
		{"guests cannot set a photo", `{"photo_id": 9}`, guest, user.Profile{}, nil, true},
		{"guests can clear one", `{"photo_id": null}`, guest, guest.Profile, nil, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req cardUpdate
			if err := json.Unmarshal([]byte(tt.body), &req); err != nil {
				t.Fatal(err)
			}
			p, photo, err := req.apply(tt.from)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if !reflect.DeepEqual(p, tt.wantP) {
				t.Errorf("profile = %+v, want %+v", p, tt.wantP)
			}
			if (photo == nil) != (tt.wantPhoto == nil) || (photo != nil && *photo != *tt.wantPhoto) {
				t.Errorf("photo = %v, want %v", photo, tt.wantPhoto)
			}
		})
	}
}

func ptr(v uint64) *uint64 { return &v }
