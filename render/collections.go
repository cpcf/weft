package render

import (
	"fmt"
	"math/rand"
	"reflect"
	"sort"
	"strings"
	"time"
)

func formatSlice(slice interface{}, separator, format string) string {
	if slice == nil {
		return ""
	}

	v := reflect.ValueOf(slice)
	if v.Kind() != reflect.Slice && v.Kind() != reflect.Array {
		return fmt.Sprintf("%v", slice)
	}

	var parts []string
	for i := 0; i < v.Len(); i++ {
		item := v.Index(i).Interface()
		if format != "" {
			parts = append(parts, fmt.Sprintf(format, item))
		} else {
			parts = append(parts, fmt.Sprintf("%v", item))
		}
	}

	return strings.Join(parts, separator)
}

func filterSlice(slice interface{}, predicate func(interface{}) bool) interface{} {
	if slice == nil {
		return slice
	}

	v := reflect.ValueOf(slice)
	if v.Kind() != reflect.Slice && v.Kind() != reflect.Array {
		return slice
	}

	sliceType := v.Type()
	result := reflect.MakeSlice(sliceType, 0, v.Len())

	for i := 0; i < v.Len(); i++ {
		item := v.Index(i).Interface()
		if predicate(item) {
			result = reflect.Append(result, v.Index(i))
		}
	}

	return result.Interface()
}

func mapSlice(slice interface{}, mapper func(interface{}) interface{}) interface{} {
	if slice == nil {
		return slice
	}

	v := reflect.ValueOf(slice)
	if v.Kind() != reflect.Slice && v.Kind() != reflect.Array {
		return slice
	}

	elementType := reflect.TypeOf(mapper(v.Index(0).Interface()))
	resultType := reflect.SliceOf(elementType)
	result := reflect.MakeSlice(resultType, 0, v.Len())

	for i := 0; i < v.Len(); i++ {
		item := v.Index(i).Interface()
		mapped := mapper(item)
		result = reflect.Append(result, reflect.ValueOf(mapped))
	}

	return result.Interface()
}

func getFirst(slice interface{}) interface{} {
	if slice == nil {
		return nil
	}

	v := reflect.ValueOf(slice)
	if v.Kind() != reflect.Slice && v.Kind() != reflect.Array {
		return slice
	}

	if v.Len() == 0 {
		return nil
	}

	return v.Index(0).Interface()
}

func getLast(slice interface{}) interface{} {
	if slice == nil {
		return nil
	}

	v := reflect.ValueOf(slice)
	if v.Kind() != reflect.Slice && v.Kind() != reflect.Array {
		return slice
	}

	if v.Len() == 0 {
		return nil
	}

	return v.Index(v.Len() - 1).Interface()
}

func getRest(slice interface{}) interface{} {
	if slice == nil {
		return slice
	}

	v := reflect.ValueOf(slice)
	if v.Kind() != reflect.Slice && v.Kind() != reflect.Array {
		return slice
	}

	if v.Len() <= 1 {
		return reflect.MakeSlice(v.Type(), 0, 0).Interface()
	}

	return v.Slice(1, v.Len()).Interface()
}

func reverseSlice(slice interface{}) interface{} {
	if slice == nil {
		return slice
	}

	v := reflect.ValueOf(slice)
	if v.Kind() != reflect.Slice && v.Kind() != reflect.Array {
		return slice
	}

	length := v.Len()
	result := reflect.MakeSlice(v.Type(), length, length)

	for i := 0; i < length; i++ {
		result.Index(i).Set(v.Index(length - 1 - i))
	}

	return result.Interface()
}

