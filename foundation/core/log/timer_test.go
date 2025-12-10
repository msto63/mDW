// File: timer_test.go
// Title: Timer Tests
// Description: Tests for performance timing functionality and integration
//              with the logging system.
// Author: msto63 with Claude Sonnet 4.0
// Version: v0.1.0
// Created: 2025-01-24
// Modified: 2025-01-24
//
// Change History:
// - 2025-01-24 v0.1.0: Initial implementation with comprehensive timer tests

package log

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestNewTimer(t *testing.T) {
	logger := New()
	timer := NewTimer(logger, "test-operation")
	
	if timer == nil {
		t.Fatal("NewTimer() should not return nil")
	}
	
	if timer.logger != logger {
		t.Error("Timer should reference the provided logger")
	}
	
	if timer.operation != "test-operation" {
		t.Errorf("Timer operation = %v, want test-operation", timer.operation)
	}
	
	if timer.startTime.IsZero() {
		t.Error("Timer should have a start time")
	}
	
	if timer.stopped {
		t.Error("New timer should not be stopped")
	}
	
	if timer.level != LevelDebug {
		t.Errorf("Timer default level = %v, want %v", timer.level, LevelDebug)
	}
}

func TestTimerWithLevel(t *testing.T) {
	logger := New()
	timer := NewTimer(logger, "test-operation")
	
	result := timer.WithLevel(LevelInfo)
	
	if result != timer {
		t.Error("WithLevel() should return the same timer instance")
	}
	
	if timer.level != LevelInfo {
		t.Errorf("WithLevel() level = %v, want %v", timer.level, LevelInfo)
	}
}

func TestTimerWithField(t *testing.T) {
	logger := New()
	timer := NewTimer(logger, "test-operation")
	
	result := timer.WithField("user_id", "123")
	
	if result != timer {
		t.Error("WithField() should return the same timer instance")
	}
	
	if timer.fields["user_id"] != "123" {
		t.Error("WithField() should add the field")
	}
}

func TestTimerWithFields(t *testing.T) {
	logger := New()
	timer := NewTimer(logger, "test-operation")
	
	fields := Fields{"user_id": "123", "request_id": "req-456"}
	result := timer.WithFields(fields)
	
	if result != timer {
		t.Error("WithFields() should return the same timer instance")
	}
	
	for k, v := range fields {
		if timer.fields[k] != v {
			t.Errorf("WithFields() should add field %s=%v", k, v)
		}
	}
}

func TestTimerElapsed(t *testing.T) {
	logger := New()
	timer := NewTimer(logger, "test-operation")
	
	// Wait a small amount of time
	time.Sleep(time.Millisecond)
	
	elapsed := timer.Elapsed()
	
	if elapsed <= 0 {
		t.Error("Elapsed() should return positive duration")
	}
	
	if elapsed < time.Millisecond {
		t.Error("Elapsed() should be at least 1ms")
	}
}

func TestTimerStop(t *testing.T) {
	var buf bytes.Buffer
	logger := New().WithOutput(&buf).WithFormat(FormatJSON).WithLevel(LevelDebug)
	timer := NewTimer(logger, "test-operation")
	
	// Wait a small amount
	time.Sleep(time.Millisecond)
	
	elapsed := timer.Stop()
	
	if elapsed <= 0 {
		t.Error("Stop() should return positive duration")
	}
	
	if !timer.stopped {
		t.Error("Stop() should mark timer as stopped")
	}
	
	// Should have logged completion
	if buf.Len() == 0 {
		t.Error("Stop() should log completion")
		return
	}
	
	var result map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("Failed to parse JSON output: %v", err)
	}
	
	if !strings.Contains(result["message"].(string), "completed") {
		t.Error("Stop() should log completion message")
	}
	
	if result["operation"] != "test-operation" {
		t.Error("Stop() should include operation name")
	}
	
	if _, exists := result["duration_ms"]; !exists {
		t.Error("Stop() should include duration in milliseconds")
	}
	
	// Second stop should return 0
	elapsed2 := timer.Stop()
	if elapsed2 != 0 {
		t.Error("Second Stop() should return 0")
	}
}

