package radar_test

import (
	"testing"
	"time"

	"github.com/lacsar712/adsbhub/internal/clock"
	"github.com/lacsar712/adsbhub/internal/radar"
)

func TestMatchesSegmentBoundary(t *testing.T) {
	cases := []struct {
		prefix string
		kind   string
		want   bool
	}{
		{"adsb", "adsb.position", true},
		{"adsb.", "adsb.position", true},
		{"adsb.position", "adsb.position", true},
		{"", "anything.else", true},
		{"ad", "adsb.position", false},
		{"ads", "adsb.position", false},
		{"adsb.p", "adsb.position", false},
		{"mode_s", "adsb.position", false},
		{"adsb", "adsbx.position", false},
	}
	for _, tc := range cases {
		d := radar.Radar{
			Enabled:      true,
			KindPrefixes: []string{tc.prefix},
		}
		got := d.Matches(tc.kind)
		if got != tc.want {
			t.Fatalf("prefix %q vs kind %q: Matches=%v want %v", tc.prefix, tc.kind, got, tc.want)
		}
	}
}

func TestMatchesDisabledNeverFires(t *testing.T) {
	d := radar.Radar{
		Enabled:      false,
		KindPrefixes: []string{""},
	}
	if d.Matches("adsb.position") {
		t.Fatal("disabled radar must not match")
	}
}

func TestRegistryMatchingSkipsDisabled(t *testing.T) {
	clk := clock.NewFrozen(time.Unix(0, 0))
	reg := radar.NewRegistry(clk)
	off := false
	d, err := reg.Create(radar.CreateInput{
		Name:         "off",
		URL:          "http://127.0.0.1:8080/api/v1/sink",
		Secret:       "abcdefgh",
		KindPrefixes: []string{"adsb"},
		Enabled:      &off,
	})
	if err != nil {
		t.Fatal(err)
	}
	if d.Enabled {
		t.Fatal("expected disabled")
	}
	got := reg.Matching("adsb.position")
	if len(got) != 0 {
		t.Fatalf("disabled radar leaked into matching: %+v", got)
	}
}
