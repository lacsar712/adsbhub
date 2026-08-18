package icao_test

import (
	"testing"

	"github.com/lacsar712/adsbhub/internal/icao"
)

func TestNormalize(t *testing.T) {
	a, err := icao.Normalize("abc123")
	if err != nil {
		t.Fatal(err)
	}
	if a != "ABC123" {
		t.Fatalf("got %s", a)
	}
	if _, err := icao.Normalize(""); err == nil {
		t.Fatal("empty should fail")
	}
	if _, err := icao.Normalize("GGGGGG"); err == nil {
		t.Fatal("non-hex should fail")
	}
}
