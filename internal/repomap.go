package internal

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/alvinunreal/tmuxai/logger"
)

const repoMapDir = ".tmuxai"
const repoMapCacheFile = "repomap.json"

// RepoMapCache stores the cached data for the repo map.
type RepoMapCache struct {
	RepoMap    string               `json:"repo_map"`
	FileMtimes map[string]time.Time `json:"file_mtimes"`
}

// RepoMapHandler manages the creation and caching of the repository map.
type RepoMapHandler struct {
	enabled     bool
	projectRoot string
	cachePath   string
}

// NewRepoMapHandler creates a new handler. It returns nil if not in a git repo.
func NewRepoMapHandler() *RepoMapHandler {
	gitRoot, err := findGitRoot()
	if err != nil {
		logger.Info("Not a git repository. RepoMap feature disabled.")
		return nil // Not a git repo, so disable the feature.
	}

	// Check for universal-ctags executable before enabling the handler
	if _, err := exec.LookPath("ctags"); err != nil {
		// This is logged for debugging, but the user is not warned repeatedly.
		logger.Info("universal-ctags not found in PATH. RepoMap feature will be disabled. Please install it to use this feature.")
		return nil
	}

	sessionDir := filepath.Join(gitRoot, repoMapDir)
	if err := os.MkdirAll(sessionDir, 0755); err != nil {
		logger.Error("Failed to create repomap directory: %v", err)
		return nil
	}

	return &RepoMapHandler{
		enabled:     true,
		projectRoot: gitRoot,
		cachePath:   filepath.Join(sessionDir, repoMapCacheFile),
	}
}

// IsEnabled returns true if the handler is active.
func (h *RepoMapHandler) IsEnabled() bool {
	return h != nil && h.enabled
}

// GetMap generates or retrieves from cache the repository map.
func (h *RepoMapHandler) GetMap() (string, error) {
	if !h.IsEnabled() {
		return "", nil
	}

	files, err := h.getTrackedFiles()
	if err != nil {
		return "", fmt.Errorf("could not get tracked files: %w", err)
	}

	currentMtimes, err := h.getFileMtimes(files)
	if err != nil {
		return "", fmt.Errorf("could not get file modification times: %w", err)
	}

	cached, err := h.loadCache()
	if err == nil && h.isCacheValid(cached, currentMtimes) {
		logger.Debug("RepoMap cache is valid, using cached version.")
		return cached.RepoMap, nil
	}
	if err != nil && !os.IsNotExist(err) {
		logger.Info("Could not load RepoMap cache: %v. Regenerating.", err)
	}

	logger.Info("Generating new repo map...")

	repoMap, err := h.generateMap(files)
	if err != nil {
		return "", fmt.Errorf("failed to generate repo map: %w", err)
	}

	newCache := &RepoMapCache{
		RepoMap:    repoMap,
		FileMtimes: currentMtimes,
	}

	if err := h.saveCache(newCache); err != nil {
		logger.Error("Failed to save repo map cache: %v", err)
	}

	return repoMap, nil
}

func (h *RepoMapHandler) generateMap(files []string) (string, error) {
	// Create a temporary file list for ctags
	tmpFile, err := os.CreateTemp("", "ctags-file-list-*.txt")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file for ctags: %w", err)
	}
	defer os.Remove(tmpFile.Name())

	for _, file := range files {
		// ctags needs paths relative to the project root
		relPath, err := filepath.Rel(h.projectRoot, file)
		if err != nil {
			// should not happen if files come from git ls-files from project root
			continue
		}
		if _, err := tmpFile.WriteString(relPath + "\n"); err != nil {
			return "", fmt.Errorf("failed to write to ctags temp file: %w", err)
		}
	}
	tmpFile.Close()

	// Run ctags
	cmd := exec.Command("ctags", "--output-format=json", "--fields=+n", "-L", tmpFile.Name())
	cmd.Dir = h.projectRoot
	output, err := cmd.Output()
	if err != nil {
		// ctags returns non-zero exit code if no tags are found, which is not an error for us.
		// We check if there's output. If not, it's a real error.
		if len(output) == 0 {
			if ee, ok := err.(*exec.ExitError); ok {
				return "", fmt.Errorf("ctags failed with exit code %d: %s", ee.ExitCode(), string(ee.Stderr))
			}
			return "", fmt.Errorf("ctags failed: %w", err)
		}
	}

	return h.parseCtagsOutput(output), nil
}

