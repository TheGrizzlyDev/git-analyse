package bisect

import (
	"testing"
	"time"
)

func TestNextReturnsNilWhenDone(t *testing.T) {
	revs := []string{
		"ebaf211260",
		"1c05d39abc",
		"416b0374fb",
	}
	target := NewBisectState(revs)

	for i := 0; i < len(revs); i++ {
		nextRev := target.Next()

		if nextRev == nil {
			t.Fatal("expected a revision but instead got nil")
		}
	}

	if target.Next() != nil {
		t.Fatal("unexpected revision after extracting all of the revs")
	}
}

func TestCancelsSmartRevWhenOutOfCurrentSearchInterval(t *testing.T) {
	revs := []string{
		"ebaf211260",
		"1c05d39abc",
		"416b0374fb",
	}
	target := NewBisectState(revs)

	_ = target.Next()
	rev_416b0374fb := target.Next()
	rev_1c05d39abc := target.Next()

	go func() {
		rev_1c05d39abc.Bad()
	}()

	select {
	case <-rev_416b0374fb.Cancel:
	case <-time.NewTimer(time.Second).C:
		t.Fatal("test timed out before revision was cancelled")
	}
}
