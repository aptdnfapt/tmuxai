package internal

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// GetReadFileConfirm returns the read file confirmation setting
func (m *Manager) GetReadFileConfirm() bool {
	if override, exists := m.SessionOverrides["read_file_confirm"]; exists {
		if val, ok := override.(bool); ok {
			return val
		}
	}
	return m.Config.ReadFileConfirm
}

// GetMaxReadFileSize returns the maximum file size for reading
func (m *Manager) GetMaxReadFileSize() int {
	if override, exists := m.SessionOverrides["max_read_file_size"]; exists {
		if val, ok := override.(int); ok {
			return val
		}
	}
	return m.Config.MaxReadFileSize
}

// validateReadFile checks if a file is valid for reading and returns its info.
func (m *Manager) validateReadFile(filePath string) (os.FileInfo, string, error) {
	filePath = strings.TrimSpace(filePath)

	// Convert relative paths to absolute
	if !filepath.IsAbs(filePath) {
		cwd, err := os.Getwd()
		if err != nil {
			return nil, "", fmt.Errorf("failed to get current directory: %w", err)
		}
		filePath = filepath.Join(cwd, filePath)
	}

	// Check if file exists
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, filePath, fmt.Errorf("file does not exist: %s", filePath)
		}
		return nil, filePath, fmt.Errorf("failed to access file: %w", err)
	}

	// Check if it's a directory
	if fileInfo.IsDir() {
		return nil, filePath, fmt.Errorf("path is a directory, not a file: %s", filePath)
	}

	// Check file size
	maxSize := int64(m.GetMaxReadFileSize())
	if fileInfo.Size() > maxSize {
		return nil, filePath, fmt.Errorf("file too large (%d bytes, max %d bytes): %s", fileInfo.Size(), maxSize, filePath)
	}

	// Check if it's a binary file (simple heuristic)
	if !isTextFile(filePath) {
		return nil, filePath, fmt.Errorf("file appears to be binary: %s", filePath)
	}

	return fileInfo, filePath, nil
}

// isTextFile performs a simple check to determine if a file is likely text
func isTextFile(filePath string) bool {
	// Check file extension
	ext := strings.ToLower(filepath.Ext(filePath))
	textExtensions := []string{
		".txt", ".md", ".go", ".py", ".js", ".ts", ".html", ".css", ".json", ".xml", ".yaml", ".yml",
		".sh", ".bash", ".zsh", ".fish", ".ps1", ".bat", ".cmd", ".c", ".cpp", ".h", ".hpp",
		".java", ".php", ".rb", ".rs", ".swift", ".kt", ".scala", ".clj", ".hs", ".ml", ".fs",
		".sql", ".r", ".m", ".pl", ".lua", ".vim", ".emacs", ".cfg", ".conf", ".ini", ".toml",
		".dockerfile", ".makefile", ".cmake", ".gradle", ".properties", ".log", ".csv", ".tsv",
	}
	
	for _, textExt := range textExtensions {
		if ext == textExt {
			return true
		}
	}
	
	// Check for files without extension that are commonly text
	baseName := strings.ToLower(filepath.Base(filePath))
	textFiles := []string{
		"readme", "license", "changelog", "makefile", "dockerfile", "gemfile", "rakefile",
		"procfile", "vagrantfile", "gruntfile", "gulpfile", "webpack", "package", "composer",
	}
	
	for _, textFile := range textFiles {
		if baseName == textFile {
			return true
		}
	}
	
	// If no extension and not a known text file, do a quick binary check
	if ext == "" {
		file, err := os.Open(filePath)
		if err != nil {
			return false
		}
		defer file.Close()
		
		// Read first 512 bytes to check for null bytes
		buffer := make([]byte, 512)
		n, err := file.Read(buffer)
		if err != nil && n == 0 {
			return false
		}
		
		// If we find null bytes, it's likely binary
		for i := 0; i < n; i++ {
			if buffer[i] == 0 {
				return false
			}
		}
		return true
	}
	
	return false
}
