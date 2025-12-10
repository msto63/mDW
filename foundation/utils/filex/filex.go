// File: filex.go
// Title: Core File Utilities
// Description: Implements comprehensive file operation utilities including
//              safe file operations, path manipulation, directory management,
//              file type detection, and content processing for the mDW platform.
// Author: msto63 with Claude Sonnet 4.0
// Version: v0.1.0
// Created: 2025-01-25
// Modified: 2025-01-25
//
// Change History:
// - 2025-01-25 v0.1.0: Initial implementation with comprehensive file utilities

package filex

import (
	"bufio"
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// FileInfo represents extended file information
type FileInfo struct {
	Name    string      // File name
	Path    string      // Full file path
	Size    int64       // File size in bytes
	Mode    os.FileMode // File mode
	ModTime time.Time   // Last modification time
	IsDir   bool        // Whether it's a directory
	Ext     string      // File extension
	MimeType string     // MIME type (if detected)
}

// DirEntry represents a directory entry with extended information
type DirEntry struct {
	FileInfo
	Children []DirEntry // Child entries (for directories)
}

// Buffer pools for efficient memory management in file operations
var (
	// smallBufferPool for small file operations (8KB)
	smallBufferPool = sync.Pool{
		New: func() interface{} {
			return make([]byte, 8*1024) // 8KB
		},
	}
	
	// largeBufferPool for copy operations (32KB)
	largeBufferPool = sync.Pool{
		New: func() interface{} {
			return make([]byte, 32*1024) // 32KB
		},
	}
	
	// hugeBufferPool for large file operations (64KB)
	hugeBufferPool = sync.Pool{
		New: func() interface{} {
			return make([]byte, 64*1024) // 64KB
		},
	}
)

// getPooledBuffer returns a buffer from the appropriate pool based on size requirements
func getPooledBuffer(size int) ([]byte, func()) {
	switch {
	case size <= 8*1024:
		buf := smallBufferPool.Get().([]byte)
		return buf[:size], func() { smallBufferPool.Put(buf) }
	case size <= 32*1024:
		buf := largeBufferPool.Get().([]byte)
		return buf[:size], func() { largeBufferPool.Put(buf) }
	default:
		buf := hugeBufferPool.Get().([]byte)
		return buf[:min(size, 64*1024)], func() { hugeBufferPool.Put(buf) }
	}
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// FileCopyOptions represents options for file copy operations
type FileCopyOptions struct {
	PreserveMode    bool // Preserve file permissions
	PreserveTime    bool // Preserve modification time
	CreateDirs      bool // Create parent directories if they don't exist
	OverwriteTarget bool // Overwrite target if it exists
	BufferSize      int  // Buffer size for copying (0 = default)
}

// DefaultCopyOptions returns default options for file copying
func DefaultCopyOptions() FileCopyOptions {
	return FileCopyOptions{
		PreserveMode:    true,
		PreserveTime:    true,
		CreateDirs:      true,
		OverwriteTarget: false,
		BufferSize:      32 * 1024, // 32KB default buffer
	}
}

// WalkFunc represents the type of function called for each file or directory
// visited by Walk. The path argument contains the argument to Walk as a prefix.
type WalkFunc func(path string, info FileInfo, err error) error

// ===============================
// File Existence and Basic Info
// ===============================

// Exists checks if a file or directory exists
func Exists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

// IsFile checks if the path exists and is a regular file
func IsFile(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.Mode().IsRegular()
}

// IsDir checks if the path exists and is a directory
func IsDir(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

// IsSymlink checks if the path is a symbolic link
func IsSymlink(path string) bool {
	info, err := os.Lstat(path)
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeSymlink != 0
}

// IsReadable checks if the file is readable
func IsReadable(path string) bool {
	file, err := os.Open(path)
	if err != nil {
		return false
	}
	defer file.Close()
	return true
}

// IsWritable checks if the file is writable
func IsWritable(path string) bool {
	// Try opening for write access
	file, err := os.OpenFile(path, os.O_WRONLY, 0)
	if err != nil {
		return false
	}
	defer file.Close()
	return true
}

// IsExecutable checks if the file is executable
func IsExecutable(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.Mode()&0111 != 0
}

// GetFileInfo returns extended file information
func GetFileInfo(path string) (FileInfo, error) {
	info, err := os.Stat(path)
	if err != nil {
		return FileInfo{}, fmt.Errorf("failed to get file info for %s: %w", path, err)
	}
	
	absPath, _ := filepath.Abs(path)
	ext := filepath.Ext(path)
	
	fileInfo := FileInfo{
		Name:    info.Name(),
		Path:    absPath,
		Size:    info.Size(),
		Mode:    info.Mode(),
		ModTime: info.ModTime(),
		IsDir:   info.IsDir(),
		Ext:     ext,
		MimeType: DetectMimeType(path),
	}
	
	return fileInfo, nil
}

// ===============================
// File Size and Space
// ===============================

// Size returns the size of a file in bytes
func Size(path string) (int64, error) {
	info, err := os.Stat(path)
	if err != nil {
		return 0, fmt.Errorf("failed to get size of %s: %w", path, err)
	}
	return info.Size(), nil
}

// DirSize calculates the total size of a directory and its contents
func DirSize(path string) (int64, error) {
	var totalSize int64
	
	err := filepath.Walk(path, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			totalSize += info.Size()
		}
		return nil
	})
	
	if err != nil {
		return 0, fmt.Errorf("failed to calculate directory size for %s: %w", path, err)
	}
	
	return totalSize, nil
}

// FormatSize formats a size in bytes to a human-readable string
func FormatSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	
	units := []string{"KB", "MB", "GB", "TB", "PB"}
	return fmt.Sprintf("%.1f %s", float64(bytes)/float64(div), units[exp])
}

