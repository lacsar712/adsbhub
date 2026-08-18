// Package beast parses Mode-S Beast-like hex frames used by dump1090-style feeders.
//
// Wire form: ASCII '*' + even-length hex + ';' (Mode-S long or short).
// DF17 ADS-B extracts ICAO, type code, and when present altitude / squawk bits.
package beast

import (
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/lacsar712/adsbhub/internal/altitude"
	"github.com/lacsar712/adsbhub/internal/icao"
)

type Frame struct {
	Raw      []byte
	DF       int
	CA       int
	ICAO     icao.Address
	TypeCode int
	ME       []byte
	AltFt    *int
	Squawk   string
}

func LooksLike(body []byte) bool {
	s := strings.TrimSpace(string(body))
	return strings.HasPrefix(s, "*") && strings.HasSuffix(s, ";")
}

func Parse(body []byte) (Frame, error) {
	s := strings.TrimSpace(string(body))
	if !LooksLike(body) {
		return Frame{}, fmt.Errorf("not a beast hex frame")
	}
	inner := s[1 : len(s)-1]
	inner = strings.ReplaceAll(inner, " ", "")
	if len(inner)%2 != 0 {
		return Frame{}, fmt.Errorf("beast hex has odd length")
	}
	if len(inner) != 14 && len(inner) != 28 {
		return Frame{}, fmt.Errorf("beast frame must be 7 or 14 bytes, got %d hex chars", len(inner))
	}
	raw, err := hex.DecodeString(inner)
	if err != nil {
		return Frame{}, fmt.Errorf("beast hex: %w", err)
	}
	f := Frame{Raw: raw}
	f.DF = int(raw[0] >> 3)
	f.CA = int(raw[0] & 0x07)
	if len(raw) >= 4 {
		addr, err := icao.FromUint(uint32(raw[1])<<16 | uint32(raw[2])<<8 | uint32(raw[3]))
		if err != nil {
			return Frame{}, err
		}
		f.ICAO = addr
	}
	if f.DF == 17 && len(raw) >= 11 {
		f.ME = append([]byte(nil), raw[4:11]...)
		f.TypeCode = int(raw[4] >> 3)
		if f.TypeCode >= 9 && f.TypeCode <= 18 {
			if ft, ok := decodeAlt(raw[5], raw[6]); ok {
				f.AltFt = &ft
			}
		}
	}
	if f.DF == 5 && len(raw) >= 3 {
		ident := uint16(raw[2])<<4 | uint16(raw[3]>>4)
		_ = ident
	}
	return f, nil
}

func decodeAlt(b5, b6 byte) (int, bool) {
	n := (int(b5&0xFF) << 4) | int(b6>>4)
	q := (n & 0x10) != 0
	n = ((n & 0xFFE0) >> 1) | (n & 0x0F)
	ft, err := altitude.FromQBit(n, q)
	if err != nil {
		return 0, false
	}
	return ft, true
}

func Encode(raw []byte) (string, error) {
	if len(raw) != 7 && len(raw) != 14 {
		return "", fmt.Errorf("mode-s payload must be 7 or 14 bytes")
	}
	return "*" + strings.ToUpper(hex.EncodeToString(raw)) + ";", nil
}
