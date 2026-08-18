package replay_test

import (
	"bytes"
	"testing"
	"time"

	"github.com/lacsar712/adsbhub/internal/journal"
	"github.com/lacsar712/adsbhub/internal/redact"
	"github.com/lacsar712/adsbhub/internal/replay"
)

func TestFromJournalUsesOriginalBody(t *testing.T) {
	body := []byte(`{"kind":"adsb.position","icao":"ABC123","lat":51.47,"lon":-0.45,"squawk":"7000","payload":{"password":"hunter2"}}`)
	e := journal.Entry{
		ReportID:     "rpt1",
		ForwardID:    "fwd1",
		RadarID:      "rad1",
		ReportKind:   "adsb.position",
		Body:         body,
		BodyRedacted: redact.JSON(body),
	}
	j, err := replay.FromJournal(e, time.Unix(1, 0))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(j.Body, body) {
		t.Fatalf("replay body %s want original, not redacted %s", j.Body, e.BodyRedacted)
	}
	if bytes.Contains(j.Body, []byte("***")) {
		t.Fatal("replay used masked payload")
	}
}

func TestFromJournalRejectsEmptyBody(t *testing.T) {
	_, err := replay.FromJournal(journal.Entry{
		ReportID:   "rpt1",
		ForwardID:  "fwd-empty",
		RadarID:    "rad1",
		ReportKind: "adsb.position",
	}, time.Unix(1, 0))
	if err == nil {
		t.Fatal("empty journal body must not become a replay job")
	}
}