// ===============================
// File Reading Operations
// ===============================

// ReadFile reads the entire file and returns its contents
func ReadFile(path string) ([]byte, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", path, err)
	}
	return content, nil
}

// ReadString reads the entire file and returns its contents as a string
func ReadString(path string) (string, error) {
	content, err := ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

// ReadLines reads the file and returns its contents as a slice of lines
func ReadLines(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open file %s: %w", path, err)
	}
	defer file.Close()
	
	var lines []string
	scanner := bufio.NewScanner(file)
	
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading lines from %s: %w", path, err)
	}
	
	return lines, nil
}

// ReadFirstLines reads the first n lines of a file
func ReadFirstLines(path string, n int) ([]string, error) {
	if n <= 0 {
		return []string{}, nil
	}
	
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open file %s: %w", path, err)
	}
	defer file.Close()
	
	var lines []string
	scanner := bufio.NewScanner(file)
	
	for i := 0; i < n && scanner.Scan(); i++ {
		lines = append(lines, scanner.Text())
	}
	
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading first %d lines from %s: %w", n, path, err)
	}
	
	return lines, nil
}

// ReadLastLines reads the last n lines of a file
func ReadLastLines(path string, n int) ([]string, error) {
	if n <= 0 {
		return []string{}, nil
	}
	
	// For simplicity, read all lines and return the last n
	// For very large files, a more efficient implementation would be needed
	allLines, err := ReadLines(path)
	if err != nil {
		return nil, err
	}
	
	if len(allLines) <= n {
		return allLines, nil
	}
	
	return allLines[len(allLines)-n:], nil
}

// ===============================
// File Writing Operations
// ===============================

// WriteFile writes data to a file, creating it if necessary
func WriteFile(path string, data []byte, perm os.FileMode) error {
	err := os.WriteFile(path, data, perm)
	if err != nil {
		return fmt.Errorf("failed to write file %s: %w", path, err)
	}
	return nil
}

// WriteString writes a string to a file
func WriteString(path, content string, perm os.FileMode) error {
	return WriteFile(path, []byte(content), perm)
}

// WriteLines writes a slice of strings as lines to a file
func WriteLines(path string, lines []string, perm os.FileMode) error {
	content := strings.Join(lines, "\n")
	if len(lines) > 0 {
		content += "\n" // Add final newline
	}
	return WriteString(path, content, perm)
}

// AppendFile appends data to a file, creating it if necessary
func AppendFile(path string, data []byte, perm os.FileMode) error {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, perm)
	if err != nil {
		return fmt.Errorf("failed to open file for append %s: %w", path, err)
	}
	defer file.Close()
	
	_, err = file.Write(data)
	if err != nil {
		return fmt.Errorf("failed to append to file %s: %w", path, err)
	}
	
	return nil
}

