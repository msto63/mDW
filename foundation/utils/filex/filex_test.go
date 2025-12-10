// File: filex_test.go
// Title: File Utilities Tests
// Description: Comprehensive test suite for all filex utility functions including
//              unit tests, edge cases, and integration scenarios.
// Author: msto63 with Claude Sonnet 4.0
// Version: v0.1.0
// Created: 2025-01-25
// Modified: 2025-01-25
//
// Change History:
// - 2025-01-25 v0.1.0: Initial test implementation with comprehensive coverage

package filex

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// setupTestDir creates a temporary directory with test files
func setupTestDir(t *testing.T) string {
	tmpDir, err := os.MkdirTemp("", "filex_test_")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	
	// Create test files
	testFiles := map[string]string{
		"test.txt":         "Hello, World!\nThis is a test file.\n",
		"empty.txt":        "",
		"numbers.txt":      "1\n2\n3\n4\n5\n",
		"long.txt":         strings.Repeat("line\n", 100),
		"subdir/nested.txt": "Nested file content",
		"image.jpg":        "fake jpg content",
		"data.json":        `{"key": "value"}`,
	}
	
	for path, content := range testFiles {
		fullPath := filepath.Join(tmpDir, path)
		dir := filepath.Dir(fullPath)
		
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("Failed to create dir %s: %v", dir, err)
		}
		
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create test file %s: %v", fullPath, err)
		}
	}
	
	return tmpDir
}

// cleanupTestDir removes the test directory
func cleanupTestDir(t *testing.T, dir string) {
	if err := os.RemoveAll(dir); err != nil {
		t.Errorf("Failed to cleanup test dir %s: %v", dir, err)
	}
}

// ===============================
// File Existence and Basic Info Tests
// ===============================

func TestExists(t *testing.T) {
	tmpDir := setupTestDir(t)
	defer cleanupTestDir(t, tmpDir)
	
	testCases := []struct {
		name     string
		path     string
		expected bool
	}{
		{"existing file", filepath.Join(tmpDir, "test.txt"), true},
		{"existing directory", tmpDir, true},
		{"non-existing file", filepath.Join(tmpDir, "nonexistent.txt"), false},
		{"non-existing directory", filepath.Join(tmpDir, "nonexistent"), false},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := Exists(tc.path)
			if result != tc.expected {
				t.Errorf("Exists(%s) = %v, want %v", tc.path, result, tc.expected)
			}
		})
	}
}

func TestIsFile(t *testing.T) {
	tmpDir := setupTestDir(t)
	defer cleanupTestDir(t, tmpDir)
	
	testCases := []struct {
		name     string
		path     string
		expected bool
	}{
		{"regular file", filepath.Join(tmpDir, "test.txt"), true},
		{"directory", tmpDir, false},
		{"non-existing", filepath.Join(tmpDir, "nonexistent.txt"), false},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := IsFile(tc.path)
			if result != tc.expected {
				t.Errorf("IsFile(%s) = %v, want %v", tc.path, result, tc.expected)
			}
		})
	}
}

func TestIsDir(t *testing.T) {
	tmpDir := setupTestDir(t)
	defer cleanupTestDir(t, tmpDir)
	
	testCases := []struct {
		name     string
		path     string
		expected bool
	}{
		{"directory", tmpDir, true},
		{"regular file", filepath.Join(tmpDir, "test.txt"), false},
		{"non-existing", filepath.Join(tmpDir, "nonexistent"), false},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := IsDir(tc.path)
			if result != tc.expected {
				t.Errorf("IsDir(%s) = %v, want %v", tc.path, result, tc.expected)
			}
		})
	}
}

func TestGetFileInfo(t *testing.T) {
	tmpDir := setupTestDir(t)
	defer cleanupTestDir(t, tmpDir)
	
	testPath := filepath.Join(tmpDir, "test.txt")
	info, err := GetFileInfo(testPath)
	
	if err != nil {
		t.Fatalf("GetFileInfo() failed: %v", err)
	}
	
	if info.Name != "test.txt" {
		t.Errorf("Expected name 'test.txt', got '%s'", info.Name)
	}
	
	if info.IsDir {
		t.Error("Expected file to not be directory")
	}
	
	if info.Ext != ".txt" {
		t.Errorf("Expected extension '.txt', got '%s'", info.Ext)
	}
	
	if info.Size <= 0 {
		t.Errorf("Expected file size > 0, got %d", info.Size)
	}
}

// ===============================
// File Size Tests
// ===============================

func TestSize(t *testing.T) {
	tmpDir := setupTestDir(t)
	defer cleanupTestDir(t, tmpDir)
	
	testCases := []struct {
		name         string
		file         string
		expectedSize int64
	}{
		{"regular file", "test.txt", -1}, // Size will be checked dynamically
		{"empty file", "empty.txt", 0},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(tmpDir, tc.file)
			size, err := Size(path)
			
			if err != nil {
				t.Fatalf("Size() failed: %v", err)
			}
			
			if tc.expectedSize == -1 {
				// For regular file, just check it's greater than 0
				if size <= 0 {
					t.Errorf("Size() = %d, want > 0", size)
				}
			} else if size != tc.expectedSize {
				t.Errorf("Size() = %d, want %d", size, tc.expectedSize)
			}
		})
	}
}

