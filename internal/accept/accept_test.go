package accept_test

import (
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/lacsar712/adsbhub/internal/accept"
	"github.com/lacsar712/adsbhub/internal/clock"
	"github.com/lacsar712/adsbhub/internal/hashutil"
	"github.com/lacsar712/adsbhub/internal/headers"
	"github.com/lacsar712/adsbhub/internal/idempotency"
	"github.com/lacsar712/adsbhub/internal/nonce"
	"github.com/lacsar712/adsbhub/internal/queue"
	"github.com/lacsar712/adsbhub/internal/radar"
	"github.com/lacsar712/adsbhub/internal/sign"
	"github.com/lacsar712/adsbhub/internal/station"
)

func sampleBody(id int) []byte {
	if id == 2 {
		return []byte(`{"kind":"adsb.position","icao":"ABC123","lat":51.47,"lon":-0.45,"squawk":"7000","payload":{"id":2}}`)
	}
	return []byte(`{"kind":"adsb.position","icao":"ABC123","lat":51.47,"lon":-0.45,"squawk":"7000","payload":{"id":1}}`)
}

func TestPipelineAcceptsSignedReport(t *testing.T) {
	clk := clock.NewFrozen(time.Unix(1_700_000_000, 0))
	radars := radar.NewRegistry(clk)
	enabled := true
	_, err := radars.Create(radar.CreateInput{
		Name:         "loop",
		URL:          "http://127.0.0.1:8080/api/v1/sink",
		Secret:       "abcdefgh",
		KindPrefixes: []string{"adsb"},
		Enabled:      &enabled,
	})
	if err != nil {
		t.Fatal(err)
	}
	p := &accept.Pipeline{
		Clk:    clk,
		Window: 5 * time.Minute,
		Keys:   station.New("station", "supersecret"),
		Nonces: nonce.New(clk, 5*time.Minute),
		Idem:   idempotency.New(clk, time.Hour),
		Radars: radars,
		Broker: queue.NewBroker(clk),
	}
	body := sampleBody(1)
	n := "abcdefghijklmnop"
	ts := clk.Now().Unix()
	sig, err := sign.Sign("supersecret", ts, n, body)
	if err != nil {
		t.Fatal(err)
	}
	h := make(http.Header)
	h.Set(headers.Timestamp, "1700000000")
	h.Set(headers.Nonce, n)
	h.Set(headers.Signature, sig)
	h.Set(headers.Idempotency, "idemkey1")
	h.Set(headers.StationKey, "station")
	res, code, err := p.Handle(h, body)
	if err != nil {
		t.Fatal(err)
	}
	if code != http.StatusAccepted {
		t.Fatalf("code %d", code)
	}
	if res.Matched != 1 {
		t.Fatalf("matched %d hash %s", res.Matched, hashutil.SHA256Hex(body))
	}
}

func TestPipelineRejectsPartialKindPrefix(t *testing.T) {
	clk := clock.NewFrozen(time.Unix(1_700_000_000, 0))
	radars := radar.NewRegistry(clk)
	enabled := true
	_, err := radars.Create(radar.CreateInput{
		Name:         "too-short",
		URL:          "http://127.0.0.1:8080/api/v1/sink",
		Secret:       "abcdefgh",
		KindPrefixes: []string{"ad"},
		Enabled:      &enabled,
	})
	if err != nil {
		t.Fatal(err)
	}
	p := &accept.Pipeline{
		Clk:    clk,
		Window: 5 * time.Minute,
		Keys:   station.New("station", "supersecret"),
		Nonces: nonce.New(clk, 5*time.Minute),
		Idem:   idempotency.New(clk, time.Hour),
		Radars: radars,
		Broker: queue.NewBroker(clk),
	}
	body := sampleBody(1)
	n := "qrstuvwxyzabcdef"
	ts := clk.Now().Unix()
	sig, err := sign.Sign("supersecret", ts, n, body)
	if err != nil {
		t.Fatal(err)
	}
	h := make(http.Header)
	h.Set(headers.Timestamp, "1700000000")
	h.Set(headers.Nonce, n)
	h.Set(headers.Signature, sig)
	h.Set(headers.Idempotency, "idemkey-partial")
	h.Set(headers.StationKey, "station")
	res, code, err := p.Handle(h, body)
	if err != nil {
		t.Fatal(err)
	}
	if code != http.StatusAccepted {
		t.Fatalf("code %d", code)
	}
	if res.Matched != 0 {
		t.Fatalf("prefix %q must not match %q, matched=%d ids=%v", "ad", "adsb.position", res.Matched, res.ForwardIDs)
	}
}

func newPipeline(t *testing.T, clk *clock.Frozen) *accept.Pipeline {
	t.Helper()
	radars := radar.NewRegistry(clk)
	enabled := true
	if _, err := radars.Create(radar.CreateInput{
		Name:         "loop",
		URL:          "http://127.0.0.1:8080/api/v1/sink",
		Secret:       "abcdefgh",
		KindPrefixes: []string{"adsb"},
		Enabled:      &enabled,
	}); err != nil {
		t.Fatal(err)
	}
	return &accept.Pipeline{
		Clk:    clk,
		Window: 5 * time.Minute,
		Keys:   station.New("station", "supersecret"),
		Nonces: nonce.New(clk, 5*time.Minute),
		Idem:   idempotency.New(clk, time.Hour),
		Radars: radars,
		Broker: queue.NewBroker(clk),
	}
}

