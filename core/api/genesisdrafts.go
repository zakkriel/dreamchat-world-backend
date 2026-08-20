package main

// genesisdrafts.go — the between-phases home of an authored-but-uncommitted world.
//
// A build now pauses for the user's kickstart answers, so the genesis document must live
// somewhere between authoring and commit. It lives HERE, in memory, and nowhere else: the
// doc holds every secret and every knowledge path, so it never crosses the wire (spec AC-7),
// and it is not worth a table because an abandoned build is simply lost — the posture the
// PRD already takes for watched builds. Claim semantics (remove-on-read) make concurrent
// requests on one handle race-free without holding a lock across a seat call or a commit.

import (
	"crypto/rand"
	"errors"
	"fmt"
	"sync"
	"time"
)

const genesisDraftTTL = 15 * time.Minute

var errDraftExpired = errors.New("that build has expired — write the brief again and rebuild")

type kickstartIdentity struct {
	Descriptor    string `json:"descriptor"`
	CanonicalName string `json:"canonical_name"`
}

type kickstartScenario struct {
	Label       string `json:"label"`
	Place       string `json:"place"`
	Why         string `json:"why"`
	Stated      string `json:"stated"`
	Recommended bool   `json:"recommended,omitempty"`
}

type draftTally struct {
	usd           float64
	tokIn, tokOut int64
	cached        int64
	calls         int
}

func (t *draftTally) add(usd float64, in, out, cached int64, calls int) {
	t.usd += usd
	t.tokIn += in
	t.tokOut += out
	t.cached += cached
	t.calls += calls
}

type genesisDraft struct {
	doc       *genesisDoc
	brief     string
	artStyle  string
	identity  *kickstartIdentity  // nil until the character question is answered
	scenarios []kickstartScenario // authored options awaiting the scenario answer
	tally     draftTally
	deadline  time.Time
}

type draftStore struct {
	mu  sync.Mutex
	ttl time.Duration
	m   map[string]*genesisDraft
}

func newDraftStore(ttl time.Duration) *draftStore {
	return &draftStore{ttl: ttl, m: make(map[string]*genesisDraft)}
}

func (s *draftStore) mint() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(err) // crypto/rand failing means the process has no entropy; nothing sensible continues
	}
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

func (s *draftStore) put(handle string, d *genesisDraft) {
	s.mu.Lock()
	defer s.mu.Unlock()
	d.deadline = time.Now().Add(s.ttl)
	s.m[handle] = d
}

func (s *draftStore) claim(handle string) (*genesisDraft, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	for h, d := range s.m { // lazy sweep: expired drafts leave on any access
		if now.After(d.deadline) {
			delete(s.m, h)
		}
	}
	d, ok := s.m[handle]
	if !ok {
		return nil, false
	}
	delete(s.m, handle)
	return d, true
}
