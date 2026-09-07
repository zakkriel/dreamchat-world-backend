package main

// Governed-by: B-1 — every surface renders from the holder's perception, never raw canon. Also I-3.
// Promoted from this file's own citations (2026-08-26), not newly decided. Change what this
// file decides and those decisions change with it (D-9).

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

// NamingWall is the mechanical enforcement of naming reach (RULINGS-2026-07-23 §3, B-1, I-3): the set
// of canonical names this viewer has NOT earned, and the label he holds instead.
//
// The wall was prompt-discipline until the founder caught it failing in play — narration reading
// "Jonas planted between her and the room" to a player who has only ever perceived "the muscle by the
// bar". The assembly seam is fixed at its source (migration 20260809090005 renders perception content
// per holder), and this is the belt: the world KNOWS which names a viewer has not earned, so a
// player-facing string containing one is a checkable defect rather than a matter of trust.
//
// Hearing is not identification: a listener can genuinely hear a canonical name spoken without
// learning who it belongs to (the shared speech-perception design's ruling). The guarded set is
// therefore narrower than "every unearned name" — a name this viewer's own recorded perceived
// speech already contains literally is not a leak merely for being quoted back. The LABEL a viewer
// holds is untouched either way (fn_display_name, not this wall, decides it): a name surviving in
// a quote unidentified never becomes an identity, so its owner still renders as his descriptor
// everywhere else.
//
// Two uses, deliberately different:
//   - Violations() during narration validation — a seat that leaks gets REJECTED and asked again,
//     because a model can rewrite the sentence better than any substitution can.
//   - Scrub() at the emit boundary and on seat text with no retry loop (NPC telegraphs) — the last
//     resort, deterministic, never letting the breach reach the player even if every attempt leaked.
type NamingWall struct {
	re     *regexp.Regexp    // (?i)\b(name|name|…)\b — nil when the viewer has earned everything
	labels map[string]string // lower(canonical) → the label this viewer actually holds
}

// loadNamingWall reads the unheard names for one viewer. Called once per beat: the set changes when
// the viewer learns a name (a canon event) or hears one spoken for the first time.
//
// "Unheard" (fn_unheard_names) layers one exemption onto "unearned" (fn_unearned_names: no
// knowledge path, not the viewer himself, and the label he holds does not already contain the name
// — the "ballast crate" case, migration 20260809090006): a canonical name already present,
// literally, in this viewer's own recorded perceived speech is removed from the guarded set. Hearing
// a name is not learning whose it is, so fn_viewer_text (the perception-content rendering seam)
// deliberately keeps the stricter fn_unearned_names — an ACCOUNT of an event is not a live quote of
// it, and only the quote gets the hearing exemption.
func loadNamingWall(ctx context.Context, pool *pgxpool.Pool, worldID, viewerID string) (*NamingWall, error) {
	// fn_unheard_names is built on fn_unearned_names, the SAME identity definition the perception
	// seam (fn_viewer_text) rewrites from — restating that predicate here is what produced the
	// "ballast crate" false positive in the first hour, so the belt checks the seam's own shared
	// function rather than a second copy of the rule.
	rows, err := pool.Query(ctx,
		`SELECT canonical_name, label FROM fn_unheard_names($1, $2::uuid)`,
		worldID, viewerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	w := &NamingWall{labels: map[string]string{}}
	var names []string
	for rows.Next() {
		var canon, label string
		if err := rows.Scan(&canon, &label); err != nil {
			return nil, err
		}
		names = append(names, canon)
		w.labels[strings.ToLower(canon)] = label
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(names) == 0 {
		return w, nil // a viewer who has earned every name: Violations is empty, Scrub is identity
	}

	// Already longest-first from fn_unearned_names, so "Hooded Companion" is matched before "Hooded"
	// can bite into it — the ORDER BY is part of the shared definition, not an incidental detail.
	quoted := make([]string, len(names))
	for i, n := range names {
		quoted[i] = regexp.QuoteMeta(n)
	}
	// \b so a name never matches inside a longer word ("jonasberry" is not Jonas); (?i) because prose
	// capitalises at a sentence start and models are inconsistent about it.
	w.re = regexp.MustCompile(`(?i)\b(` + strings.Join(quoted, "|") + `)\b`)
	return w, nil
}

// Violations returns the unearned names present in text, de-duplicated, in first-appearance order.
// Empty (and nil-safe) when the text is clean.
func (w *NamingWall) Violations(text string) []string {
	if w == nil || w.re == nil || text == "" {
		return nil
	}
	var out []string
	seen := map[string]bool{}
	for _, m := range w.re.FindAllString(text, -1) {
		k := strings.ToLower(m)
		if !seen[k] {
			seen[k] = true
			out = append(out, m)
		}
	}
	return out
}

// Scrub rewrites every unearned name into the label the viewer holds. Deterministic and total: after
// Scrub the text cannot breach the wall, whatever the model wrote.
//
// Case is not preserved from the match — the label is world data ("the muscle by the bar") and is
// written as stored. A capital at a sentence start is a cosmetic loss and the only alternative is
// guessing at grammar; the wall is worth a lowercase article.
func (w *NamingWall) Scrub(text string) string {
	if w == nil || w.re == nil || text == "" {
		return text
	}
	return w.re.ReplaceAllStringFunc(text, func(m string) string {
		if label, ok := w.labels[strings.ToLower(m)]; ok && label != "" {
			return label
		}
		return m
	})
}

// scrubAll applies Scrub across a slice in place, for seat text that reaches the player with no
// validation loop of its own (NPC telegraph wind-ups). Returns the names it had to remove so the
// caller can log a breach that the source seam should have prevented.
func (w *NamingWall) scrubAll(texts []string) []string {
	if w == nil || w.re == nil {
		return nil
	}
	var breached []string
	for i, t := range texts {
		if v := w.Violations(t); len(v) > 0 {
			breached = append(breached, v...)
			texts[i] = w.Scrub(t)
		}
	}
	return breached
}

// namingWallError is the validation rejection: it names the offending words so the repair prompt can
// tell the model exactly what it may not say.
func namingWallError(i int, names []string) error {
	return fmt.Errorf("segment %d names %s, which this viewer has not earned — refer to them only as "+
		"they are known to him", i, strings.Join(quoteEach(names), ", "))
}

func quoteEach(ss []string) []string {
	out := make([]string, len(ss))
	for i, s := range ss {
		out[i] = `"` + s + `"`
	}
	return out
}
