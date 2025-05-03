package util

func FilterArray[T any](array []T, test func(T) bool) []T {
	var output []T
	for _, needle := range array {
		if test(needle) {
			output = append(output, needle)
		}
	}
	return output
}
