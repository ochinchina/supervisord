package util

// return true if the elem is in the array arr
func InArray(elem interface{}, arr []interface{}) bool {
	for _, e := range arr {
		if e == elem {
			return true
		}
	}
	return false
}

//return true if the array arr1 contains all elements of array arr2
func HasAllElements(arr1 []interface{}, arr2 []interface{}) bool {
	for _, e2 := range arr2 {
		if !InArray(e2, arr1) {
			return false
		}
	}
	return true
}

func StringArrayToInterfacArray(arr []string) []interface{} {
	result := make([]interface{}, 0)
	for _, s := range arr {
		result = append(result, s)
	}
	return result
}

func Sub(arr_1 []string, arr_2 []string) []string {
	result := make([]string, 0)
	for _, s := range arr_1 {
		exist := false
		for _, s2 := range arr_2 {
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

func IsSameStringArray(arr_1 []string, arr_2 []string) bool {
	if len(arr_1) != len(arr_2) {
		return false
	}
	for _, s := range arr_1 {
		exist := false
		for _, s2 := range arr_2 {
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