func TestFormatSize(t *testing.T) {
	testCases := []struct {
		name     string
		bytes    int64
		expected string
	}{
		{"bytes", 512, "512 B"},
		{"kilobytes", 1536, "1.5 KB"},
		{"megabytes", 1536 * 1024, "1.5 MB"},
		{"gigabytes", 1536 * 1024 * 1024, "1.5 GB"},
		{"zero", 0, "0 B"},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := FormatSize(tc.bytes)
			if result != tc.expected {
				t.Errorf("FormatSize(%d) = %s, want %s", tc.bytes, result, tc.expected)
			}
		})
	}
}

// ===============================
// File Reading Tests
// ===============================

func TestReadFile(t *testing.T) {
	tmpDir := setupTestDir(t)
	defer cleanupTestDir(t, tmpDir)
	
	path := filepath.Join(tmpDir, "test.txt")
	content, err := ReadFile(path)
	
	if err != nil {
		t.Fatalf("ReadFile() failed: %v", err)
	}
	
	expected := "Hello, World!\nThis is a test file.\n"
	if string(content) != expected {
		t.Errorf("ReadFile() = %q, want %q", string(content), expected)
	}
}

func TestReadString(t *testing.T) {
	tmpDir := setupTestDir(t)
	defer cleanupTestDir(t, tmpDir)
	
	path := filepath.Join(tmpDir, "test.txt")
	content, err := ReadString(path)
	
	if err != nil {
		t.Fatalf("ReadString() failed: %v", err)
	}
	
	expected := "Hello, World!\nThis is a test file.\n"
	if content != expected {
		t.Errorf("ReadString() = %q, want %q", content, expected)
	}
}

func TestReadLines(t *testing.T) {
	tmpDir := setupTestDir(t)
	defer cleanupTestDir(t, tmpDir)
	
	path := filepath.Join(tmpDir, "numbers.txt")
	lines, err := ReadLines(path)
	
	if err != nil {
		t.Fatalf("ReadLines() failed: %v", err)
	}
	
	expected := []string{"1", "2", "3", "4", "5"}
	if len(lines) != len(expected) {
		t.Errorf("ReadLines() returned %d lines, want %d", len(lines), len(expected))
		return
	}
	
	for i, line := range lines {
		if line != expected[i] {
			t.Errorf("ReadLines()[%d] = %q, want %q", i, line, expected[i])
		}
	}
}

func TestReadFirstLines(t *testing.T) {
	tmpDir := setupTestDir(t)
	defer cleanupTestDir(t, tmpDir)
	
	path := filepath.Join(tmpDir, "numbers.txt")
	lines, err := ReadFirstLines(path, 3)
	
	if err != nil {
		t.Fatalf("ReadFirstLines() failed: %v", err)
	}
	
	expected := []string{"1", "2", "3"}
	if len(lines) != len(expected) {
		t.Errorf("ReadFirstLines() returned %d lines, want %d", len(lines), len(expected))
		return
	}
	
	for i, line := range lines {
		if line != expected[i] {
			t.Errorf("ReadFirstLines()[%d] = %q, want %q", i, line, expected[i])
		}
	}
}

func TestReadLastLines(t *testing.T) {
	tmpDir := setupTestDir(t)
	defer cleanupTestDir(t, tmpDir)
	
	path := filepath.Join(tmpDir, "numbers.txt")
	lines, err := ReadLastLines(path, 2)
	
	if err != nil {
		t.Fatalf("ReadLastLines() failed: %v", err)
	}
	
	expected := []string{"4", "5"}
	if len(lines) != len(expected) {
		t.Errorf("ReadLastLines() returned %d lines, want %d", len(lines), len(expected))
		return
	}
	
	for i, line := range lines {
		if line != expected[i] {
			t.Errorf("ReadLastLines()[%d] = %q, want %q", i, line, expected[i])
		}
	}
}

// ===============================
// File Writing Tests
// ===============================

func TestWriteFile(t *testing.T) {
	tmpDir := setupTestDir(t)
	defer cleanupTestDir(t, tmpDir)
	
	path := filepath.Join(tmpDir, "write_test.txt")
	content := []byte("Test content")
	
	err := WriteFile(path, content, 0644)
	if err != nil {
		t.Fatalf("WriteFile() failed: %v", err)
	}
	
	// Verify file was written correctly
	readContent, err := ReadFile(path)
	if err != nil {
		t.Fatalf("Failed to read written file: %v", err)
	}
	
	if string(readContent) != string(content) {
		t.Errorf("Written content = %q, want %q", string(readContent), string(content))
	}
}

func TestWriteString(t *testing.T) {
	tmpDir := setupTestDir(t)
	defer cleanupTestDir(t, tmpDir)
	
	path := filepath.Join(tmpDir, "write_string_test.txt")
	content := "Test string content"
	
	err := WriteString(path, content, 0644)
	if err != nil {
		t.Fatalf("WriteString() failed: %v", err)
	}
	
	// Verify file was written correctly
	readContent, err := ReadString(path)
	if err != nil {
		t.Fatalf("Failed to read written file: %v", err)
	}
	
	if readContent != content {
		t.Errorf("Written content = %q, want %q", readContent, content)
	}
}

func TestWriteLines(t *testing.T) {
	tmpDir := setupTestDir(t)
	defer cleanupTestDir(t, tmpDir)
	
	path := filepath.Join(tmpDir, "write_lines_test.txt")
	lines := []string{"line1", "line2", "line3"}
	
	err := WriteLines(path, lines, 0644)
	if err != nil {
		t.Fatalf("WriteLines() failed: %v", err)
	}
	
	// Verify file was written correctly
	readLines, err := ReadLines(path)
	if err != nil {
		t.Fatalf("Failed to read written file: %v", err)
	}
	
	if len(readLines) != len(lines) {
		t.Errorf("Read %d lines, want %d", len(readLines), len(lines))
		return
	}
	
	for i, line := range readLines {
		if line != lines[i] {
			t.Errorf("Line %d = %q, want %q", i, line, lines[i])
		}
	}
}

