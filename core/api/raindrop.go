package main

// raindrop.go — Raindrop/Workshop observability for the beat loop.
//
// One process-wide client, initialized once in main(). It is deliberately KEYLESS-safe: the SDK
// resolves a local Workshop daemon (RAINDROP_LOCAL_DEBUGGER / RAINDROP_WORKSHOP env, else a one-time
// 100ms probe of 127.0.0.1:5899 at New()) and mirrors telemetry there during development; with no
// RAINDROP_WRITE_KEY and no local Workshop it is a complete no-op. Telemetry can therefore never
// break a boot or a beat — the same fail-open posture the naming wall load takes.
//
// The client lives in a package var rather than being threaded through NewBeatsStreamHandler
// because ~20 existing test callsites construct that handler directly; tests never run main(), so
// the var stays nil there and every use is nil-guarded — instrumentation is strictly additive.

import (
	"context"
	"log"
	"os"
	"time"

	raindrop "github.com/raindrop-ai/go"
)

// raindropClient is set once in main() and read (nil-guarded) by the beat handler.
var raindropClient *raindrop.Client

// initRaindrop builds the client and returns its shutdown func. Errors disable telemetry loudly
// (one log line) instead of failing the boot.
func initRaindrop() func() {
	client, err := raindrop.New(
		raindrop.WithWriteKey(os.Getenv("RAINDROP_WRITE_KEY")),
	)
	if err != nil {
		log.Printf("raindrop: telemetry disabled: %v", err)
		return func() {}
	}
	raindropClient = client
	log.Printf("raindrop: telemetry enabled=%v (workshop mirror auto-detected when local)", client.Enabled())
	return func() { _ = client.Close() }
}

// rdInteractionKey threads the beat's interaction through the request context so the seat
// decorator (bridge.go's timedDriver) can hang per-seat spans off it without widening the Driver
// interface — the same "instrument once in the binding path" reasoning timedDriver itself documents.
type rdInteractionKey struct{}

func withInteraction(ctx context.Context, in *raindrop.Interaction) context.Context {
	return context.WithValue(ctx, rdInteractionKey{}, in)
}

func interactionFrom(ctx context.Context) *raindrop.Interaction {
	in, _ := ctx.Value(rdInteractionKey{}).(*raindrop.Interaction)
	return in
}

// trackSeatCall records ONE seat LLM call on the beat's interaction — seat name, bound model, the
// full prompt in, the raw model text (or error) out, and the real duration. Nil-safe: a context
// with no interaction (tests, non-beat callers like world creation) is a no-op.
func trackSeatCall(ctx context.Context, seat, model, prompt, out string, start time.Time, err error) {
	in := interactionFrom(ctx)
	if in == nil {
		return
	}
	opts := raindrop.TrackToolOptions{
		Name:       "seat:" + seat,
		Input:      prompt,
		Duration:   time.Since(start),
		Error:      err,
		Properties: map[string]any{"seat": seat, "model": model},
	}
	if err == nil {
		opts.Output = out
	}
	in.TrackTool(opts)
}
