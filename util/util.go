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
