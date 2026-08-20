package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// A configured timeout is only real if the outbound HTTP call honors it: a slow provider must be
// interrupted by the driver deadline rather than waiting for the full server response.
func TestOpenAICompat_RequestTimeoutBoundsHTTPCall(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(1500 * time.Millisecond)
		_, _ = w.Write([]byte(cannedChatCompletionsBody("late")))
	}))
	defer srv.Close()

	drv, err := newOpenAICompatDriver(DriverConfig{Model: "m", Params: map[string]string{
		"base_url": srv.URL, "request_timeout_seconds": "1",
	}})
	if err != nil {
		t.Fatalf("newOpenAICompatDriver: %v", err)
	}

	start := time.Now()
	_, err = drv.Generate(t.Context(), GenRequest{Prompt: "p"})
	elapsed := time.Since(start)
	if err == nil {
		t.Fatal("Generate succeeded; want timeout error from slow provider")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "timeout") {
		t.Fatalf("err = %v, want timeout-related failure", err)
	}
	if elapsed > 1400*time.Millisecond {
		t.Fatalf("elapsed = %v, want timeout near configured 1s budget", elapsed)
	}
}
