package main

import (
	"encoding/json"
	"net/http"
	"regexp"
)

// GET /worlds/art-styles — the looks a world may be created in.
//
// It exists so the picker is not a hardcoded list in a frontend. The catalogue is going to be reused
// by every surface that ever offers a style — creation, a per-character override, a regenerate-in-a-
// different-look button — and a second copy of "anime means cel shaded" in a client is how those
// surfaces start disagreeing with what a world already chose. One module answers, everyone reads it.
//
// WHAT IT DELIBERATELY DOES NOT SEND: the prompt prose. ArtStyle keeps its look unexported, so
// marshalling a catalogue entry can only ever emit key, label and blurb. A client renders what a
// style is CALLED and picks by key; what that key means to an image model stays server-side, where
// it can be tuned without shipping a frontend.
//
// No world id in the path and no auth-scoped data in the answer: this is a fixed list, identical for
// everyone, and asking for it before you have made a world is the normal case.
var worldArtStylesRoute = regexp.MustCompile(`^/worlds/art-styles$`)

type worldArtStylesHandler struct{}

func NewWorldArtStylesHandler() http.Handler { return &worldArtStylesHandler{} }

func (h *worldArtStylesHandler) Match(r *http.Request) bool {
	return r.Method == http.MethodGet && worldArtStylesRoute.MatchString(r.URL.Path)
}

func (h *worldArtStylesHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !h.Match(r) {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"schema_version": "art_styles/1",
		"styles":         ArtStyleCatalogue(),
	})
}
