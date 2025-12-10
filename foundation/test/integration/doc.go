// Package integration provides comprehensive integration tests for the mDW Foundation library.
//
// Package: integration
// Title: mDW Foundation Integration Tests
// Description: This package contains integration tests that verify the correct
//              interaction between different mDW foundation modules, ensuring
//              consistent behavior, error handling, and performance characteristics
//              across module boundaries.
// Author: msto63 with Claude Sonnet 4.0
// Version: v0.1.0
// Created: 2025-01-25
// Modified: 2025-01-25
//
// Change History:
// - 2025-01-25 v0.1.0: Initial implementation of integration test suite
//
// Test Categories:
//
// Module Integration Tests (module_integration_test.go):
// - Cross-module error handling consistency
// - Data flow between modules (stringx → mathx, stringx → timex)
// - Validation pattern integration
// - Performance characteristics under realistic loads
// - Error recovery and graceful degradation
//
// Error Integration Tests (error_integration_test.go):
// - Standardized error format compliance across all modules
// - Error severity level consistency
// - Error code pattern verification
// - Error wrapping and unwrapping through module boundaries
// - Context preservation in error chains
// - Error builder pattern integration
//
// Performance Integration Tests (performance_test.go):
// - Cross-module operation benchmarks
// - Memory allocation analysis
// - Scalability testing with varying data sizes
// - Concurrency performance verification
// - Real-world scenario performance testing
//
// Test Coverage:
//
// The integration tests cover the following critical integration points:
//
// 1. Error Handling Integration:
//    - All modules use standardized mDW error types
//    - Consistent severity levels across modules
//    - Error context preservation through module boundaries
//    - Error wrapping and unwrapping chains
//
// 2. Data Flow Validation:
//    - String validation → decimal conversion (stringx → mathx)
//    - String validation → time parsing (stringx → timex)
//    - Error propagation through processing pipelines
//    - Validation chain integration
//
// 3. Performance Integration:
//    - Cross-module operation performance
//    - Memory allocation patterns
//    - Scalability under load
//    - Thread-safety verification
//
// 4. Real-World Scenarios:
//    - Financial transaction processing
//    - Date range calculations
//    - Input validation pipelines
//    - Error recovery patterns
//
// Running Integration Tests:
//
// To run all integration tests:
//   go test -v ./test/integration/
//
// To run specific test categories:
//   go test -v ./test/integration/ -run TestErrorHandling
//   go test -v ./test/integration/ -run TestCrossModule
//   go test -v ./test/integration/ -run TestPerformance
//
// To run performance benchmarks:
//   go test -v ./test/integration/ -bench=.
//   go test -v ./test/integration/ -bench=BenchmarkCrossModule
//
// Integration Test Requirements:
//
// 1. All modules must pass error handling integration tests
// 2. Cross-module data flows must be validated
// 3. Performance benchmarks must meet defined thresholds
// 4. Memory allocation patterns must be reasonable
// 5. Thread-safety must be verified under concurrency
//
// Failure Investigation:
//
// When integration tests fail, check:
// 1. Module-specific unit tests are passing
// 2. Error handling patterns are consistent
// 3. API changes haven't broken integration points
// 4. Performance regressions in cross-module operations
// 5. Memory leaks or excessive allocations
//
// Dependencies:
//
// These integration tests depend on:
// - pkg/core/error: mDW error framework
// - pkg/core/errors: Shared error utilities
// - pkg/utils/stringx: String manipulation utilities
// - pkg/utils/mathx: Mathematical operations
// - pkg/utils/timex: Time utilities
// - pkg/utils/mapx: Map operations
// - pkg/utils/slicex: Slice operations
// - pkg/utils/validationx: Validation utilities
// - pkg/utils/filex: File operations
//
// Best Practices:
//
// 1. Integration tests should focus on module boundaries
// 2. Test realistic usage patterns and data flows
// 3. Verify error propagation and context preservation
// 4. Include performance verification for critical paths
// 5. Test both success and failure scenarios
// 6. Verify thread-safety under concurrent access
//
package integration