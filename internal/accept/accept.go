package accept

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/lacsar712/adsbhub/internal/clock"
	"github.com/lacsar712/adsbhub/internal/hashutil"
	"github.com/lacsar712/adsbhub/internal/headers"
	"github.com/lacsar712/adsbhub/internal/idempotency"
	"github.com/lacsar712/adsbhub/internal/idgen"
	"github.com/lacsar712/adsbhub/internal/job"
	"github.com/lacsar712/adsbhub/internal/nonce"
	"github.com/lacsar712/adsbhub/internal/queue"
	"github.com/lacsar712/adsbhub/internal/radar"
	"github.com/lacsar712/adsbhub/internal/report"
	"github.com/lacsar712/adsbhub/internal/route"
	"github.com/lacsar712/adsbhub/internal/sign"
	"github.com/lacsar712/adsbhub/internal/station"
)

type Pipeline struct {
	Clk    clock.Clock
	Window time.Duration
	Keys   *station.Keys
	Nonces *nonce.Book
	Idem   *idempotency.Store
	Radars *radar.Registry
	Broker *queue.Broker
}

type Result struct {
	ReportID   string   `json:"report_id"`
	Replay     bool     `json:"replay"`
	Matched    int      `json:"matched"`
	ForwardIDs []string `json:"forward_ids"`
}

func (p *Pipeline) Handle(h http.Header, body []byte) (Result, int, error) {
	in, err := headers.ParseInbound(h)
	if err != nil {
		return Result{}, http.StatusBadRequest, err
	}
	if err := hashutil.ValidIdempotencyKey(in.IdemKey); err != nil {
		return Result{}, http.StatusBadRequest, err
	}
	secrets := p.Keys.Secrets(in.StationKey)

	if err := sign.Verify(p.Clk, p.Window, secrets, sign.Headers{
		Timestamp: in.Timestamp,
		Nonce:     in.Nonce,
		Signature: in.Signature,
	}, body); err != nil {
		if errors.Is(err, sign.ErrSkew) {
			return Result{}, http.StatusBadRequest, err
		}
		return Result{}, http.StatusUnauthorized, err
	}
	env, err := report.Parse(body)
	if err != nil {
		var syn *json.SyntaxError
		if errors.As(err, &syn) {
			return Result{}, http.StatusBadRequest, err
		}
		return Result{}, http.StatusUnprocessableEntity, err
	}
	if err := p.Nonces.CheckAndRemember(in.Nonce); err != nil {
		return Result{}, http.StatusConflict, err
	}
	now := p.Clk.Now()
	reportID := idgen.New("rpt", now)
	bodyHash := hashutil.SHA256Hex(body)
	existing, replay, err := p.Idem.Remember(in.IdemKey, bodyHash, reportID)
	if err != nil {
		if errors.Is(err, idempotency.ErrConflict) {
			return Result{}, http.StatusConflict, err
		}
		return Result{}, http.StatusBadRequest, err
	}
	if replay {
		return Result{ReportID: existing, Replay: true}, http.StatusOK, nil
	}
	matched := p.Radars.Matching(env.Kind)
	plan := route.Fanout(reportID, env.Kind, body, matched, now)
	ids := make([]string, 0, len(plan.Items))
	for _, item := range plan.Items {
		d, ok := p.Radars.Get(item.RadarID)
		if !ok {
			continue
		}
		p.Broker.Ensure(d.ID, d.Ordered, d.MaxInFlight)
		p.Broker.Enqueue(job.Job{
			ReportID:  reportID,
			ForwardID: item.ForwardID,
			RadarID:   item.RadarID,
			Kind:      env.Kind,
			Body:      append([]byte(nil), body...),
			Attempt:   0,
			NotBefore: now,
			CreatedAt: now,
		}, d.Ordered, d.MaxInFlight)
		ids = append(ids, item.ForwardID)
	}
	return Result{
		ReportID:   reportID,
		Matched:    len(ids),
		ForwardIDs: ids,
	}, http.StatusAccepted, nil
}
