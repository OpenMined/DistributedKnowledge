package handlers

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
)

// DownloadLinuxHandler serves the Linux binary for download
func DownloadLinuxHandler(w http.ResponseWriter, r *http.Request) {
	// Specify the path to the binary file you wish to serve.
	filePath := "./install/binaries/linux_dk"
	fileName := "dk"

	// Open the file
	file, err := os.Open(filePath)
	if err != nil {
		http.Error(w, "File not found", http.StatusNotFound)
		log.Printf("Error opening file %s: %v", filePath, err)
		return
	}
	defer file.Close()

	// Set the headers to indicate a file download.
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", fileName))
	w.Header().Set("Content-Type", "application/octet-stream")

	// Stream the file to the client.
	if _, err := io.Copy(w, file); err != nil {
		http.Error(w, "Error downloading file", http.StatusInternalServerError)
		log.Printf("Error copying file data: %v", err)
	}
}

// DownloadMacHandler serves the Mac binary for download
func DownloadMacHandler(w http.ResponseWriter, r *http.Request) {
	// Specify the path to the binary file you wish to serve.
	filePath := "./install/binaries/mac_dk"
	fileName := "dk"

	// Open the file
	file, err := os.Open(filePath)
	if err != nil {
		http.Error(w, "File not found", http.StatusNotFound)
		log.Printf("Error opening file %s: %v", filePath, err)
		return
	}
	defer file.Close()

	// Set the headers to indicate a file download.
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", fileName))
	w.Header().Set("Content-Type", "application/octet-stream")

	// Stream the file to the client.
	if _, err := io.Copy(w, file); err != nil {
		http.Error(w, "Error downloading file", http.StatusInternalServerError)
		log.Printf("Error copying file data: %v", err)
	}
}

// DownloadWindowsHandler serves the Windows binary for download
func DownloadWindowsHandler(w http.ResponseWriter, r *http.Request) {
	// Specify the path to the binary file you wish to serve.
	filePath := "./install/binaries/windows_dk.exe"
	fileName := "windows_dk.exe"

	// Open the file
	file, err := os.Open(filePath)
	if err != nil {
		http.Error(w, "File not found", http.StatusNotFound)
		log.Printf("Error opening file %s: %v", filePath, err)
		return
	}
	defer file.Close()

	// Set the headers to indicate a file download.
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", fileName))
	w.Header().Set("Content-Type", "application/octet-stream")

	// Stream the file to the client.
	if _, err := io.Copy(w, file); err != nil {
		http.Error(w, "Error downloading file", http.StatusInternalServerError)
		log.Printf("Error copying file data: %v", err)
	}
}

// DownloadLinuxAppHandler serves the Linux app binary for download
func DownloadLinuxAppHandler(w http.ResponseWriter, r *http.Request) {
	// Specify the path to the binary file you wish to serve.
	filePath := "./install/binaries/dk-app/linux_dk"
	fileName := "dk-app"

	// Open the file
	file, err := os.Open(filePath)
	if err != nil {
		http.Error(w, "File not found", http.StatusNotFound)
		log.Printf("Error opening file %s: %v", filePath, err)
		return
	}
	defer file.Close()

	// Set the headers to indicate a file download.
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", fileName))
	w.Header().Set("Content-Type", "application/octet-stream")

	// Stream the file to the client.
	if _, err := io.Copy(w, file); err != nil {
		http.Error(w, "Error downloading file", http.StatusInternalServerError)
		log.Printf("Error copying file data: %v", err)
	}
}

// DownloadMacAppHandler serves the Mac app binary for download
func DownloadMacAppHandler(w http.ResponseWriter, r *http.Request) {
	// Specify the path to the binary file you wish to serve.
	filePath := "./install/binaries/dk-app/mac_dk"
	fileName := "dk-app"

	// Open the file
	file, err := os.Open(filePath)
	if err != nil {
		http.Error(w, "File not found", http.StatusNotFound)
		log.Printf("Error opening file %s: %v", filePath, err)
		return
	}
	defer file.Close()

	// Set the headers to indicate a file download.
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", fileName))
	w.Header().Set("Content-Type", "application/octet-stream")

	// Stream the file to the client.
	if _, err := io.Copy(w, file); err != nil {
		http.Error(w, "Error downloading file", http.StatusInternalServerError)
		log.Printf("Error copying file data: %v", err)
	}
}

// DownloadWindowsAppHandler serves the Windows app binary for download
func DownloadWindowsAppHandler(w http.ResponseWriter, r *http.Request) {
	// Specify the path to the binary file you wish to serve.
	filePath := "./install/binaries/dk-app/windows_dk.exe"
	fileName := "DK App Installer.exe"

	// Open the file
	file, err := os.Open(filePath)
	if err != nil {
		http.Error(w, "File not found", http.StatusNotFound)
		log.Printf("Error opening file %s: %v", filePath, err)
		return
	}
	defer file.Close()

	// Set the headers to indicate a file download.
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", fileName))
	w.Header().Set("Content-Type", "application/octet-stream")

	// Stream the file to the client.
	if _, err := io.Copy(w, file); err != nil {
		http.Error(w, "Error downloading file", http.StatusInternalServerError)
		log.Printf("Error copying file data: %v", err)
	}
}

// ProvideInstallationScriptHandler serves the installation script
func ProvideInstallationScriptHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/x-shellscript")
	http.ServeFile(w, r, "./install/install.sh")
}
