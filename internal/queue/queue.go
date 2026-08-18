package queue

import (
	"container/heap"
	"sync"
	"time"

	"github.com/lacsar712/adsbhub/internal/clock"
	"github.com/lacsar712/adsbhub/internal/job"
)

type item struct {
	j     job.Job
	index int
}

type dueHeap []*item

func (h dueHeap) Len() int { return len(h) }

func (h dueHeap) Less(i, j int) bool {
	if h[i].j.NotBefore.Equal(h[j].j.NotBefore) {
		return h[i].j.CreatedAt.Before(h[j].j.CreatedAt)
	}
	return h[i].j.NotBefore.Before(h[j].j.NotBefore)
}

func (h dueHeap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
	h[i].index = i
	h[j].index = j
}

func (h *dueHeap) Push(x any) {
	it := x.(*item)
	it.index = len(*h)
	*h = append(*h, it)
}

func (h *dueHeap) Pop() any {
	old := *h
	n := len(old)
	it := old[n-1]
	old[n-1] = nil
	it.index = -1
	*h = old[:n-1]
	return it
}

type RadarQueue struct {
	ID        string
	Ordered   bool
	inFlight  int
	maxFlight int
	ready     dueHeap
}

func newRadarQueue(id string, ordered bool, maxFlight int) *RadarQueue {
	if maxFlight < 1 {
		maxFlight = 1
	}
	if ordered {
		maxFlight = 1
	}
	dq := &RadarQueue{ID: id, Ordered: ordered, maxFlight: maxFlight}
	heap.Init(&dq.ready)
	return dq
}

type Broker struct {
	mu     sync.Mutex
	clk    clock.Clock
	radars map[string]*RadarQueue
}

func NewBroker(clk clock.Clock) *Broker {
	return &Broker{clk: clk, radars: make(map[string]*RadarQueue)}
}

func (b *Broker) Ensure(id string, ordered bool, maxFlight int) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if _, ok := b.radars[id]; ok {
		return
	}
	b.radars[id] = newRadarQueue(id, ordered, maxFlight)
}

func (b *Broker) Enqueue(j job.Job, ordered bool, maxFlight int) {
	b.mu.Lock()
	defer b.mu.Unlock()
	dq, ok := b.radars[j.RadarID]
	if !ok {
		dq = newRadarQueue(j.RadarID, ordered, maxFlight)
		b.radars[j.RadarID] = dq
	}
	heap.Push(&dq.ready, &item{j: j.Clone()})
}

func (b *Broker) Lease() (job.Job, bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	now := b.clk.Now()
	var best *RadarQueue
	var bestDue time.Time
	for _, dq := range b.radars {
		if dq.inFlight >= dq.maxFlight {
			continue
		}
		if dq.ready.Len() == 0 {
			continue
		}
		head := dq.ready[0].j
		if head.NotBefore.After(now) {
			continue
		}
		if best == nil || head.NotBefore.Before(bestDue) {
			best = dq
			bestDue = head.NotBefore
		}
	}
	if best == nil {
		return job.Job{}, false
	}
	it := heap.Pop(&best.ready).(*item)
	best.inFlight++
	return it.j, true
}

func (b *Broker) Release(radarID string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if dq, ok := b.radars[radarID]; ok && dq.inFlight > 0 {
		dq.inFlight--
	}
}

func (b *Broker) Depth() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	n := 0
	for _, dq := range b.radars {
		n += dq.ready.Len() + dq.inFlight
	}
	return n
}

func (b *Broker) DepthByRadar() map[string]int {
	b.mu.Lock()
	defer b.mu.Unlock()
	out := make(map[string]int, len(b.radars))
	for id, dq := range b.radars {
		out[id] = dq.ready.Len() + dq.inFlight
	}
	return out
}

func (b *Broker) Snapshot() []job.Job {
	b.mu.Lock()
	defer b.mu.Unlock()
	var out []job.Job
	for _, dq := range b.radars {
		for _, it := range dq.ready {
			out = append(out, it.j.Clone())
		}
	}
	return out
}

func (b *Broker) Restore(jobs []job.Job, meta map[string]RadarMeta) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.radars = make(map[string]*RadarQueue)
	for id, m := range meta {
		b.radars[id] = newRadarQueue(id, m.Ordered, m.MaxInFlight)
	}
	for _, j := range jobs {
		dq, ok := b.radars[j.RadarID]
		if !ok {
			dq = newRadarQueue(j.RadarID, false, 2)
			b.radars[j.RadarID] = dq
		}
		heap.Push(&dq.ready, &item{j: j.Clone()})
	}
}

type RadarMeta struct {
	Ordered     bool
	MaxInFlight int
}
