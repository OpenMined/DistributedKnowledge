package core

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sync"
)

// LLMProviderKey is a context key for the LLM provider
type LLMProviderKey struct{}

// WithLLMProvider adds an LLM provider to the context
func WithLLMProvider(ctx context.Context, provider LLMProvider) context.Context {
	return context.WithValue(ctx, LLMProviderKey{}, provider)
}

// LLMProviderFromContext extracts the LLM provider from the context
func LLMProviderFromContext(ctx context.Context) (LLMProvider, error) {
	provider, ok := ctx.Value(LLMProviderKey{}).(LLMProvider)
	if !ok {
		return nil, fmt.Errorf("LLM provider not found in context")
	}
	return provider, nil
}

// LoadModelConfig loads LLM model configuration from a JSON file
func LoadModelConfig(configFile string) (ModelConfig, error) {
	var config ModelConfig

	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		return config, fmt.Errorf("model config file does not exist: %w", err)
	}

	raw, err := os.ReadFile(configFile)
	if err != nil {
		return config, fmt.Errorf("failed to read model config file: %w", err)
	}

	if err := json.Unmarshal(raw, &config); err != nil {
		return config, fmt.Errorf("failed to unmarshal model config: %w", err)
	}

	return config, nil
}

func LoadQueries(queriesFile string) (QueriesData, error) {
	var data QueriesData
	// If file doesn't exist, initialize an empty map.
	if _, err := os.Stat(queriesFile); os.IsNotExist(err) {
		data.Queries = make(map[string]Query)
		return data, nil
	}
	raw, err := os.ReadFile(queriesFile)
	if err != nil {
		return data, fmt.Errorf("failed to read queries file: %w", err)
	}
	if err := json.Unmarshal(raw, &data); err != nil {
		return data, fmt.Errorf("failed to unmarshal queries file: %w", err)
	}
	return data, nil
}

func SaveQueries(queriesFile string, data QueriesData) error {
	// Ensure directory exists.
	dir := filepath.Dir(queriesFile)
	if err := os.MkdirAll(dir, fs.ModePerm); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}
	raw, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal queries data: %w", err)
	}
	if err := os.WriteFile(queriesFile, raw, 0644); err != nil {
		return fmt.Errorf("failed to write queries file: %w", err)
	}
	return nil
}

func generateQueryID() (string, error) {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return "qry-" + hex.EncodeToString(b), nil
}

// ScanDirToMap walks `root` recursively, reading every regular file it finds.
// It returns a map keyed by absolute path with the file's contents as []byte.
// Reading is done in parallel (up to GOMAXPROCS workers).
func ScanDirToMap(ctx context.Context, root string) (map[string]string, error) {
	// Sanity‑check the root
	info, err := os.Stat(root)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, errors.New("root path is not a directory")
	}

	// Result map protected by a mutex
	result := make(map[string]string)
	var mu sync.Mutex

	root = filepath.Clean(root)     // keep this if you already had it
	rootBase := filepath.Base(root) // <- NEW: “project”, “photos”, etc.

	// Channel of work to do (file paths)
	filesCh := make(chan string, 128)

	// Aggregate the first error we encounter
	errCh := make(chan error, 1)
	sendErr := func(e error) {
		select {
		case errCh <- e: // first error wins
		default:
		}
	}

	// Worker pool
	var wg sync.WaitGroup
	workers := runtime.GOMAXPROCS(0)
	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go func() {
			defer wg.Done()
			for path := range filesCh {
				data, readErr := os.ReadFile(path)
				if readErr != nil {
					sendErr(readErr)
					continue
				}
				rel, _ := filepath.Rel(root, path) // always safe inside WalkDir
				key := filepath.Join(rootBase, rel)

				mu.Lock()
				result[key] = string(data)
				mu.Unlock()
			}
		}()
	}

	// Walk the tree and publish work
	walkErr := filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr // propagate traversal errors
		}
		if !d.Type().IsRegular() {
			return nil // skip directories, symlinks, etc.
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case filesCh <- path:
			return nil
		}
	})
	close(filesCh)

	// Wait for workers; collect any read errors
	wg.Wait()
	close(errCh)

	// Return the earliest error: traversal first, then read/cancel errors
	if walkErr != nil {
		return nil, walkErr
	}
	if readErr := <-errCh; readErr != nil {
		return nil, readErr
	}
	return result, nil
}

// WriteMapToDir takes a destination directory and a map[path]→data, then
// creates all needed folders and files under the destination root.
//
//	destRoot ─┬─ sub/dir/a.txt
//	          └─ other/b.jpg
//
// If destRoot does not exist it is created.  The write is performed in
// parallel (one goroutine per logical CPU).
func WriteMapToDir(ctx context.Context, destRoot string, data map[string]string) error {
	if destRoot == "" {
		return errors.New("destination root must not be empty")
	}
	if len(data) == 0 {
		return nil // nothing to do
	}

	// Ensure destRoot exists and is a directory.
	if st, err := os.Stat(destRoot); err != nil {
		if os.IsNotExist(err) {
			if mkErr := os.MkdirAll(destRoot, 0o755); mkErr != nil {
				return fmt.Errorf("create dest root: %w", mkErr)
			}
		} else {
			return err
		}
	} else if !st.IsDir() {
		return fmt.Errorf("destination %q is not a directory", destRoot)
	}

	// Channel of work items
	type job struct {
		relPath string
		content []byte
	}
	jobs := make(chan job, 128)

	// One error propagated back
	errCh := make(chan error, 1)
	sendErr := func(e error) {
		select {
		case errCh <- e:
		default:
		}
	}

	// Worker pool
	var wg sync.WaitGroup
	workers := runtime.GOMAXPROCS(0)
	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go func() {
			defer wg.Done()
			for j := range jobs {
				select {
				case <-ctx.Done():
					sendErr(ctx.Err())
					return
				default:
				}

				target := filepath.Join(destRoot, j.relPath)
				parent := filepath.Dir(target)
				if mkErr := os.MkdirAll(parent, 0o755); mkErr != nil {
					sendErr(fmt.Errorf("mkdir %q: %w", parent, mkErr))
					continue
				}
				filename := filepath.Base(target)
				fileMode := fs.FileMode(0o644)
				if filename == "run.sh" {
					fileMode = fs.FileMode(0o755)
				}
				if writeErr := os.WriteFile(target, []byte(j.content), fileMode); writeErr != nil {
					sendErr(fmt.Errorf("write %q: %w", target, writeErr))
					continue
				}
			}
		}()
	}

	// Produce jobs
	for srcPath, bytes := range data {
		// Normalise key so we can join safely.
		rel := srcPath
		if filepath.IsAbs(srcPath) {
			// Strip drive/leading slash so we don't write outside destRoot.
			if r, err := filepath.Rel(string(filepath.VolumeName(srcPath)), srcPath); err == nil {
				rel = r
			} else {
				rel = srcPath[1:] // best‑effort fallback
			}
		}
		select {
		case <-ctx.Done():
			break
		case jobs <- job{relPath: rel, content: []byte(bytes)}:
		}
	}
	close(jobs)

	wg.Wait()
	close(errCh)

	// report earliest error
	if err := <-errCh; err != nil {
		return err
	}
	return ctx.Err()
}
