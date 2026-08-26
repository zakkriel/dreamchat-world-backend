# The People — clean-sheet model

My test throughout: would this person still be interesting unaddressed, and can they surprise me?

## 1. The complete model — ten topics

**Identity: the stranger's view vs. the true one.** Two layers per person: how a stranger reads
them on sight, and who they actually are underneath — a name earned only through contact, not
given for free. Without this every NPC is a labeled menu entry. AUTHOR states the first-sight
read and the true identity, as prose. ENGINE derives nothing here except *which layer renders* to
a given onlooker, based on what that onlooker has actually earned. Player feels: a first meeting
is a specific person, not a slot; earning a name is a real event, not a UI unlock.

**Pursuit, immediate and long-standing.** What someone is trying to get, stated as a goal with a
horizon class (right-now / ongoing / long-standing) and what they're presently doing about it —
distinct from temperament, which only describes *how* they act, never *toward what*. Without a
stated pursuit a person only reacts; they never initiate. AUTHOR states the goal, its horizon
class, and the current concrete step toward it. ENGINE derives the *next* step as world-time
passes unobserved — never a new goal, only advancing the authored one. Player feels: NPCs act
without being addressed, and a world left alone for a while has visibly moved.

**Current activity.** One interruptible thing a present person is doing right now, separate from
their standing pursuit. Without it, arriving anywhere triggers a greeting instead of an
interruption — the difference between a set and a place already running. AUTHOR states one
concrete action in progress. ENGINE updates it as time passes between observations, staying
consistent with the person's disposition and pursuit — never re-authored by the model per tick.
Player feels: you walk in on something.

**Relationship — asymmetric, and revisable by what happens.** How A regards B need not equal how
B regards A, and neither is fixed once stated. AUTHOR states an initial stance, per person, in
that person's own terms, grounded in one thing that already happened between them. ENGINE derives
*shifts* in stance only from a specific witnessed or told event during play — never silent drift,
never edited in place; a new stance is a new fact layered over the old, which stays true of the
past. Player feels: they can see why someone is treated oddly, and change it, and the change is
legible, not a hidden number.

