package handlers

import (
	"database/sql"
	"net/http"
	"websocketserver/auth"
	"websocketserver/ws"
)

// SetupRoutes configures all HTTP routes for the application
func SetupRoutes(mux *http.ServeMux, database *sql.DB, authService *auth.Service, wsServer *ws.Server) {
	// WebSocket routes
	mux.HandleFunc("/ws", wsServer.HandleWebSocket)
	mux.HandleFunc("/active-users", wsServer.ActiveUsersHandler)

	// Authentication routes
	mux.HandleFunc("/auth/register", authService.HandleRegistration)
	mux.HandleFunc("/auth/login", authService.HandleLogin)
	mux.HandleFunc("/auth/check-userid/", authService.HandleCheckUserID)
	mux.HandleFunc("/auth/users/", authService.HandleGetUserInfo)

	// User data routes
	mux.HandleFunc("/user/descriptions", HandleUserDescriptions(authService, database))
	mux.HandleFunc("/user/descriptions/", HandleGetUserDescriptions(database))
	mux.HandleFunc("/user/trackers", HandleUserTrackers(authService, database))
	mux.HandleFunc("/trackers", HandleGetPublicTrackers(database))
	mux.HandleFunc("/user/apis", HandleUserAPIs(authService, database))
	mux.HandleFunc("/apis", HandleGetPublicAPIs(database))
	mux.HandleFunc("/direct-message/", HandleDirectMessage(authService, wsServer))
	mux.HandleFunc("/register-document/", HandleRegisterDocument(authService, wsServer))
	mux.HandleFunc("/append-document/", HandleAppendDocument(authService, wsServer))

	// Tracker application routes
	mux.HandleFunc("/tracker-apps", HandleListTrackerApps())
	mux.HandleFunc("/tracker-folder/", HandleFetchTrackerFolder(authService))

	// Page rendering routes
	mux.HandleFunc("/", ServeHome)
	mux.HandleFunc("/download", ServeDownload)

	// Download routes
	mux.HandleFunc("/download/linux", DownloadLinuxHandler)
	mux.HandleFunc("/download/darwin", DownloadMacHandler)
	mux.HandleFunc("/download/windows", DownloadWindowsHandler)
	mux.HandleFunc("/download/app/linux", DownloadLinuxAppHandler)
	mux.HandleFunc("/download/app/darwin", DownloadMacAppHandler)
	mux.HandleFunc("/download/app/windows", DownloadWindowsAppHandler)
	mux.HandleFunc("/install.sh", ProvideInstallationScriptHandler)

	// Static file serving
	fs := http.FileServer(http.Dir("static"))
	mux.Handle("/static/", http.StripPrefix("/static/", fs))
}
