// File: example_test.go
// Title: Slice Utilities Examples
// Description: Comprehensive examples demonstrating practical usage of slicex
//              utility functions in real-world scenarios.
// Author: msto63 with Claude Sonnet 4.0
// Version: v0.1.0
// Created: 2025-01-25
// Modified: 2025-01-25
//
// Change History:
// - 2025-01-25 v0.1.0: Initial example implementation

package slicex

import (
	"fmt"
	"strings"
)

// ===============================
// Core Transformation Examples
// ===============================

func ExampleFilter() {
	numbers := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	
	// Filter even numbers
	evens := Filter(numbers, func(n int) bool {
		return n%2 == 0
	})
	
	fmt.Println("Even numbers:", evens)
	// Output: Even numbers: [2 4 6 8 10]
}

func ExampleFilter_strings() {
	words := []string{"apple", "banana", "cherry", "date", "elderberry"}
	
	// Filter words longer than 5 characters
	longWords := Filter(words, func(word string) bool {
		return len(word) > 5
	})
	
	fmt.Println("Long words:", longWords)
	// Output: Long words: [banana cherry elderberry]
}

func ExampleMap() {
	numbers := []int{1, 2, 3, 4, 5}
	
	// Convert numbers to strings
	strings := Map(numbers, func(n int) string {
		return fmt.Sprintf("Number: %d", n)
	})
	
	fmt.Println(strings)
	// Output: [Number: 1 Number: 2 Number: 3 Number: 4 Number: 5]
}

func ExampleMap_calculations() {
	prices := []float64{10.50, 25.99, 5.75, 18.25}
	
	// Calculate prices with 20% tax
	withTax := Map(prices, func(price float64) float64 {
		return price * 1.20
	})
	
	fmt.Printf("Prices with tax: %.2f\n", withTax)
	// Output: Prices with tax: [12.60 31.19 6.90 21.90]
}

func ExampleMapWithIndex() {
	items := []string{"apple", "banana", "cherry"}
	
	// Create numbered list
	numbered := MapWithIndex(items, func(i int, item string) string {
		return fmt.Sprintf("%d. %s", i+1, item)
	})
	
	fmt.Println(numbered)
	// Output: [1. apple 2. banana 3. cherry]
}

func ExampleReduce() {
	numbers := []int{1, 2, 3, 4, 5}
	
	// Sum all numbers
	sum := Reduce(numbers, 0, func(acc, n int) int {
		return acc + n
	})
	
	fmt.Println("Sum:", sum)
	// Output: Sum: 15
}

func ExampleReduce_strings() {
	words := []string{"Hello", "World", "from", "Go"}
	
	// Concatenate with spaces
	sentence := Reduce(words, "", func(acc, word string) string {
		if acc == "" {
			return word
		}
		return acc + " " + word
	})
	
	fmt.Println(sentence)
	// Output: Hello World from Go
}

// ===============================
// Slice Manipulation Examples
// ===============================

func ExampleChunk() {
	numbers := []int{1, 2, 3, 4, 5, 6, 7, 8, 9}
	
	// Split into chunks of 3
	chunks := Chunk(numbers, 3)
	
	for i, chunk := range chunks {
		fmt.Printf("Chunk %d: %v\n", i+1, chunk)
	}
	// Output:
	// Chunk 1: [1 2 3]
	// Chunk 2: [4 5 6]
	// Chunk 3: [7 8 9]
}

func ExampleFlatten() {
	nested := [][]string{
		{"red", "green"},
		{"blue", "yellow", "orange"},
		{"purple"},
	}
	
	colors := Flatten(nested)
	fmt.Println("All colors:", colors)
	// Output: All colors: [red green blue yellow orange purple]
}

func ExampleUnique() {
	numbers := []int{1, 2, 2, 3, 1, 4, 3, 5}
	
	unique := Unique(numbers)
	fmt.Println("Unique numbers:", unique)
	// Output: Unique numbers: [1 2 3 4 5]
}

func ExampleUniqueBy() {
	type Person struct {
		Name string
		Age  int
	}
	
	people := []Person{
		{"Alice", 25},
		{"Bob", 30},
		{"Alice", 35}, // Duplicate name
		{"Charlie", 25}, // Duplicate age, but different name
	}
	
	// Get unique people by name
	uniqueByName := UniqueBy(people, func(p Person) string {
		return p.Name
	})
	
	for _, person := range uniqueByName {
		fmt.Printf("%s (%d)\n", person.Name, person.Age)
	}
	// Output:
	// Alice (25)
	// Bob (30)
	// Charlie (25)
}

