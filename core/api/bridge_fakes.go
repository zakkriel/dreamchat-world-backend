package main

import (
	"context"
	"fmt"
)

// fakeStructuredDriver models CONSTRAINED DECODING for CI: it can ONLY return schema-valid chains
// (from its table) or an empty chain — it CANNOT emit out-of-vocab. This is the deterministic CI
// stand-in for the decompose leash (the live equivalent is Anthropic strict tool-use).
type fakeStructuredDriver struct {
	name  string
	table map[string]string // prompt → schema-valid chain JSON
}

func NewFakeStructuredDriver(name string, table map[string]string) Driver {
	if table == nil {
		table = map[string]string{}
	}
	return &fakeStructuredDriver{name: name, table: table}
}

func (f *fakeStructuredDriver) Name() string                { return f.name }
func (f *fakeStructuredDriver) Capabilities() CapabilitySet { return CapabilitySet{CapStructuredOutput: true} }

func (f *fakeStructuredDriver) Generate(_ context.Context, req GenRequest) (string, error) {
	if req.Schema == nil {
		return "", fmt.Errorf("%s: structured driver used without a schema", f.name)
	}
	if out, ok := f.table[req.Prompt]; ok {
		return out, nil
	}
	return "[]", nil // unknown prose → empty chain (commits nothing, C-5); never out-of-vocab
}

// fakeTextDriver: free text only; reports NO capabilities. It CANNOT bind to the decompose seat
// (capability floor fails closed) — the structural proof that out-of-vocab can't even be attempted
// through a non-constrained model.
type fakeTextDriver struct{ name string }

func NewFakeTextDriver(name string) Driver { return &fakeTextDriver{name: name} }

func (f *fakeTextDriver) Name() string                { return f.name }
func (f *fakeTextDriver) Capabilities() CapabilitySet { return CapabilitySet{} }

func (f *fakeTextDriver) Generate(_ context.Context, req GenRequest) (string, error) {
	if req.Schema != nil {
		return "", fmt.Errorf("%s: cannot do structured generation (no capability)", f.name)
	}
	out := "Scene:"
	for _, l := range req.Payload.Lines {
		out += " " + l
	}
	return out, nil
}
