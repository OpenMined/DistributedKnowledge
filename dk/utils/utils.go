package utils

import (
	"context"
	"crypto/ed25519"
	"dk/client"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/philippgille/chromem-go"
	"io/fs"
	"os"
	"path/filepath"
)

type Parameters struct {
	PrivateKeyPath        *string
	PublicKeyPath         *string
	UserID                *string
	QueriesFile           *string
	AnswersFile           *string
	AutomaticApprovalFile *string
	VectorDBPath          *string
	RagSourcesFile        *string
	ModelConfigFile       *string
	ServerURL             *string
	DescriptionSourceFile *string
	HTTPPort              *string
	SyftboxConfig         *string
}

type RemoteMessage struct {
	Type    string            `json:"type"`
	Message string            `json:"message,omitempty"`
	Files   map[string]string `json:"files,omitempty"`
}

type AnswerMessage struct {
	Answer string `json:"answer"`
	From   string `json:"from"`
	Query  string `json:"query"`
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

// UpdateDescriptions overwrites (or creates) descriptions.json with the
// supplied data. The file is written atomically:
//
//  1. Marshal the map to JSON in memory.
//  2. Write to a *.tmp file in the same directory.
//  3. Rename the tmp file onto descriptions.json.
//
// This guarantees that callers never see a partially‑written file.
func UpdateDescriptions(ctx context.Context, data []string) error {
	params, err := ParamsFromContext(ctx)
	if err != nil {
		return err
	}

	if data == nil {
		return errors.New("UpdateDescriptions: nil input")
	}

	// JSON encode with indentation for human readability.
	blob, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal JSON: %w", err)
	}

	dir := filepath.Dir(*params.DescriptionSourceFile)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create directory %q: %w", dir, err)
	}

	tmp := *params.DescriptionSourceFile + ".tmp"
	if err := os.WriteFile(tmp, blob, 0o644); err != nil {
		return fmt.Errorf("write tmp file: %w", err)
	}

	// Atomic replace (best effort on the current OS).
	if err := os.Rename(tmp, *params.DescriptionSourceFile); err != nil {
		_ = os.Remove(tmp) // clean up on failure
		return fmt.Errorf("rename tmp file: %w", err)
	}
	return nil
}

// GetDescriptions loads descriptions.json and returns its contents.
// If the file is absent, an empty map and nil error are returned.
func GetDescriptions(ctx context.Context) ([]string, error) {
	out := []string{}

	params, err := ParamsFromContext(ctx)
	if err != nil {
		return nil, err
	}

	b, err := os.ReadFile(*params.DescriptionSourceFile)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return out, nil // empty slice, no error
		}
		return nil, fmt.Errorf("read file: %w", err)
	}

	if len(b) == 0 {
		return out, nil // empty file → empty slice
	}

	if err := json.Unmarshal(b, &out); err != nil {
		return nil, fmt.Errorf("unmarshal JSON: %w", err)
	}
	return out, nil
}
