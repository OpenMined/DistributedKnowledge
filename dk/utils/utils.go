package utils

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"database/sql"
	"dk/client"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/philippgille/chromem-go"
	"net/http"
	"os"
	"strings"
	"time"
)

type Parameters struct {
	PrivateKeyPath  *string
	PublicKeyPath   *string
	UserID          *string
	VectorDBPath    *string
	RagSourcesFile  *string
	ModelConfigFile *string
	ServerURL       *string
	HTTPPort        *string
	SyftboxConfig   *string
	DBPath          *string
}

type RemoteMessage struct {
	Type     string            `json:"type"`
	Message  string            `json:"message,omitempty"`
	Files    map[string]string `json:"files,omitempty"`
	Filename string            `json:"filename,omitempty"`
	Content  string            `json:"content,omitempty"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

type AnswerMessage struct {
	Answer string `json:"answer"`
	From   string `json:"from"`
	Query  string `json:"query"`
}

// Message type constants
const (
	MessageTypeForward            = "forward"
	MessageTypeAnswer             = "answer"
	MessageTypeQuery              = "query"
	MessageTypeApp                = "app"
	MessageTypeRegisterDocument   = "register_document"
	MessageTypeAppendDocument     = "append_document"
	MessageTypeRegisterDocSuccess = "register_document_success"
	MessageTypeRegisterDocError   = "register_document_error"
)

type RegisterDocumentMessage struct {
	Type     string            `json:"type"` // Should be "register_document" or "append_document"
	Filename string            `json:"filename"`
	Content  string            `json:"content"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

func LoadOrCreateKeys(privateKeyPath, publicKeyPath string) (ed25519.PublicKey, ed25519.PrivateKey, error) {
	if _, err := os.Stat(privateKeyPath); os.IsNotExist(err) {
		publicKey, privateKey, err := ed25519.GenerateKey(nil)
		if err != nil {
			return nil, nil, err
		}
		if err := os.WriteFile(privateKeyPath, []byte(hex.EncodeToString(privateKey)), 0600); err != nil {
			return nil, nil, err
		}
		if err := os.WriteFile(publicKeyPath, []byte(hex.EncodeToString(publicKey)), 0600); err != nil {
			return nil, nil, err
		}
		return publicKey, privateKey, nil
	}

	privateKeyHex, err := os.ReadFile(privateKeyPath)
	if err != nil {
		return nil, nil, err
	}
	publicKeyHex, err := os.ReadFile(publicKeyPath)
	if err != nil {
		return nil, nil, err
	}

	privateKey, err := hex.DecodeString(string(privateKeyHex))
	if err != nil {
		return nil, nil, err
	}
	publicKey, err := hex.DecodeString(string(publicKeyHex))
	if err != nil {
		return nil, nil, err
	}
	return ed25519.PublicKey(publicKey), ed25519.PrivateKey(privateKey), nil
}

// 1. Define a key type and helper functions.
type DkKey struct{}
type ParamsKey struct{}
type chromemCollectionKey struct{}
type databaseKey struct{}

func WithDatabase(ctx context.Context, db *sql.DB) context.Context {
	return context.WithValue(ctx, databaseKey{}, db)
}

func DatabaseFromContext(ctx context.Context) (*sql.DB, error) {
	db, ok := ctx.Value(databaseKey{}).(*sql.DB)
	if !ok {
		return nil, fmt.Errorf("collection not found in context")
	}
	return db, nil
}

func WithChromemCollection(ctx context.Context, collection *chromem.Collection) context.Context {
	return context.WithValue(ctx, chromemCollectionKey{}, collection)
}

func ChromemCollectionFromContext(ctx context.Context) (*chromem.Collection, error) {
	collection, ok := ctx.Value(chromemCollectionKey{}).(*chromem.Collection)
	if !ok {
		return nil, fmt.Errorf("collection not found in context")
	}
	return collection, nil
}

func WithParams(ctx context.Context, params Parameters) context.Context {
	return context.WithValue(ctx, ParamsKey{}, params)
}

func ParamsFromContext(ctx context.Context) (Parameters, error) {
	params, ok := ctx.Value(ParamsKey{}).(Parameters)
	if !ok {
		return Parameters{}, fmt.Errorf("params not found in context")
	}
	return params, nil
}

func WithDK(ctx context.Context, dk *lib.Client) context.Context {
	return context.WithValue(ctx, DkKey{}, dk)
}

func DkFromContext(ctx context.Context) (*lib.Client, error) {
	dk, ok := ctx.Value(DkKey{}).(*lib.Client)
	if !ok {
		return nil, fmt.Errorf("dk not found in context")
	}
	return dk, nil
}

// UpdateDescriptions replaces every row in descriptions_global
// with the strings in data. It runs in a single transaction and
// ignores empty or duplicate descriptions.
func UpdateDescriptions(ctx context.Context, data []string) error {
	if data == nil {
		return errors.New("UpdateDescriptions: nil input")
	}

	db, err := DatabaseFromContext(ctx)
	if err != nil {
		return fmt.Errorf("UpdateDescriptions: get DB from context: %w", err)
	}

	tx, err := db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return fmt.Errorf("UpdateDescriptions: begin transaction: %w", err)
	}
	// ensure rollback on panic or error
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		} else if err != nil {
			_ = tx.Rollback()
		}
	}()

	// 1) clear out old descriptions
	if _, err = tx.ExecContext(ctx, `DELETE FROM descriptions_global`); err != nil {
		return fmt.Errorf("UpdateDescriptions: delete existing: %w", err)
	}

	// 2) prepare insert (IGNORE duplicates)
	stmt, err := tx.PrepareContext(ctx, `
        INSERT OR IGNORE INTO descriptions_global(description)
        VALUES (?)
    `)
	if err != nil {
		return fmt.Errorf("UpdateDescriptions: prepare insert: %w", err)
	}
	defer stmt.Close()

	// 3) insert each non‑empty, trimmed description
	for _, d := range data {
		d = strings.TrimSpace(d)
		if d == "" {
			continue
		}
		if _, err = stmt.ExecContext(ctx, d); err != nil {
			return fmt.Errorf("UpdateDescriptions: inserting %q: %w", d, err)
		}
	}

	// 4) commit once all inserts succeed
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("UpdateDescriptions: commit: %w", err)
	}
	return nil
}

