package internal

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// setupTestRepo creates a temporary directory, initializes a git repo,
// and creates some dummy files for testing.
func setupTestRepo(t *testing.T) (string, func()) {
	// Check if git and ctags are available
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not found in PATH, skipping TestRepoMapHandler")
	}
	if _, err := exec.LookPath("ctags"); err != nil {
		t.Skip("ctags not found in PATH, skipping TestRepoMapHandler")
	}

	tmpDir, err := os.MkdirTemp("", "tmuxai-repomap-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	// Change working directory to the temp dir for git commands
	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current working directory: %v", err)
	}
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change to temp dir: %v", err)
	}

	// Git init
	if err := exec.Command("git", "init", "--initial-branch=main").Run(); err != nil {
		t.Fatalf("Failed to run 'git init': %v", err)
	}

	// Create some dummy files
	file1Content := `package main
func main() {}
func anotherFunc() {}
`
	if err := os.WriteFile("main.go", []byte(file1Content), 0644); err != nil {
		t.Fatalf("Failed to write main.go: %v", err)
	}

	file2Content := `package internal
type MyStruct struct {}
func (s *MyStruct) MyMethod() {}
`
	if err := os.Mkdir("internal", 0755); err != nil {
		t.Fatalf("Failed to create internal dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join("internal", "lib.go"), []byte(file2Content), 0644); err != nil {
		t.Fatalf("Failed to write internal/lib.go: %v", err)
	}

	// Git add and commit
	if err := exec.Command("git", "config", "user.email", "test@example.com").Run(); err != nil {
		t.Fatalf("Failed to set git user.email: %v", err)
	}
	if err := exec.Command("git", "config", "user.name", "Test User").Run(); err != nil {
		t.Fatalf("Failed to set git user.name: %v", err)
	}
	if err := exec.Command("git", "add", ".").Run(); err != nil {
		t.Fatalf("Failed to run 'git add .': %v", err)
	}
	if err := exec.Command("git", "commit", "-m", "initial commit").Run(); err != nil {
		t.Fatalf("Failed to run 'git commit': %v", err)
	}

	// Teardown function to clean up
	cleanup := func() {
		os.Chdir(originalWd)
		os.RemoveAll(tmpDir)
	}

	return tmpDir, cleanup
}

func TestRepoMapHandler(t *testing.T) {
	tmpDir, cleanup := setupTestRepo(t)
	defer cleanup()

	// 1. Test NewRepoMapHandler creation
	handler := NewRepoMapHandler()
	if handler == nil {
		t.Fatal("NewRepoMapHandler() returned nil, expected a handler instance")
	}
	if !handler.IsEnabled() {
		t.Fatal("handler.IsEnabled() returned false, expected true")
	}
	if handler.projectRoot != tmpDir {
		t.Errorf("handler.projectRoot = %s; want %s", handler.projectRoot, tmpDir)
	}

	// 2. Test initial map generation
	repoMap, err := handler.GetMap()
	if err != nil {
		t.Fatalf("handler.GetMap() returned an error on initial generation: %v", err)
	}

	// Check for expected content
	expectedSymbols := []string{"main", "anotherFunc", "MyStruct", "MyMethod"}
	for _, symbol := range expectedSymbols {
		if !strings.Contains(repoMap, symbol) {
			t.Errorf("Repo map does not contain expected symbol '%s'.\nMap content:\n%s", symbol, repoMap)
		}
	}
	expectedFiles := []string{"main.go", "internal/lib.go"}
	for _, file := range expectedFiles {
		if !strings.Contains(repoMap, file) {
			t.Errorf("Repo map does not contain expected file '%s'.\nMap content:\n%s", file, repoMap)
		}
	}

	t.Logf("Initial repo map generated successfully:\n%s", repoMap)

	// 3. Test caching by calling GetMap again
	repoMapFromCache, err := handler.GetMap()
	if err != nil {
		t.Fatalf("handler.GetMap() returned an error on cached read: %v", err)
	}
	if repoMapFromCache != repoMap {
		t.Errorf("Repo map from cache differs from initial map.")
	}
	t.Log("Repo map from cache matches initial map.")

	// 4. Test cache invalidation after file modification
	time.Sleep(1 * time.Second) // Ensure mtime is different
	file1Content := `package main
func main() {}
func anotherFunc() {}
func newlyAddedFunc() {} // New function
`
	if err := os.WriteFile("main.go", []byte(file1Content), 0644); err != nil {
		t.Fatalf("Failed to modify main.go: %v", err)
	}

	repoMapAfterChange, err := handler.GetMap()
	if err != nil {
		t.Fatalf("handler.GetMap() returned an error after file modification: %v", err)
	}
	if repoMapAfterChange == repoMap {
		t.Fatal("Repo map did not change after file modification, cache was not invalidated.")
	}
	if !strings.Contains(repoMapAfterChange, "newlyAddedFunc") {
		t.Errorf("Repo map after change does not contain new symbol 'newlyAddedFunc'.\nMap content:\n%s", repoMapAfterChange)
	}
	t.Logf("Repo map successfully regenerated after file change:\n%s", repoMapAfterChange)
}
