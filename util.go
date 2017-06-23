package main

// return true if the elem is in the array arr
func inArray(elem interface{}, arr []interface{}) bool {
    for _, e := range arr {
        if e == elem {
            return true
        }
    }
    return false
}

//return true if the array arr1 contains all elements of array arr2
func hasAllElements(arr1 []interface{}, arr2 []interface{}) bool {
    for _, e2 := range arr2 {
        if !inArray(e2, arr1) {
            return false
        }
    }
    return true
}

func stringArrayToInterfacArray(arr []string) []interface{} {
    result := make([]interface{}, 0)
    for _, s := range arr {
        result = append(result, s)
    }
    return result
}


