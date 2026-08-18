package report

import (
	"encoding/json"
	"fmt"
	"strings"
	"unicode"

	"github.com/lacsar712/adsbhub/internal/beast"
	"github.com/lacsar712/adsbhub/internal/callsign"
	"github.com/lacsar712/adsbhub/internal/geo"
	"github.com/lacsar712/adsbhub/internal/icao"
	"github.com/lacsar712/adsbhub/internal/squawk"
)

const MaxBody = 256 * 1024

// Report is a station-originated aircraft position (or related) record.
type Report struct {
	Kind     string          `json:"kind"`
	ICAO     string          `json:"icao"`
	Lat      float64         `json:"lat"`
	Lon      float64         `json:"lon"`
	Squawk   string          `json:"squawk"`
	AltFt    int             `json:"alt_ft,omitempty"`
	Callsign string          `json:"callsign,omitempty"`
	Payload  json.RawMessage `json:"payload"`
}

func Parse(body []byte) (Report, error) {
	if len(body) == 0 {
		return Report{}, fmt.Errorf("empty body")
	}
	if len(body) > MaxBody {
		return Report{}, fmt.Errorf("body %d exceeds %d bytes", len(body), MaxBody)
	}
	if beast.LooksLike(body) {
		return fromBeast(body)
	}
	var r Report
	if err := json.Unmarshal(body, &r); err != nil {
		return Report{}, fmt.Errorf("json: %v", err)
	}
	if err := ValidateKind(r.Kind); err != nil {
		return Report{}, err
	}
	addr, err := icao.Normalize(r.ICAO)
	if err != nil {
		return Report{}, err
	}
	r.ICAO = string(addr)
	if err := geo.Validate(r.Lat, r.Lon); err != nil {
		return Report{}, err
	}
	sq, err := squawk.Normalize(r.Squawk)
	if err != nil {
		return Report{}, err
	}
	r.Squawk = sq
	if r.Callsign != "" {
		cs, err := callsign.Normalize(r.Callsign)
		if err != nil {
			return Report{}, err
		}
		r.Callsign = cs
	}
	if len(r.Payload) == 0 || string(r.Payload) == "null" {
		return Report{}, fmt.Errorf("payload is required")
	}
	if !json.Valid(r.Payload) {
		return Report{}, fmt.Errorf("payload is not valid json")
	}
	return r, nil
}

func fromBeast(body []byte) (Report, error) {
	fr, err := beast.Parse(body)
	if err != nil {
		return Report{}, err
	}
	r := Report{
		Kind:    "adsb.beast",
		ICAO:    string(fr.ICAO),
		Payload: json.RawMessage(`{"source":"beast"}`),
	}
	if fr.AltFt != nil {
		r.AltFt = *fr.AltFt
	}
	if fr.Squawk != "" {
		r.Squawk = fr.Squawk
	} else {
		r.Squawk = "7000"
	}
	if err := ValidateKind(r.Kind); err != nil {
		return Report{}, err
	}
	if r.ICAO == "" {
		return Report{}, fmt.Errorf("beast frame missing icao")
	}
	return r, nil
}

func ValidateKind(t string) error {
	t = strings.TrimSpace(t)
	if t == "" {
		return fmt.Errorf("report kind is required")
	}
	if len(t) > 128 {
		return fmt.Errorf("report kind too long")
	}
	parts := strings.Split(t, ".")
	for _, p := range parts {
		if p == "" {
			return fmt.Errorf("report kind has empty segment")
		}
		for _, r := range p {
			if unicode.IsUpper(r) {
				return fmt.Errorf("report kind must be lowercase")
			}
			ok := (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' || r == '-'
			if !ok {
				return fmt.Errorf("report kind has illegal character %q", r)
			}
		}
	}
	return nil
}

func MatchKind(kind, prefix string) bool {
	if prefix == "" {
		return true
	}
	if prefix == kind {
		return true
	}
	if strings.HasSuffix(prefix, ".") {
		return strings.HasPrefix(kind, prefix)
	}
	return kind == prefix || strings.HasPrefix(kind, prefix+".")
}

func SampleJSON() []byte {
	return []byte(`{"kind":"adsb.position","icao":"ABC123","lat":51.47,"lon":-0.4543,"squawk":"7000","alt_ft":35000,"payload":{"gs_kt":420}}`)
}
