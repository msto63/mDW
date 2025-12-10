// File: slicex.go
// Title: Core Slice Utilities
// Description: Implements comprehensive slice utility functions including transformation,
//              manipulation, search, validation, and conversion operations for Go slices.
//              Provides functional programming style operations with generic type support.
// Author: msto63 with Claude Sonnet 4.0
// Version: v0.1.0
// Created: 2025-01-25
// Modified: 2025-01-25
//
// Change History:
// - 2025-01-25 v0.1.0: Initial implementation with comprehensive slice utilities

package slicex

import (
	"cmp"
	"fmt"
	"slices"
)

// ===============================
// Core Transformation Functions
// ===============================

// Filter returns a new slice containing only elements that match the predicate
func Filter[T any](slice []T, predicate func(T) bool) []T {
	if slice == nil || predicate == nil {
		return nil
	}
	
	result := make([]T, 0, len(slice))
	for _, item := range slice {
		if predicate(item) {
			result = append(result, item)
		}
	}
	return result
}

// Map transforms each element in the slice using the provided function
func Map[T, R any](slice []T, mapper func(T) R) []R {
	if slice == nil || mapper == nil {
		return nil
	}
	
	result := make([]R, len(slice))
	for i, item := range slice {
		result[i] = mapper(item)
	}
	return result
}

// MapWithIndex transforms each element with its index using the provided function
func MapWithIndex[T, R any](slice []T, mapper func(int, T) R) []R {
	if slice == nil || mapper == nil {
		return nil
	}
	
	result := make([]R, len(slice))
	for i, item := range slice {
		result[i] = mapper(i, item)
	}
	return result
}

// Reduce reduces the slice to a single value using the provided function
func Reduce[T, R any](slice []T, initial R, reducer func(R, T) R) R {
	if slice == nil || reducer == nil {
		return initial
	}
	
	result := initial
	for _, item := range slice {
		result = reducer(result, item)
	}
	return result
}

// ReduceWithIndex reduces the slice to a single value with index information
func ReduceWithIndex[T, R any](slice []T, initial R, reducer func(R, int, T) R) R {
	if slice == nil || reducer == nil {
		return initial
	}
	
	result := initial
	for i, item := range slice {
		result = reducer(result, i, item)
	}
	return result
}

// ForEach executes the provided function for each element
func ForEach[T any](slice []T, fn func(T)) {
	if slice == nil || fn == nil {
		return
	}
	
	for _, item := range slice {
		fn(item)
	}
}

// ForEachWithIndex executes the provided function for each element with its index
func ForEachWithIndex[T any](slice []T, fn func(int, T)) {
	if slice == nil || fn == nil {
		return
	}
	
	for i, item := range slice {
		fn(i, item)
	}
}

// ===============================
// Slice Manipulation Functions
// ===============================

// Chunk splits the slice into chunks of the specified size
func Chunk[T any](slice []T, size int) [][]T {
	if slice == nil || size <= 0 {
		return nil
	}
	
	var chunks [][]T
	for i := 0; i < len(slice); i += size {
		end := i + size
		if end > len(slice) {
			end = len(slice)
		}
		chunks = append(chunks, slice[i:end])
	}
	return chunks
}

// Flatten flattens a slice of slices into a single slice
func Flatten[T any](slices [][]T) []T {
	if slices == nil {
		return nil
	}
	
	totalLen := 0
	for _, s := range slices {
		totalLen += len(s)
	}
	
	result := make([]T, 0, totalLen)
	for _, s := range slices {
		result = append(result, s...)
	}
	return result
}

// Unique returns a new slice with duplicate elements removed (preserves order)
func Unique[T comparable](slice []T) []T {
	if slice == nil {
		return nil
	}
	
	seen := make(map[T]bool)
	result := make([]T, 0, len(slice))
	
	for _, item := range slice {
		if !seen[item] {
			seen[item] = true
			result = append(result, item)
		}
	}
	return result
}

// UniqueBy returns a new slice with duplicates removed based on a key function
func UniqueBy[T any, K comparable](slice []T, keyFunc func(T) K) []T {
	if slice == nil || keyFunc == nil {
		return nil
	}
	
	seen := make(map[K]bool)
	result := make([]T, 0, len(slice))
	
	for _, item := range slice {
		key := keyFunc(item)
		if !seen[key] {
			seen[key] = true
			result = append(result, item)
		}
	}
	return result
}

// Union returns the union of two slices (no duplicates)
func Union[T comparable](slice1, slice2 []T) []T {
	combined := append(slice1, slice2...)
	return Unique(combined)
}