// AppendString appends a string to a file
func AppendString(path, content string, perm os.FileMode) error {
	return AppendFile(path, []byte(content), perm)
}

// AppendLine appends a line to a file (adds newline automatically)
func AppendLine(path, line string, perm os.FileMode) error {
	return AppendString(path, line+"\n", perm)
}

// ===============================
// File Copy and Move Operations
// ===============================

// Copy copies a file from source to destination with options
func Copy(src, dst string, options ...FileCopyOptions) error {
	opts := DefaultCopyOptions()
	if len(options) > 0 {
		opts = options[0]
	}
	
	// Check if source exists
	srcInfo, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("source file does not exist: %s", src)
	}
	
	// Check if destination exists and handle overwrite
	if Exists(dst) && !opts.OverwriteTarget {
		return fmt.Errorf("destination file exists and overwrite is disabled: %s", dst)
	}
	
	// Create parent directories if needed
	if opts.CreateDirs {
		if err := MkdirAll(filepath.Dir(dst), 0755); err != nil {
			return fmt.Errorf("failed to create parent directories: %w", err)
		}
	}
	
	// Open source file
	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file %s: %w", src, err)
	}
	defer srcFile.Close()
	
	// Create destination file
	dstFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("failed to create destination file %s: %w", dst, err)
	}
	defer dstFile.Close()
	
	// Copy content
	bufferSize := opts.BufferSize
	if bufferSize <= 0 {
		bufferSize = 32 * 1024 // 32KB default
	}
	
	// Use pooled buffer for efficient memory management
	buffer, returnBuffer := getPooledBuffer(bufferSize)
	defer returnBuffer()
	
	_, err = io.CopyBuffer(dstFile, srcFile, buffer)
	if err != nil {
		return fmt.Errorf("failed to copy file content: %w", err)
	}
	
	// Preserve file mode
	if opts.PreserveMode {
		if err := os.Chmod(dst, srcInfo.Mode()); err != nil {
			return fmt.Errorf("failed to preserve file mode: %w", err)
		}
	}
	
	// Preserve modification time
	if opts.PreserveTime {
		if err := os.Chtimes(dst, srcInfo.ModTime(), srcInfo.ModTime()); err != nil {
			return fmt.Errorf("failed to preserve file time: %w", err)
		}
	}
	
	return nil
}

// Move moves (renames) a file from source to destination
func Move(src, dst string) error {
	// Try simple rename first (works if on same filesystem)
	err := os.Rename(src, dst)
	if err == nil {
		return nil
	}
	
	// If rename fails, copy and delete
	if err := Copy(src, dst); err != nil {
		return fmt.Errorf("failed to copy during move: %w", err)
	}
	
	if err := os.Remove(src); err != nil {
		return fmt.Errorf("failed to remove source after copy: %w", err)
	}
	
	return nil
}

// ===============================
// Directory Operations
// ===============================

// MkdirAll creates a directory and all necessary parent directories
func MkdirAll(path string, perm os.FileMode) error {
	err := os.MkdirAll(path, perm)
	if err != nil {
		return fmt.Errorf("failed to create directory %s: %w", path, err)
	}
	return nil
}

// RemoveAll removes a directory and all its contents
func RemoveAll(path string) error {
	err := os.RemoveAll(path)
	if err != nil {
		return fmt.Errorf("failed to remove directory %s: %w", path, err)
	}
	return nil
}

// ListDir returns the contents of a directory
func ListDir(path string) ([]FileInfo, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory %s: %w", path, err)
	}
	
	var fileInfos []FileInfo
	for _, entry := range entries {
		entryPath := filepath.Join(path, entry.Name())
		info, err := GetFileInfo(entryPath)
		if err != nil {
			// Skip files that can't be read
			continue
		}
		fileInfos = append(fileInfos, info)
	}
	
	return fileInfos, nil
}

// ListFiles returns only files (not directories) in a directory
func ListFiles(path string) ([]FileInfo, error) {
	allEntries, err := ListDir(path)
	if err != nil {
		return nil, err
	}
	
	var files []FileInfo
	for _, entry := range allEntries {
		if !entry.IsDir {
			files = append(files, entry)
		}
	}
	
	return files, nil
}

