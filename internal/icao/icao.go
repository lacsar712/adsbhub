// Package icao validates 24-bit Mode-S / ADS-B aircraft addresses.
package icao

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

const (
	Min = 0x000001
	Max = 0xFFFFFF
)

// Address is a 24-bit ICAO aircraft address stored as a 6-char uppercase hex.
type Address string

func Normalize(raw string) (Address, error) {
	s := strings.TrimSpace(raw)
	s = strings.TrimPrefix(strings.ToUpper(s), "ICAO:")
	s = strings.TrimPrefix(s, "0X")
	s = strings.ReplaceAll(s, " ", "")
	s = strings.ReplaceAll(s, "-", "")
	if s == "" {
		return "", fmt.Errorf("icao address is required")
	}
	if len(s) > 6 {
		return "", fmt.Errorf("icao address %q longer than 6 hex digits", s)
	}
	for _, r := range s {
		if !unicode.Is(unicode.ASCII_Hex_Digit, r) {
			return "", fmt.Errorf("icao address %q is not hex", s)
		}
	}
	n, err := strconv.ParseUint(s, 16, 32)
	if err != nil {
		return "", fmt.Errorf("icao address %q: %w", s, err)
	}
	if n < Min || n > Max {
		return "", fmt.Errorf("icao address %q out of 24-bit range", s)
	}
	return Address(fmt.Sprintf("%06X", n)), nil
}

func (a Address) Uint() uint32 {
	n, _ := strconv.ParseUint(string(a), 16, 32)
	return uint32(n)
}

func FromUint(n uint32) (Address, error) {
	if n < Min || n > Max {
		return "", fmt.Errorf("icao %06X out of 24-bit range", n)
	}
	return Address(fmt.Sprintf("%06X", n)), nil
}

func (a Address) IsMilitaryHint() bool {
	n := a.Uint()
	// US military blocks are not a complete map; this is a coarse demo flag.
	return n >= 0xAE0000 && n <= 0xAFFFFF
}

func (a Address) IsAllZero() bool {
	return a.Uint() == 0
}

func Valid(raw string) bool {
	_, err := Normalize(raw)
	return err == nil
}
