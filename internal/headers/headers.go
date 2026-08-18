package headers

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

const (
	Timestamp   = "X-Adsb-Timestamp"
	Nonce       = "X-Adsb-Nonce"
	Signature   = "X-Adsb-Signature"
	Idempotency = "Idempotency-Key"
	StationKey  = "X-Adsb-Station-Key"
	ReportID    = "X-Adsb-Report-Id"
	ForwardID   = "X-Adsb-Forward-Id"
	Attempt     = "X-Adsb-Attempt"
	RadarID     = "X-Adsb-Radar"
	BodySHA     = "X-Adsb-Body-Sha256"
)

type Inbound struct {
	Timestamp  int64
	Nonce      string
	Signature  string
	IdemKey    string
	StationKey string
}

func ParseInbound(h http.Header) (Inbound, error) {
	var in Inbound
	ts := strings.TrimSpace(h.Get(Timestamp))
	if ts == "" {
		return in, fmt.Errorf("missing %s", Timestamp)
	}
	n, err := strconv.ParseInt(ts, 10, 64)
	if err != nil || n <= 0 {
		return in, fmt.Errorf("invalid %s", Timestamp)
	}
	in.Timestamp = n
	in.Nonce = strings.TrimSpace(h.Get(Nonce))
	if in.Nonce == "" {
		return in, fmt.Errorf("missing %s", Nonce)
	}
	in.Signature = strings.TrimSpace(h.Get(Signature))
	if in.Signature == "" {
		return in, fmt.Errorf("missing %s", Signature)
	}
	in.IdemKey = strings.TrimSpace(h.Get(Idempotency))
	if in.IdemKey == "" {
		return in, fmt.Errorf("missing %s", Idempotency)
	}
	in.StationKey = strings.TrimSpace(h.Get(StationKey))
	if in.StationKey == "" {
		in.StationKey = "station"
	}
	return in, nil
}
