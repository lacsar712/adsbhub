package replay

import (
	"fmt"
	"time"

	"github.com/lacsar712/adsbhub/internal/dlq"
	"github.com/lacsar712/adsbhub/internal/idgen"
	"github.com/lacsar712/adsbhub/internal/job"
	"github.com/lacsar712/adsbhub/internal/journal"
)

func FromJournal(e journal.Entry, now time.Time) (job.Job, error) {
	if len(e.Body) == 0 {
		return job.Job{}, fmt.Errorf("journal entry %s has no stored body", e.ForwardID)
	}
	return job.Job{
		ReportID:  e.ReportID,
		ForwardID: idgen.New("fwd", now),
		RadarID:   e.RadarID,
		Kind:      e.ReportKind,
		Body:      append([]byte(nil), e.Body...),
		Attempt:   0,
		NotBefore: now,
		CreatedAt: now,
		ReplayOf:  e.ForwardID,
	}, nil
}

func FromDLQ(it dlq.Item, now time.Time) (job.Job, error) {
	if len(it.Body) == 0 {
		return job.Job{}, fmt.Errorf("dlq item %s has no stored body", it.ForwardID)
	}
	return job.Job{
		ReportID:  it.ReportID,
		ForwardID: idgen.New("fwd", now),
		RadarID:   it.RadarID,
		Kind:      it.Kind,
		Body:      append([]byte(nil), it.Body...),
		Attempt:   0,
		NotBefore: now,
		CreatedAt: now,
		ReplayOf:  it.ForwardID,
	}, nil
}