func ExampleUnion() {
	slice1 := []int{1, 2, 3, 4}
	slice2 := []int{3, 4, 5, 6}
	
	union := Union(slice1, slice2)
	fmt.Println("Union:", union)
	// Output: Union: [1 2 3 4 5 6]
}

func ExampleIntersect() {
	interests1 := []string{"reading", "swimming", "coding", "music"}
	interests2 := []string{"music", "cooking", "reading", "travel"}
	
	common := Intersect(interests1, interests2)
	fmt.Println("Common interests:", common)
	// Output: Common interests: [reading music]
}

func ExampleDifference() {
	allFeatures := []string{"auth", "payment", "chat", "notifications"}
	implementedFeatures := []string{"auth", "payment"}
	
	remaining := Difference(allFeatures, implementedFeatures)
	fmt.Println("Features to implement:", remaining)
	// Output: Features to implement: [chat notifications]
}

func ExampleReverse() {
	steps := []string{"Start", "Process", "Validate", "Complete"}
	
	reversed := Reverse(steps)
	fmt.Println("Reversed steps:", reversed)
	// Output: Reversed steps: [Complete Validate Process Start]
}

// ===============================
// Search and Validation Examples
// ===============================

func ExampleContains() {
	fruits := []string{"apple", "banana", "cherry", "date"}
	
	fmt.Println("Contains banana:", Contains(fruits, "banana"))
	fmt.Println("Contains grape:", Contains(fruits, "grape"))
	// Output:
	// Contains banana: true
	// Contains grape: false
}

func ExampleFind() {
	numbers := []int{1, 3, 5, 8, 9, 12}
	
	// Find first even number
	even, found := Find(numbers, func(n int) bool {
		return n%2 == 0
	})
	
	if found {
		fmt.Println("First even number:", even)
	} else {
		fmt.Println("No even number found")
	}
	// Output: First even number: 8
}

func ExampleFind_struct() {
	type Product struct {
		Name  string
		Price float64
	}
	
	products := []Product{
		{"Laptop", 999.99},
		{"Mouse", 25.50},
		{"Keyboard", 75.00},
		{"Monitor", 299.99},
	}
	
	// Find product under $50
	affordable, found := Find(products, func(p Product) bool {
		return p.Price < 50.0
	})
	
	if found {
		fmt.Printf("Affordable product: %s ($%.2f)\n", affordable.Name, affordable.Price)
	}
	// Output: Affordable product: Mouse ($25.50)
}

func ExampleEvery() {
	grades := []int{85, 90, 78, 92, 88}
	
	allPassing := Every(grades, func(grade int) bool {
		return grade >= 70
	})
	
	fmt.Println("All students passed:", allPassing)
	// Output: All students passed: true
}

func ExampleSome() {
	scores := []int{45, 52, 38, 91, 44}
	
	hasExcellent := Some(scores, func(score int) bool {
		return score >= 90
	})
	
	fmt.Println("Has excellent score:", hasExcellent)
	// Output: Has excellent score: true
}

// ===============================
// Utility Examples
// ===============================

func ExampleCount() {
	responses := []string{"yes", "no", "maybe", "yes", "no", "yes"}
	
	yesCount := Count(responses, func(response string) bool {
		return response == "yes"
	})
	
	fmt.Printf("Yes responses: %d out of %d\n", yesCount, len(responses))
	// Output: Yes responses: 3 out of 6
}

func ExampleMin() {
	temperatures := []float64{23.5, 18.2, 31.7, 15.9, 28.3}
	
	minTemp, found := Min(temperatures)
	if found {
		fmt.Printf("Minimum temperature: %.1f°C\n", minTemp)
	}
	// Output: Minimum temperature: 15.9°C
}

func ExampleMax() {
	scores := []int{78, 85, 92, 88, 95, 73}
	
	maxScore, found := Max(scores)
	if found {
		fmt.Println("Highest score:", maxScore)
	}
	// Output: Highest score: 95
}

func ExampleSum() {
	expenses := []float64{45.50, 23.75, 67.25, 12.00, 89.50}
	
	total := Sum(expenses)
	fmt.Printf("Total expenses: $%.2f\n", total)
	// Output: Total expenses: $238.00
}

// ===============================
// Creation and Conversion Examples
// ===============================

func ExampleRange() {
	// Create a range of page numbers
	pages := Range(1, 6)
	fmt.Println("Pages:", pages)
	// Output: Pages: [1 2 3 4 5]
}

