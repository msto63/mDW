// File: example_test.go
// Title: Map Utilities Examples
// Description: Comprehensive examples demonstrating the usage of map utility
//              functions in practical scenarios.
// Author: msto63 with Claude Sonnet 4.0
// Version: v0.1.0
// Created: 2025-01-24
// Modified: 2025-01-24
//
// Change History:
// - 2025-01-24 v0.1.0: Initial implementation with practical examples

package mapx

import (
	"fmt"
	"sort"
	"strings"
)

func ExampleKeys() {
	userRoles := map[string]string{
		"alice": "admin",
		"bob":   "user",
		"carol": "moderator",
	}
	
	users := Keys(userRoles)
	sort.Strings(users) // Sort for consistent output
	
	fmt.Println("Users:", users)
	// Output: Users: [alice bob carol]
}

func ExampleValues() {
	userRoles := map[string]string{
		"alice": "admin",
		"bob":   "user",
		"carol": "moderator",
	}
	
	roles := Values(userRoles)
	sort.Strings(roles) // Sort for consistent output
	
	fmt.Println("Roles:", roles)
	// Output: Roles: [admin moderator user]
}

func ExampleInvert() {
	userRoles := map[string]string{
		"alice": "admin",
		"bob":   "user",
		"carol": "moderator",
	}
	
	roleUsers := Invert(userRoles)
	
	fmt.Printf("Admin user: %s\n", roleUsers["admin"])
	fmt.Printf("User user: %s\n", roleUsers["user"])
	// Output: Admin user: alice
	// User user: bob
}

func ExampleFilter() {
	products := map[string]int{
		"laptop":     1200,
		"mouse":      25,
		"keyboard":   80,
		"monitor":    300,
		"webcam":     60,
	}
	
	// Filter products over $100
	expensiveProducts := Filter(products, func(name string, price int) bool {
		return price > 100
	})
	
	fmt.Printf("Expensive products: %d items\n", len(expensiveProducts))
	
	// Sort keys for consistent output
	keys := Keys(expensiveProducts)
	sort.Strings(keys)
	for _, name := range keys {
		fmt.Printf("%s: $%d\n", name, expensiveProducts[name])
	}
	// Output: Expensive products: 2 items
	// laptop: $1200
	// monitor: $300
}

func ExampleFilterKeys() {
	inventory := map[string]int{
		"apple":  100,
		"banana": 50,
		"orange": 75,
		"grape":  30,
	}
	
	// Filter fruits that start with 'a' or 'b'
	abFruits := FilterKeys(inventory, func(fruit string) bool {
		return strings.HasPrefix(fruit, "a") || strings.HasPrefix(fruit, "b")
	})
	
	keys := Keys(abFruits)
	sort.Strings(keys)
	fmt.Println("A/B fruits:", keys)
	// Output: A/B fruits: [apple banana]
}

func ExampleFilterValues() {
	inventory := map[string]int{
		"apple":  100,
		"banana": 50,
		"orange": 75,
		"grape":  30,
	}
	
	// Filter fruits with high inventory (>60)
	highStock := FilterValues(inventory, func(count int) bool {
		return count > 60
	})
	
	keys := Keys(highStock)
	sort.Strings(keys)
	fmt.Println("High stock fruits:", keys)
	// Output: High stock fruits: [apple orange]
}

func ExampleMerge() {
	defaultConfig := map[string]interface{}{
		"timeout":    30,
		"retries":    3,
		"debug":      false,
		"cache_size": 100,
	}
	
	userConfig := map[string]interface{}{
		"timeout": 60,
		"debug":   true,
	}
	
	finalConfig := Merge(defaultConfig, userConfig)
	
	fmt.Printf("Timeout: %v\n", finalConfig["timeout"])
	fmt.Printf("Debug: %v\n", finalConfig["debug"])
	fmt.Printf("Retries: %v\n", finalConfig["retries"])
	// Output: Timeout: 60
	// Debug: true
	// Retries: 3
}

