// Package gifs searches a GIF library (GIPHY or Tenor) on the server's key
// and brings a chosen GIF into Drift as an ordinary upload, so it is stored,
// reported and removed like any other image and never hotlinked in chat.
//
// The browser only ever names a GIF by the library's id: the server asks the
// library for that id's file and fetches it from the library's own media
// hosts, so a client cannot point the server at any other URL.
package gifs

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/v3danth/free-chat/internal/apperr"
	"github.com/v3danth/free-chat/internal/media"
)

// Result is one GIF in a search page. PreviewURL is a small rendition on
// the library's CDN, shown only inside the picker.
type Result struct {
	ID         string `json:"id"`
	Title      string `json:"title,omitempty"`
	PreviewURL string `json:"preview_url"`
	Width      int    `json:"width"`
	Height     int    `json:"height"`
}

type Page struct {
	Results []Result `json:"results"`
	Next    string   `json:"next,omitempty"`
	// Provider names the library, for the attribution both require.
	Provider string `json:"provider"`
}

// provider is one library's API. fileURL returns the rendition to import.
type provider interface {
	search(ctx context.Context, q, next string) (Page, error)
	fileURL(ctx context.Context, id string) (string, error)
	mediaHost(host string) bool
}

var (
	ErrDisabled = apperr.New(apperr.NotFound, "GIF search is not set up on this server")
	ErrBadID    = apperr.New(apperr.Invalid, "unknown GIF")
	ErrLibrary  = apperr.New(apperr.Unsupported, "the GIF library did not answer, try again")
)

const pageSize = 24

// validID covers both libraries' ids; anything else never reaches them.
var validID = regexp.MustCompile(`^[A-Za-z0-9_-]{1,64}$`)

// Library is safe for concurrent use. A nil *Library is a disabled one.
type Library struct {
	p      provider
	name   string
	client *http.Client

	mu    sync.Mutex
	cache map[string]cached
}

type cached struct {
	page Page
	at   time.Time
}

const (
	cacheTTL  = 10 * time.Minute
	cacheSize = 512
)

// New returns nil (disabled) when name or key is empty.
func New(name, key string) (*Library, error) {
	if name == "" || key == "" {
		return nil, nil
	}
	client := &http.Client{Timeout: 15 * time.Second}
	var p provider
	switch name {
	case "giphy":
		p = giphy{key: key, client: client}
	case "tenor":
		p = tenor{key: key, client: client}
	default:
		return nil, fmt.Errorf("GIF_PROVIDER must be giphy or tenor, not %q", name)
	}
	l := &Library{p: p, name: name, cache: map[string]cached{}}
	// Files may only come from the library's media hosts, redirects included.
	l.client = &http.Client{
		Timeout: 20 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 3 || !l.allowed(req.URL) {
				return errors.New("gifs: redirect off the library's media hosts")
			}
			return nil
		},
	}
	return l, nil
}

func (l *Library) allowed(u *url.URL) bool {
	return u.Scheme == "https" && l.p.mediaHost(strings.ToLower(u.Hostname()))
}

// Search returns trending GIFs for an empty query. Pages are cached, which
// also keeps a busy room inside the library's rate limits.
func (l *Library) Search(ctx context.Context, q, next string) (Page, error) {
	if l == nil {
		return Page{}, ErrDisabled
	}
	q = strings.TrimSpace(q)
	if len(q) > 50 || len(next) > 64 {
		return Page{}, apperr.New(apperr.Invalid, "search is too long")
	}
	key := strings.ToLower(q) + "\x00" + next
	l.mu.Lock()
	if c, ok := l.cache[key]; ok && time.Since(c.at) < cacheTTL {
		l.mu.Unlock()
		return c.page, nil
	}
	l.mu.Unlock()

	page, err := l.p.search(ctx, q, next)
	if err != nil {
		return Page{}, err
	}
	page.Provider = l.name
	l.mu.Lock()
	if len(l.cache) >= cacheSize {
		clear(l.cache)
	}
	l.cache[key] = cached{page: page, at: time.Now()}
	l.mu.Unlock()
	return page, nil
}

// Fetch downloads GIF id from the library, at most limit bytes.
func (l *Library) Fetch(ctx context.Context, id string, limit int64) ([]byte, error) {
	if l == nil {
		return nil, ErrDisabled
	}
	if !validID.MatchString(id) {
		return nil, ErrBadID
	}
	raw, err := l.p.fileURL(ctx, id)
	if err != nil {
		return nil, err
	}
	u, err := url.Parse(raw)
	if err != nil || !l.allowed(u) {
		return nil, ErrLibrary
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, ErrLibrary
	}
	res, err := l.client.Do(req)
	if err != nil {
		return nil, ErrLibrary
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, ErrLibrary
	}
	data, err := media.ReadCapped(res.Body, limit)
	switch {
	case errors.Is(err, media.ErrTooLarge):
		return nil, err
	case err != nil:
		return nil, ErrLibrary
	}
	return data, nil
}

