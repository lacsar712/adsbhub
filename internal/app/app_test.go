package app_test

import (
	"testing"
	"time"

	"github.com/lacsar712/adsbhub/internal/app"
	"github.com/lacsar712/adsbhub/internal/config"
	"github.com/lacsar712/adsbhub/internal/journal"
)

func TestReplayEmptyJournalBodyFails(t *testing.T) {
	a, err := app.New(config.Config{
		Addr:          ":0",
		DataDir:       t.TempDir(),
		StationSecret: "dev-station-secret",
		Window:        5 * time.Minute,
		IdemTTL:       time.Hour,
		Workers:       1,
		PublicBase:    "http://127.0.0.1:8080",
		SinkPath:      "/api/v1/sink",
	})
	if err != nil {
		t.Fatal(err)
	}
	radars := a.Radars.List()
	if len(radars) == 0 {
		t.Fatal("expected seeded radar")
	}
	a.Log.Append(journal.Entry{
		ReportID:   "rpt-empty",
		ForwardID:  "fwd-empty-body",
		RadarID:    radars[0].ID,
		ReportKind: "adsb.position",
	})
	if _, err := a.Replay("fwd-empty-body"); err == nil {
		t.Fatal("replay of empty journal body must fail")
	}
	if a.Broker.Depth() != 0 {
		t.Fatalf("failed replay must not enqueue, depth=%d", a.Broker.Depth())
	}
}
