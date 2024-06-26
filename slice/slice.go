package slice

import (
	"golang.org/x/exp/constraints"
	"reflect"
)

func Contain[T comparable](slice []T, target T) bool {
	for _, item := range slice {
		if item == target {
			return true
		}
	}
	return false
}

func ContainBy[T any](slice []T, predicate func(item T) bool) bool {
	for _, item := range slice {
		if predicate(item) {
			return true
		}
	}
	return false
}

func Chunk[T any](slice []T, size int) [][]T {
	var result [][]T
	if len(slice) == 0 || size <= 0 {
		return result
	}
	for _, item := range slice {
		l := len(result)
		if l == 0 || len(result[l-1]) == size {
			result = append(result, []T{})
			l++
		}

		result[l-1] = append(result[l-1], item)
	}
	return result
}

// Compact creates a slice with all falsey values removed. The values false, nil, 0, and "" are falsey.
func Compact[T comparable](slice []T) []T {
	var zero T
	var result []T
	for _, v := range slice {
		if v != zero {
			result = append(result, v)
		}
	}
	return result
}

func Filter[T any](slice []T, predicate func(index int, item T) bool) []T {
	result := make([]T, 0)
	for i, v := range slice {
		if predicate(i, v) {
			result = append(result, v)
		}
	}
	return result
}

func CountBy[T any](slice []T, predicate func(index int, item T) bool) int {
	count := 0
	for i, v := range slice {
		if predicate(i, v) {
			count++
		}
	}
	return count
}

func DeleteAt[T any](slice []T, index int) []T {
	if index >= len(slice) {
		index = len(slice) - 1
	}

	result := make([]T, len(slice)-1)
	copy(result, slice[:index])
	copy(result[index:], slice[index+1:])
	return result
}

func Unique[T comparable](slice []T) []T {
	var result []T
	for i := 0; i < len(slice); i++ {
		v := slice[i]
		skip := true
		for j := range result {
			if v == result[j] {
				skip = false
				break
			}
		}
		if skip {
			result = append(result, v)
		}
	}
	return result
}

func UniqueBy[T comparable](slice []T, iteratee func(item T) T) []T {
	var result []T
	for _, v := range slice {
		val := iteratee(v)
		result = append(result, val)
	}
	return Unique(result)
}

func Reverse[T any](slice []T) {
	for i, j := 0, len(slice)-1; i < j; i, j = i+1, j-1 {
		slice[i], slice[j] = slice[j], slice[i]
	}
}

func Sort[T constraints.Ordered](slice []T, sortOrder ...string) {
	if len(sortOrder) > 0 && sortOrder[0] == "desc" {
		quickSort(slice, 0, len(slice)-1, "desc")
	} else {
		quickSort(slice, 0, len(slice)-1, "asc")
	}
}

func SortBy[T any](slice []T, less func(a, b T) bool) {
	quickSortBy(slice, 0, len(slice)-1, less)
}

func IsEmpty(results any) bool {
	v := reflect.ValueOf(results)
	if v.Kind() != reflect.Slice {
		return true
	}
	return v.Len() == 0
}