func ExampleRangeStep() {
	// Create even numbers from 0 to 10
	evens := RangeStep(0, 11, 2)
	fmt.Println("Even numbers:", evens)
	
	// Create countdown
	countdown := RangeStep(10, 0, -2)
	fmt.Println("Countdown:", countdown)
	// Output:
	// Even numbers: [0 2 4 6 8 10]
	// Countdown: [10 8 6 4 2]
}

func ExampleRepeat() {
	// Create placeholder data
	placeholders := Repeat("Lorem ipsum", 3)
	fmt.Println(placeholders)
	// Output: [Lorem ipsum Lorem ipsum Lorem ipsum]
}

func ExampleFill() {
	// Generate sequential IDs
	ids := Fill(5, func(i int) string {
		return fmt.Sprintf("ID-%03d", i+1)
	})
	
	fmt.Println("Generated IDs:", ids)
	// Output: Generated IDs: [ID-001 ID-002 ID-003 ID-004 ID-005]
}

func ExampleClone() {
	original := []string{"apple", "banana", "cherry"}
	copy := Clone(original)
	
	// Modify the copy
	copy[0] = "apricot"
	
	fmt.Println("Original:", original)
	fmt.Println("Copy:", copy)
	// Output:
	// Original: [apple banana cherry]
	// Copy: [apricot banana cherry]
}

// ===============================
// Advanced Operations Examples
// ===============================

func ExampleGroupBy() {
	type Student struct {
		Name  string
		Grade string
	}
	
	students := []Student{
		{"Alice", "A"},
		{"Bob", "B"},
		{"Charlie", "A"},
		{"Diana", "C"},
		{"Eve", "B"},
	}
	
	byGrade := GroupBy(students, func(s Student) string {
		return s.Grade
	})
	
	for grade, studentList := range byGrade {
		names := Map(studentList, func(s Student) string { return s.Name })
		fmt.Printf("Grade %s: %v\n", grade, names)
	}
	// Output may vary in order due to map iteration:
	// Grade A: [Alice Charlie]
	// Grade B: [Bob Eve]
	// Grade C: [Diana]
}

func ExamplePartition() {
	numbers := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	
	evens, odds := Partition(numbers, func(n int) bool {
		return n%2 == 0
	})
	
	fmt.Println("Even numbers:", evens)
	fmt.Println("Odd numbers:", odds)
	// Output:
	// Even numbers: [2 4 6 8 10]
	// Odd numbers: [1 3 5 7 9]
}

func ExampleTake() {
	urls := []string{
		"https://api.service.com/users",
		"https://api.service.com/products",
		"https://api.service.com/orders",
		"https://api.service.com/analytics",
		"https://api.service.com/reports",
	}
	
	// Process only first 3 URLs
	priority := Take(urls, 3)
	fmt.Println("Priority URLs:", priority)
	// Output: Priority URLs: [https://api.service.com/users https://api.service.com/products https://api.service.com/orders]
}

func ExampleTakeWhile() {
	scores := []int{95, 87, 92, 88, 76, 85, 90}
	
	// Take scores while they're above 80
	highScores := TakeWhile(scores, func(score int) bool {
		return score > 80
	})
	
	fmt.Println("Consecutive high scores:", highScores)
	// Output: Consecutive high scores: [95 87 92 88]
}

func ExampleDrop() {
	logEntries := []string{
		"DEBUG: App started",
		"DEBUG: Config loaded",
		"INFO: Server listening on :8080",
		"WARN: High memory usage",
		"ERROR: Database connection failed",
	}
	
	// Skip debug messages
	important := Drop(logEntries, 2)
	fmt.Println("Important logs:", important)
	// Output: Important logs: [INFO: Server listening on :8080 WARN: High memory usage ERROR: Database connection failed]
}

func ExampleDropWhile() {
	temperatures := []int{-5, -2, 0, 3, 8, 12, 15, 18}
	
	// Drop while temperature is below freezing
	aboveFreezing := DropWhile(temperatures, func(temp int) bool {
		return temp < 0
	})
	
	fmt.Println("Above/at freezing:", aboveFreezing)
	// Output: Above/at freezing: [0 3 8 12 15 18]
}

// ===============================
// Sorting Examples
// ===============================

func ExampleSort() {
	names := []string{"Charlie", "Alice", "Bob", "Diana"}
	
	sorted := Sort(names)
	fmt.Println("Sorted names:", sorted)
	fmt.Println("Original:", names) // Unchanged
	// Output:
	// Sorted names: [Alice Bob Charlie Diana]
	// Original: [Charlie Alice Bob Diana]
}