// Intersect returns elements present in both slices
func Intersect[T comparable](slice1, slice2 []T) []T {
	if slice1 == nil || slice2 == nil {
		return nil
	}
	
	set := make(map[T]bool)
	for _, item := range slice2 {
		set[item] = true
	}
	
	var result []T
	seen := make(map[T]bool)
	for _, item := range slice1 {
		if set[item] && !seen[item] {
			seen[item] = true
			result = append(result, item)
		}
	}
	return result
}

// Difference returns elements in slice1 that are not in slice2
func Difference[T comparable](slice1, slice2 []T) []T {
	if slice1 == nil {
		return nil
	}
	if slice2 == nil {
		return append([]T(nil), slice1...)
	}
	
	set := make(map[T]bool)
	for _, item := range slice2 {
		set[item] = true
	}
	
	var result []T
	for _, item := range slice1 {
		if !set[item] {
			result = append(result, item)
		}
	}
	return result
}

// Reverse returns a new slice with elements in reverse order
func Reverse[T any](slice []T) []T {
	if slice == nil {
		return nil
	}
	
	result := make([]T, len(slice))
	for i, item := range slice {
		result[len(slice)-1-i] = item
	}
	return result
}

// ===============================
// Search and Validation Functions
// ===============================

// Contains checks if the slice contains the specified element
func Contains[T comparable](slice []T, element T) bool {
	if slice == nil {
		return false
	}
	
	for _, item := range slice {
		if item == element {
			return true
		}
	}
	return false
}

// ContainsBy checks if the slice contains an element matching the predicate
func ContainsBy[T any](slice []T, predicate func(T) bool) bool {
	if slice == nil || predicate == nil {
		return false
	}
	
	for _, item := range slice {
		if predicate(item) {
			return true
		}
	}
	return false
}

// IndexOf returns the first index of the element, or -1 if not found
func IndexOf[T comparable](slice []T, element T) int {
	if slice == nil {
		return -1
	}
	
	for i, item := range slice {
		if item == element {
			return i
		}
	}
	return -1
}

// IndexOfBy returns the first index where predicate returns true, or -1
func IndexOfBy[T any](slice []T, predicate func(T) bool) int {
	if slice == nil || predicate == nil {
		return -1
	}
	
	for i, item := range slice {
		if predicate(item) {
			return i
		}
	}
	return -1
}

// LastIndexOf returns the last index of the element, or -1 if not found
func LastIndexOf[T comparable](slice []T, element T) int {
	if slice == nil {
		return -1
	}
	
	for i := len(slice) - 1; i >= 0; i-- {
		if slice[i] == element {
			return i
		}
	}
	return -1
}

// Find returns the first element matching the predicate, or zero value if none
func Find[T any](slice []T, predicate func(T) bool) (T, bool) {
	var zero T
	if slice == nil || predicate == nil {
		return zero, false
	}
	
	for _, item := range slice {
		if predicate(item) {
			return item, true
		}
	}
	return zero, false
}

// FindLast returns the last element matching the predicate, or zero value if none
func FindLast[T any](slice []T, predicate func(T) bool) (T, bool) {
	var zero T
	if slice == nil || predicate == nil {
		return zero, false
	}
	
	for i := len(slice) - 1; i >= 0; i-- {
		if predicate(slice[i]) {
			return slice[i], true
		}
	}
	return zero, false
}

// Every checks if all elements match the predicate
func Every[T any](slice []T, predicate func(T) bool) bool {
	if slice == nil || predicate == nil {
		return false
	}
	
	for _, item := range slice {
		if !predicate(item) {
			return false
		}
	}
	return true
}

// Some checks if at least one element matches the predicate
func Some[T any](slice []T, predicate func(T) bool) bool {
	if slice == nil || predicate == nil {
		return false
	}
	
	for _, item := range slice {
		if predicate(item) {
			return true
		}
	}
	return false
}

// ===============================
// Utility Functions
// ===============================

// IsEmpty checks if the slice is nil or has no elements
func IsEmpty[T any](slice []T) bool {
	return len(slice) == 0
}

// IsNotEmpty checks if the slice has at least one element
func IsNotEmpty[T any](slice []T) bool {
	return len(slice) > 0
}

// Count returns the number of elements matching the predicate
func Count[T any](slice []T, predicate func(T) bool) int {
	if slice == nil || predicate == nil {
		return 0
	}
	
	count := 0
	for _, item := range slice {
		if predicate(item) {
			count++
		}
	}
	return count
}

