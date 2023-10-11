package utils

import (
	"testing"

	"golang.org/x/exp/maps"
)

func TestNewCOWMapOfCOWChildInvalidation(t *testing.T) {
	mockMapOfMaps := map[string]map[int]int{"test_set": {1: 1}, "test_delete": {2: 2}}
	mockCowFactory := func(value map[int]int) *COWMap[int, int] {
		return NewCOWMapComparable(value)
	}
	cowMapOfCow := NewCOWMapOfCOW(mockMapOfMaps, mockCowFactory)
	releasedCowMapOfCow, changed := cowMapOfCow.Release()

	if !isEqualMapsOfMaps(releasedCowMapOfCow, mockMapOfMaps) || changed {
		t.Error("Map should not have changed prior to utilization")
		return
	}

	cowMapOfCow = NewCOWMapOfCOW(mockMapOfMaps, mockCowFactory)

	_, ok := cowMapOfCow.GetCOW("missing")
	if ok {
		t.Error("GetCOW() with missing key should not have found anything")
	}

	childCow, ok := cowMapOfCow.GetCOW("test_set")
	if !ok {
		t.Error("GetCOW() failed with key for existing value")
	}

	childCow.Set(1, 1)
	if childCow.IsChanged() {
		t.Error("Set(1, 1) should not have changed the map as it already exists")
		return
	}

	childCow.Set(3, 3)
	if !childCow.IsChanged() {
		t.Error("Set(3, 3) should have changed the map as it does not exist")
		return
	}

	deleted := cowMapOfCow.Delete("missing")
	if deleted {
		t.Error("Delete() with missing key should not have done anything")
	}

	deleted = cowMapOfCow.Delete("test_delete")
	if !deleted {
		t.Error("Delete() failed with key for existing value")
	}

	releasedCowMapOfCow, changed = cowMapOfCow.Release()
	if !changed {
		t.Error("Set() and Delete() should have changed the map")
		return
	}

	if isEqualMapsOfMaps(releasedCowMapOfCow, mockMapOfMaps) {
		t.Error("Set() and Delete() should have changed the map")
	}

	if !isEqualMapsOfMaps(releasedCowMapOfCow, map[string]map[int]int{"test_set": {1: 1, 3: 3}}) {
		t.Error("Set() and Delete() should have changed the map")
	}
}

func isEqualMapsOfMaps[M comparable, K comparable, V comparable](a, b map[M]map[K]V) bool {
	if len(a) != len(b) {
		return false
	}

	for keyA, valueA := range a {
		valueB := b[keyA]
		if !maps.Equal(valueA, valueB) {
			return false
		}
	}

	return true
}
