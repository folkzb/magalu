package utils

import (
	"slices"

	"golang.org/x/exp/constraints"
)

type MapEntry[K comparable, V any] struct {
	Key   K
	Value V
}

func SortedMapIterator[K constraints.Integer | string, V any](m map[K]V) []MapEntry[K, V] {
	if len(m) == 0 {
		return nil
	}
	pairs := make([]MapEntry[K, V], 0, len(m))
	for k, v := range m {
		pairs = append(pairs, MapEntry[K, V]{Key: k, Value: v})
	}

	slices.SortFunc(pairs, func(a, b MapEntry[K, V]) int {
		if a.Key < b.Key {
			return -1
		}
		if a.Key > b.Key {
			return 1
		}
		return 0
	})
	return pairs
}
