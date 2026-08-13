package main

import (
	"context"
	"math"
	"os"
	"strconv"
	"sync"
	"sync/atomic"
)

// costsink.go — what a beat actually costs, in dollars, measured rather than modelled.
//
// Written after a $20 OpenRouter balance vanished and answering "where did it go" took hours of
// forensics: the per-seat instrument logged `ms` and `chars` and never money, so the only spend number
// anyone could quote was an account-wide total that mixed in an unrelated project. (It had: this key
// had spent $0.71 of the $20.) A beat that cannot say what it cost cannot be budgeted, and an engine
// that bills real money should not need an external dashboard to report its own spend.
//
// The provider already sends the number. OpenRouter returns `usage.cost` — the true charge for that
// call, after provider selection and cache discounts — when the request asks for it. This captures it,
// attributes it to the seat that spent it, totals it per beat, and keeps a process-lifetime running
// sum so a long session can be watched without adding a database table.
type costSink struct {
	mu    sync.Mutex
	usd   float64
	in    int64
	out   int64
	calls int
}

type costSinkKey struct{}

// sessionUSD is the process-lifetime total: what this server instance has spent since boot. Atomic
// bits rather than a mutex because every beat reads it and it is only ever added to.
var sessionUSD atomic.Uint64 // float64 bits

func addSessionUSD(d float64) float64 {
	for {
		old := sessionUSD.Load()
		nv := math.Float64frombits(old) + d
		if sessionUSD.CompareAndSwap(old, math.Float64bits(nv)) {
			return nv
		}
	}
}

func withCostSink(ctx context.Context) (context.Context, *costSink) {
	s := &costSink{}
	return context.WithValue(ctx, costSinkKey{}, s), s
}

func costSinkFrom(ctx context.Context) *costSink {
	s, _ := ctx.Value(costSinkKey{}).(*costSink)
	return s // nil is fine: a direct unit call has no sink and every method below is nil-safe
}

// add records one provider call. Called by the driver, which is the only place that sees the bill.
func (c *costSink) add(usd float64, in, out int64) {
	if c == nil {
		return
	}
	c.mu.Lock()
	c.usd += usd
	c.in += in
	c.out += out
	c.calls++
	c.mu.Unlock()
	addSessionUSD(usd)
}

// snapshot reads the running totals. The per-seat log line takes one before and after a call and
// reports the difference: the driver knows the cost but not which seat it was, and the seat wrapper
// knows the seat but not the cost.
//
// Delta attribution is exact while seats run sequentially, which they do today (one call at a time
// per beat). If seats are ever parallelised, this must become a per-call value threaded out of the
// driver — noted here because the failure would be silent misattribution between seats, never a wrong
// beat total.
func (c *costSink) snapshot() (usd float64, in, out int64, calls int) {
	if c == nil {
		return 0, 0, 0, 0
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.usd, c.in, c.out, c.calls
}

// beatCostWarnUSD is the per-beat ceiling that turns a quiet overspend into a loud log line. A beat
// measured at ~$0.015 crossing 5c means something pathological — a repair storm, a prompt that grew,
// a seat silently routed to an expensive model — and the founder should hear about it from the engine
// rather than from his balance. 0 disables.
func beatCostWarnUSD() float64 {
	if v := os.Getenv("DREAMCHAT_BEAT_COST_WARN_USD"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		}
	}
	return 0.05
}

// sessionTotalUSD is what this server instance has spent since boot.
func sessionTotalUSD() float64 { return math.Float64frombits(sessionUSD.Load()) }