// GetDescriptions reads all descriptions out of descriptions_global,
// ordered by their primary key (in insertion order).
func GetDescriptions(ctx context.Context) ([]string, error) {
	db, err := DatabaseFromContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("GetDescriptions: get DB from context: %w", err)
	}

	rows, err := db.QueryContext(ctx, `
        SELECT description
          FROM descriptions_global
         ORDER BY id
    `)
	if err != nil {
		return nil, fmt.Errorf("GetDescriptions: query: %w", err)
	}
	defer rows.Close()

	var out []string
	for rows.Next() {
		var desc string
		if err := rows.Scan(&desc); err != nil {
			return nil, fmt.Errorf("GetDescriptions: scan row: %w", err)
		}
		out = append(out, desc)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("GetDescriptions: iterate: %w", err)
	}
	return out, nil
}

// APIInfo represents an API for HTTP communication with the websocket server
type APIInfo struct {
	APIName   string   `json:"api_name"`
	Documents []string `json:"documents,omitempty"`
	Policy    string   `json:"policy,omitempty"`
}

// TrackerDocuments represents the structure for tracker documents
type TrackerDocuments struct {
	Datasets  map[string]string `json:"datasets,omitempty"`
	Templates map[string]string `json:"templates,omitempty"`
}

// TrackerInfo represents a tracker for HTTP communication with the websocket server
type TrackerInfo struct {
	TrackerName        string           `json:"tracker_name"`
	TrackerDescription string           `json:"tracker_description,omitempty"`
	TrackerVersion     string           `json:"tracker_version,omitempty"`
	TrackerDocuments   TrackerDocuments `json:"tracker_documents,omitempty"`
}

// TrackerData represents the data stored for a single tracker
type TrackerData struct {
	TrackerDescription string          `json:"tracker_description,omitempty"`
	TrackerVersion     string          `json:"tracker_version,omitempty"`
	TrackerDocuments   TrackerDocuments `json:"tracker_documents,omitempty"`
}

// TrackerListPayload represents a list of trackers for a user
// Used in the new tracker registration endpoint format
type TrackerListPayload struct {
	UserID   string                 `json:"user_id"`
	Trackers map[string]TrackerData `json:"trackers"` // Map of tracker name to tracker data
}

