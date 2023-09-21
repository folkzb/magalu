package cmd

func splitSlice[T comparable](s []T, sep T) [][]T {
	if s == nil {
		return nil
	}

	result := [][]T{}
	addNewSlice := true

	for _, v := range s {
		if v == sep {
			addNewSlice = true
			continue
		}

		if addNewSlice {
			result = append(result, []T{})
			addNewSlice = false
		}

		lastSliceIdx := len(result) - 1
		result[lastSliceIdx] = append(result[lastSliceIdx], v)
	}

	return result
}
