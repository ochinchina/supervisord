package util

// Sub returns a slice with the elements from arr1 that are absent from arr2.
func Sub(arr1, arr2 []string) []string {
	result := make([]string, 0)
	for _, s := range arr1 {
		exist := false
		for _, s2 := range arr2 {
			if s == s2 {
				exist = true
			}
		}
		if !exist {
			result = append(result, s)
		}
	}
	return result
}

// ElementsMatchString returns true if arr1 and arr2 have the same elements without regard for order.
func ElementsMatchString(arr1, arr2 []string) bool {
	if len(arr1) != len(arr2) {
		return false
	}
	for _, s := range arr1 {
		exist := false
		for _, s2 := range arr2 {
			if s2 == s {
				exist = true
				break
			}
		}
		if !exist {
			return false
		}
	}
	return true
}
