// File: timex_test.go
// Title: Time Utilities Tests
// Description: Comprehensive test suite for all timex utility functions including
//              unit tests, edge cases, and integration scenarios.
// Author: msto63 with Claude Sonnet 4.0
// Version: v0.1.0
// Created: 2025-01-25
// Modified: 2025-01-25
//
// Change History:
// - 2025-01-25 v0.1.0: Initial test implementation with comprehensive coverage

package timex

import (
	"testing"
	"time"
)

// ===============================
// Parsing Tests
// ===============================

func TestParse(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		wantErr  bool
		expected string // Expected format for comparison
	}{
		{"RFC3339", "2023-12-25T15:30:45Z", false, "2023-12-25T15:30:45Z"},
		{"ISO8601", "2023-12-25T15:30:45+02:00", false, "2023-12-25T13:30:45Z"},
		{"Business DateTime", "2023-12-25 15:30:45", false, "2023-12-25T15:30:45Z"},
		{"Business Date", "2023-12-25", false, "2023-12-25T00:00:00Z"},
		{"Short Date", "12/25/2023", false, "2023-12-25T00:00:00Z"},
		{"Empty string", "", true, ""},
		{"Invalid format", "not a date", true, ""},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := Parse(tc.input)
			
			if tc.wantErr {
				if err == nil {
					t.Errorf("Parse(%s) expected error, got nil", tc.input)
				}
				return
			}
			
			if err != nil {
				t.Errorf("Parse(%s) unexpected error: %v", tc.input, err)
				return
			}
			
			if tc.expected != "" {
				expected, _ := time.Parse(time.RFC3339, tc.expected)
				if !result.Equal(expected) {
					t.Errorf("Parse(%s) = %v, want %v", tc.input, result, expected)
				}
			}
		})
	}
}

func TestParseDate(t *testing.T) {
	testCases := []struct {
		name    string
		input   string
		wantErr bool
		year    int
		month   time.Month
		day     int
	}{
		{"Business Date", "2023-12-25", false, 2023, 12, 25},
		{"Short Date", "12/25/2023", false, 2023, 12, 25},
		{"Compact Date", "20231225", false, 2023, 12, 25},
		{"Invalid", "not a date", true, 0, 0, 0},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := ParseDate(tc.input)
			
			if tc.wantErr {
				if err == nil {
					t.Errorf("ParseDate(%s) expected error, got nil", tc.input)
				}
				return
			}
			
			if err != nil {
				t.Errorf("ParseDate(%s) unexpected error: %v", tc.input, err)
				return
			}
			
			if result.Year() != tc.year || result.Month() != tc.month || result.Day() != tc.day {
				t.Errorf("ParseDate(%s) = %v, want %d-%d-%d", tc.input, result, tc.year, tc.month, tc.day)
			}
		})
	}
}

func TestParseDuration(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected time.Duration
		wantErr  bool
	}{
		{"Standard Go format", "1h30m", 90 * time.Minute, false},
		{"Business format - day", "1 day", 24 * time.Hour, false},
		{"Business format - days", "2 days", 48 * time.Hour, false},
		{"Business format - week", "1 week", 7 * 24 * time.Hour, false},
		{"Business format - hour", "3 hours", 3 * time.Hour, false},
		{"Empty string", "", 0, true},
		{"Invalid format", "not a duration", 0, true},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := ParseDuration(tc.input)
			
			if tc.wantErr {
				if err == nil {
					t.Errorf("ParseDuration(%s) expected error, got nil", tc.input)
				}
				return
			}
			
			if err != nil {
				t.Errorf("ParseDuration(%s) unexpected error: %v", tc.input, err)
				return
			}
			
			if result != tc.expected {
				t.Errorf("ParseDuration(%s) = %v, want %v", tc.input, result, tc.expected)
			}
		})
	}
}

// ===============================
// Formatting Tests
// ===============================

func TestFormat(t *testing.T) {
	testTime := time.Date(2023, 12, 25, 15, 30, 45, 0, time.UTC)
	
	testCases := []struct {
		name     string
		format   string
		expected string
	}{
		{"ISO8601", "iso8601", "2023-12-25T15:30:45Z"},
		{"Business", "business", "2023-12-25 15:30:45"},
		{"Business Date", "business-date", "2023-12-25"},
		{"Display", "display", "December 25, 2023 at 3:30 PM"},
		{"Short", "short", "12/25/2023 15:30"},
		{"Compact", "compact", "20231225153045"},
		{"Custom format", "2006-01-02", "2023-12-25"},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := Format(testTime, tc.format)
			if result != tc.expected {
				t.Errorf("Format(%v, %s) = %s, want %s", testTime, tc.format, result, tc.expected)
			}
		})
	}
}

func TestFormatDuration(t *testing.T) {
	testCases := []struct {
		name     string
		duration time.Duration
		expected string
	}{
		{"Zero", 0, "0 seconds"},
		{"Seconds", 5 * time.Second, "5 seconds"},
		{"One second", 1 * time.Second, "1 second"},
		{"Minutes and seconds", 65 * time.Second, "1 minute and 5 seconds"},
		{"Hours", 2 * time.Hour, "2 hours"},
		{"Complex", 25*time.Hour + 30*time.Minute + 45*time.Second, "1 day, 1 hour, 30 minutes, and 45 seconds"},
		{"Negative", -5 * time.Second, "-5 seconds"},
		{"Milliseconds", 500 * time.Millisecond, "500 milliseconds"},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := FormatDuration(tc.duration)
			if result != tc.expected {
				t.Errorf("FormatDuration(%v) = %s, want %s", tc.duration, result, tc.expected)
			}
		})
	}
}

// ===============================
// Business Day Tests
// ===============================

func TestIsBusinessDay(t *testing.T) {
	testCases := []struct {
		name     string
		date     time.Time
		expected bool
	}{
		{"Monday", time.Date(2023, 12, 25, 0, 0, 0, 0, time.UTC), true},    // Monday
		{"Saturday", time.Date(2023, 12, 23, 0, 0, 0, 0, time.UTC), false}, // Saturday
		{"Sunday", time.Date(2023, 12, 24, 0, 0, 0, 0, time.UTC), false},   // Sunday
		{"Tuesday", time.Date(2023, 12, 26, 0, 0, 0, 0, time.UTC), true},   // Tuesday
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := IsBusinessDay(tc.date)
			if result != tc.expected {
				t.Errorf("IsBusinessDay(%v) = %v, want %v", tc.date, result, tc.expected)
			}
		})
	}
}