func TestAppendString(t *testing.T) {
	tmpDir := setupTestDir(t)
	defer cleanupTestDir(t, tmpDir)
	
	path := filepath.Join(tmpDir, "append_test.txt")
	
	// Write initial content
	initial := "Initial content\n"
	err := WriteString(path, initial, 0644)
	if err != nil {
		t.Fatalf("Failed to write initial content: %v", err)
	}
	
	// Append content
	append1 := "Appended content\n"
	err = AppendString(path, append1, 0644)
	if err != nil {
		t.Fatalf("AppendString() failed: %v", err)
	}
	
	// Verify content
	final, err := ReadString(path)
	if err != nil {
		t.Fatalf("Failed to read final content: %v", err)
	}
	
	expected := initial + append1
	if final != expected {
		t.Errorf("Final content = %q, want %q", final, expected)
	}
}

// ===============================
// File Copy and Move Tests
// ===============================

func TestCopy(t *testing.T) {
	tmpDir := setupTestDir(t)
	defer cleanupTestDir(t, tmpDir)
	
	src := filepath.Join(tmpDir, "test.txt")
	dst := filepath.Join(tmpDir, "test_copy.txt")
	
	err := Copy(src, dst)
	if err != nil {
		t.Fatalf("Copy() failed: %v", err)
	}
	
	// Verify destination exists
	if !Exists(dst) {
		t.Error("Destination file does not exist after copy")
	}
	
	// Verify content is identical
	equal, err := Equal(src, dst)
	if err != nil {
		t.Fatalf("Failed to compare files: %v", err)
	}
	
	if !equal {
		t.Error("Source and destination files are not equal")
	}
}

func TestMove(t *testing.T) {
	tmpDir := setupTestDir(t)
	defer cleanupTestDir(t, tmpDir)
	
	src := filepath.Join(tmpDir, "test.txt")
	dst := filepath.Join(tmpDir, "test_moved.txt")
	
	// Read original content
	originalContent, err := ReadString(src)
	if err != nil {
		t.Fatalf("Failed to read original content: %v", err)
	}
	
	err = Move(src, dst)
	if err != nil {
		t.Fatalf("Move() failed: %v", err)
	}
	
	// Verify source no longer exists
	if Exists(src) {
		t.Error("Source file still exists after move")
	}
	
	// Verify destination exists and has correct content
	if !Exists(dst) {
		t.Error("Destination file does not exist after move")
	}
	
	movedContent, err := ReadString(dst)
	if err != nil {
		t.Fatalf("Failed to read moved content: %v", err)
	}
	
	if movedContent != originalContent {
		t.Error("Moved file content differs from original")
	}
}

// ===============================
// Directory Operations Tests
// ===============================

func TestMkdirAll(t *testing.T) {
	tmpDir := setupTestDir(t)
	defer cleanupTestDir(t, tmpDir)
	
	newDir := filepath.Join(tmpDir, "new", "nested", "directory")
	
	err := MkdirAll(newDir, 0755)
	if err != nil {
		t.Fatalf("MkdirAll() failed: %v", err)
	}
	
	if !IsDir(newDir) {
		t.Error("Created directory does not exist or is not a directory")
	}
}

func TestListDir(t *testing.T) {
	tmpDir := setupTestDir(t)
	defer cleanupTestDir(t, tmpDir)
	
	entries, err := ListDir(tmpDir)
	if err != nil {
		t.Fatalf("ListDir() failed: %v", err)
	}
	
	// Should have at least the files we created
	if len(entries) < 6 { // test.txt, empty.txt, numbers.txt, long.txt, subdir, image.jpg, data.json
		t.Errorf("ListDir() returned %d entries, expected at least 6", len(entries))
	}
	
	// Check that we have both files and directories
	hasFiles := false
	hasDirs := false
	
	for _, entry := range entries {
		if entry.IsDir {
			hasDirs = true
		} else {
			hasFiles = true
		}
	}
	
	if !hasFiles {
		t.Error("ListDir() should return files")
	}
	
	if !hasDirs {
		t.Error("ListDir() should return directories")
	}
}

func TestListFiles(t *testing.T) {
	tmpDir := setupTestDir(t)
	defer cleanupTestDir(t, tmpDir)
	
	files, err := ListFiles(tmpDir)
	if err != nil {
		t.Fatalf("ListFiles() failed: %v", err)
	}
	
	// Should only return files, not directories
	for _, file := range files {
		if file.IsDir {
			t.Errorf("ListFiles() returned directory: %s", file.Name)
		}
	}
	
	// Should have at least some files
	if len(files) < 5 {
		t.Errorf("ListFiles() returned %d files, expected at least 5", len(files))
	}
}

func TestListDirs(t *testing.T) {
	tmpDir := setupTestDir(t)
	defer cleanupTestDir(t, tmpDir)
	
	dirs, err := ListDirs(tmpDir)
	if err != nil {
		t.Fatalf("ListDirs() failed: %v", err)
	}
	
	// Should only return directories, not files
	for _, dir := range dirs {
		if !dir.IsDir {
			t.Errorf("ListDirs() returned file: %s", dir.Name)
		}
	}
	
	// Should have at least the subdir we created
	if len(dirs) < 1 {
		t.Errorf("ListDirs() returned %d directories, expected at least 1", len(dirs))
	}
}