func sortSlice(slice interface{}) interface{} {
	if slice == nil {
		return slice
	}

	v := reflect.ValueOf(slice)
	if v.Kind() != reflect.Slice && v.Kind() != reflect.Array {
		return slice
	}

	length := v.Len()
	if length == 0 {
		return slice
	}

	result := reflect.MakeSlice(v.Type(), length, length)
	reflect.Copy(result, v)

	elementType := v.Type().Elem()
	switch elementType.Kind() {
	case reflect.String:
		sort.Slice(result.Interface(), func(i, j int) bool {
			return result.Index(i).String() < result.Index(j).String()
		})
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		sort.Slice(result.Interface(), func(i, j int) bool {
			return result.Index(i).Int() < result.Index(j).Int()
		})
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		sort.Slice(result.Interface(), func(i, j int) bool {
			return result.Index(i).Uint() < result.Index(j).Uint()
		})
	case reflect.Float32, reflect.Float64:
		sort.Slice(result.Interface(), func(i, j int) bool {
			return result.Index(i).Float() < result.Index(j).Float()
		})
	default:
		sort.Slice(result.Interface(), func(i, j int) bool {
			return fmt.Sprintf("%v", result.Index(i).Interface()) < 
				   fmt.Sprintf("%v", result.Index(j).Interface())
		})
	}

	return result.Interface()
}

func uniqueSlice(slice interface{}) interface{} {
	if slice == nil {
		return slice
	}

	v := reflect.ValueOf(slice)
	if v.Kind() != reflect.Slice && v.Kind() != reflect.Array {
		return slice
	}

	seen := make(map[interface{}]bool)
	result := reflect.MakeSlice(v.Type(), 0, v.Len())

	for i := 0; i < v.Len(); i++ {
		item := v.Index(i).Interface()
		key := makeComparableKey(item)
		if !seen[key] {
			seen[key] = true
			result = reflect.Append(result, v.Index(i))
		}
	}

	return result.Interface()
}

func makeComparableKey(item interface{}) interface{} {
	v := reflect.ValueOf(item)
	
	switch v.Kind() {
	case reflect.Slice, reflect.Array:
		return fmt.Sprintf("%v", item)
	case reflect.Map:
		return fmt.Sprintf("%v", item)
	case reflect.Struct:
		return fmt.Sprintf("%v", item)
	case reflect.Ptr:
		if v.IsNil() {
			return "<nil>"
		}
		return makeComparableKey(v.Elem().Interface())
	default:
		return item
	}
}

func shuffleSlice(slice interface{}) interface{} {
	if slice == nil {
		return slice
	}

	v := reflect.ValueOf(slice)
	if v.Kind() != reflect.Slice && v.Kind() != reflect.Array {
		return slice
	}

	length := v.Len()
	result := reflect.MakeSlice(v.Type(), length, length)
	reflect.Copy(result, v)

	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(length, func(i, j int) {
		temp := result.Index(i).Interface()
		result.Index(i).Set(result.Index(j))
		result.Index(j).Set(reflect.ValueOf(temp))
	})

	return result.Interface()
}

func sliceContains(slice interface{}, item interface{}) bool {
	if slice == nil {
		return false
	}

	v := reflect.ValueOf(slice)
	if v.Kind() != reflect.Slice && v.Kind() != reflect.Array {
		return false
	}

	for i := 0; i < v.Len(); i++ {
		if reflect.DeepEqual(v.Index(i).Interface(), item) {
			return true
		}
	}

	return false
}

func sliceIndexOf(slice interface{}, item interface{}) int {
	if slice == nil {
		return -1
	}

	v := reflect.ValueOf(slice)
	if v.Kind() != reflect.Slice && v.Kind() != reflect.Array {
		return -1
	}

	for i := 0; i < v.Len(); i++ {
		if reflect.DeepEqual(v.Index(i).Interface(), item) {
			return i
		}
	}

	return -1
}

func sliceGet(slice interface{}, index int) interface{} {
	if slice == nil {
		return nil
	}

	v := reflect.ValueOf(slice)
	if v.Kind() != reflect.Slice && v.Kind() != reflect.Array {
		return nil
	}

	if index < 0 || index >= v.Len() {
		return nil
	}

	return v.Index(index).Interface()
}

