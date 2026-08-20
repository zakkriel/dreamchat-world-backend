package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// The catalogue is what a picker renders, so the wire shape is the contract: keys in display order,
// a label and a blurb for each, and nothing else.
func TestArtStylesEndpoint_ServesTheCatalogueInOrder(t *testing.T) {
	rec := httptest.NewRecorder()
	NewWorldArtStylesHandler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/worlds/art-styles", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	var body struct {
		SchemaVersion string `json:"schema_version"`
		Styles        []struct {
			Key   string `json:"key"`
			Label string `json:"label"`
			Blurb string `json:"blurb"`
		} `json:"styles"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.SchemaVersion != "art_styles/1" {
		t.Errorf("schema_version = %q", body.SchemaVersion)
	}

	want := []string{"anime", "realistic", "manhwa", "comic", "3d"}
	if len(body.Styles) != len(want) {
		t.Fatalf("got %d styles, want %d", len(body.Styles), len(want))
	}
	for i, key := range want {
		if body.Styles[i].Key != key {
			t.Errorf("style %d = %q, want %q", i, body.Styles[i].Key, key)
		}
		if body.Styles[i].Label == "" || body.Styles[i].Blurb == "" {
			t.Errorf("%s arrives with nothing to render", key)
		}
	}
}

// The prompt prose is server-side. A client picks by key; what the key MEANS to an image model must
// stay tunable without shipping a frontend, and a leaked prompt is also a leaked latitude.
func TestArtStylesEndpoint_NeverLeaksThePromptProse(t *testing.T) {
	rec := httptest.NewRecorder()
	NewWorldArtStylesHandler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/worlds/art-styles", nil))

	body := rec.Body.String()
	for _, leaked := range []string{"cel shaded", "subsurface scattering", artStyleLatitude, artStyleNegative} {
		if strings.Contains(body, leaked) {
			t.Errorf("the catalogue leaked prompt prose: %q", leaked)
		}
	}
}

// The literal path must not be read as a world id by the routes that follow it.
func TestArtStylesEndpoint_IsNotSwallowedByAnIdRoute(t *testing.T) {
	rt := newRouter(nil, false, nil, nil)

	rec := httptest.NewRecorder()
	rt.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/worlds/art-styles", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 — an id matcher claimed the catalogue path", rec.Code)
	}

	// And it answers only its own path and method.
	for _, req := range []*http.Request{
		httptest.NewRequest(http.MethodPost, "/worlds/art-styles", nil),
		httptest.NewRequest(http.MethodGet, "/worlds/art-styles/extra", nil),
	} {
		rec := httptest.NewRecorder()
		NewWorldArtStylesHandler().ServeHTTP(rec, req)
		if rec.Code == http.StatusOK {
			t.Errorf("%s %s was answered by the catalogue", req.Method, req.URL.Path)
		}
	}
}
