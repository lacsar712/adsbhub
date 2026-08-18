package beast_test

import (
	"testing"

	"github.com/lacsar712/adsbhub/internal/beast"
)

func TestLooksLikeAndParse(t *testing.T) {
	raw := make([]byte, 14)
	raw[0] = 17 << 3
	raw[1], raw[2], raw[3] = 0xAB, 0xC1, 0x23
	enc, err := beast.Encode(raw)
	if err != nil {
		t.Fatal(err)
	}
	if !beast.LooksLike([]byte(enc)) {
		t.Fatal("encoded frame should look like beast")
	}
	fr, err := beast.Parse([]byte(enc))
	if err != nil {
		t.Fatal(err)
	}
	if string(fr.ICAO) != "ABC123" {
		t.Fatalf("icao %s", fr.ICAO)
	}
	if fr.DF != 17 {
		t.Fatalf("df %d", fr.DF)
	}
}