func ExampleClone() {
	original := map[string]int{
		"a": 1,
		"b": 2,
		"c": 3,
	}
	
	copy := Clone(original)
	copy["d"] = 4
	
	fmt.Printf("Original size: %d\n", len(original))
	fmt.Printf("Copy size: %d\n", len(copy))
	// Output: Original size: 3
	// Copy size: 4
}

func ExamplePick() {
	user := map[string]interface{}{
		"id":       123,
		"name":     "John Doe",
		"email":    "john@example.com",
		"password": "secret",
		"role":     "user",
		"active":   true,
	}
	
	// Pick only safe fields for API response
	publicUser := Pick(user, "id", "name", "email", "role", "active")
	
	fmt.Printf("Public fields: %d\n", len(publicUser))
	fmt.Printf("Has password: %t\n", HasKey(publicUser, "password"))
	// Output: Public fields: 5
	// Has password: false
}

func ExampleOmit() {
	user := map[string]interface{}{
		"id":       123,
		"name":     "John Doe",
		"email":    "john@example.com",
		"password": "secret",
		"token":    "abc123",
	}
	
	// Remove sensitive fields
	safeUser := Omit(user, "password", "token")
	
	fmt.Printf("Safe fields: %d\n", len(safeUser))
	fmt.Printf("Has password: %t\n", HasKey(safeUser, "password"))
	fmt.Printf("Has token: %t\n", HasKey(safeUser, "token"))
	// Output: Safe fields: 3
	// Has password: false
	// Has token: false
}

func ExampleRename() {
	apiResponse := map[string]interface{}{
		"user_id":    123,
		"user_name":  "John Doe",
		"user_email": "john@example.com",
	}
	
	keyMapping := map[string]string{
		"user_id":    "id",
		"user_name":  "name",
		"user_email": "email",
	}
	
	normalizedResponse := Rename(apiResponse, keyMapping)
	
	fmt.Printf("ID: %v\n", normalizedResponse["id"])
	fmt.Printf("Name: %v\n", normalizedResponse["name"])
	fmt.Printf("Has user_id: %t\n", HasKey(normalizedResponse, "user_id"))
	// Output: ID: 123
	// Name: John Doe
	// Has user_id: false
}

func ExampleTransform() {
	scores := map[string]int{
		"alice": 85,
		"bob":   92,
		"carol": 78,
	}
	
	// Transform to grade letters
	grades := Transform(scores, func(name string, score int) string {
		switch {
		case score >= 90:
			return "A"
		case score >= 80:
			return "B"
		case score >= 70:
			return "C"
		default:
			return "F"
		}
	})
	
	fmt.Printf("Alice: %s\n", grades["alice"])
	fmt.Printf("Bob: %s\n", grades["bob"])
	fmt.Printf("Carol: %s\n", grades["carol"])
	// Output: Alice: B
	// Bob: A
	// Carol: C
}

func ExampleTransformValues() {
	prices := map[string]float64{
		"laptop":  999.99,
		"mouse":   24.99,
		"keyboard": 79.99,
	}
	
	// Apply 10% discount
	discountedPrices := TransformValues(prices, func(price float64) float64 {
		return price * 0.9
	})
	
	fmt.Printf("Laptop: $%.2f\n", discountedPrices["laptop"])
	fmt.Printf("Mouse: $%.2f\n", discountedPrices["mouse"])
	// Output: Laptop: $899.99
	// Mouse: $22.49
}

func ExampleIntersect() {
	teamA := map[string]string{
		"alice": "developer",
		"bob":   "designer",
		"carol": "manager",
	}
	
	teamB := map[string]string{
		"bob":   "designer",
		"carol": "manager",
		"david": "developer",
	}
	
	sharedMembers := Intersect(teamA, teamB)
	
	members := Keys(sharedMembers)
	sort.Strings(members)
	fmt.Println("Shared members:", members)
	// Output: Shared members: [bob carol]
}

func ExampleDifference() {
	allFeatures := map[string]bool{
		"feature_a": true,
		"feature_b": true,
		"feature_c": true,
		"feature_d": true,
	}
	
	implementedFeatures := map[string]bool{
		"feature_a": true,
		"feature_c": true,
	}
	
	pendingFeatures := Difference(allFeatures, implementedFeatures)
	
	features := Keys(pendingFeatures)
	sort.Strings(features)
	fmt.Println("Pending features:", features)
	// Output: Pending features: [feature_b feature_d]
}

