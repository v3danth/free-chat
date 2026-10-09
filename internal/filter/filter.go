// Package filter is a pure content filter. A Filter is immutable after New;
// Live holds the current one so moderators can swap it atomically while
// messages are being filtered concurrently.
package filter

import (
	"strings"
	"sync/atomic"
	"unicode"
	"unicode/utf8"

	"github.com/v3danth/free-chat/internal/text"
)

type Action string

const (
	Mask  Action = "mask"  // replace the word with its first letter and '*'s
	Block Action = "block" // reject the whole message
)

type Rule struct {
	Word   string
	Action Action
}

const (
	ViolationTooLong    = "message exceeds maximum length"
	ViolationProhibited = "contains prohibited content"
	ViolationBlocked    = "contains blocked content"
)

type Result struct {
	Content    string
	Filtered   bool
	Blocked    bool
	Violations []string
}

type Filter struct {
	maxLen  int
	words   map[string]Action // single words
	phrases []string          // multi-word, always Block, stored " a b "
}

// Normalize is the canonical form of a rule: lower case, runs of anything
// that is not part of a word collapsed to one space. "Telegram.me" becomes
// "telegram me", which is a phrase.
func Normalize(s string) string {
	return strings.Join(strings.FieldsFunc(strings.ToLower(s), notWord), " ")
}

func New(maxLen int, rules []Rule) Filter {
	f := Filter{maxLen: maxLen, words: make(map[string]Action, len(rules))}
	for _, r := range rules {
		w := Normalize(r.Word)
		switch {
		case w == "":
		case strings.Contains(w, " "):
			f.phrases = append(f.phrases, " "+w+" ")
		default:
			f.words[w] = r.Action
		}
	}
	return f
}

func (f Filter) Apply(content string) Result {
	var violations []string

	if utf8.RuneCountInString(content) > f.maxLen {
		content = text.Truncate(content, f.maxLen)
		violations = append(violations, ViolationTooLong)
	}

	if len(f.phrases) > 0 {
		padded := " " + Normalize(content) + " "
		for _, p := range f.phrases {
			if strings.Contains(padded, p) {
				return Result{Content: content, Filtered: true, Blocked: true, Violations: append(violations, ViolationBlocked)}
			}
		}
	}

	masked, hit, blocked := f.scan(content)
	switch {
	case blocked:
		return Result{Content: content, Filtered: true, Blocked: true, Violations: append(violations, ViolationBlocked)}
	case hit:
		violations = append(violations, ViolationProhibited)
	}
	return Result{Content: masked, Filtered: len(violations) > 0, Violations: violations}
}

// Screen adapts Apply to user.Screen for profile text.
func (f Filter) Screen(s string) (string, bool, bool) {
	r := f.Apply(s)
	return r.Content, r.Filtered && !r.Blocked, r.Blocked
}

// scan masks every Mask word as its first rune plus '*'s (e.g. "f***"),
// leaving punctuation and spacing untouched, and reports any Block word.
func (f Filter) scan(s string) (out string, masked, blocked bool) {
	if len(f.words) == 0 {
		return s, false, false
	}
	runes := []rune(s)
	for i := 0; i < len(runes); {
		if notWord(runes[i]) {
			i++
			continue
		}
		end := i
		for end < len(runes) && !notWord(runes[end]) {
			end++
		}
		switch f.words[strings.ToLower(string(runes[i:end]))] {
		case Block:
			return s, false, true
		case Mask:
			masked = true
			for k := i + 1; k < end; k++ {
				runes[k] = '*'
			}
		}
		i = end
	}
	if !masked {
		return s, false, false
	}
	return string(runes), true, false
}

// notWord splits words. Combining marks belong to the word: Hindi vowel
// signs (matras) are marks, and splitting on them would break every word.
func notWord(r rune) bool {
	return !unicode.IsLetter(r) && !unicode.IsMark(r) && !unicode.IsNumber(r)
}

// Live is the filter in force; Store swaps it for every later Load.
type Live struct {
	p atomic.Pointer[Filter]
}

func NewLive(f Filter) *Live {
	l := &Live{}
	l.Store(f)
	return l
}

func (l *Live) Load() Filter   { return *l.p.Load() }
func (l *Live) Store(f Filter) { l.p.Store(&f) }