// getJSON calls a library API and decodes its answer into v.
func getJSON(ctx context.Context, client *http.Client, endpoint string, q url.Values, v any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint+"?"+q.Encode(), nil)
	if err != nil {
		return err
	}
	res, err := client.Do(req)
	if err != nil {
		return ErrLibrary
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return ErrLibrary
	}
	if err := json.NewDecoder(io.LimitReader(res.Body, 4<<20)).Decode(v); err != nil {
		return ErrLibrary
	}
	return nil
}

// ---------------------------------------------------------------------------
// GIPHY: rating=r is the most permissive rating the API serves.
// ---------------------------------------------------------------------------

type giphy struct {
	key    string
	client *http.Client
}

type giphyRendition struct {
	URL    string `json:"url"`
	Width  string `json:"width"`
	Height string `json:"height"`
}

type giphyGIF struct {
	ID     string `json:"id"`
	Title  string `json:"title"`
	Images struct {
		FixedWidth giphyRendition `json:"fixed_width"`
		Downsized  giphyRendition `json:"downsized"`
		Original   giphyRendition `json:"original"`
	} `json:"images"`
}

func (g giphy) search(ctx context.Context, q, next string) (Page, error) {
	offset, _ := strconv.Atoi(next)
	vals := url.Values{"api_key": {g.key}, "limit": {strconv.Itoa(pageSize)}, "offset": {strconv.Itoa(offset)}, "rating": {"r"}}
	endpoint := "https://api.giphy.com/v1/gifs/trending"
	if q != "" {
		endpoint = "https://api.giphy.com/v1/gifs/search"
		vals.Set("q", q)
	}
	var body struct {
		Data []giphyGIF `json:"data"`
	}
	if err := getJSON(ctx, g.client, endpoint, vals, &body); err != nil {
		return Page{}, err
	}
	page := Page{Results: make([]Result, 0, len(body.Data))}
	for _, d := range body.Data {
		w, _ := strconv.Atoi(d.Images.FixedWidth.Width)
		h, _ := strconv.Atoi(d.Images.FixedWidth.Height)
		page.Results = append(page.Results, Result{ID: d.ID, Title: d.Title, PreviewURL: d.Images.FixedWidth.URL, Width: w, Height: h})
	}
	if len(body.Data) == pageSize {
		page.Next = strconv.Itoa(offset + pageSize)
	}
	return page, nil
}

func (g giphy) fileURL(ctx context.Context, id string) (string, error) {
	var body struct {
		Data giphyGIF `json:"data"`
	}
	if err := getJSON(ctx, g.client, "https://api.giphy.com/v1/gifs/"+id, url.Values{"api_key": {g.key}}, &body); err != nil {
		return "", err
	}
	if u := body.Data.Images.Downsized.URL; u != "" {
		return u, nil
	}
	if u := body.Data.Images.Original.URL; u != "" {
		return u, nil
	}
	return "", ErrBadID
}

func (giphy) mediaHost(host string) bool {
	return host == "i.giphy.com" || (strings.HasPrefix(host, "media") && strings.HasSuffix(host, ".giphy.com"))
}

// ---------------------------------------------------------------------------
// Tenor: contentfilter=off is its most permissive setting.
// ---------------------------------------------------------------------------

type tenor struct {
	key    string
	client *http.Client
}

type tenorPost struct {
	ID           string `json:"id"`
	Description  string `json:"content_description"`
	MediaFormats map[string]struct {
		URL  string `json:"url"`
		Dims []int  `json:"dims"`
	} `json:"media_formats"`
}

func (t tenor) search(ctx context.Context, q, next string) (Page, error) {
	vals := url.Values{"key": {t.key}, "client_key": {"drift"}, "limit": {strconv.Itoa(pageSize)},
		"contentfilter": {"off"}, "media_filter": {"tinygif,gif"}}
	if next != "" {
		vals.Set("pos", next)
	}
	endpoint := "https://tenor.googleapis.com/v2/featured"
	if q != "" {
		endpoint = "https://tenor.googleapis.com/v2/search"
		vals.Set("q", q)
	}
	var body struct {
		Results []tenorPost `json:"results"`
		Next    string      `json:"next"`
	}
	if err := getJSON(ctx, t.client, endpoint, vals, &body); err != nil {
		return Page{}, err
	}
	page := Page{Results: make([]Result, 0, len(body.Results)), Next: body.Next}
	for _, p := range body.Results {
		f, ok := p.MediaFormats["tinygif"]
		if !ok || len(f.Dims) != 2 {
			continue
		}
		page.Results = append(page.Results, Result{ID: p.ID, Title: p.Description, PreviewURL: f.URL, Width: f.Dims[0], Height: f.Dims[1]})
	}
	return page, nil
}

func (t tenor) fileURL(ctx context.Context, id string) (string, error) {
	var body struct {
		Results []tenorPost `json:"results"`
	}
	vals := url.Values{"key": {t.key}, "client_key": {"drift"}, "ids": {id}, "media_filter": {"gif"}}
	if err := getJSON(ctx, t.client, "https://tenor.googleapis.com/v2/posts", vals, &body); err != nil {
		return "", err
	}
	if len(body.Results) == 0 || body.Results[0].MediaFormats["gif"].URL == "" {
		return "", ErrBadID
	}
	return body.Results[0].MediaFormats["gif"].URL, nil
}

func (tenor) mediaHost(host string) bool { return host == "media.tenor.com" }