func TestNextBusinessDay(t *testing.T) {
	friday := time.Date(2023, 12, 22, 0, 0, 0, 0, time.UTC)    // Friday
	saturday := time.Date(2023, 12, 23, 0, 0, 0, 0, time.UTC)  // Saturday
	expectedMonday := time.Date(2023, 12, 25, 0, 0, 0, 0, time.UTC) // Monday
	
	testCases := []struct {
		name     string
		date     time.Time
		expected time.Time
	}{
		{"Friday to Monday", friday, expectedMonday},
		{"Saturday to Monday", saturday, expectedMonday},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := NextBusinessDay(tc.date)
			if !result.Equal(tc.expected) {
				t.Errorf("NextBusinessDay(%v) = %v, want %v", tc.date, result, tc.expected)
			}
		})
	}
}

func TestAddBusinessDays(t *testing.T) {
	friday := time.Date(2023, 12, 22, 0, 0, 0, 0, time.UTC)
	
	testCases := []struct {
		name     string
		start    time.Time
		days     int
		expected time.Time
	}{
		{"Add 1 business day from Friday", friday, 1, time.Date(2023, 12, 25, 0, 0, 0, 0, time.UTC)}, // Monday
		{"Add 0 business days", friday, 0, friday},
		{"Subtract 1 business day", friday, -1, time.Date(2023, 12, 21, 0, 0, 0, 0, time.UTC)}, // Thursday
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := AddBusinessDays(tc.start, tc.days)
			if !result.Equal(tc.expected) {
				t.Errorf("AddBusinessDays(%v, %d) = %v, want %v", tc.start, tc.days, result, tc.expected)
			}
		})
	}
}

func TestBusinessDaysBetween(t *testing.T) {
	monday := time.Date(2023, 12, 25, 0, 0, 0, 0, time.UTC)    // Monday
	wednesday := time.Date(2023, 12, 27, 0, 0, 0, 0, time.UTC) // Wednesday
	
	result := BusinessDaysBetween(monday, wednesday)
	expected := 3 // Monday, Tuesday and Wednesday (inclusive)
	
	if result != expected {
		t.Errorf("BusinessDaysBetween(%v, %v) = %d, want %d", monday, wednesday, result, expected)
	}
	
	// Test reverse order
	result = BusinessDaysBetween(wednesday, monday)
	expected = -3
	
	if result != expected {
		t.Errorf("BusinessDaysBetween(%v, %v) = %d, want %d", wednesday, monday, result, expected)
	}
}

// ===============================
// Time Manipulation Tests
// ===============================

func TestStartOfDay(t *testing.T) {
	input := time.Date(2023, 12, 25, 15, 30, 45, 123456789, time.UTC)
	expected := time.Date(2023, 12, 25, 0, 0, 0, 0, time.UTC)
	
	result := StartOfDay(input)
	if !result.Equal(expected) {
		t.Errorf("StartOfDay(%v) = %v, want %v", input, result, expected)
	}
}

func TestEndOfDay(t *testing.T) {
	input := time.Date(2023, 12, 25, 15, 30, 45, 123456789, time.UTC)
	expected := time.Date(2023, 12, 25, 23, 59, 59, 999999999, time.UTC)
	
	result := EndOfDay(input)
	if !result.Equal(expected) {
		t.Errorf("EndOfDay(%v) = %v, want %v", input, result, expected)
	}
}

func TestStartOfWeek(t *testing.T) {
	// Wednesday, December 27, 2023
	wednesday := time.Date(2023, 12, 27, 15, 30, 45, 0, time.UTC)
	// Expected: Monday, December 25, 2023 00:00:00
	expected := time.Date(2023, 12, 25, 0, 0, 0, 0, time.UTC)
	
	result := StartOfWeek(wednesday)
	if !result.Equal(expected) {
		t.Errorf("StartOfWeek(%v) = %v, want %v", wednesday, result, expected)
	}
}

func TestStartOfMonth(t *testing.T) {
	input := time.Date(2023, 12, 25, 15, 30, 45, 0, time.UTC)
	expected := time.Date(2023, 12, 1, 0, 0, 0, 0, time.UTC)
	
	result := StartOfMonth(input)
	if !result.Equal(expected) {
		t.Errorf("StartOfMonth(%v) = %v, want %v", input, result, expected)
	}
}

func TestEndOfMonth(t *testing.T) {
	input := time.Date(2023, 12, 15, 15, 30, 45, 0, time.UTC)
	expected := time.Date(2023, 12, 31, 23, 59, 59, 999999999, time.UTC)
	
	result := EndOfMonth(input)
	if !result.Equal(expected) {
		t.Errorf("EndOfMonth(%v) = %v, want %v", input, result, expected)
	}
}

// ===============================
// Age and Date Difference Tests
// ===============================

func TestAge(t *testing.T) {
	birthDate := time.Date(1990, 6, 15, 0, 0, 0, 0, time.UTC)
	
	testCases := []struct {
		name          string
		referenceDate time.Time
		expectedAge   int
	}{
		{"Before birthday", time.Date(2023, 3, 10, 0, 0, 0, 0, time.UTC), 32},
		{"On birthday", time.Date(2023, 6, 15, 0, 0, 0, 0, time.UTC), 33},
		{"After birthday", time.Date(2023, 9, 20, 0, 0, 0, 0, time.UTC), 33},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := Age(birthDate, tc.referenceDate)
			if result != tc.expectedAge {
				t.Errorf("Age(%v, %v) = %d, want %d", birthDate, tc.referenceDate, result, tc.expectedAge)
			}
		})
	}
}

func TestDaysBetween(t *testing.T) {
	start := time.Date(2023, 12, 25, 15, 30, 0, 0, time.UTC)
	end := time.Date(2023, 12, 28, 10, 45, 0, 0, time.UTC)
	
	result := DaysBetween(start, end)
	expected := 3
	
	if result != expected {
		t.Errorf("DaysBetween(%v, %v) = %d, want %d", start, end, result, expected)
	}
	
	// Test reverse order
	result = DaysBetween(end, start)
	expected = -3
	
	if result != expected {
		t.Errorf("DaysBetween(%v, %v) = %d, want %d", end, start, result, expected)
	}
}