// RegisterAPI sends an API registration request to the websocket server
func RegisterAPI(ctx context.Context, api APIInfo) error {
	params, err := ParamsFromContext(ctx)
	if err != nil {
		return fmt.Errorf("RegisterAPI: failed to get params from context: %w", err)
	}

	if params.ServerURL == nil || *params.ServerURL == "" {
		return fmt.Errorf("RegisterAPI: server URL is not configured")
	}

	url := fmt.Sprintf("%s/user/apis", *params.ServerURL)
	
	// Get the client from context
	dk, err := DkFromContext(ctx)
	if err != nil {
		return fmt.Errorf("RegisterAPI: failed to get DK client from context: %w", err)
	}

	// Access the JWT token directly from the client
	token := dk.Token()
	if token == "" {
		return fmt.Errorf("RegisterAPI: no JWT token available, please login first")
	}

	// Prepare the request body
	jsonBody, err := json.Marshal(api)
	if err != nil {
		return fmt.Errorf("RegisterAPI: error marshaling API data: %w", err)
	}

	// Create the request
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return fmt.Errorf("RegisterAPI: error creating request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	// Send the request
	client := &http.Client{Timeout: time.Second * 30}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("RegisterAPI: error sending request: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		var errorResponse struct {
			Error string `json:"error"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&errorResponse); err == nil && errorResponse.Error != "" {
			return fmt.Errorf("RegisterAPI: server returned error: %s", errorResponse.Error)
		}
		return fmt.Errorf("RegisterAPI: server returned status code %d", resp.StatusCode)
	}

	return nil
}

// RegisterTracker sends a tracker registration request to the websocket server
// This function maintains backward compatibility with the old format
func RegisterTracker(ctx context.Context, tracker TrackerInfo) error {
	params, err := ParamsFromContext(ctx)
	if err != nil {
		return fmt.Errorf("RegisterTracker: failed to get params from context: %w", err)
	}

	if params.ServerURL == nil || *params.ServerURL == "" {
		return fmt.Errorf("RegisterTracker: server URL is not configured")
	}

	url := fmt.Sprintf("%s/user/trackers", *params.ServerURL)
	
	// Get the client from context
	dk, err := DkFromContext(ctx)
	if err != nil {
		return fmt.Errorf("RegisterTracker: failed to get DK client from context: %w", err)
	}

	// Access the JWT token directly from the client
	token := dk.Token()
	if token == "" {
		return fmt.Errorf("RegisterTracker: no JWT token available, please login first")
	}

	// Prepare the request body
	jsonBody, err := json.Marshal(tracker)
	if err != nil {
		return fmt.Errorf("RegisterTracker: error marshaling tracker data: %w", err)
	}

	// Create the request
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return fmt.Errorf("RegisterTracker: error creating request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	// Send the request
	client := &http.Client{Timeout: time.Second * 30}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("RegisterTracker: error sending request: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		var errorResponse struct {
			Error string `json:"error"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&errorResponse); err == nil && errorResponse.Error != "" {
			return fmt.Errorf("RegisterTracker: server returned error: %s", errorResponse.Error)
		}
		return fmt.Errorf("RegisterTracker: server returned status code %d", resp.StatusCode)
	}

	return nil
}

// RegisterTrackerList sends an updated tracker list to the websocket server
// This uses the new format where the entire tracker list is sent at once
func RegisterTrackerList(ctx context.Context, trackerList TrackerListPayload) error {
	params, err := ParamsFromContext(ctx)
	if err != nil {
		return fmt.Errorf("RegisterTrackerList: failed to get params from context: %w", err)
	}

	if params.ServerURL == nil || *params.ServerURL == "" {
		return fmt.Errorf("RegisterTrackerList: server URL is not configured")
	}

	url := fmt.Sprintf("%s/user/trackers", *params.ServerURL)
	
	// Get the client from context
	dk, err := DkFromContext(ctx)
	if err != nil {
		return fmt.Errorf("RegisterTrackerList: failed to get DK client from context: %w", err)
	}

	// Access the JWT token directly from the client
	token := dk.Token()
	if token == "" {
		return fmt.Errorf("RegisterTrackerList: no JWT token available, please login first")
	}

	// Prepare the request body
	jsonBody, err := json.Marshal(trackerList)
	if err != nil {
		return fmt.Errorf("RegisterTrackerList: error marshaling tracker list data: %w", err)
	}

	// Create the request
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return fmt.Errorf("RegisterTrackerList: error creating request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	// Send the request
	client := &http.Client{Timeout: time.Second * 30}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("RegisterTrackerList: error sending request: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		var errorResponse struct {
			Error string `json:"error"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&errorResponse); err == nil && errorResponse.Error != "" {
			return fmt.Errorf("RegisterTrackerList: server returned error: %s", errorResponse.Error)
		}
		return fmt.Errorf("RegisterTrackerList: server returned status code %d", resp.StatusCode)
	}

	return nil
}
