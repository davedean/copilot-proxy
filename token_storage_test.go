package main

import (
	"os"
	"testing"
	"time"
)

func TestTokenStorage(t *testing.T) {
	// Create a temporary file for testing
	tempFile := "test_tokens.json"
	defer os.Remove(tempFile) // Clean up after test

	// Create some test tokens
	tokens := map[string]CopilotToken{
		"token1": {
			Token:  "test-copilot-token-1",
			Expiry: time.Now().Add(1 * time.Hour).Unix(),
		},
		"token2": {
			Token:  "test-copilot-token-2",
			Expiry: time.Now().Add(2 * time.Hour).Unix(),
		},
		// Add an expired token to test filtering
		"expired": {
			Token:  "expired-token",
			Expiry: time.Now().Add(-1 * time.Hour).Unix(),
		},
	}

	// Test saving tokens
	err := SaveTokensToFile(tempFile, tokens)
	if err != nil {
		t.Fatalf("Failed to save tokens: %v", err)
	}

	// Test loading tokens
	loadedTokens, err := LoadTokensFromFile(tempFile)
	if err != nil {
		t.Fatalf("Failed to load tokens: %v", err)
	}

	// Verify that we have the expected number of tokens (expired token should be filtered)
	if len(loadedTokens) != 2 {
		t.Errorf("Expected 2 tokens, got %d", len(loadedTokens))
	}

	// Verify token content
	for key, expectedToken := range tokens {
		if key == "expired" {
			// Skip expired token, it should not be loaded
			continue
		}

		loadedToken, exists := loadedTokens[key]
		if !exists {
			t.Errorf("Token %s not found in loaded tokens", key)
			continue
		}

		if loadedToken.Token != expectedToken.Token {
			t.Errorf("Token mismatch for %s: expected %s, got %s",
				key, expectedToken.Token, loadedToken.Token)
		}

		if loadedToken.Expiry != expectedToken.Expiry {
			t.Errorf("Expiry mismatch for %s: expected %d, got %d",
				key, expectedToken.Expiry, loadedToken.Expiry)
		}
	}
}

func TestTokenCachePersistence(t *testing.T) {
	// Create a temporary file for testing
	tempFile := "test_cache.json"
	defer os.Remove(tempFile) // Clean up after test

	// Create a token cache with the temp file
	cache := NewTokenCache(tempFile)

	// Add a token
	token := CopilotToken{
		Token:  "test-token",
		Expiry: time.Now().Add(1 * time.Hour).Unix(),
	}
	cache.Set("test-key", token)

	// Wait a moment for the goroutine to complete saving the file
	time.Sleep(100 * time.Millisecond)

	// Create a new cache instance that should load from the file
	newCache := NewTokenCache(tempFile)

	// Check if the token was loaded
	loadedToken, exists := newCache.Get("test-key")
	if !exists {
		t.Fatal("Token not found in new cache instance")
	}

	if loadedToken.Token != token.Token {
		t.Errorf("Token mismatch: expected %s, got %s", token.Token, loadedToken.Token)
	}

	if loadedToken.Expiry != token.Expiry {
		t.Errorf("Expiry mismatch: expected %d, got %d", token.Expiry, loadedToken.Expiry)
	}
}
