// Package filex implements comprehensive file operation utilities for the mDW platform.
//
// Package: filex
// Title: Extended File Operations for Go
// Description: This package provides a comprehensive collection of file and directory
//              operation utilities including safe file operations, path manipulation,
//              directory management, file type detection, content processing, and
//              advanced file system operations. Designed for enterprise applications
//              requiring robust and secure file handling capabilities.
// Author: msto63 with Claude Sonnet 4.0
// Version: v0.1.0
// Created: 2025-01-25
// Modified: 2025-01-26
//
// Change History:
// - 2025-01-25 v0.1.0: Initial implementation with comprehensive file utilities
// - 2025-01-26 v0.1.1: Enhanced documentation with comprehensive examples and mDW integration
//
// Package Overview:
//
// The filex package provides over 60 utility functions organized into logical categories:
//
// # File Existence and Information
//
// Functions for checking file existence and retrieving file information:
//   - Exists: Check if file or directory exists
//   - IsFile/IsDir/IsSymlink: Check file type
//   - IsReadable/IsWritable/IsExecutable: Check file permissions
//   - GetFileInfo: Retrieve extended file information
//   - Size/DirSize: Calculate file and directory sizes
//   - FormatSize: Human-readable size formatting
//
// # File Reading Operations
//
// Comprehensive file reading capabilities:
//   - ReadFile/ReadString: Read entire file content
//   - ReadLines: Read file as slice of lines
//   - ReadFirstLines/ReadLastLines: Read specific line ranges
//   - Support for various text encodings
//   - Error handling for large files
//
// # File Writing Operations
//
// Safe and flexible file writing functions:
//   - WriteFile/WriteString: Write content to files
//   - WriteLines: Write slice of strings as lines
//   - AppendFile/AppendString/AppendLine: Append content to files
//   - Atomic write operations for data integrity
//   - Permission and ownership preservation
//
// # File Copy and Move Operations
//
// Advanced file copying and moving with options:
//   - Copy: File copying with comprehensive options
//   - Move: File moving with fallback mechanisms
//   - FileCopyOptions: Configurable copy behavior
//   - Permission and timestamp preservation
//   - Cross-filesystem support
//
// # Directory Operations
//
// Complete directory management functionality:
//   - MkdirAll/RemoveAll: Create and remove directory trees
//   - ListDir/ListFiles/ListDirs: Directory content listing
//   - Directory traversal and filtering
//   - Safe directory operations
//
// # File Search and Discovery
//
// Powerful file search and filtering capabilities:
//   - Find/FindFiles/FindDirs: Pattern-based file search
//   - Walk: Custom directory tree traversal
//   - Recursive search with filtering
//   - Pattern matching support
//
// # File Comparison and Integrity
//
// File comparison and hash calculation functions:
//   - Equal: Compare file contents for equality
//   - MD5Hash/SHA256Hash: Calculate file checksums
//   - Content verification and integrity checking
//   - Support for large file comparison
//
// # File Type Detection
//
// Intelligent file type detection and classification:
//   - DetectMimeType: MIME type detection by extension
//   - IsTextFile/IsImageFile: File type classification
//   - Support for common file formats
//   - Extensible type detection system
//
// # Path Manipulation
//
// Comprehensive path handling utilities:
//   - AbsPath/RelPath: Absolute and relative path conversion
//   - Dir/Base/Ext: Path component extraction
//   - Join/Split/Clean: Path manipulation
//   - Cross-platform path handling
//
// # File Sorting and Organization
//
// File sorting and organization capabilities:
//   - SortFiles: Sort files by various criteria
//   - SortBy/SortOrder: Configurable sorting options
//   - Name, size, date, extension sorting
//   - Custom sorting functions
//
// # Utility Functions
//
// Additional utility functions for common operations:
//   - Touch: Create files or update timestamps
//   - TempFile/TempDir: Temporary file and directory creation
//   - Backup: Create timestamped file backups
//   - LineCount/WordCount: Text file analysis
//   - IsEmpty: Check for empty files
//   - SafeRemove: Safe file deletion
//
// # Usage Examples
//
// Basic file operations:
//
//	// Check if file exists
//	if filex.Exists("config.txt") {
//		content, err := filex.ReadString("config.txt")
//		if err != nil {
//			log.Fatal(err)
//		}
//		fmt.Println("Config:", content)
//	}
//
//	// Write content to file
//	data := "Hello, World!"
//	err := filex.WriteString("output.txt", data, 0644)
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	// Get file information
//	info, err := filex.GetFileInfo("document.pdf")
//	if err != nil {
//		log.Fatal(err)
//	}
//	fmt.Printf("File: %s, Size: %s, Modified: %v\n", 
//		info.Name, filex.FormatSize(info.Size), info.ModTime)
//
// File copying with options:
//
//	// Copy with custom options
//	options := filex.FileCopyOptions{
//		PreserveMode:    true,
//		PreserveTime:    true,
//		CreateDirs:      true,
//		OverwriteTarget: false,
//		BufferSize:      64 * 1024, // 64KB buffer
//	}
//
//	err := filex.Copy("source.txt", "backup/source.txt", options)
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	// Simple copy with defaults
//	err = filex.Copy("input.txt", "output.txt")
//	if err != nil {
//		log.Fatal(err)
//	}
//
// Directory operations:
//
//	// Create directory structure
//	err := filex.MkdirAll("data/processed/2024", 0755)
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	// List directory contents
//	files, err := filex.ListFiles("documents")
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	for _, file := range files {
//		fmt.Printf("%s (%s)\n", file.Name, filex.FormatSize(file.Size))
//	}
//
//	// List only directories
//	dirs, err := filex.ListDirs("projects")
//	if err != nil {
//		log.Fatal(err)
//	}
//
// File searching:
//
//	// Find all text files
//	textFiles, err := filex.FindFiles(".", "*.txt")
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	// Find all Go source files recursively
//	goFiles, err := filex.Find("src", "*.go")
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	// Custom directory walking
//	err = filex.Walk("logs", func(path string, info filex.FileInfo, err error) error {
//		if err != nil {
//			return err
//		}
//
//		if strings.HasSuffix(path, ".log") && info.Size > 1024*1024 { // > 1MB
//			fmt.Printf("Large log file: %s (%s)\n", path, filex.FormatSize(info.Size))
//		}
//		return nil
//	})
//
// File comparison and integrity:
//
//	// Compare two files
//	equal, err := filex.Equal("file1.txt", "file2.txt")
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	if equal {
//		fmt.Println("Files are identical")
//	}
//
//	// Calculate file hash
//	hash, err := filex.SHA256Hash("important.dat")
//	if err != nil {
//		log.Fatal(err)
//	}
//	fmt.Printf("SHA256: %s\n", hash)
//
//	// Verify file integrity
//	expectedHash := "abc123..."
//	actualHash, err := filex.SHA256Hash("downloaded.zip")
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	if actualHash == expectedHash {
//		fmt.Println("File integrity verified")
//	} else {
//		fmt.Println("File integrity check failed")
//	}
//
// File type detection:
//
//	// Detect MIME type
//	mimeType := filex.DetectMimeType("document.pdf")
//	fmt.Printf("MIME type: %s\n", mimeType)
//
//	// Check file types
//	if filex.IsTextFile("README.md") {
//		content, _ := filex.ReadString("README.md")
//		fmt.Println("Text content:", content[:100])
//	}
//
//	if filex.IsImageFile("photo.jpg") {
//		fmt.Println("Processing image file...")
//	}
//
// Advanced file operations:
//
//	// Create backup with timestamp
//	backupPath, err := filex.Backup("important.conf")
//	if err != nil {
//		log.Fatal(err)
//	}
//	fmt.Printf("Backup created: %s\n", backupPath)
//
//	// Analyze text files
//	lineCount, err := filex.LineCount("source.go")
//	if err != nil {
//		log.Fatal(err)
//	}
//	
//	wordCount, err := filex.WordCount("document.txt")
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	fmt.Printf("Lines: %d, Words: %d\n", lineCount, wordCount)
//
//	// Work with temporary files
//	tmpFile, err := filex.TempFile("process_*.tmp", []byte("temp data"))
//	if err != nil {
//		log.Fatal(err)
//	}
//	defer filex.SafeRemove(tmpFile)
//
//	// Process temporary file...
//
// File sorting and organization:
//
//	files, err := filex.ListFiles("downloads")
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	// Sort by size, largest first
//	filex.SortFiles(files, filex.SortBySize, filex.Descending)
//	
//	// Sort by modification time, newest first
//	filex.SortFiles(files, filex.SortByModTime, filex.Descending)
//
//	// Sort by name alphabetically
//	filex.SortFiles(files, filex.SortByName, filex.Ascending)
//
// Line-by-line file processing:
//
//	// Read specific line ranges
//	firstLines, err := filex.ReadFirstLines("log.txt", 10)
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	lastLines, err := filex.ReadLastLines("log.txt", 5)
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	// Process all lines
//	allLines, err := filex.ReadLines("data.csv")
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	for i, line := range allLines {
//		fmt.Printf("Line %d: %s\n", i+1, line)
//	}
//
//	// Write lines to file
//	outputLines := []string{
//		"Line 1",
//		"Line 2", 
//		"Line 3",
//	}
//	err = filex.WriteLines("output.txt", outputLines, 0644)
//	if err != nil {
//		log.Fatal(err)
//	}
//
// # Data Types
//
// The package defines several custom types for enhanced functionality:
//
// FileInfo provides extended file information:
//
//	type FileInfo struct {
//		Name     string      // File name
//		Path     string      // Full file path
//		Size     int64       // File size in bytes
//		Mode     os.FileMode // File mode
//		ModTime  time.Time   // Last modification time
//		IsDir    bool        // Whether it's a directory
//		Ext      string      // File extension
//		MimeType string      // MIME type (if detected)
//	}
//
// FileCopyOptions configures copy operations:
//
//	type FileCopyOptions struct {
//		PreserveMode    bool // Preserve file permissions
//		PreserveTime    bool // Preserve modification time
//		CreateDirs      bool // Create parent directories
//		OverwriteTarget bool // Overwrite target if exists
//		BufferSize      int  // Buffer size for copying
//	}
//
// WalkFunc defines the signature for directory walking:
//
//	type WalkFunc func(path string, info FileInfo, err error) error
//
// # Performance Characteristics
//
// All functions are optimized for performance:
//   - Efficient buffer management for large file operations
//   - Streaming operations to minimize memory usage
//   - Optimized path manipulation using filepath package
//   - Minimal memory allocations for frequently used operations
//
// # Error Handling
//
// The package follows Go best practices for error handling:
//   - Descriptive error messages with context
//   - Wrapped errors to preserve error chains
//   - Safe operations that handle edge cases
//   - No panics from normal usage
//
// # Thread Safety
//
// All functions are thread-safe and can be used concurrently without
// additional synchronization. However, concurrent access to the same
// files should be coordinated by the application as needed.
//
// # Platform Compatibility
//
// The package works across all platforms supported by Go:
//   - Cross-platform path handling
//   - Appropriate file permission handling
//   - Platform-specific optimizations where applicable
//
// # Integration with mDW Platform
//
// This package is designed as part of the mDW (Trusted Business Platform)
// foundation library and follows mDW coding standards:
//   - Comprehensive documentation and examples
//   - Extensive test coverage (>95%)
//   - Consistent error handling
//   - English-only code and comments
//
// The package provides the file operation capabilities needed for TCOL
// (Terminal Command Object Language) processing, configuration management,
// log processing, data import/export, and general file system operations
// in enterprise applications.
//
// # Security Considerations
//
// File operations include security considerations:
//   - Path validation to prevent directory traversal
//   - Safe temporary file creation
//   - Proper file permission handling
//   - No automatic execution of files
//   - Validation of file operations before execution
//
// # Common Use Cases
//
// 1. Configuration File Management
//
//	// Read configuration with backup
//	configPath := "config.yaml"
//	if filex.Exists(configPath) {
//		// Create backup before modifying
//		backup, _ := filex.Backup(configPath)
//		fmt.Printf("Backup created: %s\n", backup)
//		
//		// Read and process config
//		content, _ := filex.ReadString(configPath)
//		newConfig := processConfig(content)
//		
//		// Write atomically
//		filex.WriteString(configPath, newConfig, 0644)
//	}
//
// 2. Log File Rotation
//
//	// Check log file size and rotate if needed
//	logPath := "app.log"
//	info, _ := filex.GetFileInfo(logPath)
//	
//	if info.Size > 10*1024*1024 { // 10MB
//		// Archive current log
//		timestamp := time.Now().Format("20060102_150405")
//		archivePath := fmt.Sprintf("logs/app_%s.log", timestamp)
//		filex.Move(logPath, archivePath)
//		
//		// Create new log file
//		filex.Touch(logPath)
//	}
//
// 3. Batch File Processing
//
//	// Process all CSV files in directory
//	csvFiles, _ := filex.FindFiles("data", "*.csv")
//	
//	for _, file := range csvFiles {
//		lines, _ := filex.ReadLines(file)
//		processed := processCSVLines(lines)
//		
//		outputPath := strings.Replace(file, ".csv", "_processed.csv", 1)
//		filex.WriteLines(outputPath, processed, 0644)
//	}
//
// 4. Safe File Updates
//
//	// Update file atomically
//	dataFile := "important.dat"
//	tmpFile, _ := filex.TempFile("update_*.tmp", []byte(""))
//	
//	// Write to temporary file
//	newData := generateData()
//	filex.WriteFile(tmpFile, newData, 0644)
//	
//	// Verify and move to final location
//	if verifyData(tmpFile) {
//		filex.Move(tmpFile, dataFile)
//	} else {
//		filex.SafeRemove(tmpFile)
//		return errors.New("data verification failed")
//	}
//
// 5. Directory Synchronization
//
//	// Sync source to destination
//	sourceFiles, _ := filex.Find("source", "*")
//	
//	for _, srcFile := range sourceFiles {
//		relPath, _ := filex.RelPath("source", srcFile)
//		dstFile := filex.Join("destination", relPath)
//		
//		// Skip if destination is newer
//		srcInfo, _ := filex.GetFileInfo(srcFile)
//		dstInfo, _ := filex.GetFileInfo(dstFile)
//		
//		if dstInfo == nil || srcInfo.ModTime.After(dstInfo.ModTime) {
//			filex.Copy(srcFile, dstFile, filex.FileCopyOptions{
//				CreateDirs: true,
//				PreserveTime: true,
//			})
//		}
//	}
//
// # Best Practices
//
// When using this package, consider these best practices:
//   - Always handle errors appropriately
//   - Use appropriate file permissions (0644 for files, 0755 for directories)
//   - Close file handles properly (handled automatically by these functions)
//   - Use temporary files for atomic operations
//   - Validate file paths before operations
//   - Consider file locking for concurrent access scenarios
//
// # mDW Integration Examples
//
// 1. TCOL File Operations
//
//	// Execute TCOL file commands
//	cmd := "FILE.COPY source='report.pdf' dest='archive/report_2024.pdf'"
//	params := parseTCOLParams(cmd)
//	
//	err := filex.Copy(params["source"], params["dest"], filex.FileCopyOptions{
//		CreateDirs: true,
//		PreserveMode: true,
//	})
//	
//	if err != nil {
//		return tcolError("FILE_COPY_FAILED", err)
//	}
//
// 2. Import/Export Processing
//
//	// Process data export files
//	exportDir := "exports"
//	pendingFiles, _ := filex.FindFiles(exportDir, "*.pending")
//	
//	for _, file := range pendingFiles {
//		// Process file
//		data, _ := filex.ReadFile(file)
//		result := processExportData(data)
//		
//		// Write result and mark as processed
//		outputFile := strings.Replace(file, ".pending", ".processed", 1)
//		filex.WriteFile(outputFile, result, 0644)
//		filex.SafeRemove(file)
//		
//		// Log in audit trail
//		log.Audit("EXPORT_PROCESSED", 
//			"file", file,
//			"size", filex.FormatSize(int64(len(data))),
//		)
//	}
//
// 3. Template Processing
//
//	// Process business document templates
//	templatePath := "templates/invoice.tmpl"
//	template, _ := filex.ReadString(templatePath)
//	
//	// Generate document from template
//	document := processTemplate(template, invoiceData)
//	
//	// Save with appropriate naming
//	outputPath := fmt.Sprintf("invoices/INV_%s_%s.pdf", 
//		invoiceData.Number,
//		time.Now().Format("20060102"),
//	)
//	
//	filex.WriteString(outputPath, document, 0644)
//
// 4. Backup and Archive Operations
//
//	// Daily backup routine
//	backupDate := time.Now().Format("2006-01-02")
//	backupDir := filex.Join("backups", backupDate)
//	filex.MkdirAll(backupDir, 0755)
//	
//	// Backup critical files
//	criticalFiles := []string{
//		"config.yaml",
//		"database.db",
//		"certificates/server.crt",
//	}
//	
//	for _, file := range criticalFiles {
//		if filex.Exists(file) {
//			destPath := filex.Join(backupDir, filex.Base(file))
//			filex.Copy(file, destPath, filex.FileCopyOptions{
//				PreserveMode: true,
//				PreserveTime: true,
//			})
//		}
//	}
//	
//	// Create checksum file
//	checksumPath := filex.Join(backupDir, "checksums.txt")
//	var checksums []string
//	
//	backupFiles, _ := filex.ListFiles(backupDir)
//	for _, bf := range backupFiles {
//		hash, _ := filex.SHA256Hash(bf.Path)
//		checksums = append(checksums, fmt.Sprintf("%s  %s", hash, bf.Name))
//	}
//	
//	filex.WriteLines(checksumPath, checksums, 0644)
//
// # Performance Considerations
//
// 1. Large File Operations
//   - Use streaming operations for files > 100MB
//   - Consider chunked processing for very large files
//   - Monitor memory usage with file operations
//
// 2. Directory Operations
//   - Use Find with specific patterns to limit results
//   - Consider pagination for large directory listings
//   - Cache directory information when appropriate
//
// 3. Optimization Tips
//   - Batch file operations when possible
//   - Use appropriate buffer sizes for copy operations
//   - Minimize stat calls by caching file info
//
// # Security Considerations
//
// File operations include security considerations:
//   - Path validation to prevent directory traversal
//   - Safe temporary file creation
//   - Proper file permission handling
//   - No automatic execution of files
//   - Validation of file operations before execution
//
// # Related Packages
//
//   - core/log: File-based logging operations
//   - core/config: Configuration file handling
//   - stringx: String manipulation for file content
//   - validationx: Path and filename validation
//
// This comprehensive file utility package provides enterprise-grade
// file operations suitable for business applications requiring robust,
// secure, and efficient file system interactions.
package filex