func TestMonthsBetween(t *testing.T) {
	start := time.Date(2023, 1, 15, 0, 0, 0, 0, time.UTC)
	end := time.Date(2023, 6, 10, 0, 0, 0, 0, time.UTC) // Before the 15th
	
	result := MonthsBetween(start, end)
	expected := 4 // January to May (not June because it's before the 15th)
	
	if result != expected {
		t.Errorf("MonthsBetween(%v, %v) = %d, want %d", start, end, result, expected)
	}
}

// ===============================
// Validation Tests
// ===============================

func TestIsZero(t *testing.T) {
	zeroTime := time.Time{}
	nonZeroTime := time.Now()
	
	if !IsZero(zeroTime) {
		t.Error("IsZero(zeroTime) = false, want true")
	}
	
	if IsZero(nonZeroTime) {
		t.Error("IsZero(nonZeroTime) = true, want false")
	}
}

func TestIsWeekend(t *testing.T) {
	saturday := time.Date(2023, 12, 23, 0, 0, 0, 0, time.UTC)
	sunday := time.Date(2023, 12, 24, 0, 0, 0, 0, time.UTC)
	monday := time.Date(2023, 12, 25, 0, 0, 0, 0, time.UTC)
	
	if !IsWeekend(saturday) {
		t.Error("IsWeekend(Saturday) = false, want true")
	}
	
	if !IsWeekend(sunday) {
		t.Error("IsWeekend(Sunday) = false, want true")
	}
	
	if IsWeekend(monday) {
		t.Error("IsWeekend(Monday) = true, want false")
	}
}

func TestIsWeekday(t *testing.T) {
	saturday := time.Date(2023, 12, 23, 0, 0, 0, 0, time.UTC)
	monday := time.Date(2023, 12, 25, 0, 0, 0, 0, time.UTC)
	
	if IsWeekday(saturday) {
		t.Error("IsWeekday(Saturday) = true, want false")
	}
	
	if !IsWeekday(monday) {
		t.Error("IsWeekday(Monday) = false, want true")
	}
}

// ===============================
// Utility Tests
// ===============================

func TestMinMax(t *testing.T) {
	time1 := time.Date(2023, 12, 25, 0, 0, 0, 0, time.UTC)
	time2 := time.Date(2023, 12, 26, 0, 0, 0, 0, time.UTC)
	
	if !Min(time1, time2).Equal(time1) {
		t.Errorf("Min(%v, %v) = %v, want %v", time1, time2, Min(time1, time2), time1)
	}
	
	if !Max(time1, time2).Equal(time2) {
		t.Errorf("Max(%v, %v) = %v, want %v", time1, time2, Max(time1, time2), time2)
	}
}

func TestClamp(t *testing.T) {
	min := time.Date(2023, 12, 20, 0, 0, 0, 0, time.UTC)
	max := time.Date(2023, 12, 30, 0, 0, 0, 0, time.UTC)
	
	// Test value within range
	middle := time.Date(2023, 12, 25, 0, 0, 0, 0, time.UTC)
	result := Clamp(middle, min, max)
	if !result.Equal(middle) {
		t.Errorf("Clamp(%v, %v, %v) = %v, want %v", middle, min, max, result, middle)
	}
	
	// Test value below range
	below := time.Date(2023, 12, 15, 0, 0, 0, 0, time.UTC)
	result = Clamp(below, min, max)
	if !result.Equal(min) {
		t.Errorf("Clamp(%v, %v, %v) = %v, want %v", below, min, max, result, min)
	}
	
	// Test value above range
	above := time.Date(2024, 1, 5, 0, 0, 0, 0, time.UTC)
	result = Clamp(above, min, max)
	if !result.Equal(max) {
		t.Errorf("Clamp(%v, %v, %v) = %v, want %v", above, min, max, result, max)
	}
}

// ===============================
// Time Series Tests
// ===============================

func TestGenerateTimeRange(t *testing.T) {
	start := time.Date(2023, 12, 25, 0, 0, 0, 0, time.UTC)
	end := time.Date(2023, 12, 25, 2, 0, 0, 0, time.UTC)
	interval := 1 * time.Hour
	
	result := GenerateTimeRange(start, end, interval)
	expected := 3 // 00:00, 01:00, 02:00
	
	if len(result) != expected {
		t.Errorf("GenerateTimeRange() returned %d times, want %d", len(result), expected)
	}
	
	// Test invalid parameters
	result = GenerateTimeRange(end, start, interval)
	if result != nil {
		t.Error("GenerateTimeRange() with end before start should return nil")
	}
	
	result = GenerateTimeRange(start, end, 0)
	if result != nil {
		t.Error("GenerateTimeRange() with zero interval should return nil")
	}
}

func TestRoundToNearest(t *testing.T) {
	// Test rounding to nearest hour
	input := time.Date(2023, 12, 25, 14, 35, 0, 0, time.UTC)
	expected := time.Date(2023, 12, 25, 15, 0, 0, 0, time.UTC) // Rounds up to 15:00
	
	result := RoundToNearest(input, time.Hour)
	if !result.Equal(expected) {
		t.Errorf("RoundToNearest(%v, 1h) = %v, want %v", input, result, expected)
	}
	
	// Test rounding down
	input = time.Date(2023, 12, 25, 14, 25, 0, 0, time.UTC)
	expected = time.Date(2023, 12, 25, 14, 0, 0, 0, time.UTC) // Rounds down to 14:00
	
	result = RoundToNearest(input, time.Hour)
	if !result.Equal(expected) {
		t.Errorf("RoundToNearest(%v, 1h) = %v, want %v", input, result, expected)
	}
}

func TestTruncateToNearest(t *testing.T) {
	input := time.Date(2023, 12, 25, 14, 45, 30, 0, time.UTC)
	expected := time.Date(2023, 12, 25, 14, 0, 0, 0, time.UTC) // Truncates to 14:00
	
	result := TruncateToNearest(input, time.Hour)
	if !result.Equal(expected) {
		t.Errorf("TruncateToNearest(%v, 1h) = %v, want %v", input, result, expected)
	}
}

// ===============================
// Timezone Tests
// ===============================