// ===============================
// File Search Tests
// ===============================

func TestFind(t *testing.T) {
	tmpDir := setupTestDir(t)
	defer cleanupTestDir(t, tmpDir)
	
	// Find all .txt files
	matches, err := Find(tmpDir, "*.txt")
	if err != nil {
		t.Fatalf("Find() failed: %v", err)
	}
	
	if len(matches) < 4 { // test.txt, empty.txt, numbers.txt, long.txt, nested.txt
		t.Errorf("Find() returned %d matches, expected at least 4", len(matches))
	}
	
	// Verify all matches end with .txt
	for _, match := range matches {
		if !strings.HasSuffix(match, ".txt") {
			t.Errorf("Find() returned non-.txt file: %s", match)
		}
	}
}

func TestFindFiles(t *testing.T) {
	tmpDir := setupTestDir(t)
	defer cleanupTestDir(t, tmpDir)
	
	// Find all .txt files (files only)
	matches, err := FindFiles(tmpDir, "*.txt")
	if err != nil {
		t.Fatalf("FindFiles() failed: %v", err)
	}
	
	// Verify all matches are files (not directories)
	for _, match := range matches {
		if !IsFile(match) {
			t.Errorf("FindFiles() returned non-file: %s", match)
		}
	}
}

// ===============================
// File Comparison and Hashing Tests
// ===============================

func TestEqual(t *testing.T) {
	tmpDir := setupTestDir(t)
	defer cleanupTestDir(t, tmpDir)
	
	file1 := filepath.Join(tmpDir, "test.txt")
	file2 := filepath.Join(tmpDir, "test_copy.txt")
	file3 := filepath.Join(tmpDir, "numbers.txt")
	
	// Copy file1 to file2
	err := Copy(file1, file2)
	if err != nil {
		t.Fatalf("Failed to copy file: %v", err)
	}
	
	// Test equal files
	equal, err := Equal(file1, file2)
	if err != nil {
		t.Fatalf("Equal() failed: %v", err)
	}
	
	if !equal {
		t.Error("Equal files should be reported as equal")
	}
	
	// Test different files
	equal, err = Equal(file1, file3)
	if err != nil {
		t.Fatalf("Equal() failed: %v", err)
	}
	
	if equal {
		t.Error("Different files should not be reported as equal")
	}
}

func TestMD5Hash(t *testing.T) {
	tmpDir := setupTestDir(t)
	defer cleanupTestDir(t, tmpDir)
	
	path := filepath.Join(tmpDir, "test.txt")
	hash1, err := MD5Hash(path)
	if err != nil {
		t.Fatalf("MD5Hash() failed: %v", err)
	}
	
	if len(hash1) != 32 { // MD5 hash is 32 hex characters
		t.Errorf("MD5Hash() returned hash of length %d, expected 32", len(hash1))
	}
	
	// Hash of same file should be identical
	hash2, err := MD5Hash(path)
	if err != nil {
		t.Fatalf("MD5Hash() failed on second call: %v", err)
	}
	
	if hash1 != hash2 {
		t.Error("MD5Hash() should return same hash for same file")
	}
}

func TestSHA256Hash(t *testing.T) {
	tmpDir := setupTestDir(t)
	defer cleanupTestDir(t, tmpDir)
	
	path := filepath.Join(tmpDir, "test.txt")
	hash, err := SHA256Hash(path)
	if err != nil {
		t.Fatalf("SHA256Hash() failed: %v", err)
	}
	
	if len(hash) != 64 { // SHA256 hash is 64 hex characters
		t.Errorf("SHA256Hash() returned hash of length %d, expected 64", len(hash))
	}
}

// ===============================
// File Type Detection Tests
// ===============================

func TestDetectMimeType(t *testing.T) {
	testCases := []struct {
		filename string
		expected string
	}{
		{"test.txt", "text/plain"},
		{"image.jpg", "image/jpeg"},
		{"data.json", "application/json"},
		{"page.html", "text/html"},
		{"style.css", "text/css"},
		{"unknown.xyz", "application/octet-stream"},
	}
	
	for _, tc := range testCases {
		t.Run(tc.filename, func(t *testing.T) {
			result := DetectMimeType(tc.filename)
			if result != tc.expected {
				t.Errorf("DetectMimeType(%s) = %s, want %s", tc.filename, result, tc.expected)
			}
		})
	}
}

func TestIsTextFile(t *testing.T) {
	testCases := []struct {
		filename string
		expected bool
	}{
		{"test.txt", true},
		{"data.json", true},
		{"page.html", true},
		{"script.js", true},
		{"source.go", true},
		{"image.jpg", false},
		{"video.mp4", false},
		{"archive.zip", false},
	}
	
	for _, tc := range testCases {
		t.Run(tc.filename, func(t *testing.T) {
			result := IsTextFile(tc.filename)
			if result != tc.expected {
				t.Errorf("IsTextFile(%s) = %v, want %v", tc.filename, result, tc.expected)
			}
		})
	}
}

