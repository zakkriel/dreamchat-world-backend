# Schema stress test — Sueño Común (detailed tier)

## 1. The encoding attempt

```json
{
  "world_model": "1",
  "world": { "name": "Orbe",
    "premise": "A city of twenty-eight registered neighbourhoods. Everyone asleep in the same one enters the identical dream, appears as themselves, and cannot hide what they want. A transcriber writes down what happened; the archive is public and permanent." },

  "vocabulary": {
    "media": [ { "name": "waking air", "descriptor": "ordinary daylight and weight", "resists": [], "affords": [] } ],
    "movements": [ { "name": "walk", "pace_class": "steady" } ],
    "channels": [
      { "name": "sight" }, { "name": "hearsay", "descriptor": "what a neighbour tells another, informally, indefinitely" },
      { "name": "the shared dream", "descriptor": "a nightly collective perception every sleeper in one neighbourhood enters at once, unmasked" } ],
    "conditions": [
      { "name": "waking-projection", "hinders": [ { "channel": "sight", "class": "moderate", "note": "PROJECTS rather than hinders — see report §2" } ] } ],
    "substances": []
  },

  "law": [
    { "name": "nothing hides in the shared dream",
      "stated": "Inside the shared dream nobody can conceal face, body, or age, and what a person wants renders visibly around them.",
      "governs": "hiding" }
  ],

  "places": [
    { "name": "Barrio Doce", "parent": "Orbe", "extent_class": "large", "sort": "neighbourhood", "medium": "waking air", "tension": "normal" },
    { "name": "Sala de lectura", "parent": "Barrio Doce", "extent_class": "small", "medium": "waking air", "tension": "tense" },
    { "name": "Pensión Tarma", "parent": "Barrio Doce", "extent_class": "small", "medium": "waking air", "tension": "normal" },
    { "name": "El Sueño del Doce", "parent": "Barrio Doce", "extent_class": "vast", "medium": "waking air", "tension": "tense",
      "ambient_demand": [ { "requires": "30% of the neighbourhood asleep", "absent_effect": "?", "onset": "?" } ] }
  ],

  "things": [
    { "name": "Rem's list", "bulk_class": "small", "integrity": "sound", "where": { "carried_by": "Rem Salas" } },
    { "name": "Ossen's incomplete volumes", "bulk_class": "moderate", "integrity": "sound", "where": { "in_place": "Barrio Doce" } },
    { "name": "district card", "bulk_class": "small", "integrity": "sound", "where": { "carried_by": "Inspector Bald" } }
  ],

  "processes": [
    { "name": "the forgetting", "acts_on": "?", "direction": "degrade", "rate_class": "fast", "terminus": "erased" }
  ],

  "accumulators": [
    { "name": "Vira Cor's sleeplessness", "stated": "How many nights running she has stayed awake.",
      "starts_at": "low", "raised_by": ["another night without sleep"],
      "threshold": { "class": "high", "then": "she projects continuously; the neighbourhood sees what she thinks" } }
  ],

  "collectives": [
    { "name": "the Watch Office", "legibility": "marked", "interest": "that nobody audits the transcriptions", "speaks_through": "the district's transcriber" },
    { "name": "the Registry of the Solitary", "legibility": "marked", "interest": "keep the count at eleven", "speaks_through": "?" },
    { "name": "the Sleepless", "legibility": "concealed", "interest": "not to be seen", "speaks_through": null },
    { "name": "the Twenty-Ninth", "legibility": "concealed", "interest": "?", "speaks_through": null }
  ],

  "people": [
    { "name": "Rem Salas", "seen_as": "the district's transcriber, twelve years at it", "role": "writes what the neighbourhood dreamed",
      "belongs_to": ["the Watch Office"], "starts_in": "Barrio Doce",
      "capability": { "moves_by": ["walk"], "carry_class": "light" }, "senses": { "sight": "normal" },
      "disposition": [ { "trait": "careful to a fault", "strength": "defining", "manner": "reads a passage back before archiving it" } ],
      "doing": "shelving last night's volume", "pursuing": [ { "horizon": "long_standing", "toward": "never be wrong once", "progress": "advanced" } ],
      "obligation": [ { "owed_to": "the Watch Office", "stated": "decides alone what enters the record" } ],
      "regard": [], "hiding": "eleven prior volumes have things she left out, always to protect someone; the list is at home" },
    { "name": "Onel", "seen_as": "solitary, registrant four", "role": "dreams alone and is seen by no one; sees everyone",
      "belongs_to": ["the Registry of the Solitary"], "starts_in": "Barrio Doce",
      "capability": { "moves_by": ["walk"], "carry_class": "light" }, "senses": { "the shared dream": "acute" },
      "disposition": [ { "trait": "withheld", "strength": "strong", "manner": "answers less than he saw" } ],
      "doing": "waiting out the morning alone, by rule",
      "pursuing": [ { "horizon": "long_standing", "toward": "be released from the registry", "progress": "early", "step": "his fifteenth request, already expecting refusal" } ],
      "obligation": [ { "owed_to": "the Registry", "stated": "cannot resign" } ], "regard": [],
      "hiding": "he can follow one person through the whole night in more detail than he ever declares" },
    { "name": "Inspector Bald", "seen_as": "Watch Office, foreign-sleep section", "role": "investigates sleeping outside your registered district",
      "belongs_to": ["the Watch Office"], "starts_in": "Barrio Doce",
      "capability": { "moves_by": ["walk"], "carry_class": "moderate" }, "senses": { "sight": "normal" },
      "disposition": [ { "trait": "dogged", "strength": "strong", "manner": "has wanted the Pensión closed for six years" } ],
      "doing": "watching the Pensión Tarma from across the street", "pursuing": [ { "horizon": "ongoing", "toward": "close the Pensión Tarma", "progress": "advanced" } ],
      "obligation": [ { "owed_to": "the Watch Office", "stated": "enforces the ban on sleeping outside your registered district" } ],
      "regard": [], "hiding": "his own card says a different district than the one he actually sleeps in, and has for four years" }
  ],

  "opposition": [
    { "between": ["Inspector Bald", "Inspector Bald's own card"], "incompatible": "?",
      "stakes": "the man enforcing this exact law is breaking it" }
  ],

  "norms": [
    { "name": "no sleeping outside your district", "binds": [],
      "stated": "Sleeping in a district you are not registered to is foreign sleep — punished by fine, reassignment, and publication of the record.",
      "precedent": "?" },
    { "name": "nobody asks a solitary what they saw", "binds": [], "stated": "It is not asked.", "precedent": "?" }
  ],

  "history": [
    { "name": "the-dreamed-killing", "what_happened": "?",
      "where": "El Sueño del Doce", "who": ["you"],
      "knowledge": [ { "holder": "you", "channel": "the shared dream", "path": "direct",
        "believes": "I was standing next to a body, in a place I don't recognise, and I don't remember doing anything." } ] }
  ],

  "arrivals": [
    { "premise": "Barrio Doce, thirty-one, no record. Forty thousand people saw you standing over a body last night, in a dream. It hasn't happened. Nobody recognises the place.",
      "seen_as": "yourself, exactly", "place": "Barrio Doce",
      "capability": { "moves_by": ["walk"], "carry_class": "moderate" }, "senses": { "sight": "normal" } }
  ]
}
```