func (h *RepoMapHandler) parseCtagsOutput(output []byte) string {
	type Ctag struct {
		Name string `json:"name"`
		Path string `json:"path"`
		Line int    `json:"line"`
		Kind string `json:"kind"`
	}

	tagsByFile := make(map[string][]Ctag)
	lines := bytes.Split(output, []byte("\n"))

	for _, line := range lines {
		if len(line) == 0 {
			continue
		}
		var tag Ctag
		if err := json.Unmarshal(line, &tag); err != nil {
			logger.Debug("Failed to unmarshal ctag line: %s, error: %v", string(line), err)
			continue
		}
		tagsByFile[tag.Path] = append(tagsByFile[tag.Path], tag)
	}

	var sortedFiles []string
	for file := range tagsByFile {
		sortedFiles = append(sortedFiles, file)
	}
	sort.Strings(sortedFiles)

	var builder strings.Builder
	builder.WriteString("Repository Map:\n")

	for _, file := range sortedFiles {
		builder.WriteString(fmt.Sprintf("--- %s ---\n", file))
		tags := tagsByFile[file]
		// sort tags by line number
		sort.Slice(tags, func(i, j int) bool {
			return tags[i].Line < tags[j].Line
		})
		for _, tag := range tags {
			builder.WriteString(fmt.Sprintf("  %s (%s)\n", tag.Name, tag.Kind))
		}
	}

	return builder.String()
}

func (h *RepoMapHandler) getTrackedFiles() ([]string, error) {
	cmd := exec.Command("git", "ls-files")
	cmd.Dir = h.projectRoot
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	files := strings.Split(strings.TrimSpace(string(output)), "\n")
	var absolutePaths []string
	for _, f := range files {
		absolutePaths = append(absolutePaths, filepath.Join(h.projectRoot, f))
	}
	return absolutePaths, nil
}

func (h *RepoMapHandler) getFileMtimes(files []string) (map[string]time.Time, error) {
	mtimes := make(map[string]time.Time)
	for _, file := range files {
		info, err := os.Stat(file)
		if err != nil {
			// File might have been deleted since ls-files ran, which is fine.
			if os.IsNotExist(err) {
				continue
			}
			return nil, err
		}
		mtimes[file] = info.ModTime()
	}
	return mtimes, nil
}

func (h *RepoMapHandler) isCacheValid(cached *RepoMapCache, currentMtimes map[string]time.Time) bool {
	if len(cached.FileMtimes) != len(currentMtimes) {
		logger.Debug("RepoMap cache invalid: file count changed.")
		return false // Number of files has changed.
	}
	for file, mtime := range currentMtimes {
		cachedMtime, ok := cached.FileMtimes[file]
		if !ok || !cachedMtime.Equal(mtime) {
			logger.Debug("RepoMap cache invalid: mtime for %s changed.", file)
			return false // File is new or has been modified.
		}
	}
	return true
}

func (h *RepoMapHandler) loadCache() (*RepoMapCache, error) {
	data, err := os.ReadFile(h.cachePath)
	if err != nil {
		return nil, err
	}
	var cache RepoMapCache
	if err := json.Unmarshal(data, &cache); err != nil {
		return nil, err
	}
	return &cache, nil
}

func (h *RepoMapHandler) saveCache(cache *RepoMapCache) error {
	data, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(h.cachePath, data, 0644)
}

// findGitRoot traverses up from the CWD to find the .git directory.
func findGitRoot() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	dir := cwd
	for {
		gitPath := filepath.Join(dir, ".git")
		if _, err := os.Stat(gitPath); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir { // Reached root directory
			return "", fmt.Errorf("not a git repository")
		}
		dir = parent
	}
}
