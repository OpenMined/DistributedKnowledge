module dk

go 1.24.1

replace dk/client => ../websocketclient

require (
	dk/client v0.0.0-00010101000000-000000000000
	github.com/mark3labs/mcp-go v0.18.0
	github.com/philippgille/chromem-go v0.7.0
)

require (
	github.com/google/uuid v1.6.0 // indirect
	github.com/gorilla/websocket v1.5.3 // indirect
	github.com/sashabaranov/go-openai v1.38.1 // indirect
	github.com/yosida95/uritemplate/v3 v3.0.2 // indirect
)
