package utils

import (
	"context"
	"crypto/ed25519"
	"dk/client"
	"encoding/hex"
	"fmt"
	"github.com/philippgille/chromem-go"
	"os"
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
}

type RemoteMessage struct {
	Type    string `json:"type"`
	Message string `json:"message"`
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
