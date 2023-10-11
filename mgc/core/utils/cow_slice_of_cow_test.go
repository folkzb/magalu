package utils

import (
	"testing"

	"golang.org/x/exp/slices"
)

func TestCOWSliceOfCOWChildInvalidation(t *testing.T) {
	mockSliceOfSlices := [][]int{{1}, {2}}
	mockCowFactory := func(val []int) *COWSlice[int] {
		return NewCOWSliceComparable(val)
	}
	cowSliceOfCow := NewCOWSliceOfCOW(mockSliceOfSlices, mockCowFactory)
	releasedCowSliceOfCow, changed := cowSliceOfCow.Release()

	if !isEqualSlicesOfSlices(releasedCowSliceOfCow, mockSliceOfSlices) || changed {
		t.Error("Slice should not have changed prior to utilization")
		return
	}

	cowSliceOfCow = NewCOWSliceOfCOW(mockSliceOfSlices, mockCowFactory)

	_, ok := cowSliceOfCow.GetCOW(-1)
	if ok {
		t.Error("GetCOW() with negative index should not have found anything")
	}

	childCow, ok := cowSliceOfCow.GetCOW(0)
	if !ok {
		t.Error("GetCOW() failed with index for existing value")
	}

	childCow.Add(1)
	if childCow.IsChanged() {
		t.Error("Add(1) should not have changed the slice as it already exists")
		return
	}

	childCow.Add(3)
	if !childCow.IsChanged() {
		t.Error("Add(3) should have changed the slice as it does not exist")
		return
	}

	deleted := cowSliceOfCow.Delete(-1)
	if deleted {
		t.Error("Delete() with negative index should not have done anything")
	}

	deleted = cowSliceOfCow.Delete(1)
	if !deleted {
		t.Error("Delete() failed with index for existing value")
	}

	releasedCowSliceOfCow, changed = cowSliceOfCow.Release()
	if !changed {
		t.Error("Add() and Delete() should have changed the slice")
		return
	}

	if isEqualSlicesOfSlices(releasedCowSliceOfCow, mockSliceOfSlices) {
		t.Error("Add() and Delete() should have changed the slice")
	}

	if !isEqualSlicesOfSlices(releasedCowSliceOfCow, [][]int{{1, 3}}) {
		t.Error("Add() and Delete() should have changed the slice")
	}
}

func isEqualSlicesOfSlices[V comparable, S []V](a, b []S) bool {
	if len(a) != len(b) {
		return false
	}

	for i := range a {
		if !slices.Equal(a[i], b[i]) {
			return false
		}
	}

	return true
}
