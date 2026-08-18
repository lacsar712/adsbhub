// Package callsign decodes AIS 6-bit aircraft identity used in DF17 TC=1..4.
package callsign

import (
	"fmt"
	"strings"
	"unicode"
)

const charset = "?ABCDEFGHIJKLMNOPQRSTUVWXYZ????? ???????????????0123456789??????"

func DecodeAIS(bits uint64, nchars int) (string, error) {
	if nchars < 1 || nchars > 8 {
		return "", fmt.Errorf("ais identity length %d not in [1,8]", nchars)
	}
	var b strings.Builder
	for i := nchars - 1; i >= 0; i-- {
		idx := int((bits >> (uint(i) * 6)) & 0x3F)
		if idx >= len(charset) {
			return "", fmt.Errorf("ais index %d out of charset", idx)
		}
		b.WriteByte(charset[idx])
	}
	s := strings.TrimRight(b.String(), " ")
	if s == "" {
		return "", fmt.Errorf("empty callsign")
	}
	return s, nil
}

func Validate(cs string) error {
	cs = strings.TrimSpace(cs)
	if cs == "" {
		return fmt.Errorf("callsign is required")
	}
	if len(cs) > 8 {
		return fmt.Errorf("callsign %q longer than 8", cs)
	}
	for _, r := range cs {
		if unicode.IsSpace(r) {
			return fmt.Errorf("callsign has interior space")
		}
		ok := (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9')
		if !ok {
			return fmt.Errorf("callsign %q has illegal rune %q", cs, r)
		}
	}
	return nil
}

func Normalize(cs string) (string, error) {
	s := strings.ToUpper(strings.TrimSpace(cs))
	if err := Validate(s); err != nil {
		return "", err
	}
	return s, nil
}
