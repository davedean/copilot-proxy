package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"time"
)

// TokenStorage represents the persistent storage for tokens
type TokenStorage struct {
	Tokens map[string]CopilotToken `json:"tokens"`
}

// SaveTokensToFile saves the current token cache to a file
func SaveTokensToFile(filename string, tokens map[string]CopilotToken) error {
	// Create directory if it doesn't exist
	dir := filepath.Dir(filename)
	if err := os.MkdirAll(dir, 0755); err != nil {
		log.Printf("Failed to create directory for token storage: %v", err)
		return err
	}

	// Create storage object
	storage := TokenStorage{
		Tokens: make(map[string]CopilotToken),
	}

	// Only save non-expired tokens
	now := time.Now().Unix()
	for k, v := range tokens {
		if v.Expiry > now {
			storage.Tokens[k] = v
		}
	}

	// Marshal to JSON
	data, err := json.MarshalIndent(storage, "", "  ")
	if err != nil {
		log.Printf("Failed to marshal tokens to JSON: %v", err)
		return err
	}

	// Write to file with secure permissions
	err = ioutil.WriteFile(filename, data, 0600)
	if err != nil {
		log.Printf("Failed to write tokens to file: %v", err)
		return err
	}

	log.Printf("Saved %d tokens to %s", len(storage.Tokens), filename)
	return nil
}

// LoadTokensFromFile loads tokens from a file into the provided map
func LoadTokensFromFile(filename string) (map[string]CopilotToken, error) {
	// Check if file exists
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		log.Printf("Token file %s does not exist, starting with empty cache", filename)
		return make(map[string]CopilotToken), nil
	}

	// Read file
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		log.Printf("Failed to read token file: %v", err)
		return make(map[string]CopilotToken), err
	}

	// Unmarshal JSON
	var storage TokenStorage
	if err := json.Unmarshal(data, &storage); err != nil {
		log.Printf("Failed to unmarshal tokens from JSON: %v", err)
		return make(map[string]CopilotToken), err
	}

	// Filter out expired tokens
	tokens := make(map[string]CopilotToken)
	now := time.Now().Unix()
	for k, v := range storage.Tokens {
		if v.Expiry > now {
			tokens[k] = v
		}
	}

	log.Printf("Loaded %d valid tokens from %s", len(tokens), filename)
	return tokens, nil
}