func TestIsImageFile(t *testing.T) {
	testCases := []struct {
		filename string
		expected bool
	}{
		{"photo.jpg", true},
		{"image.png", true},
		{"graphic.gif", true},
		{"icon.svg", true},
		{"text.txt", false},
		{"video.mp4", false},
		{"document.pdf", false},
	}
	
	for _, tc := range testCases {
		t.Run(tc.filename, func(t *testing.T) {
			result := IsImageFile(tc.filename)
			if result != tc.expected {
				t.Errorf("IsImageFile(%s) = %v, want %v", tc.filename, result, tc.expected)
			}
		})
	}
}

// ===============================
// Path Manipulation Tests
// ===============================

func TestPathFunctions(t *testing.T) {
	testPath := filepath.Join("path", "to", "file.txt")
	
	t.Run("Dir", func(t *testing.T) {
		result := Dir(testPath)
		expected := filepath.Join("path", "to")
		if result != expected {
			t.Errorf("Dir(%s) = %s, want %s", testPath, result, expected)
		}
	})
	
	t.Run("Base", func(t *testing.T) {
		result := Base(testPath)
		expected := "file.txt"
		if result != expected {
			t.Errorf("Base(%s) = %s, want %s", testPath, result, expected)
		}
	})
	
	t.Run("Ext", func(t *testing.T) {
		result := Ext(testPath)
		expected := ".txt"
		if result != expected {
			t.Errorf("Ext(%s) = %s, want %s", testPath, result, expected)
		}
	})
	
	t.Run("Join", func(t *testing.T) {
		result := Join("path", "to", "file.txt")
		expected := filepath.Join("path", "to", "file.txt")
		if result != expected {
			t.Errorf("Join() = %s, want %s", result, expected)
		}
	})
}

// ===============================
// Utility Function Tests
// ===============================

func TestTouch(t *testing.T) {
	tmpDir := setupTestDir(t)
	defer cleanupTestDir(t, tmpDir)
	
	newFile := filepath.Join(tmpDir, "touched.txt")
	
	// Touch non-existing file (should create it)
	err := Touch(newFile)
	if err != nil {
		t.Fatalf("Touch() failed: %v", err)
	}
	
	if !Exists(newFile) {
		t.Error("Touch() should create non-existing file")
	}
	
	// Get initial mod time
	info1, err := os.Stat(newFile)
	if err != nil {
		t.Fatalf("Failed to stat touched file: %v", err)
	}
	
	// Wait a bit then touch again
	time.Sleep(10 * time.Millisecond)
	
	err = Touch(newFile)
	if err != nil {
		t.Fatalf("Touch() failed on existing file: %v", err)
	}
	
	// Check that mod time was updated
	info2, err := os.Stat(newFile)
	if err != nil {
		t.Fatalf("Failed to stat touched file: %v", err)
	}
	
	if !info2.ModTime().After(info1.ModTime()) {
		t.Error("Touch() should update modification time")
	}
}

func TestTempFile(t *testing.T) {
	content := []byte("temporary content")
	
	path, err := TempFile("test_*", content)
	if err != nil {
		t.Fatalf("TempFile() failed: %v", err)
	}
	defer os.Remove(path) // Clean up
	
	// Verify file exists
	if !Exists(path) {
		t.Error("TempFile() should create file")
	}
	
	// Verify content
	readContent, err := ReadFile(path)
	if err != nil {
		t.Fatalf("Failed to read temp file: %v", err)
	}
	
	if string(readContent) != string(content) {
		t.Error("TempFile() content differs from expected")
	}
}

func TestTempDir(t *testing.T) {
	path, err := TempDir("test_*")
	if err != nil {
		t.Fatalf("TempDir() failed: %v", err)
	}
	defer os.RemoveAll(path) // Clean up
	
	// Verify directory exists
	if !IsDir(path) {
		t.Error("TempDir() should create directory")
	}
}

func TestLineCount(t *testing.T) {
	tmpDir := setupTestDir(t)
	defer cleanupTestDir(t, tmpDir)
	
	path := filepath.Join(tmpDir, "numbers.txt")
	count, err := LineCount(path)
	if err != nil {
		t.Fatalf("LineCount() failed: %v", err)
	}
	
	expected := 5 // numbers.txt has 5 lines
	if count != expected {
		t.Errorf("LineCount() = %d, want %d", count, expected)
	}
}

func TestWordCount(t *testing.T) {
	tmpDir := setupTestDir(t)
	defer cleanupTestDir(t, tmpDir)
	
	path := filepath.Join(tmpDir, "test.txt")
	count, err := WordCount(path)
	if err != nil {
		t.Fatalf("WordCount() failed: %v", err)
	}
	
	// "Hello, World!\nThis is a test file.\n" = 7 words
	expected := 7
	if count != expected {
		t.Errorf("WordCount() = %d, want %d", count, expected)
	}
}

func TestIsEmpty(t *testing.T) {
	tmpDir := setupTestDir(t)
	defer cleanupTestDir(t, tmpDir)
	
	emptyFile := filepath.Join(tmpDir, "empty.txt")
	nonEmptyFile := filepath.Join(tmpDir, "test.txt")
	
	// Test empty file
	empty, err := IsEmpty(emptyFile)
	if err != nil {
		t.Fatalf("IsEmpty() failed for empty file: %v", err)
	}
	
	if !empty {
		t.Error("IsEmpty() should return true for empty file")
	}
	
	// Test non-empty file
	empty, err = IsEmpty(nonEmptyFile)
	if err != nil {
		t.Fatalf("IsEmpty() failed for non-empty file: %v", err)
	}
	
	if empty {
		t.Error("IsEmpty() should return false for non-empty file")
	}
}

