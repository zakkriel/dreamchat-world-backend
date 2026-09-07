# Product observability — the numbers worth watching, and what they do not mean

**Status:** living. One entry per number. Each says what it measures, how to read it, and — the part that
matters — **what it over-counts**, so nobody acts on a figure that looks alarming and is not.

This file exists because a number without its caveat becomes folklore. A count of "altered entities"
that includes *someone learned a name* will one day be quoted as a reason not to ship tiered world
creation, and by then nobody will remember it counted name-learning.

---

## OBS-1 — Gameplay alteration inside the world-creation window

**The question it answers.** World creation at depth 4–5 is intended to run in **tiers**: commit an
early tier so the world exists and can be walked into, and keep authoring the rest in the background.
That opens a race — background authoring works from a snapshot, and gameplay can change canon underneath
it. Founder's example: creation authors two people, play kills one in the first second, and a later tier
happily writes more content about them.

So: **how often does gameplay alter an entity while a build would still be running?**

**How to read it.** Genesis writes canon with `origin = 'fast_path'` (the one documented exception to
`D-1`; templated worlds use `'template'`). Everything else — `freeform`, `ruling`, `threshold`,
`backstage`, `compensation`, `telegraph` — is gameplay. So an entity is "gameplay-altered" if it
participates in a canon event whose origin is neither of the genesis two. **No column and no migration
are needed; this is derivable, and retroactive over every world already stored.**

Run it: `ci/obs_alteration_window.sh`

**Measured 2026-08-30, 11 worlds, of which 5 had ever been played:**

| world | ≤1 min | ≤5 min | ≤19 min | ever |
|---|---|---|---|---|
| The Ironmoor Reach | 0 | 0 | 3 | 3 |
| The Drowned Lantern | 0 | 0 | 2 | 8 |
| The Stone Lantern | 0 | 1 | 1 | 1 |
| Mara 0A Fixture | 0 | 0 | 0 | 2 |

**What it said.** Two things, pulling in opposite directions.

The **blast radius is tiny** — 1 to 3 entities, never more, from worlds born with 4 to 14. A freshness
check at merge would be rejecting a handful of references, not reconciling a world.

The **timing overlaps** — nothing in the first 60 seconds, but three of the five played worlds were
altered well inside a 19-minute window. On a world someone actually plays, this is the common case
rather than a rare race. So the danger is not committing early; it is how long authoring continues
afterwards. A tier that finishes in a minute is effectively safe.

### What this number OVER-COUNTS, and why you must not act on it alone

**Not every alteration is an alteration that matters.** Founder, 2026-08-30:

> we don't even have a real alteration. for example, learning someone's name might be an alteration.
> that is hardly relevant to the world creation next tier

Exactly right, and it is the main defect in this figure. It counts **any** canon event naming the
entity. Someone learning a name, someone speaking, someone glancing at a room — all touch an entity and
**none** of them invalidates what a later tier is about to author.

The alterations that genuinely conflict are the ones that change an entity's **present state**: death,
relocation, destruction, a change of holder. And the distinction is sharper than "severity" — it is the
canon/state split the engine already has:

- **Authoring the PAST of an altered entity is legitimate.** Writing the history of someone killed a
  minute ago is correct, and learning it afterwards is good play. Canon is history; history does not
  change when someone dies.
- **Authoring their PRESENT is the bug.** `starts_in` for a corpse, an object placed in a dead hand, a
  person put in a scene they have left.

So **treat OBS-1 as an upper bound on a risk whose real size is unknown.** Before anyone designs around
it, it needs splitting by whether the event changed present state. Until then it answers "could a
collision have happened" and not "would it have mattered".

### Other caveats, stated rather than buried

- **Sample of five.** Eleven worlds, six never played. Trust the shape, not the ratios.
- **`accepted_at` is wall-clock.** A session paused overnight and resumed inflates a delay; the
  seven-hour and fifteen-day first-touch figures in the raw data are almost certainly that, not
  patience.
- **Entities, not events.** An entity altered ten times counts once.
- **It cannot see the future case it exists for.** Today genesis commits in one transaction and play
  cannot begin until it has, so a *live* window measurement would read zero by construction. This
  number is a retrospective proxy: it asks what gameplay did to worlds of this age, and infers what it
  would do to a build still running.

### When to look at it

When tiered creation ships. The number to watch is not the total but **whether alterations start
appearing inside the first minute** — that would mean the safe window has closed and the freshness rule
has to become strict rather than best-effort.