func signedHeaders(t *testing.T, body []byte, nonce, idem string, ts int64) http.Header {
	t.Helper()
	sig, err := sign.Sign("supersecret", ts, nonce, body)
	if err != nil {
		t.Fatal(err)
	}
	h := make(http.Header)
	h.Set(headers.Timestamp, strconv.FormatInt(ts, 10))
	h.Set(headers.Nonce, nonce)
	h.Set(headers.Signature, sig)
	h.Set(headers.Idempotency, idem)
	h.Set(headers.StationKey, "station")
	return h
}

func TestPipelineSkewIsBadRequest(t *testing.T) {
	clk := clock.NewFrozen(time.Unix(1_700_000_000, 0))
	p := newPipeline(t, clk)
	body := sampleBody(1)
	ts := clk.Now().Add(-20 * time.Minute).Unix()
	_, code, err := p.Handle(signedHeaders(t, body, "skewnonce16chars", "idem-skew-01", ts), body)
	if err == nil {
		t.Fatal("expected skew error")
	}
	if code != http.StatusBadRequest {
		t.Fatalf("code %d want 400", code)
	}
}

func TestPipelineIdempotencyConflict(t *testing.T) {
	clk := clock.NewFrozen(time.Unix(1_700_000_000, 0))
	p := newPipeline(t, clk)
	body1 := sampleBody(1)
	body2 := sampleBody(2)
	ts := clk.Now().Unix()
	_, code1, err := p.Handle(signedHeaders(t, body1, "idemnonce16charA", "same-idem-key", ts), body1)
	if err != nil || code1 != http.StatusAccepted {
		t.Fatalf("first: code=%d err=%v", code1, err)
	}
	_, code2, err := p.Handle(signedHeaders(t, body2, "idemnonce16charB", "same-idem-key", ts), body2)
	if err == nil {
		t.Fatal("expected conflict")
	}
	if code2 != http.StatusConflict {
		t.Fatalf("code %d want 409", code2)
	}
}

func TestPipelineUnknownSourceKeyUnauthorized(t *testing.T) {
	clk := clock.NewFrozen(time.Unix(1_700_000_000, 0))
	p := newPipeline(t, clk)
	body := sampleBody(1)
	h := signedHeaders(t, body, "unknownsrc16char", "idem-unknown-01", clk.Now().Unix())
	h.Set(headers.StationKey, "no-such-key")
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("unknown station key panicked: %v", r)
		}
	}()
	_, code, err := p.Handle(h, body)
	if err == nil {
		t.Fatal("expected unauthorized")
	}
	if code != http.StatusUnauthorized {
		t.Fatalf("code %d want 401", code)
	}
}

func TestPipelineInvalidJSONIsBadRequest(t *testing.T) {
	clk := clock.NewFrozen(time.Unix(1_700_000_000, 0))
	p := newPipeline(t, clk)
	body := []byte(`{`)
	_, code, err := p.Handle(signedHeaders(t, body, "jsonnonce16chars", "idem-json-01", clk.Now().Unix()), body)
	if err == nil {
		t.Fatal("expected parse error")
	}
	if code != http.StatusBadRequest {
		t.Fatalf("broken json want 400, got %d", code)
	}
}

func TestPipelineMissingPayloadUnprocessable(t *testing.T) {
	clk := clock.NewFrozen(time.Unix(1_700_000_000, 0))
	p := newPipeline(t, clk)
	body := []byte(`{"kind":"adsb.position","icao":"ABC123","lat":51.47,"lon":-0.45,"squawk":"7000"}`)
	_, code, err := p.Handle(signedHeaders(t, body, "paylnonce16chars", "idem-payload-01", clk.Now().Unix()), body)
	if err == nil {
		t.Fatal("expected payload error")
	}
	if code != http.StatusUnprocessableEntity {
		t.Fatalf("missing payload want 422, got %d", code)
	}
}

func TestPipelineDuplicateNonceConflict(t *testing.T) {
	clk := clock.NewFrozen(time.Unix(1_700_000_000, 0))
	p := newPipeline(t, clk)
	body := sampleBody(1)
	ts := clk.Now().Unix()
	n := "dupnonce16charsx"
	_, code1, err := p.Handle(signedHeaders(t, body, n, "idem-nonce-a", ts), body)
	if err != nil || code1 != http.StatusAccepted {
		t.Fatalf("first: code=%d err=%v", code1, err)
	}
	_, code2, err := p.Handle(signedHeaders(t, body, n, "idem-nonce-b", ts), body)
	if err == nil {
		t.Fatal("expected nonce conflict")
	}
	if code2 != http.StatusConflict {
		t.Fatalf("duplicate nonce want 409, got %d", code2)
	}
}