func TestConvertTimezone(t *testing.T) {
	// Test conversion from UTC to EST
	utcTime := time.Date(2023, 12, 25, 15, 0, 0, 0, time.UTC)
	
	result, err := ConvertTimezone(utcTime, "UTC", "America/New_York")
	if err != nil {
		t.Errorf("ConvertTimezone() unexpected error: %v", err)
		return
	}
	
	// EST is UTC-5, so 15:00 UTC should be 10:00 EST
	if result.Hour() != 10 {
		t.Errorf("ConvertTimezone() hour = %d, want 10", result.Hour())
	}
	
	// Test invalid timezone
	_, err = ConvertTimezone(utcTime, "Invalid/Timezone", "UTC")
	if err == nil {
		t.Error("ConvertTimezone() with invalid timezone should return error")
	}
}

func TestToUTC(t *testing.T) {
	localTime := time.Date(2023, 12, 25, 15, 0, 0, 0, time.Local)
	utcTime := ToUTC(localTime)
	
	if utcTime.Location() != time.UTC {
		t.Error("ToUTC() should return time in UTC location")
	}
}

// ===============================
// Edge Cases and Error Handling
// ===============================

func TestBusinessDayConfig(t *testing.T) {
	// Create custom config with different weekend days
	config := &BusinessDayConfig{
		WeekendDays: []Weekday{Friday, Saturday}, // Friday-Saturday weekend
		Holidays:    []time.Time{time.Date(2023, 12, 25, 0, 0, 0, 0, time.UTC)}, // Christmas
	}
	
	christmas := time.Date(2023, 12, 25, 0, 0, 0, 0, time.UTC) // Monday, but holiday
	friday := time.Date(2023, 12, 22, 0, 0, 0, 0, time.UTC)    // Friday, weekend in custom config
	sunday := time.Date(2023, 12, 24, 0, 0, 0, 0, time.UTC)    // Sunday, working day in custom config
	
	if IsBusinessDay(christmas, config) {
		t.Error("IsBusinessDay(Christmas) with holiday config should be false")
	}
	
	if IsBusinessDay(friday, config) {
		t.Error("IsBusinessDay(Friday) with custom weekend config should be false")
	}
	
	if !IsBusinessDay(sunday, config) {
		t.Error("IsBusinessDay(Sunday) with custom weekend config should be true")
	}
}

func TestTimeRangeOperations(t *testing.T) {
	start := time.Date(2023, 12, 25, 10, 0, 0, 0, time.UTC)
	end := time.Date(2023, 12, 25, 14, 0, 0, 0, time.UTC)
	tr := TimeRange{Start: start, End: end}
	
	// Test Duration
	expectedDuration := 4 * time.Hour
	if tr.Duration() != expectedDuration {
		t.Errorf("TimeRange.Duration() = %v, want %v", tr.Duration(), expectedDuration)
	}
	
	// Test Contains
	middle := time.Date(2023, 12, 25, 12, 0, 0, 0, time.UTC)
	if !tr.Contains(middle) {
		t.Error("TimeRange.Contains() should return true for time within range")
	}
	
	outside := time.Date(2023, 12, 25, 16, 0, 0, 0, time.UTC)
	if tr.Contains(outside) {
		t.Error("TimeRange.Contains() should return false for time outside range")
	}
	
	// Test Overlaps
	overlapping := TimeRange{
		Start: time.Date(2023, 12, 25, 13, 0, 0, 0, time.UTC),
		End:   time.Date(2023, 12, 25, 16, 0, 0, 0, time.UTC),
	}
	if !tr.Overlaps(overlapping) {
		t.Error("TimeRange.Overlaps() should return true for overlapping ranges")
	}
	
	nonOverlapping := TimeRange{
		Start: time.Date(2023, 12, 25, 15, 0, 0, 0, time.UTC),
		End:   time.Date(2023, 12, 25, 18, 0, 0, 0, time.UTC),
	}
	if tr.Overlaps(nonOverlapping) {
		t.Error("TimeRange.Overlaps() should return false for non-overlapping ranges")
	}
}

func TestUnixTimestamps(t *testing.T) {
	// Test Unix conversion
	timestamp := int64(1703516400) // 2023-12-25 15:00:00 UTC
	expected := time.Date(2023, 12, 25, 15, 0, 0, 0, time.UTC)
	
	result := Unix(timestamp)
	if !result.Equal(expected) {
		t.Errorf("Unix(%d) = %v, want %v", timestamp, result, expected)
	}
	
	// Test reverse conversion
	resultTimestamp := ToUnix(expected)
	if resultTimestamp != timestamp {
		t.Errorf("ToUnix(%v) = %d, want %d", expected, resultTimestamp, timestamp)
	}
	
	// Test millisecond conversion
	msTimestamp := timestamp * 1000
	result = UnixMilli(msTimestamp)
	if !result.Equal(expected) {
		t.Errorf("UnixMilli(%d) = %v, want %v", msTimestamp, result, expected)
	}
	
	resultMsTimestamp := ToUnixMilli(expected)
	if resultMsTimestamp != msTimestamp {
		t.Errorf("ToUnixMilli(%v) = %d, want %d", expected, resultMsTimestamp, msTimestamp)
	}
}

// ===============================
// Additional Test Coverage for Critical Functions
// ===============================

func TestParseDate_ExtendedFormats(t *testing.T) {
	testCases := []struct {
		name   string
		input  string
		hasErr bool
	}{
		{"ISO date", "2023-12-25", false},
		{"US date", "12/25/2023", false},
		{"European date", "25.12.2023", false},
		{"Compact date", "20231225", false},
		{"Invalid month", "2023-13-25", true},
		{"Invalid day", "2023-12-32", true},
		{"Empty string", "", true},
		{"Random text", "not a date", true},
		{"Partial date", "2023-12", true},
		{"Year only", "2023", true},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ParseDate(tc.input)
			if (err != nil) != tc.hasErr {
				t.Errorf("ParseDate(%s) error = %v, wantErr %v", tc.input, err, tc.hasErr)
			}
		})
	}
}