`"?"` marks a value the schema demands but the brief gives no honest answer for — the breaks below.

## 2. Every break

**The shared dream's own existence, gated by a population threshold.** *"Empieza cuando duerme el
30% del barrio. Termina cuando queda menos del 10%."* Section: `places[].ambient_demand`.
**(iii) wrong-shaped.** `absent_effect` only inflicts a *condition* on occupants already there
(the worked example's `chalk-lung`); it cannot toggle whether the place *exists at all*. I could
not fill `absent_effect`/`onset` honestly — a place either is or isn't there depending on how many
people, elsewhere, are asleep.

**Desire made compulsorily visible, and identity un-hideable, inside the dream only.**
*"Nadie puede ocultarse... Lo que uno desea se manifiesta alrededor suyo, visible para todos."*
Section: `law[].governs`. **(iii) wrong-shaped.** `governs` names a property of a *thing* (the
worked example: `integrity`). "Hiding" isn't a thing's property, it's a rule about how the
*epistemic layer itself* behaves inside one channel — the exact inversion of every other world's
default (privacy unless earned). `law[]` has no vocabulary for a policy on perception itself.

**Personal memory erased on a fixed clock, distinct from an indefinite rumour and a permanent
archive.** *"Nadie recuerda más de tres noches atrás... indefinida... permanente."* Section:
`processes[].acts_on`. **(i) inexpressible.** `processes` act on things with `integrity` in a
medium; a person's *recollection* is not a `things[]` entry, it lives in
`history[].knowledge[].believes`, which `processes` cannot reach at all. I left `acts_on` as `"?"`
because there is nothing to point it at.

**Staged, escalating, involuntary broadcast of a specific person's mind, worsening over a named
table (noches 1-3 normal / 4 leve / 5-8 frecuente / 9-13 continua / 14+ colapso).** Section:
`accumulators[]` and `conditions[]`, tried both. **(iii) wrong-shaped, both times.**
`accumulators` has exactly one threshold and one `then`; the brief's table is five ordered stages
with different effects each. `conditions[].hinders` only *hinders* a movement or channel — Vira's
condition does the opposite of hinder, it *broadcasts*, which is a category the vocabulary doesn't
have.

**A norm whose enforcer is himself the ongoing violator.** *"Duerme en el Doce. Su carné dice
Barrio Siete. Lleva cuatro años en falta."* Section: `norms[]`, `people[].obligation`. **(ii)
expressible only as inert prose.** `norms[]` has no `enforced_by`, only `binds`; nothing links
Bald's `obligation` (he enforces this law) to his own violation of the same law as a structural
fact — I had to invent an `opposition[]` entry between a person and his own card, which the schema
was never shaped for (`between` wants two *people*).

**A faction's institutional vulnerability, stated as such.** *"Registro de Solitarios... Débil en:
son once, están identificados, y ninguno quiere estar ahí."* Section: `collectives[]`. **(ii)
inert prose.** `collectives[]` has `interest` but no `vulnerability` — I could not place "eleven,
identified, none of them want this" anywhere structured; it degrades to a fact nobody reads.

**Evidentiary objects that, if read, correct a false belief.** *Rem's list of eleven omissions;
Ossen's forty-one incomplete volumes; the one solitary who predates the reform.* Section:
`things[]`. **(ii) inert prose.** `things[]` carries `bulk_class`/`integrity`/`where` — physical
properties only. Nothing connects an object to the specific `history[].knowledge[]` entries
reading it would resolve. The entire investigation arc the brief structures around (§ Fase 1/2/3)
has no home; it is decoration on top of props that happen to be findable.

**The central hook itself: a mass-witnessed event whose status as fact is exactly what's
undetermined.** *"El sueño no predice... Si no fue predicción, alguien puso una escena adentro."*
Section: `history[]`. **(i) inexpressible, and the sharpest break.** `history[]` entries are
settled past fact by construction — "every perception cites a source event" assumes the event is
known to have happened. Here forty thousand people share one perception and *whether it grounds in
any real event at all is the entire plot.* I left `what_happened` as `"?"` because writing anything
there — "a killing occurred" or "a killing did not occur" — pre-answers the mystery the brief
explicitly forbids answering (*"El caso del usuario es una anomalía y debe seguir siendo la
única"*).

## 3. Minimal fixes — agnostic, two-world tested

**Primitive 1 — an event can be authored with disputed standing.** Give `history[]` entries an
optional `"standing": "occurred" | "disputed"` (default `occurred`, closed vocabulary). A disputed
entry is still a real event *that many people directly perceived* — it just has no confirmed truth
value until a later event resolves it (a new `history[]` entry, never an edit — append-only holds).
*Sueño Común:* `"the-dreamed-killing"` gets `"standing": "disputed"`; play resolves it forward, not
backward. *An unrelated world — a shipping company:* a ledger entry for a lost cargo gets
`"standing": "disputed"` the moment two crews give contradictory accounts, resolved months later
when the cargo turns up, or doesn't.

**Primitive 2 — a channel can declare a concealment policy.** Give `vocabulary.channels[]` an
optional `"conceals": "all" | "identity" | "none"` (default `"all"`, matching every other world's
earned-privacy norm). Inside a channel declared `"none"`, every present person's `pursuing` and
`hiding` render to everyone else automatically — engine-derived from what's already authored, no
new per-fact authoring. *Sueño Común:* `"the shared dream"` channel gets `"conceals": "none"`.
*An unrelated world — a courtroom under oath with a truth-compulsion:* a `"sworn testimony"`
channel gets the same value; outside it, ordinary privacy returns.

## 4. Tier fidelity

The basic tier already carries every hard rule (1-6) that matters mechanically — a sparse world
here is *not* impoverished on rules, it's impoverished on population: four people, no factions, no
objects. That part degrades gracefully.

What the detailed tier reveals is different in kind, not just volume: **collectives with a stated
weakness** (`débil en`), **objects that exist to correct a belief rather than to be carried**, and
**an event whose standing is itself contested** — none of these are "more of what tier 1 already
implied," they're facts the schema has no field for at *any* tier. Numeric detail (400,000
habitantes, 41 tomos, 6 barrios) degrades the same way at every tier regardless, since the schema
forbids raw numbers outright — more precision in the brief buys nothing there. The real finding:
detail doesn't strain capacity, it exposes that three sections (`collectives`, `things`,
`history`) are shallower than the genre-agnostic shape they claim to be.

## 5. The one shape change

**The document assumes one reality.** Every break above is a symptom of the same missing
dimension: `law[]` is one global set, `places[]` is one tree, there is no notion that a *channel*
or a *subtree of places* belongs to a different, concurrently-active rule-layer with its own
defaults for privacy, harm, and persistence — while people move between layers on a schedule and
keep a foothold in both. Add a named **layer** a `law[]` entry, a `places[]` subtree, and a
`channel` can declare membership in (default: the one waking layer everyone already assumes). Not
a field on one section — a scoping dimension that cuts across `law`, `places`, and `vocabulary`
at once. Everything else I found — the gated dream, the inverted privacy, the fixed forgetting —
is one layer's rules leaking into a shape built for exactly one.
