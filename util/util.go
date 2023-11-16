package util

func FilterSlice(slice []string, f func(string) bool) []string {
	filtered := make([]string, 0)
	for _, v := range slice {
		if f(v) {
			filtered = append(filtered, v)
		}
	}
	return filtered
}

func ArrayContains(slice []int32, searchElement int32) bool {
	for _, element := range slice {
		if element == searchElement {
			return true
		}
	}
	return false
}
