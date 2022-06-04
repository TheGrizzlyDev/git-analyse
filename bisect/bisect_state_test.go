package bisect

import (
	"testing"
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