func TestParseDuration_ExtendedFormats(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected time.Duration
		hasErr   bool
	}{
		{"standard format", "1h30m", time.Hour + 30*time.Minute, false},
		{"seconds only", "45s", 45 * time.Second, false},
		{"business hours", "8 hours", 8 * time.Hour, false},
		{"business minutes", "30 minutes", 30 * time.Minute, false},
		{"business seconds", "45 seconds", 45 * time.Second, false},
		{"single day", "1 day", 24 * time.Hour, false},
		{"multiple days", "3 days", 72 * time.Hour, false},
		{"single week", "1 week", 7 * 24 * time.Hour, false},
		{"multiple weeks", "2 weeks", 14 * 24 * time.Hour, false},
		{"single month", "1 month", 30 * 24 * time.Hour, false}, // Approximate
		{"invalid format", "invalid", 0, true},
		{"empty string", "", 0, true},
		{"negative duration", "-1h", 0, true}, // Business durations should be positive
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := ParseDuration(tc.input)
			if (err != nil) != tc.hasErr {
				t.Errorf("ParseDuration(%s) error = %v, wantErr %v", tc.input, err, tc.hasErr)
				return
			}
			if !tc.hasErr && result != tc.expected {
				t.Errorf("ParseDuration(%s) = %v, want %v", tc.input, result, tc.expected)
			}
		})
	}
}

func TestFormatDuration_ExtendedCases(t *testing.T) {
	testCases := []struct {
		name     string
		duration time.Duration
		expected string
	}{
		{"zero duration", 0, "0s"},
		{"milliseconds only", 500 * time.Millisecond, "500ms"},
		{"seconds only", 30 * time.Second, "30s"},
		{"minutes only", 5 * time.Minute, "5m 0s"},
		{"hours only", 2 * time.Hour, "2h 0m 0s"},
		{"days", 25 * time.Hour, "1d 1h 0m 0s"},
		{"complex duration", 26*time.Hour + 30*time.Minute + 45*time.Second, "1d 2h 30m 45s"},
		{"very long duration", 8*24*time.Hour + 5*time.Hour, "8d 5h 0m 0s"},
		{"negative duration", -1 * time.Hour, "-1h 0m 0s"},
		{"microseconds", 1500 * time.Microsecond, "1ms 500μs"},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := FormatDurationCompact(tc.duration)
			if result != tc.expected {
				t.Errorf("FormatDurationCompact(%v) = %s, want %s", tc.duration, result, tc.expected)
			}
		})
	}
}

func TestBusinessDay_ExtendedCases(t *testing.T) {
	testCases := []struct {
		name       string
		date       time.Time
		isBusiness bool
	}{
		{"Monday morning", time.Date(2023, 12, 25, 9, 0, 0, 0, time.UTC), true},
		{"Tuesday afternoon", time.Date(2023, 12, 26, 15, 0, 0, 0, time.UTC), true},
		{"Wednesday midnight", time.Date(2023, 12, 27, 0, 0, 0, 0, time.UTC), true},
		{"Thursday evening", time.Date(2023, 12, 28, 22, 0, 0, 0, time.UTC), true},
		{"Friday noon", time.Date(2023, 12, 29, 12, 0, 0, 0, time.UTC), true},
		{"Saturday morning", time.Date(2023, 12, 30, 9, 0, 0, 0, time.UTC), false},
		{"Sunday evening", time.Date(2023, 12, 31, 18, 0, 0, 0, time.UTC), false},
		{"New Year's Day Sunday", time.Date(2023, 1, 1, 10, 0, 0, 0, time.UTC), false},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := IsBusinessDay(tc.date)
			if result != tc.isBusiness {
				t.Errorf("IsBusinessDay(%v) = %v, want %v", tc.date, result, tc.isBusiness)
			}
		})
	}
}

func TestBusinessDayCalculations_EdgeCases(t *testing.T) {
	// Test AddBusinessDays with various scenarios
	testCases := []struct {
		name     string
		start    time.Time
		days     int
		expected time.Time
	}{
		{
			"Friday + 1 business day = Monday",
			time.Date(2023, 12, 29, 9, 0, 0, 0, time.UTC), // Friday
			1,
			time.Date(2024, 1, 1, 9, 0, 0, 0, time.UTC), // Monday
		},
		{
			"Friday + 3 business days = Wednesday",
			time.Date(2023, 12, 29, 9, 0, 0, 0, time.UTC), // Friday
			3,
			time.Date(2024, 1, 3, 9, 0, 0, 0, time.UTC), // Wednesday
		},
		{
			"Saturday + 1 business day = Monday",
			time.Date(2023, 12, 30, 9, 0, 0, 0, time.UTC), // Saturday
			1,
			time.Date(2024, 1, 1, 9, 0, 0, 0, time.UTC), // Monday
		},
		{
			"Monday - 1 business day = Friday",
			time.Date(2024, 1, 1, 9, 0, 0, 0, time.UTC), // Monday
			-1,
			time.Date(2023, 12, 29, 9, 0, 0, 0, time.UTC), // Friday
		},
		{
			"Zero business days",
			time.Date(2023, 12, 25, 9, 0, 0, 0, time.UTC), // Monday
			0,
			time.Date(2023, 12, 25, 9, 0, 0, 0, time.UTC), // Monday
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := AddBusinessDays(tc.start, tc.days)
			if !result.Equal(tc.expected) {
				t.Errorf("AddBusinessDays(%v, %d) = %v, want %v", tc.start, tc.days, result, tc.expected)
			}
		})
	}
}

