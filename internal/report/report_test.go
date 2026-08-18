package report_test

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/lacsar712/adsbhub/internal/report"
)

func TestParseAndKind(t *testing.T) {
	r, err := report.Parse([]byte(`{"kind":"adsb.position","icao":"ABC123","lat":51.47,"lon":-0.45,"squawk":"7000","payload":{"id":1}}`))
	if err != nil {
		t.Fatal(err)
	}
	if r.Kind != "adsb.position" {
		t.Fatalf("kind %s", r.Kind)
	}
	if r.ICAO != "ABC123" {
		t.Fatalf("icao %s", r.ICAO)
	}
	if !report.MatchKind("adsb.position", "adsb") {
		t.Fatal("adsb should match adsb.position")
	}
	if report.MatchKind("adsb.position", "mode_s") {
		t.Fatal("mode_s should not match")
	}
	if _, err := report.Parse([]byte(`{"kind":"Adsb.Position","icao":"ABC123","lat":1,"lon":1,"squawk":"7000","payload":{}}`)); err == nil {
		t.Fatal("uppercase kind should fail")
	}
}

func TestMatchKindSegmentBoundary(t *testing.T) {
	cases := []struct {
		kind   string
		prefix string
		want   bool
	}{
		{"adsb.position", "adsb", true},
		{"adsb.position", "adsb.", true},
		{"adsb.position", "adsb.position", true},
		{"adsb.position", "", true},
		{"adsb.position", "ad", false},
		{"adsb.position", "ads", false},
		{"adsb.position", "adsb.p", false},
		{"adsb.position", "mode_s", false},
		{"adsbx.position", "adsb", false},
	}
	for _, tc := range cases {
		got := report.MatchKind(tc.kind, tc.prefix)
		if got != tc.want {
			t.Fatalf("MatchKind(%q, %q)=%v want %v", tc.kind, tc.prefix, got, tc.want)
		}
	}
}

func TestParseWrapsSyntaxError(t *testing.T) {
	_, err := report.Parse([]byte(`{`))
	if err == nil {
		t.Fatal("expected syntax error")
	}
	var syn *json.SyntaxError
	if !errors.As(err, &syn) {
		t.Fatalf("want json.SyntaxError via errors.As, got %v", err)
	}
}
