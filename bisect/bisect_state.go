package bisect

import (
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

	indexes map[string]int

	start int

	end int

	// bisect tracker
	bisectSteps     []int
	bisectIteration int

	activeListeners map[int]*smartRev

	mu sync.RWMutex
}

func NewBisectState(revs []string) *BisectState {
	state := &BisectState{
		revs:            revs,
		end:             len(revs) - 1,
		indexes:         make(map[string]int, len(revs)),
		bisectSteps:     make([]int, len(revs)),
		activeListeners: make(map[int]*smartRev, len(revs)),
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
	b.mu.Lock()
	defer b.mu.Unlock()

	if (b.end - b.start) <= 1 {
		return nil
	}

	for ; b.bisectIteration < len(b.bisectSteps); b.bisectIteration++ {
		step := b.bisectSteps[b.bisectIteration]
		if step >= b.start && step <= b.end {

			b.bisectIteration++
			rev := &smartRev{
				Rev:    b.revs[step],
				Cancel: make(chan interface{}),
				state:  b,
			}

			b.activeListeners[step] = rev

			return rev
		}
	}
	return nil
}

func (b *BisectState) getIndex(rev string) int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.indexes[rev]
}

func (b *BisectState) markAsGood(rev string) error {
	i := b.getIndex(rev)

	b.mu.Lock()
	defer b.mu.Unlock()

	if i <= b.start {
		return nil
	}

	b.start = i

	defer b.notifyActiveListeners()
	return nil
}

func (b *BisectState) markAsBad(rev string) error {
	i := b.getIndex(rev)

	b.mu.Lock()
	defer b.mu.Unlock()

	if i >= b.end {
		return nil
	}

	b.end = i

	defer b.notifyActiveListeners()
	return nil
}

func (b *BisectState) notifyActiveListeners() {
	for i := range b.activeListeners {
		if i < b.start || i > b.end {
			defer delete(b.activeListeners, i)
			b.activeListeners[i].Cancel <- struct{}{}
		}
	}
}

func (b *BisectState) FirstBadRev() *string {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if (b.end - b.start) <= 1 {
		return &b.revs[b.end]
	}

	return nil
}
