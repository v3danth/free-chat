package websocket

import (
	"bytes"
	"compress/flate"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/v3danth/free-chat/internal/user"
)

// TestHelloCompressesWell records why big frames are deflated: a hello is
// mostly repeated card keys and face URLs.
func TestHelloCompressesWell(t *testing.T) {
	cards := make([]json.RawMessage, 0, 1000)
	for i := range 1000 {
		u := user.User{ID: uint64(i + 1), Kind: user.KindGuest, Role: user.RoleUser,
			Profile:  user.Profile{Name: fmt.Sprintf("person %d", i), Gender: user.GenderFemale, Age: 25, Tags: []string{"just talk", "switch"}, About: "Night owl, bad at sleeping", Location: "Lisbon"},
			Settings: user.Settings{Aura: "rose"}}
		cards = append(cards, encode(user.ToCard(u, time.Now())))
	}
	frame := helloFrame(helloEvent{Type: "hello", Doors: []doorView{}}, cards)
	var buf bytes.Buffer
	zw, _ := flate.NewWriter(&buf, flate.BestSpeed)
	_, _ = zw.Write(frame)
	_ = zw.Close()
	ratio := float64(len(frame)) / float64(buf.Len())
	t.Logf("hello for 1000 online: %d KB raw, %d KB deflated (%.1fx)", len(frame)>>10, buf.Len()>>10, ratio)
	if ratio < 3 {
		t.Fatalf("hello deflates only %.1fx; revisit compressFrom", ratio)
	}
}
