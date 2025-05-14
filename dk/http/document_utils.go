package http

import (
	"mime"
	"path/filepath"
	"strings"
)

// DocumentType returns the MIME type for a given file name or path
// This is used by both document handlers and API management handlers
func DocumentType(filename string) string {
	// Get the file extension
	ext := strings.ToLower(filepath.Ext(filename))

	// Get MIME type from extension
	mimeType := mime.TypeByExtension(ext)
	if mimeType == "" {
		// Default to application/octet-stream for unknown types
		return "application/octet-stream"
	}

	return mimeType
}

// InferContentType infers the content type based on file extension
func InferContentType(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))

	switch ext {
	case ".pdf":
		return "application/pdf"
	case ".txt", ".md":
		return "text/plain"
	case ".htm", ".html":
		return "text/html"
	case ".doc", ".docx":
		return "application/msword"
	case ".xls", ".xlsx":
		return "application/vnd.ms-excel"
	case ".ppt", ".pptx":
		return "application/vnd.ms-powerpoint"
	case ".json":
		return "application/json"
	case ".xml":
		return "application/xml"
	case ".csv":
		return "text/csv"
	default:
		return "application/octet-stream"
	}
}
