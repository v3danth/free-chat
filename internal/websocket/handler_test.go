package websocket

import (
	"encoding/json"
	"testing"

	"github.com/v3danth/free-chat/internal/user"
)

func TestCardUpdateApply(t *testing.T) {
	photo, key := uint64(5), "key"
	current := user.User{
		Profile:  user.Profile{Intent: user.IntentNightOwl, About: "cannot sleep", Location: "Pune"},
		PhotoID:  &photo,
		PhotoKey: &key,
	}
	removed := current
	removed.PhotoKey = nil

	tests := []struct {
		name      string
		body      string
		from      user.User
		wantP     user.Profile
		wantPhoto *uint64
		wantErr   bool
	}{
		{"photo only keeps the card", `{"photo_id": 9}`, current,
			user.Profile{Intent: user.IntentNightOwl, About: "cannot sleep", Location: "Pune"}, ptr(9), false},
		{"intent only keeps the photo", `{"intent": "flirt"}`, current,
			user.Profile{Intent: user.IntentFlirt, About: "cannot sleep", Location: "Pune"}, ptr(5), false},
		{"null clears the photo", `{"photo_id": null}`, current, current.Profile, nil, false},
		{"empty strings clear text", `{"about": "", "location": ""}`, current,
			user.Profile{Intent: user.IntentNightOwl}, ptr(5), false},
		{"a removed photo is dropped", `{}`, removed, current.Profile, nil, false},
		{"bad photo id", `{"photo_id": "x"}`, current, user.Profile{}, nil, true},
		{"bad intent", `{"intent": "party"}`, current, user.Profile{}, nil, true},
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
			if p != tt.wantP {
				t.Errorf("profile = %+v, want %+v", p, tt.wantP)
			}
			if (photo == nil) != (tt.wantPhoto == nil) || (photo != nil && *photo != *tt.wantPhoto) {
				t.Errorf("photo = %v, want %v", photo, tt.wantPhoto)
			}
		})
	}
}

func ptr(v uint64) *uint64 { return &v }
