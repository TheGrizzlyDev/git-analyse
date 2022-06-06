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

	good int
	bad  int

	// bisect tracker
	bisectSteps     []int
	bisectIteration int

	activeListeners map[int]*smartRev

	mu sync.RWMutex

	Done chan string
}

func NewBisectState(revs []string) *BisectState {
	state := &BisectState{
		revs:            revs,
		good:            len(revs) - 1,
		indexes:         make(map[string]int, len(revs)),
		bisectSteps:     make([]int, len(revs)),
		activeListeners: make(map[int]*smartRev, len(revs)),
		Done:            make(chan string),
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

	// Fully random
	rand.Seed(1)
	rand.Shuffle(len(b.bisectSteps), func(i, j int) {
		b.bisectSteps[i], b.bisectSteps[j] = b.bisectSteps[j], b.bisectSteps[i]
	})

	// Middle chunk moved to start and randomized
	// rand.Seed(1)
	// chunk := len(b.bisectSteps) / 4
	// middleChunk := b.bisectSteps[chunk : chunk*3]
	// rand.Shuffle(len(middleChunk), func(i, j int) {
	// 	middleChunk[i], middleChunk[j] = middleChunk[j], middleChunk[i]
	// })
	// b.bisectSteps = append(middleChunk, append(b.bisectSteps[:chunk-1], b.bisectSteps[chunk*3+1:]...)...)
}

func (b *BisectState) Next() *smartRev {
	b.mu.Lock()
	defer b.mu.Unlock()

	for ; b.bisectIteration < len(b.bisectSteps); b.bisectIteration++ {
		step := b.bisectSteps[b.bisectIteration]
		if step >= b.bad && step <= b.good {

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

	if i >= b.good {
		return nil
	}

	b.good = i

	defer b.notifyActiveListeners()
	return nil
}

func (b *BisectState) markAsBad(rev string) error {
	i := b.getIndex(rev)

	b.mu.Lock()
	defer b.mu.Unlock()

	if i <= b.bad {
		return nil
	}

	b.bad = i

	defer b.notifyActiveListeners()
	return nil
}

func (b *BisectState) notifyActiveListeners() {
	for i := range b.activeListeners {
		if i < b.bad || i > b.good {
			defer delete(b.activeListeners, i)
			b.activeListeners[i].Cancel <- struct{}{}
		}
	}

	if (b.good - b.bad) <= 1 {
		b.Done <- b.revs[b.bad]
	}
}

func (b *BisectState) Stats() *BisectStats {
	b.mu.RLock()
	defer b.mu.RUnlock()
	revs := len(b.revs)
	left := revs - b.bisectIteration
	if left < 0 {
		left = 0
	}
	return &BisectStats{
		Pending: len(b.activeListeners),
		Left:    left,
		Total:   revs,
	}
}