func TestSafeRemove(t *testing.T) {
	tmpDir := setupTestDir(t)
	defer cleanupTestDir(t, tmpDir)
	
	existingFile := filepath.Join(tmpDir, "test.txt")
	nonExistingFile := filepath.Join(tmpDir, "nonexistent.txt")
	
	// Remove existing file
	err := SafeRemove(existingFile)
	if err != nil {
		t.Fatalf("SafeRemove() failed for existing file: %v", err)
	}
	
	if Exists(existingFile) {
		t.Error("SafeRemove() should remove existing file")
	}
	
	// Remove non-existing file (should not error)
	err = SafeRemove(nonExistingFile)
	if err != nil {
		t.Errorf("SafeRemove() should not error for non-existing file: %v", err)
	}
}

// ===============================
// Additional Test Coverage for Critical Functions
// ===============================

func TestIsSymlink(t *testing.T) {
	tmpDir := setupTestDir(t)
	defer cleanupTestDir(t, tmpDir)
	
	// Test with regular file
	regularFile := filepath.Join(tmpDir, "test.txt")
	if IsSymlink(regularFile) {
		t.Error("IsSymlink() should return false for regular file")
	}
	
	// Test with non-existing file
	nonExistingFile := filepath.Join(tmpDir, "nonexistent.txt")
	if IsSymlink(nonExistingFile) {
		t.Error("IsSymlink() should return false for non-existing file")
	}
}

func TestIsReadable(t *testing.T) {
	tmpDir := setupTestDir(t)
	defer cleanupTestDir(t, tmpDir)
	
	testFile := filepath.Join(tmpDir, "test.txt")
	
	// Test readable file
	if !IsReadable(testFile) {
		t.Error("IsReadable() should return true for readable file")
	}
	
	// Test non-existing file
	nonExistingFile := filepath.Join(tmpDir, "nonexistent.txt")
	if IsReadable(nonExistingFile) {
		t.Error("IsReadable() should return false for non-existing file")
	}
}

func TestIsWritable(t *testing.T) {
	tmpDir := setupTestDir(t)
	defer cleanupTestDir(t, tmpDir)
	
	testFile := filepath.Join(tmpDir, "test.txt")
	
	// Test writable file
	if !IsWritable(testFile) {
		t.Error("IsWritable() should return true for writable file")
	}
	
	// Test non-existing file
	nonExistingFile := filepath.Join(tmpDir, "nonexistent.txt")
	if IsWritable(nonExistingFile) {
		t.Error("IsWritable() should return false for non-existing file")
	}
}

func TestIsExecutable(t *testing.T) {
	tmpDir := setupTestDir(t)
	defer cleanupTestDir(t, tmpDir)
	
	testFile := filepath.Join(tmpDir, "test.txt")
	
	// Test non-executable file (regular text file)
	if IsExecutable(testFile) {
		t.Error("IsExecutable() should return false for regular text file")
	}
	
	// Test non-existing file
	nonExistingFile := filepath.Join(tmpDir, "nonexistent.txt")
	if IsExecutable(nonExistingFile) {
		t.Error("IsExecutable() should return false for non-existing file")
	}
}

func TestDirSize(t *testing.T) {
	tmpDir := setupTestDir(t)
	defer cleanupTestDir(t, tmpDir)
	
	// Test directory size calculation
	size, err := DirSize(tmpDir)
	if err != nil {
		t.Fatalf("DirSize() failed: %v", err)
	}
	
	if size <= 0 {
		t.Error("DirSize() should return positive size for directory with files")
	}
	
	// Test non-existing directory
	nonExistingDir := filepath.Join(tmpDir, "nonexistent")
	_, err = DirSize(nonExistingDir)
	if err == nil {
		t.Error("DirSize() should return error for non-existing directory")
	}
}

func TestAbsPath(t *testing.T) {
	tmpDir := setupTestDir(t)
	defer cleanupTestDir(t, tmpDir)
	
	testFile := filepath.Join(tmpDir, "test.txt")
	
	// Test absolute path
	absPath, err := AbsPath(testFile)
	if err != nil {
		t.Fatalf("AbsPath() failed: %v", err)
	}
	
	if !filepath.IsAbs(absPath) {
		t.Error("AbsPath() should return absolute path")
	}
	
	// Test relative path conversion
	relPath := "test.txt"
	absPath2, err := AbsPath(relPath)
	if err != nil {
		t.Fatalf("AbsPath() failed for relative path: %v", err)
	}
	
	if !filepath.IsAbs(absPath2) {
		t.Error("AbsPath() should convert relative path to absolute")
	}
}

func TestRelPath(t *testing.T) {
	tmpDir := setupTestDir(t)
	defer cleanupTestDir(t, tmpDir)
	
	baseDir := tmpDir
	targetFile := filepath.Join(tmpDir, "test.txt")
	
	// Test relative path calculation
	relPath, err := RelPath(baseDir, targetFile)
	if err != nil {
		t.Fatalf("RelPath() failed: %v", err)
	}
	
	if relPath == "" {
		t.Error("RelPath() should return non-empty relative path")
	}
	
	if filepath.IsAbs(relPath) {
		t.Error("RelPath() should return relative path, not absolute")
	}
}

func TestSplit(t *testing.T) {
	testPath := filepath.Join("path", "to", "file.txt")
	
	dir, file := Split(testPath)
	
	expectedDir := filepath.Join("path", "to") + string(filepath.Separator)
	expectedFile := "file.txt"
	
	if dir != expectedDir {
		t.Errorf("Split() dir = %s, want %s", dir, expectedDir)
	}
	
	if file != expectedFile {
		t.Errorf("Split() file = %s, want %s", file, expectedFile)
	}
}

