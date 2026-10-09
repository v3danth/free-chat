package media

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/v3danth/free-chat/internal/mediapath"
)

func TestHideRestoreDelete(t *testing.T) {
	root := filepath.Join(t.TempDir(), "uploads")
	s, err := NewStorage(root)
	if err != nil {
		t.Fatal(err)
	}
	served := func() bool {
		_, err := os.Stat(filepath.Join(root, mediapath.File(mediapath.Full, "k")))
		return err == nil
	}
	if err := s.SaveAll("k", map[mediapath.Variant][]byte{mediapath.Full: []byte("f"), mediapath.Thumb: []byte("t")}); err != nil {
		t.Fatal(err)
	}
	if err := s.Hide("k"); err != nil || served() {
		t.Fatalf("hide: err=%v, still served=%v", err, served())
	}
	if ok, err := s.Restore("k"); !ok || err != nil || !served() {
		t.Fatalf("restore: ok=%v err=%v served=%v", ok, err, served())
	}
	if ok, _ := s.Restore("k"); ok {
		t.Fatal("restoring an image that is not hidden must report false")
	}
	s.Hide("k")
	if err := s.DeleteAll("k"); err != nil {
		t.Fatal(err)
	}
	if ok, _ := s.Restore("k"); ok || served() {
		t.Fatal("delete must remove hidden files too")
	}
}