func ExampleEqual() {
	config1 := map[string]int{
		"timeout": 30,
		"retries": 3,
	}
	
	config2 := map[string]int{
		"retries": 3,
		"timeout": 30,
	}
	
	config3 := map[string]int{
		"timeout": 60,
		"retries": 3,
	}
	
	fmt.Printf("config1 == config2: %t\n", Equal(config1, config2))
	fmt.Printf("config1 == config3: %t\n", Equal(config1, config3))
	// Output: config1 == config2: true
	// config1 == config3: false
}

func ExampleToSlice() {
	data := map[string]int{
		"a": 1,
		"b": 2,
		"c": 3,
	}
	
	entries := ToSlice(data)
	
	// Sort by key for consistent output
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Key < entries[j].Key
	})
	
	for _, entry := range entries {
		fmt.Printf("%s: %d\n", entry.Key, entry.Value)
	}
	// Output: a: 1
	// b: 2
	// c: 3
}

func ExampleFromSlice() {
	entries := []Entry[string, int]{
		{Key: "apple", Value: 5},
		{Key: "banana", Value: 3},
		{Key: "orange", Value: 8},
	}
	
	inventory := FromSlice(entries)
	
	fmt.Printf("Apple count: %d\n", inventory["apple"])
	fmt.Printf("Total items: %d\n", len(inventory))
	// Output: Apple count: 5
	// Total items: 3
}

func ExampleSize() {
	data := map[string]int{
		"a": 1,
		"b": 2,
		"c": 3,
	}
	
	fmt.Printf("Map size: %d\n", Size(data))
	fmt.Printf("Empty map size: %d\n", Size(map[string]int{}))
	fmt.Printf("Nil map size: %d\n", Size(map[string]int(nil)))
	// Output: Map size: 3
	// Empty map size: 0
	// Nil map size: 0
}

func ExampleForEach() {
	inventory := map[string]int{
		"apple":  10,
		"banana": 5,
		"orange": 8,
	}
	
	total := 0
	ForEach(inventory, func(fruit string, count int) {
		total += count
		if fruit == "apple" {
			fmt.Printf("Found %d %ss\n", count, fruit)
		}
	})
	
	fmt.Printf("Total fruits: %d\n", total)
	// Output: Found 10 apples
	// Total fruits: 23
}

// Advanced example: Data processing pipeline
func Example_dataProcessingPipeline() {
	// Simulate API response data
	rawData := map[string]interface{}{
		"user_001": map[string]interface{}{"name": "Alice", "score": 85, "active": true},
		"user_002": map[string]interface{}{"name": "Bob", "score": 92, "active": true},
		"user_003": map[string]interface{}{"name": "Carol", "score": 78, "active": false},
		"user_004": map[string]interface{}{"name": "David", "score": 95, "active": true},
	}
	
	// Step 1: Extract just the user data
	userData := TransformValues(rawData, func(data interface{}) map[string]interface{} {
		return data.(map[string]interface{})
	})
	
	// Step 2: Filter only active users
	activeUsers := Filter(userData, func(id string, user map[string]interface{}) bool {
		return user["active"].(bool)
	})
	
	// Step 3: Transform to simplified structure with grades
	userGrades := Transform(activeUsers, func(id string, user map[string]interface{}) string {
		score := user["score"].(int)
		name := user["name"].(string)
		
		var grade string
		switch {
		case score >= 90:
			grade = "A"
		case score >= 80:
			grade = "B"
		default:
			grade = "C"
		}
		
		return name + " (" + grade + ")"
	})
	
	// Step 4: Show results
	fmt.Printf("Active users with grades: %d\n", Size(userGrades))
	
	// Get user_002 (Bob) as example
	if bobGrade, exists := userGrades["user_002"]; exists {
		fmt.Printf("Bob's grade: %s\n", bobGrade)
	}
	
	// Output: Active users with grades: 3
	// Bob's grade: Bob (A)
}