func TestClean(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{"simple path", "a/b/c", filepath.Join("a", "b", "c")},
		{"path with dots", "a/./b/../c", filepath.Join("a", "c")},
		{"path with double slashes", "a//b/c", filepath.Join("a", "b", "c")},
		{"empty path", "", "."},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := Clean(tc.input)
			if result != tc.expected {
				t.Errorf("Clean(%s) = %s, want %s", tc.input, result, tc.expected)
			}
		})
	}
}

func TestSortFiles(t *testing.T) {
	tmpDir := setupTestDir(t)
	defer cleanupTestDir(t, tmpDir)
	
	files, err := ListFiles(tmpDir)
	if err != nil {
		t.Fatalf("ListFiles() failed: %v", err)
	}
	
	if len(files) == 0 {
		t.Skip("No files to sort")
	}
	
	// Test sorting by name
	originalOrder := make([]FileInfo, len(files))
	copy(originalOrder, files)
	
	SortFiles(files, SortByName, Ascending)
	
	// Verify sorting worked (files should be in alphabetical order)
	for i := 1; i < len(files); i++ {
		if files[i-1].Name > files[i].Name {
			t.Error("SortFiles() by name ascending failed")
			break
		}
	}
	
	// Test sorting by size
	SortFiles(files, SortBySize, Descending)
	
	// Test sorting by modification time
	SortFiles(files, SortByModTime, Ascending)
	
	// Test sorting by extension
	SortFiles(files, SortByExt, Ascending)
}

func TestBackup(t *testing.T) {
	tmpDir := setupTestDir(t)
	defer cleanupTestDir(t, tmpDir)
	
	originalFile := filepath.Join(tmpDir, "test.txt")
	
	// Test backup creation
	backupPath, err := Backup(originalFile)
	if err != nil {
		t.Fatalf("Backup() failed: %v", err)
	}
	
	// Verify backup file exists
	if !Exists(backupPath) {
		t.Error("Backup() should create backup file")
	}
	
	// Verify backup content matches original
	equal, err := Equal(originalFile, backupPath)
	if err != nil {
		t.Fatalf("Failed to compare original and backup: %v", err)
	}
	
	if !equal {
		t.Error("Backup content should match original file")
	}
	
	// Clean up backup
	defer SafeRemove(backupPath)
	
	// Test backup of non-existing file
	nonExistingFile := filepath.Join(tmpDir, "nonexistent.txt")
	_, err = Backup(nonExistingFile)
	if err == nil {
		t.Error("Backup() should return error for non-existing file")
	}
}

func TestRemoveAll(t *testing.T) {
	tmpDir := setupTestDir(t)
	
	// Create a subdirectory to test recursive removal
	subDir := filepath.Join(tmpDir, "subdir", "nested")
	err := MkdirAll(subDir, 0755)
	if err != nil {
		t.Fatalf("MkdirAll() failed: %v", err)
	}
	
	// Test removing entire directory tree
	err = RemoveAll(tmpDir)
	if err != nil {
		t.Fatalf("RemoveAll() failed: %v", err)
	}
	
	// Verify directory was removed
	if Exists(tmpDir) {
		t.Error("RemoveAll() should remove entire directory tree")
	}
}

func TestAppendFile(t *testing.T) {
	tmpDir := setupTestDir(t)
	defer cleanupTestDir(t, tmpDir)
	
	testFile := filepath.Join(tmpDir, "append_test.txt")
	
	// Write initial content
	initialContent := []byte("Initial content\n")
	err := WriteFile(testFile, initialContent, 0644)
	if err != nil {
		t.Fatalf("WriteFile() failed: %v", err)
	}
	
	// Append content
	appendContent := []byte("Appended content\n")
	err = AppendFile(testFile, appendContent, 0644)
	if err != nil {
		t.Fatalf("AppendFile() failed: %v", err)
	}
	
	// Verify final content
	finalContent, err := ReadFile(testFile)
	if err != nil {
		t.Fatalf("ReadFile() failed: %v", err)
	}
	
	expectedContent := string(initialContent) + string(appendContent)
	if string(finalContent) != expectedContent {
		t.Errorf("AppendFile() result = %q, want %q", string(finalContent), expectedContent)
	}
}

func TestAppendLine(t *testing.T) {
	tmpDir := setupTestDir(t)
	defer cleanupTestDir(t, tmpDir)
	
	testFile := filepath.Join(tmpDir, "append_line_test.txt")
	
	// Write initial content
	err := WriteString(testFile, "Line 1\n", 0644)
	if err != nil {
		t.Fatalf("WriteString() failed: %v", err)
	}
	
	// Append line
	err = AppendLine(testFile, "Line 2", 0644)
	if err != nil {
		t.Fatalf("AppendLine() failed: %v", err)
	}
	
	// Verify content
	content, err := ReadString(testFile)
	if err != nil {
		t.Fatalf("ReadString() failed: %v", err)
	}
	
	expected := "Line 1\nLine 2\n"
	if content != expected {
		t.Errorf("AppendLine() result = %q, want %q", content, expected)
	}
}

