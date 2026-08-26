# A — The world as a living system

**Thesis.** A model built from nouns produces a diorama. A world is a set of **processes that need no
observer**; people participate in them, they are not their engine. My test throughout: if every
character froze, what still happens, and how does a player find out?

**One grammar rule everything obeys:** the author states a **direction, a class, and a terminus**. The
engine owns rate, quantity and instant. No author ever writes a unit, a period or a number.

## 1. The model

**1. Frame & Extent.** Places nest; each states a size class and what contains it. *Impoverished
without:* "over there" has no cost, so travel is a menu pick. *Author:* containment, size class, what
sort of place this is in its own words. *Engine:* coordinates, distance, travel time, nearest shared
parent. *Table:* leaving costs something, and two players can be genuinely far apart.

**2. Medium.** What you are immersed in *here* — thick air, flood, dark, vacuum, ash, crushing noise.
It modifies every action attempted in it, for everyone, with no actor involved. *Impoverished
without:* all worlds share one physics and differ only in adjectives. This is how worlds differ
physically **without a genre taxonomy**: a medium is an invented word with stated resistances, never a
type from a list. *Author:* each place's medium; what each resists and affords, as classes. *Engine:*
the modifier, and whether an attempt is stopped outright. *Table:* the same act is easy in one room
and impossible one door away.

**3. Barrier & Passage.** A separation states what it resists; a mover states what it moves by;
passage is the comparison. *Impoverished without:* doors are open/closed booleans and no world can
have a wall that stops one thing and not another. *Author:* what each barrier resists, what each way
affords. *Engine:* the comparison, and the refusal. *Table:* the map is different for different
bodies, and finding the way through is play.

**4. Matter & Stock — the world's accounting, discrete and continuous.** Discrete things state bulk,
volume and integrity classes; continuous quantities — water, fuel, feed, breathable air, usable light
— state where they sit, an abundance band, and who may draw. *Impoverished without:* scarcity is a
word in the prose, and possession means nothing if nothing is finite. *Author:* classes and locations.
*Engine:* numbers, arithmetic, the refusal when it runs out. *Table:* taking a thing costs capacity;
drinking the water means it is not there later.

**5. Process.** A named ongoing change: direction, rate class, what it acts on, and a terminus.
Spread, drain, ripen, corrode, heal, silt up. **Decay is the default process on anything with
integrity** — a world that only accumulates is a hoard. *Impoverished without:* the world is a still
photograph between player actions; nothing is true *while* you deliberate. *Author:* the process, its
direction and rate class, what ends it. *Engine:* ticking and resulting state. *Table:* hesitating
costs something.

**6. Cycle.** A process that returns: named ordered **phases** and a period class. Not a clock and
never a calendar — the author states phases, never "twice a day". *Impoverished without:* the world
has no rhythm older than the player, and nothing can be *waited for*. *Author:* phases in order, a
period class, what each changes. *Engine:* when the phase flips, and the event recording it. *Table:*
the world has a heartbeat you learn to read, and timing becomes a skill.

**7. Threshold & Escalation.** A process crossing a line becomes a named event. Escalation is a
threshold that raises its own process's rate or arms another. *Impoverished without:* accumulation
stays a number and never becomes drama; the world cannot have a crisis. *Author:* the line, stated as
a change in the world's condition, and what it arms. *Engine:* when it trips, once, and in what order
against every other. *Table:* the moment the situation stops being stable — and you can see it coming.

**8. Hazard.** A place property that acts on whoever is present, with nobody deciding to — cold,
current, glare, blight. Some are instant; some accrue with exposure. *Impoverished without:* only
people can hurt you, so every danger is a negotiation. *Author:* what it inflicts, intensity class,
presence or exposure. *Engine:* accrual and the condition imposed. *Table:* a place is dangerous by
being a place, and the engine still only removes options, never awards.

**9. Trace.** Every change leaves a *physical* residue that is itself perceivable, and traces age:
scorch, wear-line, empty shelf, silt, absence. *Impoverished without:* the past is only testimony, so
a player who was not there can only be told. **The single most load-bearing topic for multiplayer.**
*Author:* which changes leave what mark, and how it ages. *Engine:* the trace's state and what reading
it yields. *Table:* you enter a room and learn what happened without anyone telling you.