// ListDirs returns only directories in a directory
func ListDirs(path string) ([]FileInfo, error) {
	allEntries, err := ListDir(path)
	if err != nil {
		return nil, err
	}
	
	var dirs []FileInfo
	for _, entry := range allEntries {
		if entry.IsDir {
			dirs = append(dirs, entry)
		}
	}
	
	return dirs, nil
}

// ===============================
// File Search and Filtering
// ===============================

// Find searches for files matching a pattern in a directory tree
func Find(root, pattern string) ([]string, error) {
	var matches []string
	
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		
		matched, err := filepath.Match(pattern, filepath.Base(path))
		if err != nil {
			return err
		}
		
		if matched {
			matches = append(matches, path)
		}
		
		return nil
	})
	
	if err != nil {
		return nil, fmt.Errorf("error during file search: %w", err)
	}
	
	return matches, nil
}

// FindFiles searches for files (not directories) matching a pattern
func FindFiles(root, pattern string) ([]string, error) {
	var matches []string
	
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		
		if info.IsDir() {
			return nil
		}
		
		matched, err := filepath.Match(pattern, filepath.Base(path))
		if err != nil {
			return err
		}
		
		if matched {
			matches = append(matches, path)
		}
		
		return nil
	})
	
	if err != nil {
		return nil, fmt.Errorf("error during file search: %w", err)
	}
	
	return matches, nil
}

// FindDirs searches for directories matching a pattern
func FindDirs(root, pattern string) ([]string, error) {
	var matches []string
	
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		
		if !info.IsDir() {
			return nil
		}
		
		matched, err := filepath.Match(pattern, filepath.Base(path))
		if err != nil {
			return err
		}
		
		if matched {
			matches = append(matches, path)
		}
		
		return nil
	})
	
	if err != nil {
		return nil, fmt.Errorf("error during directory search: %w", err)
	}
	
	return matches, nil
}

// Walk walks the file tree rooted at root, calling walkFn for each file
func Walk(root string, walkFn WalkFunc) error {
	return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return walkFn(path, FileInfo{}, err)
		}
		
		fileInfo, err := GetFileInfo(path)
		if err != nil {
			return walkFn(path, FileInfo{}, err)
		}
		
		return walkFn(path, fileInfo, nil)
	})
}

// ===============================
// File Comparison and Hashing
// ===============================

// Equal checks if two files have identical content
func Equal(path1, path2 string) (bool, error) {
	// Check if files exist
	info1, err := os.Stat(path1)
	if err != nil {
		return false, fmt.Errorf("cannot stat file %s: %w", path1, err)
	}
	
	info2, err := os.Stat(path2)
	if err != nil {
		return false, fmt.Errorf("cannot stat file %s: %w", path2, err)
	}
	
	// Quick check: if sizes differ, files are different
	if info1.Size() != info2.Size() {
		return false, nil
	}
	
	// If size is 0, both files are empty
	if info1.Size() == 0 {
		return true, nil
	}
	
	// Compare content
	file1, err := os.Open(path1)
	if err != nil {
		return false, fmt.Errorf("cannot open file %s: %w", path1, err)
	}
	defer file1.Close()
	
	file2, err := os.Open(path2)
	if err != nil {
		return false, fmt.Errorf("cannot open file %s: %w", path2, err)
	}
	defer file2.Close()
	
	const bufferSize = 8192
	
	// Use pooled buffers for efficient memory management
	buf1, returnBuf1 := getPooledBuffer(bufferSize)
	defer returnBuf1()
	buf2, returnBuf2 := getPooledBuffer(bufferSize)
	defer returnBuf2()
	
	for {
		n1, err1 := file1.Read(buf1)
		n2, err2 := file2.Read(buf2)
		
		if n1 != n2 {
			return false, nil
		}
		
		if n1 == 0 {
			break
		}
		
		if string(buf1[:n1]) != string(buf2[:n2]) {
			return false, nil
		}
		
		if err1 != nil || err2 != nil {
			if err1 == io.EOF && err2 == io.EOF {
				break
			}
			if err1 != nil {
				return false, fmt.Errorf("error reading %s: %w", path1, err1)
			}
			return false, fmt.Errorf("error reading %s: %w", path2, err2)
		}
	}
	
	return true, nil
}

