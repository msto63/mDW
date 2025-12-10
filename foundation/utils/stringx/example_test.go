// File: example_test.go
// Title: Example Tests for StringX Package Documentation
// Description: Executable examples that serve as both documentation and tests.
//              These examples demonstrate typical usage patterns and appear
//              in the generated documentation.
// Author: msto63 with Claude Sonnet 4.0
// Version: v0.1.0
// Created: 2025-01-24
// Modified: 2025-01-24
//
// Change History:
// - 2025-01-24 v0.1.0: Initial example implementation

package stringx_test

import (
	"fmt"
	mdwstringx "github.com/msto63/mDW/foundation/utils/stringx"
)

func ExampleIsEmpty() {
	fmt.Println(mdwstringx.IsEmpty(""))
	fmt.Println(mdwstringx.IsEmpty("hello"))
	fmt.Println(mdwstringx.IsEmpty(" "))
	// Output:
	// true
	// false
	// false
}

func ExampleIsBlank() {
	fmt.Println(mdwstringx.IsBlank(""))
	fmt.Println(mdwstringx.IsBlank("   "))
	fmt.Println(mdwstringx.IsBlank("hello"))
	fmt.Println(mdwstringx.IsBlank(" hello "))
	// Output:
	// true
	// true
	// false
	// false
}

func ExampleTruncate() {
	text := "This is a long text that needs to be truncated"
	
	fmt.Println(mdwstringx.Truncate(text, 20, "..."))
	fmt.Println(mdwstringx.Truncate(text, 50, "..."))
	fmt.Println(mdwstringx.Truncate("short", 10, "..."))
	// Output:
	// This is a long te...
	// This is a long text that needs to be truncated
	// short
}

func ExampleTruncate_unicode() {
	text := "これは日本語のテキストです"
	
	fmt.Println(mdwstringx.Truncate(text, 8, "..."))
	// Output:
	// これは日本...
}

func ExampleReverse() {
	fmt.Println(mdwstringx.Reverse("hello"))
	fmt.Println(mdwstringx.Reverse("world"))
	fmt.Println(mdwstringx.Reverse("こんにちは"))
	// Output:
	// olleh
	// dlrow
	// はちにんこ
}

func ExampleContainsIgnoreCase() {
	text := "Hello World"
	
	fmt.Println(mdwstringx.ContainsIgnoreCase(text, "WORLD"))
	fmt.Println(mdwstringx.ContainsIgnoreCase(text, "hello"))
	fmt.Println(mdwstringx.ContainsIgnoreCase(text, "xyz"))
	// Output:
	// true
	// true
	// false
}

func ExamplePadLeft() {
	fmt.Printf("|%s|\n", mdwstringx.PadLeft("hello", 10, ' '))
	fmt.Printf("|%s|\n", mdwstringx.PadLeft("123", 5, '0'))
	// Output:
	// |     hello|
	// |00123|
}

func ExamplePadRight() {
	fmt.Printf("|%s|\n", mdwstringx.PadRight("hello", 10, ' '))
	fmt.Printf("|%s|\n", mdwstringx.PadRight("test", 8, '-'))
	// Output:
	// |hello     |
	// |test----|
}

func ExampleCenter() {
	fmt.Printf("|%s|\n", mdwstringx.Center("test", 10, ' '))
	fmt.Printf("|%s|\n", mdwstringx.Center("hi", 6, '*'))
	// Output:
	// |   test   |
	// |**hi**|
}

func ExampleSplitLines() {
	text := "line1\nline2\r\nline3\rline4"
	lines := mdwstringx.SplitLines(text)
	
	for i, line := range lines {
		fmt.Printf("Line %d: %s\n", i+1, line)
	}
	// Output:
	// Line 1: line1
	// Line 2: line2
	// Line 3: line3
	// Line 4: line4
}

func ExampleFirstNonEmpty() {
	fmt.Println(mdwstringx.FirstNonEmpty("", "", "hello", "world"))
	fmt.Println(mdwstringx.FirstNonEmpty("first", "second"))
	fmt.Println(mdwstringx.FirstNonEmpty("", "", ""))
	// Output:
	// hello
	// first
	//
}