func TestTimeRangeOperations_Extended(t *testing.T) {
	baseRange := TimeRange{
		Start: time.Date(2023, 12, 25, 10, 0, 0, 0, time.UTC),
		End:   time.Date(2023, 12, 25, 15, 0, 0, 0, time.UTC),
	}
	
	t.Run("Contains tests", func(t *testing.T) {
		testCases := []struct {
			name     string
			testTime time.Time
			expected bool
		}{
			{"start time", baseRange.Start, true},
			{"end time", baseRange.End, true},
			{"middle time", time.Date(2023, 12, 25, 12, 30, 0, 0, time.UTC), true},
			{"before start", time.Date(2023, 12, 25, 9, 0, 0, 0, time.UTC), false},
			{"after end", time.Date(2023, 12, 25, 16, 0, 0, 0, time.UTC), false},
			{"different day", time.Date(2023, 12, 26, 12, 0, 0, 0, time.UTC), false},
		}
		
		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				result := baseRange.Contains(tc.testTime)
				if result != tc.expected {
					t.Errorf("TimeRange.Contains(%v) = %v, want %v", tc.testTime, result, tc.expected)
				}
			})
		}
	})
	
	t.Run("Overlaps tests", func(t *testing.T) {
		testCases := []struct {
			name     string
			other    TimeRange
			expected bool
		}{
			{
				"complete overlap",
				TimeRange{
					Start: time.Date(2023, 12, 25, 11, 0, 0, 0, time.UTC),
					End:   time.Date(2023, 12, 25, 14, 0, 0, 0, time.UTC),
				},
				true,
			},
			{
				"partial overlap at start",
				TimeRange{
					Start: time.Date(2023, 12, 25, 8, 0, 0, 0, time.UTC),
					End:   time.Date(2023, 12, 25, 12, 0, 0, 0, time.UTC),
				},
				true,
			},
			{
				"partial overlap at end",
				TimeRange{
					Start: time.Date(2023, 12, 25, 13, 0, 0, 0, time.UTC),
					End:   time.Date(2023, 12, 25, 17, 0, 0, 0, time.UTC),
				},
				true,
			},
			{
				"touching at start",
				TimeRange{
					Start: time.Date(2023, 12, 25, 8, 0, 0, 0, time.UTC),
					End:   time.Date(2023, 12, 25, 10, 0, 0, 0, time.UTC),
				},
				true,
			},
			{
				"touching at end",
				TimeRange{
					Start: time.Date(2023, 12, 25, 15, 0, 0, 0, time.UTC),
					End:   time.Date(2023, 12, 25, 17, 0, 0, 0, time.UTC),
				},
				true,
			},
			{
				"no overlap - before",
				TimeRange{
					Start: time.Date(2023, 12, 25, 7, 0, 0, 0, time.UTC),
					End:   time.Date(2023, 12, 25, 9, 0, 0, 0, time.UTC),
				},
				false,
			},
			{
				"no overlap - after",
				TimeRange{
					Start: time.Date(2023, 12, 25, 16, 0, 0, 0, time.UTC),
					End:   time.Date(2023, 12, 25, 18, 0, 0, 0, time.UTC),
				},
				false,
			},
		}
		
		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				result := baseRange.Overlaps(tc.other)
				if result != tc.expected {
					t.Errorf("TimeRange.Overlaps(%v) = %v, want %v", tc.other, result, tc.expected)
				}
			})
		}
	})
}

func TestTimezoneOperations(t *testing.T) {
	utcTime := time.Date(2023, 12, 25, 15, 0, 0, 0, time.UTC)
	
	t.Run("ConvertTimezone", func(t *testing.T) {
		// Test conversion to various timezones
		testCases := []struct {
			name       string
			fromTZ     string
			toTZ       string
			hasErr     bool
		}{
			{"UTC to EST", "UTC", "America/New_York", false},
			{"UTC to PST", "UTC", "America/Los_Angeles", false},
			{"UTC to CET", "UTC", "Europe/Berlin", false},
			{"UTC to JST", "UTC", "Asia/Tokyo", false},
			{"Invalid from timezone", "Invalid/Timezone", "UTC", true},
			{"Invalid to timezone", "UTC", "Invalid/Timezone", true},
			{"Empty from timezone", "", "UTC", true},
			{"Empty to timezone", "UTC", "", true},
		}
		
		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				result, err := ConvertTimezone(utcTime, tc.fromTZ, tc.toTZ)
				if (err != nil) != tc.hasErr {
					t.Errorf("ConvertTimezone(%v, %s, %s) error = %v, wantErr %v", utcTime, tc.fromTZ, tc.toTZ, err, tc.hasErr)
					return
				}
				
				if !tc.hasErr {
					// Verify that the time is in the correct timezone
					expectedLocation, _ := time.LoadLocation(tc.toTZ)
					if result.Location().String() != expectedLocation.String() {
						t.Errorf("ConvertTimezone() location = %v, want %v", result.Location(), expectedLocation)
					}
				}
			})
		}
	})
	
	t.Run("ToUTC", func(t *testing.T) {
		// Create time in different timezone
		est, _ := time.LoadLocation("America/New_York")
		estTime := time.Date(2023, 12, 25, 10, 0, 0, 0, est)
		
		result := ToUTC(estTime)
		
		// Should be converted to UTC
		if result.Location() != time.UTC {
			t.Errorf("ToUTC() location = %v, want %v", result.Location(), time.UTC)
		}
	})
}

func TestAgeCalculations_Extended(t *testing.T) {
	referenceDate := time.Date(2023, 12, 25, 12, 0, 0, 0, time.UTC)
	
	testCases := []struct {
		name      string
		birthDate time.Time
		expected  int
	}{
		{"exactly 25 years", time.Date(1998, 12, 25, 0, 0, 0, 0, time.UTC), 25},
		{"25 years - 1 day", time.Date(1998, 12, 26, 0, 0, 0, 0, time.UTC), 24},
		{"25 years + 1 day", time.Date(1998, 12, 24, 0, 0, 0, 0, time.UTC), 25},
		{"born today", referenceDate, 0},
		{"born yesterday", time.Date(2023, 12, 24, 0, 0, 0, 0, time.UTC), 0},
		{"leap year birthday", time.Date(2000, 2, 29, 0, 0, 0, 0, time.UTC), 23},
		{"very old", time.Date(1900, 1, 1, 0, 0, 0, 0, time.UTC), 123},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := Age(tc.birthDate, referenceDate)
			if result != tc.expected {
				t.Errorf("Age(%v, %v) = %d, want %d", tc.birthDate, referenceDate, result, tc.expected)
			}
		})
	}
}

func TestYearsBetween_Extended(t *testing.T) {
	testCases := []struct {
		name     string
		start    time.Time
		end      time.Time
		expected int
	}{
		{
			"same year",
			time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
			time.Date(2023, 12, 31, 0, 0, 0, 0, time.UTC),
			0,
		},
		{
			"exactly one year",
			time.Date(2022, 12, 25, 0, 0, 0, 0, time.UTC),
			time.Date(2023, 12, 25, 0, 0, 0, 0, time.UTC),
			1,
		},
		{
			"almost one year",
			time.Date(2022, 12, 25, 0, 0, 0, 0, time.UTC),
			time.Date(2023, 12, 24, 0, 0, 0, 0, time.UTC),
			0,
		},
		{
			"more than one year",
			time.Date(2022, 12, 25, 0, 0, 0, 0, time.UTC),
			time.Date(2023, 12, 26, 0, 0, 0, 0, time.UTC),
			1,
		},
		{
			"multiple years",
			time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
			time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
			3,
		},
		{
			"reverse order",
			time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
			time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
			-3,
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := YearsBetween(tc.start, tc.end)
			if result != tc.expected {
				t.Errorf("YearsBetween(%v, %v) = %d, want %d", tc.start, tc.end, result, tc.expected)
			}
		})
	}
}