// MD5Hash calculates the MD5 hash of a file
func MD5Hash(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("cannot open file %s: %w", path, err)
	}
	defer file.Close()
	
	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", fmt.Errorf("error calculating MD5 hash: %w", err)
	}
	
	return hex.EncodeToString(hash.Sum(nil)), nil
}

// SHA256Hash calculates the SHA256 hash of a file
func SHA256Hash(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("cannot open file %s: %w", path, err)
	}
	defer file.Close()
	
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", fmt.Errorf("error calculating SHA256 hash: %w", err)
	}
	
	return hex.EncodeToString(hash.Sum(nil)), nil
}

// ===============================
// File Type Detection
// ===============================

// DetectMimeType attempts to detect the MIME type of a file
func DetectMimeType(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	
	// Common MIME types based on file extension
	mimeTypes := map[string]string{
		".txt":  "text/plain",
		".html": "text/html",
		".htm":  "text/html",
		".css":  "text/css",
		".js":   "application/javascript",
		".json": "application/json",
		".xml":  "application/xml",
		".pdf":  "application/pdf",
		".zip":  "application/zip",
		".gz":   "application/gzip",
		".tar":  "application/x-tar",
		".jpg":  "image/jpeg",
		".jpeg": "image/jpeg",
		".png":  "image/png",
		".gif":  "image/gif",
		".bmp":  "image/bmp",
		".svg":  "image/svg+xml",
		".ico":  "image/x-icon",
		".mp3":  "audio/mpeg",
		".wav":  "audio/wav",
		".mp4":  "video/mp4",
		".avi":  "video/x-msvideo",
		".mov":  "video/quicktime",
		".doc":  "application/msword",
		".docx": "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		".xls":  "application/vnd.ms-excel",
		".xlsx": "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		".ppt":  "application/vnd.ms-powerpoint",
		".pptx": "application/vnd.openxmlformats-officedocument.presentationml.presentation",
	}
	
	if mimeType, exists := mimeTypes[ext]; exists {
		return mimeType
	}
	
	return "application/octet-stream" // Default for unknown types
}

// IsTextFile checks if a file is likely a text file based on its extension
func IsTextFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	textExtensions := []string{
		".txt", ".md", ".rst", ".log", ".cfg", ".conf", ".ini",
		".json", ".xml", ".yaml", ".yml", ".toml",
		".html", ".htm", ".css", ".js", ".ts", ".jsx", ".tsx",
		".go", ".py", ".java", ".c", ".cpp", ".h", ".hpp",
		".rb", ".php", ".sh", ".bat", ".ps1",
		".sql", ".csv", ".tsv",
	}
	
	for _, textExt := range textExtensions {
		if ext == textExt {
			return true
		}
	}
	
	return false
}

// IsImageFile checks if a file is an image based on its extension
func IsImageFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	imageExtensions := []string{
		".jpg", ".jpeg", ".png", ".gif", ".bmp", ".tiff", ".tif",
		".svg", ".webp", ".ico", ".psd", ".raw",
	}
	
	for _, imgExt := range imageExtensions {
		if ext == imgExt {
			return true
		}
	}
	
	return false
}

// ===============================
// Path Manipulation
// ===============================

// AbsPath returns the absolute path of a file
func AbsPath(path string) (string, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute path for %s: %w", path, err)
	}
	return absPath, nil
}

// RelPath returns the relative path from base to target
func RelPath(base, target string) (string, error) {
	relPath, err := filepath.Rel(base, target)
	if err != nil {
		return "", fmt.Errorf("failed to get relative path from %s to %s: %w", base, target, err)
	}
	return relPath, nil
}

// Dir returns the directory containing the file
func Dir(path string) string {
	return filepath.Dir(path)
}

// Base returns the last element of the path
func Base(path string) string {
	return filepath.Base(path)
}

// Ext returns the file extension
func Ext(path string) string {
	return filepath.Ext(path)
}

// Join joins path elements with the appropriate separator
func Join(elements ...string) string {
	return filepath.Join(elements...)
}

// Split splits a path into directory and file components
func Split(path string) (dir, file string) {
	return filepath.Split(path)
}

// Clean cleans the path, removing redundant separators and up-level references
func Clean(path string) string {
	return filepath.Clean(path)
}

// ===============================
// File Sorting and Organization
// ===============================

// SortBy represents sorting criteria
type SortBy int