func ExampleSortBy() {
	type Person struct {
		Name string
		Age  int
	}
	
	people := []Person{
		{"Alice", 30},
		{"Bob", 25},
		{"Charlie", 35},
	}
	
	// Sort by age
	byAge := SortBy(people, func(a, b Person) bool {
		return a.Age < b.Age
	})
	
	for _, person := range byAge {
		fmt.Printf("%s (%d)\n", person.Name, person.Age)
	}
	// Output:
	// Bob (25)
	// Alice (30)
	// Charlie (35)
}

// ===============================
// String Conversion Examples
// ===============================

func ExampleJoin() {
	tags := []string{"golang", "programming", "tutorial", "slices"}
	
	// Create hashtag string
	hashtags := "#" + Join(tags, " #")
	fmt.Println(hashtags)
	// Output: #golang #programming #tutorial #slices
}

func ExampleJoin_csv() {
	data := []int{100, 250, 75, 430, 180}
	
	// Convert to CSV format
	csv := Join(data, ",")
	fmt.Println("CSV:", csv)
	// Output: CSV: 100,250,75,430,180
}

// ===============================
// Real-World Use Cases
// ===============================

func Example_realWorldDataProcessing() {
	// Example: Processing e-commerce order data
	type Order struct {
		ID     string
		Amount float64
		Status string
	}
	
	orders := []Order{
		{"ORD-001", 99.99, "completed"},
		{"ORD-002", 149.50, "pending"},
		{"ORD-003", 75.25, "completed"},
		{"ORD-004", 200.00, "cancelled"},
		{"ORD-005", 50.00, "completed"},
	}
	
	// Get completed orders
	completed := Filter(orders, func(o Order) bool {
		return o.Status == "completed"
	})
	
	// Calculate total revenue
	totalRevenue := Reduce(completed, 0.0, func(acc float64, o Order) float64 {
		return acc + o.Amount
	})
	
	// Get order IDs
	orderIds := Map(completed, func(o Order) string {
		return o.ID
	})
	
	fmt.Printf("Completed orders: %v\n", orderIds)
	fmt.Printf("Total revenue: $%.2f\n", totalRevenue)
	// Output:
	// Completed orders: [ORD-001 ORD-003 ORD-005]
	// Total revenue: $225.24
}

func Example_realWorldTextProcessing() {
	// Example: Processing log files
	logLines := []string{
		"2024-01-25 10:30:15 INFO User logged in: alice@example.com",
		"2024-01-25 10:30:20 DEBUG Cache hit for user: alice@example.com",
		"2024-01-25 10:30:25 ERROR Database connection failed",
		"2024-01-25 10:30:30 WARN High memory usage detected: 85%",
		"2024-01-25 10:30:35 INFO User logged out: alice@example.com",
	}
	
	// Extract error and warning messages
	important := Filter(logLines, func(line string) bool {
		return Contains([]string{"ERROR", "WARN"}, "ERROR") && strings.Contains(line, "ERROR") ||
			   Contains([]string{"ERROR", "WARN"}, "WARN") && strings.Contains(line, "WARN")
	})
	
	// Extract just the message part
	messages := Map(important, func(line string) string {
		parts := strings.SplitN(line, " ", 4)
		if len(parts) >= 4 {
			return parts[3]
		}
		return line
	})
	
	fmt.Println("Important messages:")
	ForEach(messages, func(msg string) {
		fmt.Printf("- %s\n", msg)
	})
	// Output:
	// Important messages:
	// - Database connection failed
	// - High memory usage detected: 85%
}

func Example_realWorldApiProcessing() {
	// Example: Processing API response data
	type APIResponse struct {
		UserID   int    `json:"user_id"`
		Username string `json:"username"`
		Status   string `json:"status"`
		Score    int    `json:"score"`
	}
	
	responses := []APIResponse{
		{1, "alice", "active", 95},
		{2, "bob", "inactive", 67},
		{3, "charlie", "active", 89},
		{4, "diana", "active", 92},
		{5, "eve", "suspended", 45},
	}
	
	// Get active users with high scores
	topActiveUsers := Filter(responses, func(r APIResponse) bool {
		return r.Status == "active" && r.Score >= 90
	})
	
	// Create summary report
	summary := Map(topActiveUsers, func(r APIResponse) string {
		return fmt.Sprintf("%s (ID: %d) - Score: %d", r.Username, r.UserID, r.Score)
	})
	
	fmt.Println("Top active users:")
	ForEach(summary, func(line string) {
		fmt.Printf("• %s\n", line)
	})
	// Output:
	// Top active users:
	// • alice (ID: 1) - Score: 95
	// • diana (ID: 4) - Score: 92
}