// Package squawk validates Mode-A/C identity codes (octal 0000–7777).
package squawk

import (
	"fmt"
	"strconv"
	"strings"
)

const (
	Emergency7500 = "7500"
	Emergency7600 = "7600"
	Emergency7700 = "7700"
)

func Normalize(raw string) (string, error) {
	s := strings.TrimSpace(raw)
	s = strings.TrimPrefix(s, "A")
	s = strings.TrimPrefix(s, "a")
	if s == "" {
		return "", fmt.Errorf("squawk is required")
	}
	if len(s) > 4 {
		return "", fmt.Errorf("squawk %q longer than 4 digits", s)
	}
	for _, r := range s {
		if r < '0' || r > '7' {
			return "", fmt.Errorf("squawk %q is not octal", s)
		}
	}
	n, err := strconv.ParseUint(s, 8, 16)
	if err != nil {
		return "", fmt.Errorf("squawk %q: %w", s, err)
	}
	if n > 0o7777 {
		return "", fmt.Errorf("squawk %q out of range", s)
	}
	return fmt.Sprintf("%04o", n), nil
}

func IsEmergency(code string) bool {
	c, err := Normalize(code)
	if err != nil {
		return false
	}
	return c == Emergency7500 || c == Emergency7600 || c == Emergency7700
}

func FromIdentityBits(bits uint16) (string, error) {
	// Mode-A C1 A1 C2 A2 C4 A4 B1 D1 B2 D2 B4 D4 packing (12 bits).
	bits &= 0x0FFF
	a := ((bits >> 9) & 1) | (((bits >> 7) & 1) << 1) | (((bits >> 5) & 1) << 2)
	b := ((bits >> 3) & 1) | (((bits >> 1) & 1) << 1) | (((bits >> 11) & 1) << 2)
	c := ((bits >> 8) & 1) | (((bits >> 6) & 1) << 1) | (((bits >> 4) & 1) << 2)
	d := ((bits >> 2) & 1) | ((bits & 1) << 1) | (((bits >> 10) & 1) << 2)
	n := (a << 9) | (b << 6) | (c << 3) | d
	return Normalize(fmt.Sprintf("%04o", n))
}
