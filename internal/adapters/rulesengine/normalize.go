package rulesengine

import (
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

// DefaultPSPSuffixes are Indian UPI PSP handles (design defaults).
var DefaultPSPSuffixes = []string{
	"okaxis", "okhdfcbank", "okicici", "oksbi", "ybl", "paytm", "ibl", "axl", "apl", "upi",
}

// NormalizePayee produces payee_norm: NFKC, lower, trim, strip @psp suffixes.
func NormalizePayee(raw string, pspSuffixes []string) string {
	if raw == "" {
		return ""
	}
	s := norm.NFKC.String(raw)
	s = strings.ToLower(strings.TrimSpace(s))
	// collapse internal whitespace
	var b strings.Builder
	prevSpace := false
	for _, r := range s {
		if unicode.IsSpace(r) {
			if !prevSpace {
				b.WriteByte(' ')
				prevSpace = true
			}
			continue
		}
		prevSpace = false
		b.WriteRune(r)
	}
	s = strings.TrimSpace(b.String())
	suffixes := pspSuffixes
	if len(suffixes) == 0 {
		suffixes = DefaultPSPSuffixes
	}
	if i := strings.LastIndex(s, "@"); i >= 0 {
		handle := s[i+1:]
		for _, suf := range suffixes {
			if handle == suf {
				s = strings.TrimSpace(s[:i])
				break
			}
		}
	}
	return s
}
