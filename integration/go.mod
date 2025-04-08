module integration

go 1.24.1

require (
	github.com/mattn/go-sqlite3 v1.14.27
	websocketclient v0.0.0-00010101000000-000000000000
	websocketserver v0.0.0-00010101000000-000000000000
)

require (
	github.com/golang-jwt/jwt v3.2.2+incompatible // indirect
	github.com/gorilla/websocket v1.5.3 // indirect
)

replace websocketserver => ../websocketserver

replace websocketclient => ../websocketclient
