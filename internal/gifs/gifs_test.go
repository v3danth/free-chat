package gifs

import (
	"context"
	"errors"
	"net/url"
	"testing"
)

type fakeProvider struct{ file string }

func (fakeProvider) search(context.Context, string, string) (Page, error) { return Page{}, nil }
func (f fakeProvider) fileURL(context.Context, string) (string, error)    { return f.file, nil }
func (fakeProvider) mediaHost(host string) bool                           { return host == "media.example.com" }

func TestDisabledLibrary(t *testing.T) {
	lib, err := New("", "")
	if err != nil || lib != nil {
		t.Fatalf("empty config must disable, got %v %v", lib, err)
	}
	if _, err := lib.Search(context.Background(), "x", ""); !errors.Is(err, ErrDisabled) {
		t.Errorf("search on a disabled library: %v", err)
	}
	if _, err := New("bogus", "k"); err == nil {
		t.Error("unknown provider accepted")
	}
}

func TestFetchOnlyFromMediaHosts(t *testing.T) {
	for _, file := range []string{
		"http://media.example.com/a.gif",  // not https
		"https://evil.example.org/a.gif",  // another host
		"https://127.0.0.1/a.gif",         // internal address
		"https://media.example.com.evil/", // look-alike
	} {
		lib := &Library{p: fakeProvider{file: file}}
		if _, err := lib.Fetch(context.Background(), "abc", 1<<20); !errors.Is(err, ErrLibrary) {
			t.Errorf("%s: fetched (%v)", file, err)
		}
	}
	lib := &Library{p: fakeProvider{}}
	if _, err := lib.Fetch(context.Background(), "../etc", 1<<20); !errors.Is(err, ErrBadID) {
		t.Errorf("bad id: %v", err)
	}
	u, _ := url.Parse("https://MEDIA.example.com/x.gif")
	if !lib.allowed(u) {
		t.Error("host match must ignore case")
	}
}

func TestProviderHosts(t *testing.T) {
	g, tn := giphy{}, tenor{}
	for host, want := range map[string]bool{"media0.giphy.com": true, "i.giphy.com": true, "giphy.com": false, "media.giphy.com.evil.io": false} {
		if g.mediaHost(host) != want {
			t.Errorf("giphy %s = %v", host, !want)
		}
	}
	if !tn.mediaHost("media.tenor.com") || tn.mediaHost("tenor.com") {
		t.Error("tenor hosts")
	}
}
