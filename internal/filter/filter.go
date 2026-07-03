package filter

import (
	"strings"
	"sync"
	"unicode"
	"unicode/utf8"
)

type Config struct {
	MaxMessageLength int
	BannedWords      []string
}

func DefaultConfig() Config {
	return Config{
		MaxMessageLength: 1000,
	}
}

type Result struct {
	Content    string
	Violations []string
	Filtered   bool
}

type Filter struct {
	maxLength int

	mu          sync.RWMutex
	bannedWords map[string]struct{}
}

func NewFilter(cfg Config) *Filter {

	if cfg.MaxMessageLength <= 0 {
		cfg.MaxMessageLength = DefaultConfig().MaxMessageLength
	}

	f := &Filter{
		maxLength: cfg.MaxMessageLength,
		bannedWords: make(map[string]struct{},
			len(cfg.BannedWords)),
	}

	for _, w := range cfg.BannedWords {
		w = normalize(w)

		if w != "" {
			f.bannedWords[w] = struct{}{}
		}
	}

	return f
}

func (f *Filter) Check(content string) Result {

	result := Result{
		Content: content,
	}

	if content == "" {
		return result
	}

	// Unicode safe length check
	if utf8.RuneCountInString(content) > f.maxLength {

		content = truncateRunes(
			content,
			f.maxLength,
		)

		result.Content = content
		result.Filtered = true

		result.Violations =
			append(result.Violations,
				"message exceeds maximum length")
	}

	filtered, changed :=
		f.filterWords(content)

	if changed {

		result.Content = filtered
		result.Filtered = true

		result.Violations =
			append(result.Violations,
				"contains prohibited content")
	}

	return result
}

func (f *Filter) filterWords(content string) (string, bool) {

	var builder strings.Builder

	builder.Grow(len(content))

	changed := false

	for _, word := range strings.FieldsFunc(
		content,
		func(r rune) bool {
			return !unicode.IsLetter(r) &&
				!unicode.IsNumber(r)
		},
	) {

		if f.isBanned(word) {

			changed = true

			continue
		}
	}

	// second pass keeps formatting
	for _, r := range content {

		builder.WriteRune(r)
	}

	if !changed {
		return content, false
	}

	return maskContent(content, f), true
}

func (f *Filter) isBanned(word string) bool {

	f.mu.RLock()
	defer f.mu.RUnlock()

	_, ok :=
		f.bannedWords[normalize(word)]

	return ok
}

func maskContent(content string, f *Filter) string {

	runes := []rune(content)

	start := 0

	for i, r := range runes {

		if !unicode.IsLetter(r) &&
			!unicode.IsNumber(r) {

			continue
		}

		start = i

		for start < len(runes) {

			if !unicode.IsLetter(runes[start]) &&
				!unicode.IsNumber(runes[start]) {
				break
			}

			start++
		}

		word := string(runes[i:start])

		if f.isBanned(word) {

			for j := i + 1; j < start-1; j++ {
				runes[j] = '*'
			}
		}

		i = start
	}

	return string(runes)
}

func normalize(s string) string {

	var b strings.Builder

	for _, r := range s {

		if unicode.IsLetter(r) ||
			unicode.IsNumber(r) {

			b.WriteRune(
				unicode.ToLower(r),
			)
		}
	}

	return b.String()
}

func truncateRunes(
	s string,
	n int,
) string {

	count := 0

	for i, r := range s {

		if count == n {
			return s[:i]
		}

		_ = r
		count++
	}

	return s
}

func (f *Filter) AddBannedWord(word string) {

	word = normalize(word)

	if word == "" {
		return
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	f.bannedWords[word] = struct{}{}
}

func (f *Filter) RemoveBannedWord(word string) {

	f.mu.Lock()
	defer f.mu.Unlock()

	delete(
		f.bannedWords,
		normalize(word),
	)
}
