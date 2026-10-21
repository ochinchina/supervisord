package util

import "slices"

// HasAllElements returns true if the array arr1 contains all elements of array arr2
func HasAllElements(arr1 []any, arr2 []any) bool {
	for _, e2 := range arr2 {
		if !slices.Contains(arr1, e2) {
			return false
		}
	}
	return true
}

// StringArrayToInterfacArray converts []string to []interface
func StringArrayToInterfacArray(arr []string) []any {
	result := make([]any, 0)
	for _, s := range arr {
		result = append(result, s)
	}
	return result
}

// Sub returns all the element in arr1 but not in arr2
func Sub(arr1 []string, arr2 []string) []string {
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

// IsSameStringArray returns true if arr1 and arr2 has exactly same elements without order care
func IsSameStringArray(arr1 []string, arr2 []string) bool {
	if len(arr1) != len(arr2) {
		return false
	}
	for _, s := range arr1 {
		exist := slices.Contains(arr2, s)
		if !exist {
			return false
		}
	}
	return true
}