**10. Propagation.** A change happens somewhere and spreads at a rate — instantly, at travel speed,
or never leaving the room. *Impoverished without:* the world is a shared newsfeed and distance means
nothing. *Author:* a propagation class per kind of change. *Engine:* where and when it is knowable.
*Table:* you can be the only one who knows, and that is worth something.

**11. Epoch.** The structurally different past: a prior state where some process ran that no longer
runs, some stock was abundant that is now scarce, some passage existed that is now shut.
*Impoverished without:* history is narration, not structure — the ruins mean nothing mechanically.
*Author:* prior epochs and what differed, in the same vocabulary as now. *Engine:* the derivation of
the present, and which traces survive. *Table:* the world is older than its own explanation, and
there is something to reconstruct.

**12. Standing State.** The world's condition, readable at any moment without replay: each cycle's
phase, each stock's band, which processes run, which thresholds are armed. *Impoverished without:*
nobody can arrive late. *Author:* genesis state only. *Engine:* everything after. *Table:* a player
arriving at any point walks into a world mid-sentence, not at chapter one.

Seams to other seats: bodies differ by what they *move by* and *resist* (3, 8); knowledge enters
through traces and propagation (9, 10); social rules are barriers whose comparison is social (3).

## 2. Worked example — **The Sift**

A settlement on a plain of drifting sand-glass. A scouring wind-front, the Grind, crosses on a cycle
and strips everything above knee height. People live in the *lee* of standing wrecks — and because the
Grind turns, the lee moves. Between passes there is a lull, the only time anyone travels or repairs.

```json
{
  "world": {
    "name": "The Sift",
    "premise": "A plain of drifting sand-glass under a scouring wind that turns. Shelter is a shadow that moves."
  },

  "epochs": [
    { "name": "Before the turning", "what_differed": [
      { "topic": "cycle", "subject": "the Grind", "then": "held one quarter for generations" },
      { "topic": "stock", "subject": "condensate", "then": "abundant", "now": "thin" } ],
      "surviving_traces": ["the old lee-town, now scoured to floor plates"] }
  ],

  "frames": [
    { "name": "The Sift", "contains": null, "extent": "vast", "sort": "open plain" },
    { "name": "The Hulk", "contains": "The Sift", "extent": "large", "sort": "grounded wreck",
      "medium": "settled air" },
    { "name": "Windward Yard", "contains": "The Hulk", "extent": "medium", "sort": "open yard",
      "medium": "abrasive air" },
    { "name": "The Deep Hold", "contains": "The Hulk", "extent": "small", "sort": "sealed hold",
      "medium": "settled air" }
  ],

  "media": [
    { "name": "abrasive air", "resists": [
        { "to": "unshielded skin", "degree": "severe" },
        { "to": "speech", "degree": "total" },
        { "to": "sight", "degree": "moderate" } ],
      "affords": [ { "to": "glass-cloth masks", "degree": "full" } ] },
    { "name": "settled air", "resists": [], "affords": [ { "to": "everything", "degree": "full" } ] }
  ],

  "barriers": [
    { "name": "the shutter wall", "between": ["Windward Yard", "The Deep Hold"],
      "resists": ["abrasive air", "bulk above hand-carried"], "affords": ["a person, stooping"],
      "integrity": "worn" }
  ],

  "stocks": [
    { "name": "condensate", "held_in": "The Deep Hold", "abundance": "thin",
      "drawn_by": "anyone present", "replenished_by": "night-catch" },
    { "name": "glass-cloth", "held_in": "The Hulk", "abundance": "adequate", "drawn_by": "anyone present" }
  ],

  "cycles": [
    { "name": "the Grind", "period": "short",
      "phases": [
        { "name": "the lull", "changes": [
            { "medium_of": "Windward Yard", "becomes": "settled air" },
            { "process": "scour", "becomes": "stopped" } ] },
        { "name": "the rise", "changes": [ { "process": "scour", "becomes": "running" } ] },
        { "name": "the pass", "changes": [
            { "medium_of": "Windward Yard", "becomes": "abrasive air" },
            { "process": "scour", "becomes": "running", "rate": "fast" } ] },
        { "name": "the turning", "changes": [
            { "note": "the sheltered side of every frame changes to the opposite face" } ] } ] }
  ],

  "processes": [
    { "name": "scour", "acts_on": "anything with integrity in abrasive air",
      "direction": "degrade", "rate": "fast", "terminus": "the lull" },
    { "name": "night-catch", "acts_on": "condensate", "direction": "replenish",
      "rate": "slow", "terminus": "when the holding is full" },
    { "name": "drift", "acts_on": "barriers on the windward face",
      "direction": "bury", "rate": "slow", "terminus": null }
  ],

  "thresholds": [
    { "name": "the shutter fails", "when": "the shutter wall's integrity falls past ruined",
      "becomes": [ { "medium_of": "The Deep Hold", "becomes": "abrasive air" } ],
      "arms": ["the hold is lost"] },
    { "name": "the hold is lost", "when": "condensate falls past empty",
      "becomes": [ { "note": "no one can remain in The Hulk between lulls" } ] }
  ],

  "hazards": [
    { "name": "scouring", "in_medium": "abrasive air", "inflicts": "flensed",
      "intensity": "severe", "mode": "exposure" }
  ],

  "traces": [
    { "of": "scour", "leaves": "a bright polished face on the windward side", "ages": "slowly" },
    { "of": "drawing condensate", "leaves": "a fallen mark on the catch-wall", "ages": "never" },
    { "of": "a person passing the shutter", "leaves": "glass dust inside the sill", "ages": "by the next pass" }
  ],

  "propagation": [
    { "of": "phase change", "spreads": "everywhere at once" },
    { "of": "a threshold tripping", "spreads": "at travel speed" },
    { "of": "a stock being drawn", "spreads": "never leaves the frame" }
  ],

  "standing_state": {
    "cycle_phases": { "the Grind": "the lull" },
    "stock_bands": { "condensate": "thin", "glass-cloth": "adequate" },
    "processes_running": ["night-catch", "drift"],
    "thresholds_armed": ["the shutter fails"]
  }
}
```