func TestTimerStopWithError(t *testing.T) {
	var buf bytes.Buffer
	logger := New().WithOutput(&buf).WithFormat(FormatJSON).WithLevel(LevelTrace)
	timer := NewTimer(logger, "test-operation")
	
	time.Sleep(time.Millisecond)
	err := errors.New("operation failed")
	elapsed := timer.StopWithError(err)
	
	if elapsed <= 0 {
		t.Error("StopWithError() should return positive duration")
	}
	
	if !timer.stopped {
		t.Error("StopWithError() should mark timer as stopped")
	}
	
	// Should have logged error
	if buf.Len() == 0 {
		t.Error("StopWithError() should log error")
		return
	}
	
	var result map[string]interface{}
	if jsonErr := json.Unmarshal(buf.Bytes(), &result); jsonErr != nil {
		t.Fatalf("Failed to parse JSON output: %v", jsonErr)
	}
	
	if result["level"] != "error" {
		t.Error("StopWithError() should log at error level")
	}
	
	if !strings.Contains(result["message"].(string), "failed") {
		t.Error("StopWithError() should log failure message")
	}
	
	if result["success"] != false {
		t.Error("StopWithError() should set success to false")
	}
	
	if result["error"] != "operation failed" {
		t.Error("StopWithError() should include error message")
	}
}

func TestTimerStopWithResult(t *testing.T) {
	var buf bytes.Buffer
	logger := New().WithOutput(&buf).WithFormat(FormatJSON).WithLevel(LevelTrace)
	timer := NewTimer(logger, "test-operation")
	
	time.Sleep(time.Millisecond)
	result := map[string]interface{}{"count": 42}
	elapsed := timer.StopWithResult(true, result)
	
	if elapsed <= 0 {
		t.Error("StopWithResult() should return positive duration")
	}
	
	if !timer.stopped {
		t.Error("StopWithResult() should mark timer as stopped")
	}
	
	// Should have logged result
	if buf.Len() == 0 {
		t.Error("StopWithResult() should log result")
		return
	}
	
	var logResult map[string]interface{}
	if jsonErr := json.Unmarshal(buf.Bytes(), &logResult); jsonErr != nil {
		t.Fatalf("Failed to parse JSON output: %v", jsonErr)
	}
	
	if !strings.Contains(logResult["message"].(string), "completed successfully") {
		t.Error("StopWithResult(true) should log success message")
	}
	
	if logResult["success"] != true {
		t.Error("StopWithResult(true) should set success to true")
	}
	
	if logResult["result"] == nil {
		t.Error("StopWithResult() should include result")
	}
}

func TestTimerStopWithResultFailure(t *testing.T) {
	var buf bytes.Buffer
	logger := New().WithOutput(&buf).WithFormat(FormatJSON)
	timer := NewTimer(logger, "test-operation").WithLevel(LevelInfo)
	
	timer.StopWithResult(false, nil)
	
	var logResult map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &logResult); err != nil {
		t.Fatalf("Failed to parse JSON output: %v", err)
	}
	
	if !strings.Contains(logResult["message"].(string), "completed with errors") {
		t.Error("StopWithResult(false) should log error message")
	}
	
	if logResult["success"] != false {
		t.Error("StopWithResult(false) should set success to false")
	}
	
	// Level should be elevated to warn for failures
	if logResult["level"] != "warn" {
		t.Error("StopWithResult(false) should elevate log level to warn")
	}
}

func TestTimerCheckpoint(t *testing.T) {
	var buf bytes.Buffer
	logger := New().WithOutput(&buf).WithFormat(FormatJSON).WithLevel(LevelDebug)
	timer := NewTimer(logger, "test-operation")
	
	time.Sleep(time.Millisecond)
	timer.Checkpoint("validation", Fields{"step": 1})
	
	if timer.stopped {
		t.Error("Checkpoint() should not stop the timer")
	}
	
	// Should have logged checkpoint
	if buf.Len() == 0 {
		t.Error("Checkpoint() should log checkpoint")
		return
	}
	
	var result map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("Failed to parse JSON output: %v", err)
	}
	
	if !strings.Contains(result["message"].(string), "checkpoint: validation") {
		t.Error("Checkpoint() should log checkpoint message")
	}
	
	if result["checkpoint"] != "validation" {
		t.Error("Checkpoint() should include checkpoint name")
	}
	
	if result["step"] != float64(1) {
		t.Error("Checkpoint() should include provided fields")
	}
	
	if _, exists := result["elapsed_ms"]; !exists {
		t.Error("Checkpoint() should include elapsed time")
	}
}

