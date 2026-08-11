package main

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
// Two uses, deliberately different:
//   - Violations() during narration validation — a seat that leaks gets REJECTED and asked again,
//     because a model can rewrite the sentence better than any substitution can.
//   - Scrub() at the emit boundary and on seat text with no retry loop (NPC telegraphs) — the last
//     resort, deterministic, never letting the breach reach the player even if every attempt leaked.
type NamingWall struct {
	re     *regexp.Regexp    // (?i)\b(name|name|…)\b — nil when the viewer has earned everything
	labels map[string]string // lower(canonical) → the label this viewer actually holds
}

// loadNamingWall reads the unearned names for one viewer. Called once per beat: the set changes only
// when the viewer learns a name, which is itself a canon event.
//
// "Unearned" is defined once, in SQL (fn_unearned_names): no knowledge path, not the viewer himself,
// and the label he holds does not already contain the name. That last clause is why "the ballast
// crate" is not a breach of "Ballast Crate" — see migration 20260809090006, which was written after
// the belt rejected exactly that sentence in live play.
func loadNamingWall(ctx context.Context, pool *pgxpool.Pool, worldID, viewerID string) (*NamingWall, error) {
	// fn_unearned_names is the SHARED definition — the same function the perception seam
	// (fn_viewer_text) rewrites from. Restating the predicate here is what produced the "ballast
	// crate" false positive in the first hour: the belt must check the seam against the seam's own
	// rule, or it is checking something else.
	rows, err := pool.Query(ctx,
		`SELECT canonical_name, label FROM fn_unearned_names($1, $2::uuid)`,
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