// Min returns the minimum element (requires ordered type)
func Min[T cmp.Ordered](slice []T) (T, bool) {
	var zero T
	if len(slice) == 0 {
		return zero, false
	}
	
	min := slice[0]
	for _, item := range slice[1:] {
		if item < min {
			min = item
		}
	}
	return min, true
}

// Max returns the maximum element (requires ordered type)
func Max[T cmp.Ordered](slice []T) (T, bool) {
	var zero T
	if len(slice) == 0 {
		return zero, false
	}
	
	max := slice[0]
	for _, item := range slice[1:] {
		if item > max {
			max = item
		}
	}
	return max, true
}

// MinBy returns the minimum element using a comparison function
func MinBy[T any](slice []T, less func(T, T) bool) (T, bool) {
	var zero T
	if len(slice) == 0 || less == nil {
		return zero, false
	}
	
	min := slice[0]
	for _, item := range slice[1:] {
		if less(item, min) {
			min = item
		}
	}
	return min, true
}

// MaxBy returns the maximum element using a comparison function
func MaxBy[T any](slice []T, less func(T, T) bool) (T, bool) {
	var zero T
	if len(slice) == 0 || less == nil {
		return zero, false
	}
	
	max := slice[0]
	for _, item := range slice[1:] {
		if less(max, item) {
			max = item
		}
	}
	return max, true
}

// Sum returns the sum of all elements (requires numeric type)
func Sum[T interface{ ~int | ~int8 | ~int16 | ~int32 | ~int64 | ~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~float32 | ~float64 }](slice []T) T {
	var sum T
	for _, item := range slice {
		sum += item
	}
	return sum
}

// ===============================
// Slice Creation and Conversion
// ===============================

// Range creates a slice of integers from start to end (exclusive)
func Range(start, end int) []int {
	if start >= end {
		return nil
	}
	
	result := make([]int, end-start)
	for i := 0; i < end-start; i++ {
		result[i] = start + i
	}
	return result
}

// RangeStep creates a slice of integers from start to end with step
func RangeStep(start, end, step int) []int {
	if step == 0 || (step > 0 && start >= end) || (step < 0 && start <= end) {
		return nil
	}
	
	var result []int
	if step > 0 {
		for i := start; i < end; i += step {
			result = append(result, i)
		}
	} else {
		for i := start; i > end; i += step {
			result = append(result, i)
		}
	}
	return result
}

// Repeat creates a slice with the element repeated n times
func Repeat[T any](element T, n int) []T {
	if n <= 0 {
		return nil
	}
	
	result := make([]T, n)
	for i := 0; i < n; i++ {
		result[i] = element
	}
	return result
}

// Fill creates a slice of specified length with elements from the generator function
func Fill[T any](length int, generator func(int) T) []T {
	if length <= 0 || generator == nil {
		return nil
	}
	
	result := make([]T, length)
	for i := 0; i < length; i++ {
		result[i] = generator(i)
	}
	return result
}

// Clone creates a shallow copy of the slice
func Clone[T any](slice []T) []T {
	if slice == nil {
		return nil
	}
	
	result := make([]T, len(slice))
	copy(result, slice)
	return result
}

// Equal checks if two slices are equal (deep comparison)
func Equal[T comparable](slice1, slice2 []T) bool {
	if len(slice1) != len(slice2) {
		return false
	}
	
	for i, item := range slice1 {
		if item != slice2[i] {
			return false
		}
	}
	return true
}

// EqualBy checks if two slices are equal using a comparison function
func EqualBy[T any](slice1, slice2 []T, equal func(T, T) bool) bool {
	if len(slice1) != len(slice2) || equal == nil {
		return false
	}
	
	for i, item := range slice1 {
		if !equal(item, slice2[i]) {
			return false
		}
	}
	return true
}

// ===============================
// Advanced Operations
// ===============================

// GroupBy groups elements by a key function
func GroupBy[T any, K comparable](slice []T, keyFunc func(T) K) map[K][]T {
	if slice == nil || keyFunc == nil {
		return nil
	}
	
	groups := make(map[K][]T)
	for _, item := range slice {
		key := keyFunc(item)
		groups[key] = append(groups[key], item)
	}
	return groups
}

// Partition splits the slice into two based on a predicate
func Partition[T any](slice []T, predicate func(T) bool) ([]T, []T) {
	if slice == nil || predicate == nil {
		return nil, nil
	}
	
	var trueSlice, falseSlice []T
	for _, item := range slice {
		if predicate(item) {
			trueSlice = append(trueSlice, item)
		} else {
			falseSlice = append(falseSlice, item)
		}
	}
	return trueSlice, falseSlice
}