func ExampleFirstNonBlank() {
	fmt.Println(mdwstringx.FirstNonBlank("", "  ", "hello", "world"))
	fmt.Println(mdwstringx.FirstNonBlank("  ", "\t", ""))
	// Output:
	// hello
	//
}

func ExampleToSnakeCase() {
	fmt.Println(mdwstringx.ToSnakeCase("HelloWorld"))
	fmt.Println(mdwstringx.ToSnakeCase("myVariableName"))
	fmt.Println(mdwstringx.ToSnakeCase("HTTP Server"))
	fmt.Println(mdwstringx.ToSnakeCase("already_snake_case"))
	// Output:
	// hello_world
	// my_variable_name
	// http_server
	// already_snake_case
}

func ExampleToCamelCase() {
	fmt.Println(mdwstringx.ToCamelCase("hello_world"))
	fmt.Println(mdwstringx.ToCamelCase("my-variable-name"))
	fmt.Println(mdwstringx.ToCamelCase("test case"))
	fmt.Println(mdwstringx.ToCamelCase("alreadyCamelCase"))
	// Output:
	// helloWorld
	// myVariableName
	// testCase
	// alreadyCamelCase
}

func ExampleToPascalCase() {
	fmt.Println(mdwstringx.ToPascalCase("hello_world"))
	fmt.Println(mdwstringx.ToPascalCase("my-variable-name"))
	fmt.Println(mdwstringx.ToPascalCase("test case"))
	// Output:
	// HelloWorld
	// MyVariableName
	// TestCase
}

func ExampleToKebabCase() {
	fmt.Println(mdwstringx.ToKebabCase("HelloWorld"))
	fmt.Println(mdwstringx.ToKebabCase("myVariableName"))
	fmt.Println(mdwstringx.ToKebabCase("HTTP_Server"))
	// Output:
	// hello-world
	// my-variable-name
	// http-server
}

func ExampleToTitleCase() {
	fmt.Println(mdwstringx.ToTitleCase("hello world"))
	fmt.Println(mdwstringx.ToTitleCase("the quick brown fox"))
	// Output:
	// Hello World
	// The Quick Brown Fox
}

func ExampleRandomString() {
	// Generate a random string with custom charset
	result, err := mdwstringx.RandomString(8, "abc123")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	
	fmt.Printf("Length: %d\n", len(result))
	fmt.Printf("Contains only allowed chars: %t\n", 
		containsOnly(result, "abc123"))
	// Output:
	// Length: 8
	// Contains only allowed chars: true
}

func ExampleRandomAlphanumeric() {
	result, err := mdwstringx.RandomAlphanumeric(12)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	
	fmt.Printf("Length: %d\n", len(result))
	fmt.Printf("Is alphanumeric: %t\n", 
		containsOnly(result, mdwstringx.Alphanumeric))
	// Output:
	// Length: 12
	// Is alphanumeric: true
}

func ExampleRandomHex() {
	result, err := mdwstringx.RandomHex(16)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	
	fmt.Printf("Length: %d\n", len(result))
	fmt.Printf("Is hex: %t\n", 
		containsOnly(result, "0123456789abcdef"))
	// Output:
	// Length: 16
	// Is hex: true
}

func ExampleRandomPassword() {
	password, err := mdwstringx.RandomPassword(12)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	
	fmt.Printf("Length: %d\n", len(password))
	fmt.Printf("Has mixed characters: %t\n", 
		hasMixedCharacters(password))
	// Output:
	// Length: 12
	// Has mixed characters: true
}

// Helper functions for examples
func containsOnly(s, charset string) bool {
	for _, char := range s {
		found := false
		for _, allowed := range charset {
			if char == allowed {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func hasMixedCharacters(s string) bool {
	hasLower := false
	hasUpper := false
	hasDigit := false
	hasSpecial := false
	
	for _, char := range s {
		switch {
		case char >= 'a' && char <= 'z':
			hasLower = true
		case char >= 'A' && char <= 'Z':
			hasUpper = true
		case char >= '0' && char <= '9':
			hasDigit = true
		default:
			hasSpecial = true
		}
	}
	
	return hasLower && hasUpper && hasDigit && hasSpecial
}