func sliceTake(slice interface{}, count int) interface{} {
	if slice == nil {
		return slice
	}

	v := reflect.ValueOf(slice)
	if v.Kind() != reflect.Slice && v.Kind() != reflect.Array {
		return slice
	}

	if count <= 0 {
		return reflect.MakeSlice(v.Type(), 0, 0).Interface()
	}

	if count >= v.Len() {
		return slice
	}

	return v.Slice(0, count).Interface()
}

func sliceDrop(slice interface{}, count int) interface{} {
	if slice == nil {
		return slice
	}

	v := reflect.ValueOf(slice)
	if v.Kind() != reflect.Slice && v.Kind() != reflect.Array {
		return slice
	}

	if count <= 0 {
		return slice
	}

	if count >= v.Len() {
		return reflect.MakeSlice(v.Type(), 0, 0).Interface()
	}

	return v.Slice(count, v.Len()).Interface()
}

func sliceConcat(slices ...interface{}) interface{} {
	if len(slices) == 0 {
		return nil
	}

	var firstSlice reflect.Value
	var totalLength int

	for _, slice := range slices {
		if slice == nil {
			continue
		}

		v := reflect.ValueOf(slice)
		if v.Kind() != reflect.Slice && v.Kind() != reflect.Array {
			continue
		}

		if !firstSlice.IsValid() {
			firstSlice = v
		}

		totalLength += v.Len()
	}

	if !firstSlice.IsValid() {
		return nil
	}

	result := reflect.MakeSlice(firstSlice.Type(), 0, totalLength)

	for _, slice := range slices {
		if slice == nil {
			continue
		}

		v := reflect.ValueOf(slice)
		if v.Kind() != reflect.Slice && v.Kind() != reflect.Array {
			continue
		}

		for i := 0; i < v.Len(); i++ {
			result = reflect.Append(result, v.Index(i))
		}
	}

	return result.Interface()
}

func sliceFlatten(slice interface{}) interface{} {
	if slice == nil {
		return slice
	}

	v := reflect.ValueOf(slice)
	if v.Kind() != reflect.Slice && v.Kind() != reflect.Array {
		return slice
	}

	if v.Len() == 0 {
		return slice
	}

	firstElement := v.Index(0)
	if firstElement.Kind() != reflect.Slice && firstElement.Kind() != reflect.Array {
		return slice
	}

	elementType := firstElement.Type().Elem()
	result := reflect.MakeSlice(reflect.SliceOf(elementType), 0, v.Len()*2)

	for i := 0; i < v.Len(); i++ {
		subSlice := v.Index(i)
		if subSlice.Kind() == reflect.Slice || subSlice.Kind() == reflect.Array {
			for j := 0; j < subSlice.Len(); j++ {
				result = reflect.Append(result, subSlice.Index(j))
			}
		}
	}

	return result.Interface()
}

func sliceGroup(slice interface{}, keyFunc func(interface{}) interface{}) interface{} {
	if slice == nil {
		return make(map[interface{}]interface{})
	}

	v := reflect.ValueOf(slice)
	if v.Kind() != reflect.Slice && v.Kind() != reflect.Array {
		return make(map[interface{}]interface{})
	}

	groups := make(map[interface{}][]interface{})

	for i := 0; i < v.Len(); i++ {
		item := v.Index(i).Interface()
		key := keyFunc(item)
		groups[key] = append(groups[key], item)
	}

	result := make(map[interface{}]interface{})
	for k, v := range groups {
		result[k] = v
	}

	return result
}

func sliceReduce(slice interface{}, initialValue interface{}, reducer func(interface{}, interface{}) interface{}) interface{} {
	if slice == nil {
		return initialValue
	}

	v := reflect.ValueOf(slice)
	if v.Kind() != reflect.Slice && v.Kind() != reflect.Array {
		return initialValue
	}

	accumulator := initialValue
	for i := 0; i < v.Len(); i++ {
		item := v.Index(i).Interface()
		accumulator = reducer(accumulator, item)
	}

	return accumulator
}

