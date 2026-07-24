package main

// tier1Registry is the engine-known closed set (contracts §5.2). It grows only when we add a check in code — never at runtime, never by mint.
var tier1Registry = map[string]string{
	"open":                "boolean",
	"locked":              "boolean",
	"connects":            "array",
	"size":                "number",
	"weight":              "number",
	"max_room":            "number",
	"occupied_room":       "number",
	"empty_weight":        "number",
	"max_load":            "number",
	"carried_weight":      "number",
	"base_speed":          "number",
	"location_id":         "string",
	"coordinates":         "object",
	"tension":             "string",
}