If every character froze: the shutter keeps wearing, the drift keeps burying it, condensate keeps
falling because people already drew from it, the Grind turns and the safe half of the world moves to
the other side of the wreck. A player notices because the room they sheltered in last time is now the
windward one.

## 3. The three hardest

**Cycles without a calendar.** Every author describes a cycle in units, and every unit is a trap: a
period in hours is a fact with no generating event, and it silently commits the product to a clock
model wrong for most worlds. The compiler must take "twice a day" and keep only *the phases and their
order* — discarding the number without discarding the meaning — and the phase flip must itself be a
real event that leaves a trace, or nothing about it can be remembered.

**Composing classes.** A process at rate *slow*, through a medium resisting *severely*, during a phase
at *peak*, on matter of *worn* integrity. Four authored classes must compose into one outcome with no
author-supplied number and no engine opinion about genre. Wrong in one direction and nothing crosses a
threshold in a session; wrong in the other and every world burns down in five beats. This decides
whether depth is felt or merely present.

**What happens where nobody is.** Ticking every process everywhere is unbounded. Ticking only where
someone stands makes the world a stage set and breaks multiplayer outright — a second player's region
must have moved on while they were away. The honest answer is deriving state on demand from the last
recorded event, but it must stay append-only and traceable without writing an event per tick per
process. Long absences, late arrivals and asynchronous players all rest on this one.

## 4. What multiplayer changes

**Stocks are the first shared truth.** Depletion is the cleanest statement that another person is
real: you drank it, it is gone, and I find the band lower without being told. Every stock needs a
contention rule — simultaneous draws serialize, and the loser perceives that someone got there first.

**Traces are the main channel between players who never meet.** Most multiplayer here is archaeology:
A acts, B reads the residue much later. That makes trace *ageing* a design problem, not flavour — a
trace that never fades makes the world a logbook; one that fades too fast makes other players ghosts.

**Propagation turns lag into content.** Two players can hold contradictory beliefs both true where and
when they formed. It is the most interesting property multiplayer offers, and it means no world-level
state may be globally readable by default.

**Standing State becomes mandatory, not a convenience.** A player arriving at any point must be handed
what the world *is*, derived, without replaying its history and without violating their earned
knowledge.

**Thresholds need global ordering.** Two players crossing one line at once is a race the engine must
serialize into a single event with one winner, both perceiving the same trip.

**Cycles become the session rhythm.** The lull is when players can meet and trade; the pass pins them
apart. A world's cycle is in practice its multiplayer schedule, which makes period classes a social
design decision, not only a physical one.
