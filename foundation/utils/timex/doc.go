// Package timex implements comprehensive time utility functions for the mDW platform.
//
// Package: timex
// Title: Extended Time Utilities for Go
// Description: This package provides a comprehensive collection of utility functions
//              for working with time in Go applications, including parsing, formatting,
//              business day calculations, timezone handling, duration operations, and
//              time series generation. All functions are designed for real-world
//              business applications with focus on performance and ease of use.
// Author: msto63 with Claude Sonnet 4.0
// Version: v0.1.0
// Created: 2025-01-25
// Modified: 2025-01-26
//
// Change History:
// - 2025-01-25 v0.1.0: Initial implementation with comprehensive time operations
// - 2025-01-26 v0.1.1: Enhanced documentation with comprehensive examples and mDW integration
//
// Package Overview:
//
// The timex package provides over 50 utility functions organized into logical categories:
//
// # Time Parsing Functions
//
// Advanced parsing capabilities that handle multiple common formats automatically:
//   - Parse: Intelligent parsing of time strings using common formats
//   - ParseInLocation: Parse time strings in specific timezones
//   - ParseDate: Parse date-only strings with various formats
//   - ParseDuration: Parse duration strings including business-friendly formats
//
// Supported formats include ISO8601, RFC3339, business formats, display formats,
// compact formats, and many others. The parser automatically detects the format
// and handles timezone information appropriately.
//
// # Time Formatting Functions
//
// Comprehensive formatting options for different use cases:
//   - Format: Format times using predefined format names or custom layouts
//   - FormatDuration: Human-readable duration formatting
//   - FormatDurationCompact: Compact duration formatting (e.g., "1d 2h 30m")
//
// Predefined formats include:
//   - Business formats: "2023-12-25 15:30:45", "2023-12-25"
//   - Display formats: "December 25, 2023 at 3:30 PM"
//   - ISO8601 formats: "2023-12-25T15:30:45Z"
//   - Compact formats: "20231225153045"
//   - Log formats: "2023-12-25 15:30:45.000"
//
// # Business Day Calculations
//
// Sophisticated business day handling with configurable rules:
//   - IsBusinessDay: Check if a date is a business day
//   - NextBusinessDay/PrevBusinessDay: Navigate to adjacent business days
//   - AddBusinessDays: Add/subtract business days
//   - BusinessDaysBetween: Count business days between dates
//   - GenerateBusinessDays: Generate sequences of business days
//
// Business day configuration supports:
//   - Custom weekend days (not just Saturday/Sunday)
//   - Fixed holidays by date
//   - Custom holiday checker functions
//   - Multiple business day calendars
//
// # Time Manipulation Functions
//
// Functions for time boundary calculations and adjustments:
//   - StartOfDay/EndOfDay: Get day boundaries
//   - StartOfWeek/EndOfWeek: Get week boundaries (Monday-Sunday)
//   - StartOfMonth/EndOfMonth: Get month boundaries
//   - StartOfYear/EndOfYear: Get year boundaries
//   - RoundToNearest/TruncateToNearest: Round/truncate to time boundaries
//
// # Age and Date Difference Calculations
//
// Accurate date arithmetic for various use cases:
//   - Age: Calculate age in years with proper handling of leap years
//   - AgeToday: Calculate current age from birth date
//   - YearsBetween/MonthsBetween/DaysBetween: Calculate differences
//
// All functions handle edge cases like leap years, month boundaries,
// and timezone changes correctly.
//
// # Timezone Handling
//
// Comprehensive timezone support for global applications:
//   - ConvertTimezone: Convert between arbitrary timezones
//   - ToUTC/ToLocal: Convert to common timezone references
//   - ParseInLocation: Parse times in specific timezones
//   - GetTimezoneOffset: Get timezone offset information
//
// # Time Validation Functions
//
// Useful validation and classification functions:
//   - IsZero: Check for zero time values
//   - IsFuture/IsPast: Check temporal relationship to now
//   - IsToday/IsYesterday/IsTomorrow: Check specific day relationships
//   - IsWeekend/IsWeekday: Check day type
//
// # Time Series and Range Operations
//
// Functions for generating and working with time sequences:
//   - GenerateTimeRange: Generate time sequences with intervals
//   - GenerateBusinessDays: Generate business day sequences
//   - TimeRange: Work with time ranges (contains, overlaps, duration)
//
// # Unix Timestamp Utilities
//
// Conversion functions for Unix timestamps:
//   - Unix/UnixMilli: Convert Unix timestamps to time.Time
//   - ToUnix/ToUnixMilli: Convert time.Time to Unix timestamps
//
// # Common Time Constants
//
// The package provides predefined format constants for common use cases:
//
//	const (
//		BusinessDate     = "2006-01-02"
//		BusinessDateTime = "2006-01-02 15:04:05"
//		DisplayDate      = "January 2, 2006"
//		ISO8601          = "2006-01-02T15:04:05Z07:00"
//		LogTimestamp     = "2006-01-02 15:04:05.000"
//		// ... and many more
//	)
//
// # Usage Examples
//
// Basic parsing and formatting:
//
//	// Intelligent parsing
//	t, err := timex.Parse("2023-12-25 15:30:45")
//	t, err := timex.Parse("December 25, 2023")
//	t, err := timex.Parse("12/25/2023")
//
//	// Business-friendly duration parsing
//	d, err := timex.ParseDuration("2 hours 30 minutes")
//	d, err := timex.ParseDuration("1 day")
//	d, err := timex.ParseDuration("2 weeks")
//
//	// Duration formatting
//	readable := timex.FormatDuration(duration)        // "2 hours and 30 minutes"
//	compact := timex.FormatDurationCompact(duration)  // "2h 30m"
//
//	// Predefined formatting
//	formatted := timex.Format(time.Now(), "business")     // "2023-12-25 15:30:45"
//	formatted := timex.Format(time.Now(), "display")     // "December 25, 2023 at 3:30 PM"
//	formatted := timex.Format(time.Now(), "iso8601")     // "2023-12-25T15:30:45Z"
//
// Business day calculations:
//
//	// Check if today is a business day
//	if timex.IsBusinessDay(time.Now()) {
//		// Process business logic
//	}
//
//	// Add 5 business days to a date
//	deadline := timex.AddBusinessDays(startDate, 5)
//
//	// Count business days between dates
//	workingDays := timex.BusinessDaysBetween(startDate, endDate)
//
//	// Custom business day configuration
//	config := &timex.BusinessDayConfig{
//		WeekendDays: []timex.Weekday{timex.Friday, timex.Saturday},
//		Holidays:    []time.Time{christmas, newYear},
//	}
//	isWorkDay := timex.IsBusinessDay(someDate, config)
//
// Time manipulation and boundaries:
//
//	// Get day boundaries
//	startOfDay := timex.StartOfDay(time.Now())
//	endOfDay := timex.EndOfDay(time.Now())
//
//	// Get month boundaries
//	startOfMonth := timex.StartOfMonth(time.Now())
//	endOfMonth := timex.EndOfMonth(time.Now())
//
//	// Round to nearest hour
//	rounded := timex.RoundToNearest(time.Now(), time.Hour)
//
// Age calculations:
//
//	birthDate := timex.ParseDate("1990-06-15")
//	age := timex.AgeToday(birthDate)
//	ageAt := timex.Age(birthDate, someReferenceDate)
//
// Timezone handling:
//
//	// Convert between timezones
//	est, err := timex.ConvertTimezone(utcTime, "UTC", "America/New_York")
//	tokyo, err := timex.ConvertTimezone(utcTime, "UTC", "Asia/Tokyo")
//
//	// Parse time in specific timezone
//	local, err := timex.ParseInLocation("2023-12-25 15:30:45", tokyoLocation)
//
// Time series generation:
//
//	// Generate hourly timestamps for a day
//	start := timex.StartOfDay(time.Now())
//	end := timex.EndOfDay(time.Now())
//	hourlyTimes := timex.GenerateTimeRange(start, end, time.Hour)
//
//	// Generate business days for a month
//	monthStart := timex.StartOfMonth(time.Now())
//	monthEnd := timex.EndOfMonth(time.Now())
//	businessDays := timex.GenerateBusinessDays(monthStart, monthEnd)
//
// Time range operations:
//
//	workingHours := timex.TimeRange{
//		Start: timex.Parse("2023-12-25 09:00:00"),
//		End:   timex.Parse("2023-12-25 17:00:00"),
//	}
//
//	if workingHours.Contains(time.Now()) {
//		// During working hours
//	}
//
//	duration := workingHours.Duration() // 8 hours
//
// # Performance Characteristics
//
// All functions are optimized for performance:
//   - Parsing functions try common formats first for speed
//   - Business day calculations use efficient algorithms
//   - Memory allocations are minimized where possible
//   - Functions handle edge cases without performance penalties
//
// # Error Handling
//
// The package follows Go best practices for error handling:
//   - Parse functions return descriptive errors for invalid input
//   - Timezone functions return errors for invalid timezone names
//   - All functions handle nil and zero values gracefully
//   - No panics are generated from normal usage
//
// # Thread Safety
//
// All functions are thread-safe and can be used concurrently without
// additional synchronization. The package does not maintain any global
// state that could cause race conditions.
//
// # Integration with mDW Platform
//
// This package is designed as part of the mDW (Trusted Business Platform)
// foundation library and follows mDW coding standards:
//   - Comprehensive documentation and examples
//   - Extensive test coverage (>95%)
//   - Performance benchmarks
//   - Consistent error handling
//   - English-only code and comments
//
// The package provides the time handling capabilities needed for TCOL
// (Terminal Command Object Language) processing, business logic operations,
// and general enterprise application requirements.
//
// # Constants Reference
//
// Common time format constants provided by the package:
//
//	ISO8601         = "2006-01-02T15:04:05Z07:00"
//	BusinessDate    = "2006-01-02"
//	BusinessDateTime = "2006-01-02 15:04:05"
//	DisplayDate     = "January 2, 2006"
//	DisplayDateTime = "January 2, 2006 at 3:04 PM"
//	LogTimestamp    = "2006-01-02 15:04:05.000"
//	CompactDate     = "20060102"
//	ShortDate       = "01/02/2006"
//
// # Common Use Cases
//
// 1. Date Range Reporting
//
//	// Generate report for last month's business days
//	lastMonth := timex.StartOfMonth(time.Now().AddDate(0, -1, 0))
//	monthEnd := timex.EndOfMonth(lastMonth)
//	
//	businessDays := timex.GenerateBusinessDays(lastMonth, monthEnd)
//	for _, day := range businessDays {
//		report := generateDailyReport(day)
//		processReport(report)
//	}
//
// 2. SLA and Deadline Calculations
//
//	// Calculate SLA deadline (5 business days)
//	ticketCreated := time.Now()
//	slaDeadline := timex.AddBusinessDays(ticketCreated, 5)
//	
//	// Check if SLA is breached
//	if time.Now().After(slaDeadline) {
//		escalateTicket()
//	}
//	
//	// Days remaining
//	daysLeft := timex.BusinessDaysBetween(time.Now(), slaDeadline)
//
// 3. Time-Based Access Control
//
//	// Check if within business hours
//	workingHours := timex.TimeRange{
//		Start: timex.Parse("09:00:00"),
//		End:   timex.Parse("17:00:00"),
//	}
//	
//	if !workingHours.Contains(time.Now()) {
//		return errors.New("access allowed only during business hours")
//	}
//
// 4. Multi-Timezone Scheduling
//
//	// Schedule meeting across timezones
//	nyTime, _ := timex.ConvertTimezone(meetingTime, "UTC", "America/New_York")
//	londonTime, _ := timex.ConvertTimezone(meetingTime, "UTC", "Europe/London")
//	tokyoTime, _ := timex.ConvertTimezone(meetingTime, "UTC", "Asia/Tokyo")
//	
//	fmt.Printf("Meeting times:\n")
//	fmt.Printf("New York: %s\n", timex.Format(nyTime, "display"))
//	fmt.Printf("London: %s\n", timex.Format(londonTime, "display"))
//	fmt.Printf("Tokyo: %s\n", timex.Format(tokyoTime, "display"))
//
// 5. Batch Job Scheduling
//
//	// Generate execution times for hourly batch job
//	today := timex.StartOfDay(time.Now())
//	tomorrow := timex.StartOfDay(time.Now().AddDate(0, 0, 1))
//	
//	executionTimes := timex.GenerateTimeRange(today, tomorrow, time.Hour)
//	for _, execTime := range executionTimes {
//		scheduleJob(execTime)
//	}
//
// # Best Practices
//
// 1. Always use UTC for storage and internal processing
// 2. Convert to local time only for display purposes
// 3. Use business day functions for SLA and deadline calculations
// 4. Handle timezone conversions explicitly when dealing with global users
// 5. Use predefined format constants for consistency
// 6. Always check errors from parsing functions
//
// # mDW Integration Examples
//
// 1. TCOL Timestamp Commands
//
//	// Parse TCOL timestamp parameters
//	cmd := "REPORT.GENERATE start='2023-12-01' end='2023-12-31'"
//	startTime, _ := timex.Parse(extractParam(cmd, "start"))
//	endTime, _ := timex.Parse(extractParam(cmd, "end"))
//	
//	// Validate date range
//	if endTime.Before(startTime) {
//		return errors.New("end date must be after start date")
//	}
//
// 2. Audit Log Formatting
//
//	// Format timestamps for audit logs
//	entry := AuditLogEntry{
//		Timestamp: time.Now(),
//		User:      currentUser,
//		Action:    "CUSTOMER.UPDATE",
//	}
//	
//	logLine := fmt.Sprintf("[%s] %s: %s",
//		timex.Format(entry.Timestamp, "log"),
//		entry.User,
//		entry.Action,
//	)
//
// 3. Business Process Timing
//
//	// Track business process duration
//	processStart := time.Now()
//	
//	// ... execute business process ...
//	
//	duration := time.Since(processStart)
//	if duration > 5*time.Minute {
//		log.Warn("Process exceeded expected duration",
//			"duration", timex.FormatDuration(duration),
//			"expected", "5 minutes",
//		)
//	}
//
// 4. Report Generation Scheduling
//
//	// Schedule monthly reports for first business day
//	nextMonth := timex.StartOfMonth(time.Now().AddDate(0, 1, 0))
//	firstBusinessDay := timex.NextBusinessDay(nextMonth.AddDate(0, 0, -1))
//	
//	scheduleReport(firstBusinessDay)
//
// # Performance Considerations
//
// 1. Parsing Performance
//   - Common formats are checked first for optimization
//   - Cache parsed times when processing large datasets
//   - Use specific parse functions when format is known
//
// 2. Business Day Calculations
//   - Pre-compute holiday lists for better performance
//   - Cache business day calculations for repeated queries
//   - Use efficient algorithms for large date ranges
//
// 3. Timezone Operations
//   - Load timezone data once and reuse
//   - Minimize timezone conversions in hot paths
//   - Use UTC internally to avoid conversions
//
// # Error Handling
//
// The package follows Go best practices for error handling:
//   - Parse functions return descriptive errors for invalid input
//   - Timezone functions return errors for invalid timezone names
//   - All functions handle nil and zero values gracefully
//   - No panics are generated from normal usage
//
// # Thread Safety
//
// All functions are thread-safe and can be used concurrently without
// additional synchronization. The package does not maintain any global
// state that could cause race conditions.
//
// # Related Packages
//
//   - core/log: Timestamp formatting for logs
//   - core/error: Time-related error handling
//   - mathx: Duration calculations and comparisons
//   - validationx: Time validation rules
//
// # Integration with mDW Platform
//
// This package is designed as part of the mDW (Trusted Business Platform)
// foundation library and follows mDW coding standards:
//   - Comprehensive documentation and examples
//   - Extensive test coverage (>95%)
//   - Performance benchmarks
//   - Consistent error handling
//   - English-only code and comments
//
// The package provides the time handling capabilities needed for TCOL
// (Terminal Command Object Language) processing, business logic operations,
// and general enterprise application requirements.
//
// # Constants Reference
//
// Common time format constants provided by the package:
//
//	ISO8601         = "2006-01-02T15:04:05Z07:00"
//	BusinessDate    = "2006-01-02"
//	BusinessDateTime = "2006-01-02 15:04:05"
//	DisplayDate     = "January 2, 2006"
//	DisplayDateTime = "January 2, 2006 at 3:04 PM"
//	LogTimestamp    = "2006-01-02 15:04:05.000"
//	CompactDate     = "20060102"
//	ShortDate       = "01/02/2006"
//
// # Business Day Configuration
//
// The BusinessDayConfig struct allows customization of business day rules:
//
//	type BusinessDayConfig struct {
//		WeekendDays []Weekday           // Custom weekend days
//		Holidays    []time.Time         // Fixed holidays
//		IsHoliday   func(time.Time) bool // Custom holiday checker
//	}
//
// This flexibility supports different business calendars and international
// requirements where weekends and holidays vary by region.
package timex