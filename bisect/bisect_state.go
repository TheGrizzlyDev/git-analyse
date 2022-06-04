package bisect

import (
	"fmt"
	"math/rand"
	"sync"
)

type smartRev struct {
	Rev    string
	Cancel chan interface{}
	state  *BisectState
}

func (s *smartRev) Good() {
	s.state.markAsGood(s.Rev)
}

func (s *smartRev) Bad() {
	s.state.markAsBad(s.Rev)
}

type BisectState struct {
	revs []string // immutable

	indexes   map[string]int // written once but it's async
	indexesMu sync.RWMutex

	start   int
	startMu sync.RWMutex

	end   int
	endMu sync.RWMutex

	// bisect tracker
	bisectSteps     []int
	bisectIteration int
	bisectMu        sync.Mutex

	activeListenersMu sync.Mutex
	activeListeners   []*smartRev
}

func NewBisectState(revs []string) *BisectState {
	state := &BisectState{
		revs:        revs,
		end:         len(revs) - 1,
		indexes:     make(map[string]int),
		bisectSteps: make([]int, len(revs)),
	}

	state.initIndexesTable()
	state.initBisectSteps()

	return state
}

func (b *BisectState) initIndexesTable() {
	for i, rev := range b.revs {
		b.indexes[rev] = i
	}
}

func (b *BisectState) initBisectSteps() {
	for i := range b.bisectSteps {
		b.bisectSteps[i] = i
	}

	rand.Seed(1)
	rand.Shuffle(len(b.bisectSteps), func(i, j int) {
		b.bisectSteps[i], b.bisectSteps[j] = b.bisectSteps[j], b.bisectSteps[i]
	})
}

func (b *BisectState) Next() *smartRev {
	b.bisectMu.Lock()
	defer b.bisectMu.Unlock()
	for ; b.bisectIteration < len(b.bisectSteps); b.bisectIteration++ {
		step := b.bisectSteps[b.bisectIteration]
		if step >= b.start && step <= b.end {
			// TODO implement ref cancellation
			rev := b.revs[step]
			b.bisectIteration++
			return &smartRev{
				Rev:    rev,
				Cancel: make(chan interface{}),
				state:  b,
			}
		}
	}
	return nil
}

func (b *BisectState) getIndex(rev string) int {
	b.indexesMu.RLock()
	defer b.indexesMu.RUnlock()
	return b.indexes[rev]
}

func (b *BisectState) markAsGood(rev string) error {
	i := b.getIndex(rev)

	if i <= b.start {
		return nil
	}

	if i >= b.end {
		return fmt.Errorf("TODO(come up with an error msg)")
	}

	b.startMu.Lock()
	defer b.startMu.Unlock()
	b.start = i
	return nil
}

func (b *BisectState) markAsBad(rev string) error {
	i := b.getIndex(rev)

	if i >= b.end {
		return nil
	}

	if i <= b.start {
		return fmt.Errorf("TODO(come up with an error msg)")
	}

	b.endMu.Lock()
	defer b.endMu.Unlock()
	b.end = i
	return nil
}

func (b *BisectState) FirstBadRev() *string {
	b.startMu.RLock()
	defer b.startMu.RUnlock()

	b.endMu.RLock()
	defer b.endMu.RUnlock()

	if b.end-b.start == 1 {
		return &b.revs[b.end]
	}

	return nil
}
