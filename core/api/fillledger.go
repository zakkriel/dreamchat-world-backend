package main

// fillledger.go — where a fill's tokens actually went, per work item, measured rather than reasoned.
//
// WHY THIS EXISTS. Asked to explain why a depth-1 world takes ~19 minutes, I could give the total
// (47,545 output tokens, 21 calls) and nothing else, so I reasoned from the total instead of measuring —
// which is arguing from the number I was asked to explain. The per-call log lines that would have
// answered it were both misattributed (see below) and gone within the hour, because Railway's log
// retention is shorter than a build.
//
// AND THE OLD NUMBERS WERE WRONG, NOT JUST MISSING. costsink.go carries an explicit warning: the
// per-seat token counts are a DELTA across the call, "exact while seats run sequentially… If seats are
// ever parallelised, this must become a per-call value threaded out of the driver — the failure would be
// silent misattribution." The layered fill runs waves of calls at once. So every per-call tok_out logged
// since that change has been some arbitrary share of whatever finished during that window. This replaces
// the delta with a per-call value, which is what that comment asked for.
//
// WHAT IT MEASURES. One row per provider call, tagged with the work item that made it: output tokens,
// input tokens, how many of those were cache hits, wall time, and the outcome. The outcome is the point —
// a retried or re-asked call produces tokens that are DISCARDED, and until now those were invisible in a
// total by construction.

import (
	"context"
	"fmt"
	"log"
	"sort"
	"strings"
	"sync"
	"time"
)

// callUsage is one provider call's own bill, threaded out of the driver rather than differenced from a
// shared running total. Created per call by the seat decorator, so two calls in flight cannot mix.
type callUsage struct {
	mu     sync.Mutex
	usd    float64
	in     int64
	out    int64
	cached int64
	seen   bool
}

type callUsageKey struct{}

func withCallUsage(ctx context.Context) (context.Context, *callUsage) {
	u := &callUsage{}
	return context.WithValue(ctx, callUsageKey{}, u), u
}

func callUsageFrom(ctx context.Context) *callUsage {
	u, _ := ctx.Value(callUsageKey{}).(*callUsage)
	return u // nil is fine; every method is nil-safe
}

func (u *callUsage) add(usd float64, in, out, cached int64) {
	if u == nil {
		return
	}
	u.mu.Lock()
	u.usd += usd
	u.in += in
	u.out += out
	u.cached += cached
	u.seen = true
	u.mu.Unlock()
}

func (u *callUsage) read() (usd float64, in, out, cached int64, seen bool) {
	if u == nil {
		return 0, 0, 0, 0, false
	}
	u.mu.Lock()
	defer u.mu.Unlock()
	return u.usd, u.in, u.out, u.cached, u.seen
}

// fillRow is one call, and `Outcome` is the column that matters: `kept` tokens are in the document,
// everything else was paid for and thrown away.
type fillRow struct {
	Item    string // the work item that made the call, e.g. "people:The Tally"
	Outcome string // kept | retried | reasked | failed
	Ms      int64
	In      int64
	Cached  int64
	Out     int64
	USD     float64
}

type fillLedger struct {
	mu    sync.Mutex
	rows  []fillRow
	start time.Time
}

type fillLedgerKey struct{}

func withFillLedger(ctx context.Context) (context.Context, *fillLedger) {
	l := &fillLedger{start: time.Now()}
	return context.WithValue(ctx, fillLedgerKey{}, l), l
}

func fillLedgerFrom(ctx context.Context) *fillLedger {
	l, _ := ctx.Value(fillLedgerKey{}).(*fillLedger)
	return l
}

func (l *fillLedger) record(r fillRow) {
	if l == nil {
		return
	}
	l.mu.Lock()
	l.rows = append(l.rows, r)
	l.mu.Unlock()
}

// markLast retags the most recent row for an item. A retry or a re-ask only becomes known to be waste
// AFTER the next call succeeds, so the row is written optimistically and corrected here.
func (l *fillLedger) markLast(item, outcome string) {
	if l == nil {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	for i := len(l.rows) - 1; i >= 0; i-- {
		if l.rows[i].Item == item {
			l.rows[i].Outcome = outcome
			return
		}
	}
}

// workTag names the work item a call belongs to. Threaded through the context so the seat decorator
// stays the only instrument — an instrument you have to remember to add at each call site is an
// instrument that will be missing from the next one.
type workTagKey struct{}

func withWorkTag(ctx context.Context, tag string) context.Context {
	return context.WithValue(ctx, workTagKey{}, tag)
}

func workTagFrom(ctx context.Context) string {
	s, _ := ctx.Value(workTagKey{}).(string)
	return s
}

// report is the line that answers "where did the time go". Grouped by work item KIND (the id before the
// colon), because "people" is the question and "people:The Tally" is one instance of it.
func (l *fillLedger) report() string {
	if l == nil {
		return ""
	}
	l.mu.Lock()
	rows := append([]fillRow{}, l.rows...)
	wall := time.Since(l.start)
	l.mu.Unlock()
	if len(rows) == 0 {
		return ""
	}

	type agg struct {
		calls, wasted       int
		out, in, cached, ms int64
		usd                 float64
		wastedOut           int64
	}
	byKind := map[string]*agg{}
	var order []string
	total := agg{}
	for _, r := range rows {
		kind := r.Item
		if i := strings.IndexByte(kind, ':'); i > 0 {
			kind = kind[:i]
		}
		if kind == "" {
			kind = "(untagged)"
		}
		a := byKind[kind]
		if a == nil {
			a = &agg{}
			byKind[kind] = a
			order = append(order, kind)
		}
		a.calls++
		a.out += r.Out
		a.in += r.In
		a.cached += r.cachedOr()
		a.ms += r.Ms
		a.usd += r.USD
		total.calls++
		total.out += r.Out
		total.in += r.In
		total.cached += r.cachedOr()
		total.usd += r.USD
		if r.Outcome != "kept" {
			a.wasted++
			a.wastedOut += r.Out
			total.wasted++
			total.wastedOut += r.Out
		}
	}
	sort.Slice(order, func(i, j int) bool { return byKind[order[i]].out > byKind[order[j]].out })

	var sb strings.Builder
	fmt.Fprintf(&sb, "fill ledger: wall=%.0fs calls=%d out=%d in=%d cached=%d usd=%.4f discarded_out=%d (%.0f%%)\n",
		wall.Seconds(), total.calls, total.out, total.in, total.cached, total.usd,
		total.wastedOut, pct(total.wastedOut, total.out))
	for _, kind := range order {
		a := byKind[kind]
		fmt.Fprintf(&sb, "fill ledger:   %-12s calls=%-3d out=%-7d (%4.1f%% of output) cached_in=%3.0f%% discarded=%d call(s)/%d tok\n",
			kind, a.calls, a.out, pct(a.out, total.out), pct(a.cached, a.in), a.wasted, a.wastedOut)
	}
	return strings.TrimRight(sb.String(), "\n")
}

func (r fillRow) cachedOr() int64 { return r.Cached }

func pct(part, whole int64) float64 {
	if whole == 0 {
		return 0
	}
	return float64(part) / float64(whole) * 100
}

// logReport writes the ledger where it will survive: one line per work-item kind, at the end of the
// fill. Railway keeps logs for well under an hour, so a summary emitted once is worth more than perfect
// per-call lines nobody can read back.
func (l *fillLedger) logReport() {
	if s := l.report(); s != "" {
		for _, line := range strings.Split(s, "\n") {
			log.Print(line)
		}
	}
}
