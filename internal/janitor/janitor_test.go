package janitor

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/v3danth/free-chat/internal/id"
)

type fakeStore struct {
	remaining int64
	cutoffs   []uint64
	online    []uint64
	keys      []string
}

func (f *fakeStore) PurgeMessages(_ context.Context, cutoff uint64, batch int) (int64, error) {
	f.cutoffs = append(f.cutoffs, cutoff)
	n := min(f.remaining, int64(batch))
	f.remaining -= n
	return n, nil
}

func (f *fakeStore) ExpireGuests(_ context.Context, _ time.Time, online []uint64) ([]string, error) {
	f.online = online
	return f.keys, nil
}

func (f *fakeStore) PurgeExpiredIPBans(context.Context) (int64, error) { return 0, nil }

type fakeFiles struct{ deleted []string }

func (f *fakeFiles) DeleteFiles(key string) { f.deleted = append(f.deleted, key) }

type fakeOnline []uint64

func (f fakeOnline) OnlineIDs() []uint64 { return f }

func TestSweep(t *testing.T) {
	store := &fakeStore{remaining: 12_000, keys: []string{"a", "b"}}
	files := &fakeFiles{}
	now := time.Date(2026, 10, 9, 12, 0, 0, 0, time.UTC)

	Sweep(context.Background(), store, files, fakeOnline{7}, 7*24*time.Hour, now)

	if len(store.cutoffs) != 3 || store.remaining != 0 {
		t.Fatalf("expected 3 batches to purge 12000 rows, got %d (left %d)", len(store.cutoffs), store.remaining)
	}
	if want := id.Floor(now.Add(-7 * 24 * time.Hour)); store.cutoffs[0] != want {
		t.Fatalf("cutoff id = %d, want %d", store.cutoffs[0], want)
	}
	if !slices.Equal(store.online, []uint64{7}) {
		t.Fatal("online users must be excluded from expiry")
	}
	if !slices.Equal(files.deleted, []string{"a", "b"}) {
		t.Fatalf("deleted files = %v", files.deleted)
	}
}
