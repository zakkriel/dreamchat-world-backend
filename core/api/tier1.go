package main

// tier1Registry is the engine-known closed set (contracts §5.2). It grows only when we add a check in code — never at runtime, never by mint.
var tier1Registry = map[string]string{
	"open":               "boolean",
	"locked":             "boolean",
	"connects":           "array",
	"size":               "number",
	"weight":             "number",
	"max_room":           "number",
	"occupied_room":      "number",
	"empty_weight":       "number",
	"max_load":           "number",
	"carried_weight":     "number",
	"base_speed":         "number",
	"location_id":        "string",
	"coordinates":        "object",
	"parent_location_id": "string", // a location's parent edge in the nested-coordinate hierarchy (§3); Tier-1 string, read by fn_distance/fn_location_depth
	"tension":            "string",
	"contained_by":       "string", // the carry edge (§4): contents of X = artifacts whose contained_by = X; read by fn_effective_weight/fn_occupied_room and the eager encumbrance rule. Carry MUST be Tier-1 — the engine reads it (Rule A).
	"weight_modifier":    "number", // a container's weight modifier (§4/§5.2): effective_weight = (empty_weight + Σ contents) × weight_modifier; Tier-1 because fn_effective_weight reads it (absent → 1, a mundane container).
}