func TestCopyWithOptions(t *testing.T) {
	tmpDir := setupTestDir(t)
	defer cleanupTestDir(t, tmpDir)
	
	src := filepath.Join(tmpDir, "test.txt")
	dst := filepath.Join(tmpDir, "copy_with_options.txt")
	
	// Test copy with custom options
	options := FileCopyOptions{
		PreserveMode:    true,
		PreserveTime:    true,
		CreateDirs:      true,
		OverwriteTarget: false,
		BufferSize:      1024,
	}
	
	err := Copy(src, dst, options)
	if err != nil {
		t.Fatalf("Copy() with options failed: %v", err)
	}
	
	// Verify file was copied
	if !Exists(dst) {
		t.Error("Copy() should create destination file")
	}
	
	// Verify content is identical
	equal, err := Equal(src, dst)
	if err != nil {
		t.Fatalf("Failed to compare files: %v", err)
	}
	
	if !equal {
		t.Error("Copied file should have identical content")
	}
}

func TestFindDirs(t *testing.T) {
	tmpDir := setupTestDir(t)
	defer cleanupTestDir(t, tmpDir)
	
	// Find directories
	dirs, err := FindDirs(tmpDir, "*")
	if err != nil {
		t.Fatalf("FindDirs() failed: %v", err)
	}
	
	// Should find at least the subdir we created in setupTestDir
	if len(dirs) == 0 {
		t.Error("FindDirs() should find at least one directory")
	}
	
	// Verify all results are directories
	for _, dir := range dirs {
		if !IsDir(dir) {
			t.Errorf("FindDirs() returned non-directory: %s", dir)
		}
	}
}

func TestWalk(t *testing.T) {
	tmpDir := setupTestDir(t)
	defer cleanupTestDir(t, tmpDir)
	
	var visitedPaths []string
	
	err := Walk(tmpDir, func(path string, info FileInfo, err error) error {
		if err != nil {
			return err
		}
		visitedPaths = append(visitedPaths, path)
		return nil
	})
	
	if err != nil {
		t.Fatalf("Walk() failed: %v", err)
	}
	
	// Should visit at least the root directory and some files
	if len(visitedPaths) == 0 {
		t.Error("Walk() should visit at least some paths")
	}
	
	// First path should be the root directory
	if len(visitedPaths) > 0 && visitedPaths[0] != tmpDir {
		t.Errorf("Walk() first path should be root directory, got %s", visitedPaths[0])
	}
}

func TestDefaultCopyOptions(t *testing.T) {
	opts := DefaultCopyOptions()
	
	// Verify default values
	if !opts.PreserveMode {
		t.Error("DefaultCopyOptions() should preserve mode by default")
	}
	
	if !opts.PreserveTime {
		t.Error("DefaultCopyOptions() should preserve time by default")
	}
	
	if !opts.CreateDirs {
		t.Error("DefaultCopyOptions() should create directories by default")
	}
	
	if opts.OverwriteTarget {
		t.Error("DefaultCopyOptions() should not overwrite target by default")
	}
	
	if opts.BufferSize != 32*1024 {
		t.Errorf("DefaultCopyOptions() buffer size = %d, want %d", opts.BufferSize, 32*1024)
	}
}

func TestEdgeCases(t *testing.T) {
	tmpDir := setupTestDir(t)
	defer cleanupTestDir(t, tmpDir)
	
	t.Run("empty file operations", func(t *testing.T) {
		emptyFile := filepath.Join(tmpDir, "empty.txt")
		
		// Test reading empty file
		content, err := ReadString(emptyFile)
		if err != nil {
			t.Fatalf("ReadString() failed for empty file: %v", err)
		}
		
		if content != "" {
			t.Error("ReadString() should return empty string for empty file")
		}
		
		// Test line count of empty file
		count, err := LineCount(emptyFile)
		if err != nil {
			t.Fatalf("LineCount() failed for empty file: %v", err)
		}
		
		if count != 0 {
			t.Errorf("LineCount() for empty file = %d, want 0", count)
		}
		
		// Test word count of empty file
		wordCount, err := WordCount(emptyFile)
		if err != nil {
			t.Fatalf("WordCount() failed for empty file: %v", err)
		}
		
		if wordCount != 0 {
			t.Errorf("WordCount() for empty file = %d, want 0", wordCount)
		}
	})
	
	t.Run("large file operations", func(t *testing.T) {
		largeFile := filepath.Join(tmpDir, "large.txt")
		
		// Create large content
		largeContent := strings.Repeat("This is a line of text.\n", 1000)
		err := WriteString(largeFile, largeContent, 0644)
		if err != nil {
			t.Fatalf("WriteString() failed for large file: %v", err)
		}
		
		// Test reading large file
		content, err := ReadString(largeFile)
		if err != nil {
			t.Fatalf("ReadString() failed for large file: %v", err)
		}
		
		if content != largeContent {
			t.Error("ReadString() should return correct content for large file")
		}
		
		// Test reading first lines of large file
		firstLines, err := ReadFirstLines(largeFile, 10)
		if err != nil {
			t.Fatalf("ReadFirstLines() failed for large file: %v", err)
		}
		
		if len(firstLines) != 10 {
			t.Errorf("ReadFirstLines() returned %d lines, want 10", len(firstLines))
		}
		
		// Test reading last lines of large file
		lastLines, err := ReadLastLines(largeFile, 5)
		if err != nil {
			t.Fatalf("ReadLastLines() failed for large file: %v", err)
		}
		
		if len(lastLines) != 5 {
			t.Errorf("ReadLastLines() returned %d lines, want 5", len(lastLines))
		}
	})
}