**Groups as agents, not labels.** A named collective — however the brief implies one — that wants,
protects, fears, and can respond as itself, not merely tag its members. AUTHOR states the group's
own interest (which need not sum its members' interests), who currently speaks for it, and how
legible membership is to an outsider. ENGINE derives *who acts in this moment* on the group's
behalf from membership and standing — never a scripted institutional response to a hypothetical.
Player feels: crossing one person of a group produces consequence attributable to the group, not
just that one person.

**Obligation, permission, and taboo — felt, not posted.** What is expected or forbidden, who it
binds, and evidence it has already mattered once. AUTHOR states the rule as one sentence, its
binding, and one precedent — someone already rewarded or punished for it — so it isn't inert on
arrival. ENGINE decides, only at the moment a bound person actually witnesses a violation, whether
and how they respond — never pre-scripted. Player feels: an act that would work anywhere else gets
answered by a person here, not a wall.

**Escalation and accumulation.** Repeated, sub-threshold experience compounds toward a visible
break; the tenth small thing lands differently than the first. AUTHOR states nothing directly here
— only enough specificity in a person's disposition that "more of the same" is a coherent,
checkable pattern. ENGINE derives accumulating pressure per person-and-subject from repeated
unresolved instances of one kind, resolving into a visible change only past a threshold set by
that person's own stated temperament. Player feels: patterns matter; a slow burn is real, not
every consequence instant.

**Knowledge and how it travels.** Who knows what, by what path (witnessed, told, overheard,
inferred, common talk), stated in each holder's own words — two witnesses to one event should not
describe it alike. AUTHOR states, for anything already past, who came away knowing what and how.
ENGINE derives further spread and distortion during play strictly along the closed set of paths
already used — never invents a new kind of knowing. Player feels: asking two people about one
night gets two stories, and the gap is where the story lives.

**Mistaken belief — sincere and load-bearing, not a bug.** Someone can be confidently wrong and
stay wrong until something specific corrects them, and act on the wrong belief the whole time.
AUTHOR states at least one person's false belief and what they actually saw that produced it.
ENGINE decides only from a later, real event whether and when it gets corrected — never
spontaneously. Player feels: trusting an account can burn you; correcting someone is a real,
satisfying act, not a lore-dump.

**Reputation — what strangers assume before meeting.** An aggregate, distinct from any one
relationship, built from facts explicitly scoped as public rather than one-to-one told, and just
as capable of being wrong as any belief. AUTHOR states at least one public-scope fact per notable
person or group. ENGINE derives what an unmet stranger assumes from public-scope facts alone, and
lets that assumption trail or outrun the truth. Player feels: people already have an attitude
before a word is exchanged.

## 2. Worked example — the Hold at Verge Ridge

A seed repository above a valley three bad harvests deep, staffed by people who no longer farm.

```json
{
  "world_premise": "A wind-scoured ridge holds the valley's seed stock against a fourth bad year. Everyone here left farming to keep this place running instead.",
  "people": [
    {
      "seen_as": "a woman who never looks up from the ledger",
      "true_name": "Osha Vell",
      "disposition": [
        {"trait": "meticulous", "class": "strong", "shows_as": "recounts a tally twice before speaking it aloud"},
        {"trait": "unforgiving of waste", "class": "strong", "shows_as": "names the exact bushel lost, never 'some'"}
      ],
      "pursuing": [
        {"horizon": "long_standing", "toward": "prove the Hold can survive without its founder", "progress": "advanced", "doing_now": "rebuilding the count her own way, quietly overriding his old method"}
      ],
      "regards": [
        {"toward": "the man who sleeps in the seed loft", "as": "trusted with the work, not with why the count keeps coming up short", "since_event": "the-spring-shortfall"}
      ]
    },
    {
      "seen_as": "a man who sleeps in the seed loft rather than the bunks",
      "true_name": "Corrin Adle",
      "disposition": [
        {"trait": "restless", "class": "moderate", "shows_as": "counts exits before he counts anything else"}
      ],
      "pursuing": [
        {"horizon": "right_now", "toward": "trade a sack of the rarer stock to a river merchant before dawn", "doing_now": "waiting by the loft door for the merchant's whistle"},
        {"horizon": "ongoing", "toward": "leave before anyone connects the shortfalls to him", "progress": "early"}
      ],
      "regards": [
        {"toward": "Osha Vell", "as": "sharper than she lets on; avoid being alone with her ledger", "since_event": "the-spring-shortfall"}
      ]
    }
  ],
  "collectives": [
    {
      "seen_as": "the Hold",
      "wants": "the valley never learns how close the count actually runs",
      "speaks_through": "Osha Vell, by default, unless she is absent",
      "legible": true
    }
  ],
  "norms": [
    {
      "stated": "nobody takes seed stock off the ridge without the count being witnessed by two",
      "binds": "the Hold",
      "precedent_event": "the-loft-boy-dismissed"
    }
  ],
  "history": [
    {
      "what_happened": "this spring's count came up short by more than the weevils could explain",
      "who_knows": [
        {"person": "Osha Vell", "path": "direct", "belief": "somebody is taking stock and covering it in the tally — she has not named who"},
        {"person": "Corrin Adle", "path": "direct", "belief": "he knows exactly who, because it is him"},
        {"person": "the loft boy dismissed last season", "path": "told", "belief": "he was blamed and let go for it, and it was not him — he still doesn't know it was Corrin"}
      ]
    }
  ],
  "reputations": [
    {
      "about": "the Hold",
      "public_belief": "the ridge keeps the valley fed no matter how bad the year — steady people, nothing gets wasted",
      "grounded_in": "public-scope facts only; does not know about the-spring-shortfall"
    }
  ]
}
```

## 3. The three hardest

**Escalation and accumulation.** Compounding sub-threshold experience into a felt break without a
visible score the author (or the player, reverse-engineering it) can game, while keeping every
step traceable to a real instance — this is the topic most likely to collapse into either a hidden
number or a fake one.

**Mistaken belief, corrected only by a real later event.** The chicken-and-egg problem: a false
belief formed *during play* — not authored at genesis — still needs a specific correcting event to
exist before it can be fixed, and nothing should be allowed to invent that event just to resolve
the story neatly. Getting the "never spontaneously" rule to hold under narrative pressure to wrap
things up is genuinely hard.

**Groups as agents.** Deciding who speaks for a collective at a given moment, and making its
response feel institutional rather than one person's opinion, without secretly requiring the group
to have its own mind and its own turn in the same budget every present person already competes
for.

## 4. What multiplayer breaks

**Relationships stop being author-seeded by default.** A second player arriving later has no
authored stance toward anyone — for most pairs in a populated world, "first meeting is the first
event" must become the normal path, not the exception it is when there's one arrival.

**An NPC's knowledge and belief are no longer single-valued.** What a person believes can now
diverge by *which player* told them what and when; two players interacting with the same person on
separate threads must not silently merge into one omniscient memory that leaks one player's
information to the other through the NPC.

**Reputation can go genuinely contested.** Two players causing contradictory public facts about the
same person at close to the same time should be able to produce a real split — part of the valley
believing one thing, part another — rather than forcing one account to silently win.

**Escalation accumulates across sources.** A person's threshold might be crossed by the sum of two
players' independent, individually-harmless actions; the pressure needs multi-source provenance,
not an assumption of one causal thread.

**A group's stance must stay coherent across concurrent scenes.** One player's conversation with a
member cannot be allowed to quietly decide the group's position for a different player's
simultaneous scene with someone else.

**Correcting a mistaken belief is per-thread, not global.** A belief one player fixes for an NPC
may still be the wrong belief as perceived by a second player who hasn't caused that correction
yet — belief state has to be evaluated at the moment of each interaction, not flipped everywhere
the instant any one player resolves it.