func slicePartition(slice interface{}, predicate func(interface{}) bool) interface{} {
	if slice == nil {
		return [2]interface{}{nil, nil}
	}

	v := reflect.ValueOf(slice)
	if v.Kind() != reflect.Slice && v.Kind() != reflect.Array {
		return [2]interface{}{slice, nil}
	}

	var truthy []interface{}
	var falsy []interface{}

	for i := 0; i < v.Len(); i++ {
		item := v.Index(i).Interface()
		if predicate(item) {
			truthy = append(truthy, item)
		} else {
			falsy = append(falsy, item)
		}
	}

	return [2]interface{}{truthy, falsy}
}

func sliceChunk(slice interface{}, size int) interface{} {
	if slice == nil || size <= 0 {
		return []interface{}{}
	}

	v := reflect.ValueOf(slice)
	if v.Kind() != reflect.Slice && v.Kind() != reflect.Array {
		return []interface{}{slice}
	}

	length := v.Len()
	if length == 0 {
		return []interface{}{}
	}

	var chunks []interface{}
	for i := 0; i < length; i += size {
		end := i + size
		if end > length {
			end = length
		}
		chunks = append(chunks, v.Slice(i, end).Interface())
	}

	return chunks
}

func sliceZip(slices ...interface{}) interface{} {
	if len(slices) == 0 {
		return []interface{}{}
	}

	var minLength int = -1
	var sliceValues []reflect.Value

	for _, slice := range slices {
		if slice == nil {
			continue
		}

		v := reflect.ValueOf(slice)
		if v.Kind() != reflect.Slice && v.Kind() != reflect.Array {
			continue
		}

		sliceValues = append(sliceValues, v)
		if minLength == -1 || v.Len() < minLength {
			minLength = v.Len()
		}
	}

	if len(sliceValues) == 0 || minLength == 0 {
		return []interface{}{}
	}

	var result []interface{}
	for i := 0; i < minLength; i++ {
		var tuple []interface{}
		for _, v := range sliceValues {
			tuple = append(tuple, v.Index(i).Interface())
		}
		result = append(result, tuple)
	}

	return result
}

func sliceDifference(slice1, slice2 interface{}) interface{} {
	if slice1 == nil {
		return slice1
	}

	v1 := reflect.ValueOf(slice1)
	if v1.Kind() != reflect.Slice && v1.Kind() != reflect.Array {
		return slice1
	}

	if slice2 == nil {
		return slice1
	}

	v2 := reflect.ValueOf(slice2)
	if v2.Kind() != reflect.Slice && v2.Kind() != reflect.Array {
		return slice1
	}

	set2 := make(map[interface{}]bool)
	for i := 0; i < v2.Len(); i++ {
		key := makeComparableKey(v2.Index(i).Interface())
		set2[key] = true
	}

	result := reflect.MakeSlice(v1.Type(), 0, v1.Len())
	for i := 0; i < v1.Len(); i++ {
		item := v1.Index(i).Interface()
		key := makeComparableKey(item)
		if !set2[key] {
			result = reflect.Append(result, v1.Index(i))
		}
	}

	return result.Interface()
}

func sliceIntersection(slice1, slice2 interface{}) interface{} {
	if slice1 == nil || slice2 == nil {
		return nil
	}

	v1 := reflect.ValueOf(slice1)
	v2 := reflect.ValueOf(slice2)

	if v1.Kind() != reflect.Slice && v1.Kind() != reflect.Array {
		return nil
	}
	if v2.Kind() != reflect.Slice && v2.Kind() != reflect.Array {
		return nil
	}

	set2 := make(map[interface{}]bool)
	for i := 0; i < v2.Len(); i++ {
		key := makeComparableKey(v2.Index(i).Interface())
		set2[key] = true
	}

	seen := make(map[interface{}]bool)
	result := reflect.MakeSlice(v1.Type(), 0, v1.Len())

	for i := 0; i < v1.Len(); i++ {
		item := v1.Index(i).Interface()
		key := makeComparableKey(item)
		if set2[key] && !seen[key] {
			seen[key] = true
			result = reflect.Append(result, v1.Index(i))
		}
	}

	return result.Interface()
}