func TestTimerCheckpointAfterStop(t *testing.T) {
	var buf bytes.Buffer
	logger := New().WithOutput(&buf).WithFormat(FormatJSON)
	timer := NewTimer(logger, "test-operation")
	
	timer.Stop()
	buf.Reset() // Clear stop message
	
	timer.Checkpoint("should-not-log")
	
	if buf.Len() != 0 {
		t.Error("Checkpoint() after Stop() should not log")
	}
}

func TestTimerCancel(t *testing.T) {
	var buf bytes.Buffer
	logger := New().WithOutput(&buf).WithFormat(FormatJSON)
	timer := NewTimer(logger, "test-operation")
	
	timer.Cancel()
	
	if !timer.stopped {
		t.Error("Cancel() should mark timer as stopped")
	}
	
	// Should not have logged anything
	if buf.Len() != 0 {
		t.Error("Cancel() should not log anything")
	}
}

func TestTimerIsRunning(t *testing.T) {
	logger := New()
	timer := NewTimer(logger, "test-operation")
	
	if !timer.IsRunning() {
		t.Error("New timer should be running")
	}
	
	timer.Stop()
	
	if timer.IsRunning() {
		t.Error("Stopped timer should not be running")
	}
}

func TestTimerReset(t *testing.T) {
	logger := New()
	timer := NewTimer(logger, "test-operation")
	
	originalStart := timer.StartTime()
	time.Sleep(time.Millisecond * 2)
	
	timer.Reset()
	
	if !timer.IsRunning() {
		t.Error("Reset() should make timer running")
	}
	
	if timer.StartTime().Equal(originalStart) {
		t.Error("Reset() should update start time")
	}
	
	if timer.StartTime().Before(originalStart) {
		t.Error("Reset() should set start time to now or later")
	}
}

func TestTimerStartTime(t *testing.T) {
	logger := New()
	before := time.Now()
	timer := NewTimer(logger, "test-operation")
	after := time.Now()
	
	startTime := timer.StartTime()
	
	if startTime.Before(before) || startTime.After(after) {
		t.Error("StartTime() should be within creation time range")
	}
}

func TestTimerWithNilLogger(t *testing.T) {
	timer := NewTimer(nil, "test-operation")
	
	// Should not panic even with nil logger
	time.Sleep(time.Millisecond)
	elapsed := timer.Stop()
	
	if elapsed <= 0 {
		t.Error("Timer with nil logger should still measure time")
	}
}

func TestTimerConcurrentAccess(t *testing.T) {
	logger := New()
	timer := NewTimer(logger, "concurrent-test")
	
	// Start multiple goroutines that access the timer
	done := make(chan bool, 3)
	
	go func() {
		timer.WithField("goroutine", 1)
		done <- true
	}()
	
	go func() {
		timer.Checkpoint("checkpoint1")
		done <- true
	}()
	
	go func() {
		time.Sleep(time.Millisecond)
		timer.Stop()
		done <- true
	}()
	
	// Wait for all goroutines
	for i := 0; i < 3; i++ {
		<-done
	}
	
	// Should not panic and timer should be stopped
	if !timer.stopped {
		t.Error("Timer should be stopped after concurrent access")
	}
}

// Benchmark tests
func BenchmarkNewTimer(b *testing.B) {
	logger := New()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = NewTimer(logger, "benchmark-operation")
	}
}

func BenchmarkTimerStop(b *testing.B) {
	var buf bytes.Buffer
	logger := New().WithOutput(&buf)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		timer := NewTimer(logger, "benchmark-operation")
		timer.Stop()
	}
}

func BenchmarkTimerElapsed(b *testing.B) {
	logger := New()
	timer := NewTimer(logger, "benchmark-operation")
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = timer.Elapsed()
	}
}

func BenchmarkTimerCheckpoint(b *testing.B) {
	var buf bytes.Buffer
	logger := New().WithOutput(&buf)
	timer := NewTimer(logger, "benchmark-operation")
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		timer.Checkpoint("benchmark", Fields{"iteration": i})
	}
}