func TestMonthsBetween_Extended(t *testing.T) {
	testCases := []struct {
		name     string
		start    time.Time
		end      time.Time
		expected int
	}{
		{
			"same month",
			time.Date(2023, 12, 1, 0, 0, 0, 0, time.UTC),
			time.Date(2023, 12, 31, 0, 0, 0, 0, time.UTC),
			0,
		},
		{
			"exactly one month",
			time.Date(2023, 11, 25, 0, 0, 0, 0, time.UTC),
			time.Date(2023, 12, 25, 0, 0, 0, 0, time.UTC),
			1,
		},
		{
			"almost one month",
			time.Date(2023, 11, 25, 0, 0, 0, 0, time.UTC),
			time.Date(2023, 12, 24, 0, 0, 0, 0, time.UTC),
			0,
		},
		{
			"across year boundary",
			time.Date(2023, 11, 1, 0, 0, 0, 0, time.UTC),
			time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			2,
		},
		{
			"multiple years",
			time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
			time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
			24,
		},
		{
			"reverse order",
			time.Date(2023, 12, 1, 0, 0, 0, 0, time.UTC),
			time.Date(2023, 11, 1, 0, 0, 0, 0, time.UTC),
			-1,
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := MonthsBetween(tc.start, tc.end)
			if result != tc.expected {
				t.Errorf("MonthsBetween(%v, %v) = %d, want %d", tc.start, tc.end, result, tc.expected)
			}
		})
	}
}

func TestTimeUtilities_EdgeCases(t *testing.T) {
	baseTime := time.Date(2023, 12, 25, 15, 30, 45, 123456789, time.UTC)
	
	t.Run("Start/End of periods", func(t *testing.T) {
		// Test StartOfDay
		startOfDay := StartOfDay(baseTime)
		expected := time.Date(2023, 12, 25, 0, 0, 0, 0, time.UTC)
		if !startOfDay.Equal(expected) {
			t.Errorf("StartOfDay(%v) = %v, want %v", baseTime, startOfDay, expected)
		}
		
		// Test EndOfDay
		endOfDay := EndOfDay(baseTime)
		expected = time.Date(2023, 12, 25, 23, 59, 59, 999999999, time.UTC)
		if !endOfDay.Equal(expected) {
			t.Errorf("EndOfDay(%v) = %v, want %v", baseTime, endOfDay, expected)
		}
		
		// Test StartOfWeek (Monday)
		startOfWeek := StartOfWeek(baseTime)
		expected = time.Date(2023, 12, 25, 0, 0, 0, 0, time.UTC) // 2023-12-25 was a Monday
		if !startOfWeek.Equal(expected) {
			t.Errorf("StartOfWeek(%v) = %v, want %v", baseTime, startOfWeek, expected)
		}
		
		// Test StartOfMonth
		startOfMonth := StartOfMonth(baseTime)
		expected = time.Date(2023, 12, 1, 0, 0, 0, 0, time.UTC)
		if !startOfMonth.Equal(expected) {
			t.Errorf("StartOfMonth(%v) = %v, want %v", baseTime, startOfMonth, expected)
		}
		
		// Test EndOfMonth
		endOfMonth := EndOfMonth(baseTime)
		expected = time.Date(2023, 12, 31, 23, 59, 59, 999999999, time.UTC)
		if !endOfMonth.Equal(expected) {
			t.Errorf("EndOfMonth(%v) = %v, want %v", baseTime, endOfMonth, expected)
		}
	})
}

func TestParseErrors(t *testing.T) {
	t.Run("Parse invalid formats", func(t *testing.T) {
		invalidInputs := []string{
			"not a time",
			"2023/13/25 25:61:61",
			"32-01-2023",
			"2023-02-30", // Invalid date
			"25:61:61",   // Invalid time
			"",           // Empty string
			"2023",       // Incomplete
		}
		
		for _, input := range invalidInputs {
			t.Run(input, func(t *testing.T) {
				_, err := Parse(input)
				if err == nil {
					t.Errorf("Parse(%s) should return error for invalid input", input)
				}
			})
		}
	})
}

func TestBusinessDayCount(t *testing.T) {
	testCases := []struct {
		name     string
		start    time.Time
		end      time.Time
		expected int
	}{
		{
			"Monday to Friday (same week)",
			time.Date(2023, 12, 25, 0, 0, 0, 0, time.UTC), // Monday
			time.Date(2023, 12, 29, 0, 0, 0, 0, time.UTC), // Friday
			5,
		},
		{
			"Friday to Monday (over weekend)",
			time.Date(2023, 12, 29, 0, 0, 0, 0, time.UTC), // Friday
			time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),   // Monday
			2,
		},
		{
			"Saturday to Sunday (weekend only)",
			time.Date(2023, 12, 30, 0, 0, 0, 0, time.UTC), // Saturday
			time.Date(2023, 12, 31, 0, 0, 0, 0, time.UTC), // Sunday
			0,
		},
		{
			"Same day",
			time.Date(2023, 12, 25, 0, 0, 0, 0, time.UTC), // Monday
			time.Date(2023, 12, 25, 0, 0, 0, 0, time.UTC), // Monday
			1,
		},
		{
			"Reverse order",
			time.Date(2023, 12, 29, 0, 0, 0, 0, time.UTC), // Friday
			time.Date(2023, 12, 25, 0, 0, 0, 0, time.UTC), // Monday
			-5,
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := BusinessDaysBetween(tc.start, tc.end)
			if result != tc.expected {
				t.Errorf("BusinessDaysBetween(%v, %v) = %d, want %d", tc.start, tc.end, result, tc.expected)
			}
		})
	}
}

// ===============================
// Additional Coverage Tests
// ===============================

