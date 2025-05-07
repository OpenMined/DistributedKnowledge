package handlers

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"websocketserver/auth"
)

// TrackerApp represents a tracker application available for installation
type TrackerApp struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	Description string `json:"description"`
	IconPath    string `json:"icon_path"`
}

// HandleListTrackerApps returns a handler that serves the list of available tracker applications
func HandleListTrackerApps() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Get current working directory
		cwd, err := os.Getwd()
		if err != nil {
			http.Error(w, "Server error: unable to determine working directory", http.StatusInternalServerError)
			return
		}

		// Read the tracker list file
		filePath := filepath.Join(cwd, "trackers", "tracker_list.json")
		file, err := os.Open(filePath)
		if err != nil {
			http.Error(w, "Server error: unable to read tracker list", http.StatusInternalServerError)
			return
		}
		defer file.Close()

		// Read the file contents
		data, err := io.ReadAll(file)
		if err != nil {
			http.Error(w, "Server error: unable to read tracker list data", http.StatusInternalServerError)
			return
		}

		// Set content type header
		w.Header().Set("Content-Type", "application/json")

		// Write the JSON data directly
		w.Write(data)
	}
}

// HandleFetchTrackerFolder returns a handler that serves pre-packaged ZIP files for tracker applications
func HandleFetchTrackerFolder(authService *auth.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Extract tracker name from URL path
		// Expected format: /tracker-folder/{trackerName}
		pathParts := strings.Split(r.URL.Path, "/")
		if len(pathParts) < 3 {
			http.Error(w, "Invalid tracker path", http.StatusBadRequest)
			return
		}

		trackerName := pathParts[2]

		// Validate the tracker name to prevent directory traversal attacks
		if !isValidFolderName(trackerName) {
			http.Error(w, "Invalid tracker name", http.StatusBadRequest)
			return
		}

		// Get current working directory
		cwd, err := os.Getwd()
		if err != nil {
			http.Error(w, "Server error: unable to determine working directory", http.StatusInternalServerError)
			return
		}

		// Construct the path to the tracker zip file
		zipFilePath := filepath.Join(cwd, "trackers", trackerName+".zip")

		// Check if the zip file exists in the trackers directory
		if _, err := os.Stat(zipFilePath); err != nil {
			// Fallback: check if it's a directory and try to serve that
			trackerFolderPath := filepath.Join(cwd, "trackers", trackerName)
			if info, err := os.Stat(trackerFolderPath); err == nil && info.IsDir() {
				// It's a directory, so try to find a zip file within it
				zipFilePath = filepath.Join(trackerFolderPath, trackerName+".zip")
				if _, err := os.Stat(zipFilePath); err != nil {
					http.Error(w, "Tracker package not found", http.StatusNotFound)
					return
				}
			} else {
				http.Error(w, "Tracker package not found", http.StatusNotFound)
				return
			}
		}

		// Open the zip file
		file, err := os.Open(zipFilePath)
		if err != nil {
			http.Error(w, "Server error: unable to open tracker package", http.StatusInternalServerError)
			return
		}
		defer file.Close()

		// Get file info for headers
		fileInfo, err := file.Stat()
		if err != nil {
			http.Error(w, "Server error: unable to get file information", http.StatusInternalServerError)
			return
		}

		// Set appropriate headers
		w.Header().Set("Content-Type", "application/zip")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s.zip", trackerName))
		w.Header().Set("Content-Length", fmt.Sprintf("%d", fileInfo.Size()))

		// Copy the file directly to the response
		_, err = io.Copy(w, file)
		if err != nil {
			log.Printf("Error sending zip file: %v", err)
			return
		}
	}
}

// isValidFolderName checks if a folder name is valid and not a path traversal attempt
func isValidFolderName(name string) bool {
	// Check for directory traversal attempts
	if strings.Contains(name, "..") || strings.Contains(name, "/") || strings.Contains(name, "\\") {
		return false
	}

	// Allow only alphanumeric characters, underscores, and hyphens
	for _, c := range name {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_' || c == '-') {
			return false
		}
	}

	return true
}
