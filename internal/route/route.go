package route

import (
	"time"

	"github.com/lacsar712/adsbhub/internal/idgen"
	"github.com/lacsar712/adsbhub/internal/radar"
)

type PlanItem struct {
	RadarID   string
	ForwardID string
	URL       string
	Ordered   bool
}

type Plan struct {
	ReportID   string
	Kind       string
	Items      []PlanItem
	DroppedOff int
}

func Fanout(reportID, kind string, body []byte, radars []radar.Radar, now time.Time) Plan {
	_ = body
	p := Plan{ReportID: reportID, Kind: kind, Items: make([]PlanItem, 0, len(radars))}
	for _, d := range radars {
		if !d.Matches(kind) {
			p.DroppedOff++
			continue
		}
		p.Items = append(p.Items, PlanItem{
			RadarID:   d.ID,
			ForwardID: idgen.New("fwd", now),
			URL:       d.URL,
			Ordered:   d.Ordered,
		})
	}
	return p
}