const (
	SortByName SortBy = iota
	SortBySize
	SortByModTime
	SortByExt
)

// SortOrder represents sorting order
type SortOrder int

const (
	Ascending SortOrder = iota
	Descending
)

// SortFiles sorts a slice of FileInfo based on criteria and order
func SortFiles(files []FileInfo, by SortBy, order SortOrder) {
	sort.Slice(files, func(i, j int) bool {
		var less bool
		
		switch by {
		case SortByName:
			less = files[i].Name < files[j].Name
		case SortBySize:
			less = files[i].Size < files[j].Size
		case SortByModTime:
			less = files[i].ModTime.Before(files[j].ModTime)
		case SortByExt:
			less = files[i].Ext < files[j].Ext
		default:
			less = files[i].Name < files[j].Name
		}
		
		if order == Descending {
			less = !less
		}
		
		return less
	})
}

// ===============================
// Utility Functions
// ===============================

// Touch creates an empty file or updates the modification time of an existing file
func Touch(path string) error {
	// If file doesn't exist, create it
	if !Exists(path) {
		file, err := os.Create(path)
		if err != nil {
			return fmt.Errorf("failed to create file %s: %w", path, err)
		}
		file.Close()
		return nil
	}
	
	// Update modification time
	now := time.Now()
	err := os.Chtimes(path, now, now)
	if err != nil {
		return fmt.Errorf("failed to update modification time for %s: %w", path, err)
	}
	
	return nil
}

// TempFile creates a temporary file with optional content
func TempFile(pattern string, content []byte) (string, error) {
	tmpFile, err := os.CreateTemp("", pattern)
	if err != nil {
		return "", fmt.Errorf("failed to create temporary file: %w", err)
	}
	
	path := tmpFile.Name()
	
	if content != nil {
		if _, err := tmpFile.Write(content); err != nil {
			tmpFile.Close()
			os.Remove(path)
			return "", fmt.Errorf("failed to write content to temporary file: %w", err)
		}
	}
	
	tmpFile.Close()
	return path, nil
}

// TempDir creates a temporary directory
func TempDir(pattern string) (string, error) {
	tmpDir, err := os.MkdirTemp("", pattern)
	if err != nil {
		return "", fmt.Errorf("failed to create temporary directory: %w", err)
	}
	return tmpDir, nil
}

// Backup creates a backup copy of a file with a timestamp suffix
func Backup(path string) (string, error) {
	if !Exists(path) {
		return "", fmt.Errorf("file does not exist: %s", path)
	}
	
	timestamp := time.Now().Format("20060102_150405")
	ext := filepath.Ext(path)
	base := strings.TrimSuffix(path, ext)
	backupPath := fmt.Sprintf("%s_%s%s", base, timestamp, ext)
	
	err := Copy(path, backupPath)
	if err != nil {
		return "", fmt.Errorf("failed to create backup: %w", err)
	}
	
	return backupPath, nil
}

// LineCount counts the number of lines in a text file
func LineCount(path string) (int, error) {
	file, err := os.Open(path)
	if err != nil {
		return 0, fmt.Errorf("failed to open file %s: %w", path, err)
	}
	defer file.Close()
	
	count := 0
	scanner := bufio.NewScanner(file)
	
	for scanner.Scan() {
		count++
	}
	
	if err := scanner.Err(); err != nil {
		return 0, fmt.Errorf("error counting lines in %s: %w", path, err)
	}
	
	return count, nil
}

// WordCount counts the number of words in a text file
func WordCount(path string) (int, error) {
	content, err := ReadString(path)
	if err != nil {
		return 0, err
	}
	
	words := strings.Fields(content)
	return len(words), nil
}

// IsEmpty checks if a file is empty (0 bytes)
func IsEmpty(path string) (bool, error) {
	info, err := os.Stat(path)
	if err != nil {
		return false, fmt.Errorf("failed to stat file %s: %w", path, err)
	}
	
	return info.Size() == 0, nil
}

// SafeRemove safely removes a file, checking if it exists first
func SafeRemove(path string) error {
	if !Exists(path) {
		return nil // Already doesn't exist
	}
	
	err := os.Remove(path)
	if err != nil {
		return fmt.Errorf("failed to remove file %s: %w", path, err)
	}
	
	return nil
}