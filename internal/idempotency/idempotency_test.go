package idempotency_test

import (
	"errors"
	"testing"
	"time"

	"github.com/lacsar712/adsbhub/internal/clock"
	"github.com/lacsar712/adsbhub/internal/idempotency"
)

func TestRememberReplayAndConflict(t *testing.T) {
	clk := clock.NewFrozen(time.Unix(0, 0))
	s := idempotency.New(clk, time.Hour)
	id, replay, err := s.Remember("key-aaaa", "hash1", "rpt1")
	if err != nil || replay || id != "rpt1" {
		t.Fatalf("first: %s replay=%v err=%v", id, replay, err)
	}
	id, replay, err = s.Remember("key-aaaa", "hash1", "rpt2")
	if err != nil || !replay || id != "rpt1" {
		t.Fatalf("replay: %s replay=%v err=%v", id, replay, err)
	}
	_, _, err = s.Remember("key-aaaa", "hash2", "rpt3")
	if !errors.Is(err, idempotency.ErrConflict) {
		t.Fatalf("want conflict, got %v", err)
	}
	clk.Advance(2 * time.Hour)
	id, replay, err = s.Remember("key-aaaa", "hash2", "rpt4")
	if err != nil || replay || id != "rpt4" {
		t.Fatalf("after ttl: %s replay=%v err=%v", id, replay, err)
	}
}
