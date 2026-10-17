package letters

import (
	"context"
	"slices"
	"testing"
	"time"
)

type memStore struct {
	letters   []Letter
	delivered []uint64
	asked     []int // recipients per Due call
}

func (m *memStore) Due(_ context.Context, now time.Time, recipients []uint64, limit int) ([]Letter, error) {
	m.asked = append(m.asked, len(recipients))
	var due []Letter
	for _, l := range m.letters {
		if slices.Contains(recipients, l.RecipientID) && !l.DeliverAt.After(now) && !slices.Contains(m.delivered, l.ID) && len(due) < limit {
			due = append(due, l)
		}
	}
	return due, nil
}

func (m *memStore) MarkDelivered(_ context.Context, ids []uint64) error {
	m.delivered = append(m.delivered, ids...)
	return nil
}

// onlineBox delivers only to recipients who are online.
type onlineBox struct {
	online map[uint64]bool
	got    []uint64
}

func (b *onlineBox) OnlineIDs() []uint64 {
	ids := make([]uint64, 0, len(b.online))
	for id, on := range b.online {
		if on {
			ids = append(ids, id)
		}
	}
	return ids
}

func (b *onlineBox) DeliverLetter(l Letter) bool {
	if !b.online[l.RecipientID] {
		return false
	}
	b.got = append(b.got, l.ID)
	return true
}

func TestDeliverHandsOverOnlyDueLettersToPeopleOnline(t *testing.T) {
	now := time.Now()
	store := &memStore{letters: []Letter{
		{ID: 1, RecipientID: 10, DeliverAt: now.Add(-time.Minute)}, // due, online
		{ID: 2, RecipientID: 11, DeliverAt: now.Add(-time.Minute)}, // due, offline
		{ID: 3, RecipientID: 10, DeliverAt: now.Add(time.Hour)},    // not yet
	}}
	box := &onlineBox{online: map[uint64]bool{10: true}}

	if err := Deliver(context.Background(), store, box, now); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(box.got, []uint64{1}) || !slices.Equal(store.delivered, []uint64{1}) {
		t.Fatalf("delivered %v, marked %v; want only letter 1", box.got, store.delivered)
	}

	box.online[11] = true
	if err := Deliver(context.Background(), store, box, now); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(store.delivered, []uint64{1, 2}) {
		t.Fatalf("a letter waits for its recipient to come online: marked %v", store.delivered)
	}
}

func TestOfflineLettersNeverHoldBackOnlineOnes(t *testing.T) {
	now := time.Now()
	store := &memStore{}
	// A full batch of letters for someone offline, then one for someone here.
	for i := range batch {
		store.letters = append(store.letters, Letter{ID: uint64(i + 1), RecipientID: 11, DeliverAt: now.Add(-time.Hour)})
	}
	store.letters = append(store.letters, Letter{ID: 9999, RecipientID: 10, DeliverAt: now.Add(-time.Minute)})
	box := &onlineBox{online: map[uint64]bool{10: true}}
	if err := Deliver(context.Background(), store, box, now); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(box.got, []uint64{9999}) {
		t.Fatalf("delivered %v, want the letter for the person online", box.got)
	}
}

func TestNobodyOnlineSkipsTheQuery(t *testing.T) {
	store := &memStore{}
	if err := Deliver(context.Background(), store, &onlineBox{}, time.Now()); err != nil {
		t.Fatal(err)
	}
	if len(store.asked) != 0 {
		t.Fatalf("Due was called %d times with nobody online", len(store.asked))
	}
}
