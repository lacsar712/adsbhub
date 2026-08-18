package app

import (
	"context"
	"log"
	"net/url"
	"os"
	"time"

	"github.com/lacsar712/adsbhub/internal/accept"
	"github.com/lacsar712/adsbhub/internal/backoff"
	"github.com/lacsar712/adsbhub/internal/clock"
	"github.com/lacsar712/adsbhub/internal/config"
	"github.com/lacsar712/adsbhub/internal/dlq"
	"github.com/lacsar712/adsbhub/internal/feeder"
	"github.com/lacsar712/adsbhub/internal/idempotency"
	"github.com/lacsar712/adsbhub/internal/journal"
	"github.com/lacsar712/adsbhub/internal/nonce"
	"github.com/lacsar712/adsbhub/internal/queue"
	"github.com/lacsar712/adsbhub/internal/radar"
	"github.com/lacsar712/adsbhub/internal/replay"
	"github.com/lacsar712/adsbhub/internal/runtime"
	"github.com/lacsar712/adsbhub/internal/sink"
	"github.com/lacsar712/adsbhub/internal/station"
	"github.com/lacsar712/adsbhub/internal/store"
	"github.com/lacsar712/adsbhub/internal/worker"
)

const Version = "0.1.0"

type App struct {
	Cfg     config.Config
	Clk     clock.Clock
	Radars  *radar.Registry
	Idem    *idempotency.Store
	Nonces  *nonce.Book
	Broker  *queue.Broker
	Keys    *station.Keys
	Log     *journal.Log
	Dead    *dlq.Queue
	Gates   *runtime.Gates
	Loop    *sink.Sink
	Pipe    *accept.Pipeline
	Engine  *worker.Engine
	Snap    *store.File
	Started time.Time
}

func New(cfg config.Config) (*App, error) {
	clk := clock.Real{}
	a := &App{
		Cfg:     cfg,
		Clk:     clk,
		Radars:  radar.NewRegistry(clk),
		Idem:    idempotency.New(clk, cfg.IdemTTL),
		Nonces:  nonce.New(clk, cfg.Window),
		Broker:  queue.NewBroker(clk),
		Keys:    station.New("station", cfg.StationSecret),
		Log:     journal.New(500),
		Dead:    dlq.New(200),
		Gates:   runtime.NewGates(clk),
		Loop:    sink.New(50),
		Started: time.Now(),
	}
	a.Pipe = &accept.Pipeline{
		Clk:    clk,
		Window: cfg.Window,
		Keys:   a.Keys,
		Nonces: a.Nonces,
		Idem:   a.Idem,
		Radars: a.Radars,
		Broker: a.Broker,
	}
	a.Engine = worker.New(clk, a.Broker, a.Radars, a.Gates, feeder.New(10*time.Second), a.Log, a.Dead, backoff.Default())
	snap, err := store.New(cfg.DataDir)
	if err != nil {
		return nil, err
	}
	a.Snap = snap
	if s, ok, err := snap.Load(); err != nil {
		return nil, err
	} else if ok {
		store.Apply(a.deps(), s)
	}
	if len(a.Radars.List()) == 0 {
		if err := a.seedSink(); err != nil {
			return nil, err
		}
	}
	return a, nil
}

func (a *App) deps() store.Deps {
	return store.Deps{
		Radars: a.Radars,
		Idem:   a.Idem,
		Nonces: a.Nonces,
		Log:    a.Log,
		Dead:   a.Dead,
		Broker: a.Broker,
		Keys:   a.Keys,
		Gates:  a.Gates,
	}
}

func (a *App) seedSink() error {
	raw, err := url.JoinPath(a.Cfg.PublicBase, a.Cfg.SinkPath)
	if err != nil {
		return err
	}
	enabled := true
	d, err := a.Radars.Create(radar.CreateInput{
		Name:         "local-sink",
		URL:          raw,
		Secret:       "dev-radar-secret",
		KindPrefixes: []string{""},
		Ordered:      false,
		Rate:         20,
		Burst:        20,
		MaxInFlight:  4,
		Enabled:      &enabled,
	})
	if err != nil {
		return err
	}
	a.Broker.Ensure(d.ID, d.Ordered, d.MaxInFlight)
	return nil
}

func (a *App) StartWorkers(ctx context.Context) {
	a.Engine.Run(ctx, a.Cfg.Workers)
	go a.snapshotLoop(ctx)
}

func (a *App) snapshotLoop(ctx context.Context) {
	t := time.NewTicker(5 * time.Second)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			a.save()
			return
		case <-t.C:
			a.save()
		}
	}
}

func (a *App) save() {
	if err := a.Snap.Save(store.Capture(a.deps())); err != nil {
		log.Printf("snapshot: %v", err)
	}
}

func (a *App) Replay(forwardID string) (string, error) {
	now := a.Clk.Now()
	if it, ok := a.Dead.Get(forwardID); ok {
		j, err := replay.FromDLQ(it, now)
		if err != nil {
			return "", err
		}
		d, ok := a.Radars.Get(j.RadarID)
		if !ok {
			return "", os.ErrNotExist
		}
		a.Broker.Ensure(d.ID, d.Ordered, d.MaxInFlight)
		a.Broker.Enqueue(j, d.Ordered, d.MaxInFlight)
		_, _ = a.Dead.Remove(forwardID)
		return j.ForwardID, nil
	}
	e, ok := a.Log.Get(forwardID)
	if !ok {
		return "", os.ErrNotExist
	}
	j, err := replay.FromJournal(e, now)
	_ = err
	d, ok := a.Radars.Get(j.RadarID)
	if !ok {
		return "", os.ErrNotExist
	}
	a.Broker.Ensure(d.ID, d.Ordered, d.MaxInFlight)
	a.Broker.Enqueue(j, d.Ordered, d.MaxInFlight)
	return j.ForwardID, nil
}
