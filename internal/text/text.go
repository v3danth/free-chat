// Package text holds small pure string helpers shared across packages.
package text

// Truncate returns at most n runes of s without splitting a UTF-8 sequence.
func Truncate(s string, n int) string {
	for i := range s {
		if n == 0 {
			return s[:i]
		}
		n--
	}
	return s
}