// Take returns the first n elements
func Take[T any](slice []T, n int) []T {
	if slice == nil || n <= 0 {
		return nil
	}
	
	if n >= len(slice) {
		return Clone(slice)
	}
	
	result := make([]T, n)
	copy(result, slice[:n])
	return result
}

// TakeWhile returns elements from the beginning while predicate is true
func TakeWhile[T any](slice []T, predicate func(T) bool) []T {
	if slice == nil || predicate == nil {
		return nil
	}
	
	var result []T
	for _, item := range slice {
		if !predicate(item) {
			break
		}
		result = append(result, item)
	}
	return result
}

// Drop returns all but the first n elements
func Drop[T any](slice []T, n int) []T {
	if slice == nil || n <= 0 {
		return Clone(slice)
	}
	
	if n >= len(slice) {
		return nil
	}
	
	result := make([]T, len(slice)-n)
	copy(result, slice[n:])
	return result
}

// DropWhile returns elements after dropping from the beginning while predicate is true
func DropWhile[T any](slice []T, predicate func(T) bool) []T {
	if slice == nil || predicate == nil {
		return Clone(slice)
	}
	
	i := 0
	for i < len(slice) && predicate(slice[i]) {
		i++
	}
	
	if i == len(slice) {
		return nil
	}
	
	result := make([]T, len(slice)-i)
	copy(result, slice[i:])
	return result
}

// Pair represents a pair of values with type safety
type Pair[T, U any] struct {
	First  T
	Second U
}

// Zip combines two slices into a slice of type-safe pairs
func Zip[T, U any](slice1 []T, slice2 []U) []Pair[T, U] {
	if slice1 == nil || slice2 == nil {
		return nil
	}
	
	minLen := len(slice1)
	if len(slice2) < minLen {
		minLen = len(slice2)
	}
	
	result := make([]Pair[T, U], minLen)
	for i := 0; i < minLen; i++ {
		result[i] = Pair[T, U]{First: slice1[i], Second: slice2[i]}
	}
	return result
}

// ZipLegacy combines two slices into a slice of interface{} pairs (for backward compatibility)
// Deprecated: Use Zip for type-safe pairs
func ZipLegacy[T, U any](slice1 []T, slice2 []U) [][2]interface{} {
	if slice1 == nil || slice2 == nil {
		return nil
	}
	
	minLen := len(slice1)
	if len(slice2) < minLen {
		minLen = len(slice2)
	}
	
	result := make([][2]interface{}, minLen)
	for i := 0; i < minLen; i++ {
		result[i] = [2]interface{}{slice1[i], slice2[i]}
	}
	return result
}

// ===============================
// Sorting Helpers
// ===============================

// Sort returns a sorted copy of the slice (requires ordered type)
func Sort[T cmp.Ordered](slice []T) []T {
	if slice == nil {
		return nil
	}
	
	result := Clone(slice)
	slices.Sort(result)
	return result
}

// SortBy returns a sorted copy using a comparison function
func SortBy[T any](slice []T, less func(T, T) bool) []T {
	if slice == nil || less == nil {
		return nil
	}
	
	result := Clone(slice)
	slices.SortFunc(result, func(a, b T) int {
		if less(a, b) {
			return -1
		}
		if less(b, a) {
			return 1
		}
		return 0
	})
	return result
}

// IsSorted checks if the slice is sorted (requires ordered type)
func IsSorted[T cmp.Ordered](slice []T) bool {
	return slices.IsSorted(slice)
}

// IsSortedBy checks if the slice is sorted using a comparison function
func IsSortedBy[T any](slice []T, less func(T, T) bool) bool {
	if slice == nil || less == nil {
		return true
	}
	
	return slices.IsSortedFunc(slice, func(a, b T) int {
		if less(a, b) {
			return -1
		}
		if less(b, a) {
			return 1
		}
		return 0
	})
}

// ===============================
// String Conversion
// ===============================

// String returns a string representation of the slice
func String[T any](slice []T) string {
	if slice == nil {
		return "[]"
	}
	
	return fmt.Sprintf("%v", slice)
}

// Join converts elements to strings and joins them with separator
func Join[T any](slice []T, separator string) string {
	if slice == nil {
		return ""
	}
	
	if len(slice) == 0 {
		return ""
	}
	
	if len(slice) == 1 {
		return fmt.Sprintf("%v", slice[0])
	}
	
	result := fmt.Sprintf("%v", slice[0])
	for _, item := range slice[1:] {
		result += separator + fmt.Sprintf("%v", item)
	}
	return result
}