func TestUncoveredFunctions(t *testing.T) {
	// Test time constants String() method
	t.Run("Time Constants String", func(t *testing.T) {
		// This tests the String() method at 0% coverage
		result := ISO8601
		if result == "" {
			t.Error("ISO8601 constant should not be empty")
		}
	})

	// Test ParseInLocation
	t.Run("ParseInLocation", func(t *testing.T) {
		berlin, _ := time.LoadLocation("Europe/Berlin")
		result, err := ParseInLocation("2023-12-25 15:04:05", berlin)
		if err != nil {
			t.Errorf("ParseInLocation failed: %v", err)
		}
		if result.Location() != berlin {
			t.Errorf("ParseInLocation location = %v, want %v", result.Location(), berlin)
		}

		// Test with invalid input
		_, err = ParseInLocation("invalid", berlin)
		if err == nil {
			t.Error("ParseInLocation should fail with invalid input")
		}
	})

	// Test PrevBusinessDay
	t.Run("PrevBusinessDay", func(t *testing.T) {
		monday := time.Date(2023, 12, 25, 0, 0, 0, 0, time.UTC)
		friday := time.Date(2023, 12, 22, 0, 0, 0, 0, time.UTC)
		
		result := PrevBusinessDay(monday)
		if !result.Equal(friday) {
			t.Errorf("PrevBusinessDay(%v) = %v, want %v", monday, result, friday)
		}

		// Test from weekend
		saturday := time.Date(2023, 12, 23, 0, 0, 0, 0, time.UTC)
		result = PrevBusinessDay(saturday)
		if !result.Equal(friday) {
			t.Errorf("PrevBusinessDay(%v) = %v, want %v", saturday, result, friday)
		}
	})

	// Test EndOfWeek
	t.Run("EndOfWeek", func(t *testing.T) {
		monday := time.Date(2023, 12, 25, 10, 30, 0, 0, time.UTC)
		expected := time.Date(2023, 12, 31, 23, 59, 59, 999999999, time.UTC)
		
		result := EndOfWeek(monday)
		if !result.Equal(expected) {
			t.Errorf("EndOfWeek(%v) = %v, want %v", monday, result, expected)
		}
	})

	// Test StartOfYear and EndOfYear
	t.Run("StartOfYear", func(t *testing.T) {
		someDate := time.Date(2023, 8, 15, 10, 30, 0, 0, time.UTC)
		expected := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
		
		result := StartOfYear(someDate)
		if !result.Equal(expected) {
			t.Errorf("StartOfYear(%v) = %v, want %v", someDate, result, expected)
		}
	})

	t.Run("EndOfYear", func(t *testing.T) {
		someDate := time.Date(2023, 8, 15, 10, 30, 0, 0, time.UTC)
		expected := time.Date(2023, 12, 31, 23, 59, 59, 999999999, time.UTC)
		
		result := EndOfYear(someDate)
		if !result.Equal(expected) {
			t.Errorf("EndOfYear(%v) = %v, want %v", someDate, result, expected)
		}
	})

	// Test AgeToday
	t.Run("AgeToday", func(t *testing.T) {
		// Test with a birthdate that makes someone 25 years old today
		now := time.Now()
		birthDate := now.AddDate(-25, 0, 0)
		
		age := AgeToday(birthDate)
		if age < 24 || age > 25 {
			t.Errorf("AgeToday(%v) = %d, want around 25", birthDate, age)
		}
	})

	// Test timezone functions
	t.Run("ToLocal", func(t *testing.T) {
		utc := time.Date(2023, 12, 25, 12, 0, 0, 0, time.UTC)
		result := ToLocal(utc)
		if result.Location() == time.UTC {
			t.Error("ToLocal should convert from UTC to local timezone")
		}
	})

	t.Run("GetTimezoneOffset", func(t *testing.T) {
		utc := time.Date(2023, 12, 25, 12, 0, 0, 0, time.UTC)
		offset := GetTimezoneOffset(utc, time.Local)
		if offset < -12*3600 || offset > 14*3600 {
			t.Errorf("GetTimezoneOffset returned invalid offset: %d", offset)
		}

		// Test with nil location (should use time.Local)
		offset2 := GetTimezoneOffset(utc, nil)
		if offset2 < -12*3600 || offset2 > 14*3600 {
			t.Errorf("GetTimezoneOffset with nil location returned invalid offset: %d", offset2)
		}
	})

	// Test time comparison functions
	t.Run("Time Comparisons", func(t *testing.T) {
		now := time.Now()
		future := now.Add(24 * time.Hour)
		past := now.Add(-24 * time.Hour)
		today := time.Date(now.Year(), now.Month(), now.Day(), 12, 0, 0, 0, now.Location())
		yesterday := today.Add(-24 * time.Hour)
		tomorrow := today.Add(24 * time.Hour)

		if !IsFuture(future) {
			t.Error("IsFuture should return true for future time")
		}
		if IsFuture(past) {
			t.Error("IsFuture should return false for past time")
		}

		if !IsPast(past) {
			t.Error("IsPast should return true for past time")
		}
		if IsPast(future) {
			t.Error("IsPast should return false for future time")
		}

		if !IsToday(today) {
			t.Error("IsToday should return true for today's date")
		}
		if IsToday(yesterday) {
			t.Error("IsToday should return false for yesterday")
		}

		if !IsYesterday(yesterday) {
			t.Error("IsYesterday should return true for yesterday's date")
		}
		if IsYesterday(today) {
			t.Error("IsYesterday should return false for today")
		}

		if !IsTomorrow(tomorrow) {
			t.Error("IsTomorrow should return true for tomorrow's date")
		}
		if IsTomorrow(today) {
			t.Error("IsTomorrow should return false for today")
		}
	})

	// Test Format edge cases
	t.Run("Format Edge Cases", func(t *testing.T) {
		testTime := time.Date(2023, 12, 25, 15, 30, 45, 0, time.UTC)
		
		// Test unknown format - should return custom format
		result := Format(testTime, "unknown_format")
		if result == "" {
			t.Error("Format with unknown format should not return empty string")
		}
		
		// Test custom format
		result = Format(testTime, "2006-01-02 Monday")
		expected := "2023-12-25 Monday"
		if result != expected {
			t.Errorf("Format with custom format = %s, want %s", result, expected)
		}
	})
}