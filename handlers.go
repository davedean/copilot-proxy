package main

import (
	"bytes"
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
)

//go:embed public/*
var content embed.FS

func handleLogin(w http.ResponseWriter, r *http.Request) {
	// Check if we have any valid tokens in the cache
	validToken := false
	var accessToken string

	// Iterate through the token cache to find a valid token
	tokenCache.mu.Lock()
	for key, token := range tokenCache.cache {
		// Check if the token is still valid
		if time.Until(time.Unix(token.Expiry, 0)) > tokenExpiryBuffer {
			validToken = true
			accessToken = key
			break
		}
	}
	tokenCache.mu.Unlock()

	if validToken {
		// We have a valid token, return it directly
		log.Println("Using existing valid token from cache")
		json.NewEncoder(w).Encode(map[string]string{
			"access_token": accessToken,
			"cached":       "true",
		})
		return
	}

	// Check if we have a valid cached device code
	if IsDeviceCodeValid(deviceCodeStorage) {
		log.Println("Using cached device code to get a fresh access token")
		at, err := pollAccessToken(deviceCodeStorage.DeviceCode)
		if err == nil && at.AccessToken != "" {
			log.Println("Successfully obtained access token using cached device code")

			// Get a fresh Copilot token
			ct, err := fetchCopilotToken(at.AccessToken)
			if err == nil {
				// Store the token in the cache in a separate goroutine to avoid blocking
				go func() {
					tokenCache.Set(at.AccessToken, ct)
				}()

				// Return the access token to the client
				json.NewEncoder(w).Encode(map[string]string{
					"access_token": at.AccessToken,
					"cached":       "device_code",
				})
				return
			} else {
				log.Printf("Failed to fetch Copilot token: %v", err)
			}
		} else {
			log.Printf("Failed to get access token using cached device code: %v", err)
		}
	}

	// No valid token or device code found, proceed with device code flow
	log.Println("No valid token or device code found, requesting new device code")
	dc, err := requestDeviceCode()
	if err != nil {
		http.Error(w, "Failed to get device code", http.StatusInternalServerError)
		return
	}

	// Store the device code in the global variable
	deviceCodeStorage = DeviceCodeStorage{
		DeviceCode:      dc.DeviceCode,
		UserCode:        dc.UserCode,
		VerificationURI: dc.VerificationURI,
		Interval:        dc.Interval,
		CreatedAt:       time.Now().Unix(),
	}

	json.NewEncoder(w).Encode(dc)
}

func handleWebsocketPoll(w http.ResponseWriter, r *http.Request) {
	log.Println("Got websocket connection")
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	var req struct {
		DeviceCode string `json:"device_code"`
		Interval   int    `json:"interval"`
	}
	if err := conn.ReadJSON(&req); err != nil {
		return
	}
	log.Println(req.DeviceCode, req.Interval)

	// Update the device code storage with the latest information
	// This ensures we have the most recent device code for future use
	deviceCodeStorage.DeviceCode = req.DeviceCode
	deviceCodeStorage.Interval = req.Interval
	deviceCodeStorage.CreatedAt = time.Now().Unix()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			conn.WriteJSON(map[string]string{"error": "timeout"})
			return
		case <-ticker.C:
			at, err := pollAccessToken(req.DeviceCode)
			if err == nil && at.AccessToken != "" {
				conn.WriteJSON(map[string]string{"access_token": at.AccessToken})

				// Save the device code to file for future use
				if deviceCodeFile := defaultDeviceCodeFile; deviceCodeFile != "" {
					go func() {
						if err := SaveDeviceCodeToFile(deviceCodeFile, deviceCodeStorage); err != nil {
							log.Printf("Failed to save device code to file: %v", err)
						}
					}()
				}

				// Get a fresh Copilot token
				ct, err := fetchCopilotToken(at.AccessToken)
				if err == nil {
					// Store the token in the cache in a separate goroutine to avoid blocking
					go func(accessToken string, copilotToken CopilotToken) {
						tokenCache.Set(accessToken, copilotToken)
					}(at.AccessToken, ct)
				}
				return
			}
		}
	}
}

func handleGitHubProxy(w http.ResponseWriter, r *http.Request) {
	log.Println("Forwarding GitHub Copilot Request")
	auth := r.Header.Get("Authorization")
	if len(auth) < 8 || auth[:7] != "Bearer " {
		log.Println("403: Missing Authoirzation header")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	accessToken := auth[7:]
	ct, ok := tokenCache.Get(accessToken)
	if !ok {
		log.Println("Token not in cache... Fetching")
		var err error
		ct, err = fetchCopilotToken(accessToken)
		if err != nil {
			log.Println("500: Failed to fetch copilot token")
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}
		// Store the token in the cache in a separate goroutine to avoid blocking
		go func(accessToken string, copilotToken CopilotToken) {
			tokenCache.Set(accessToken, copilotToken)
		}(accessToken, ct)
	}

	// Forward the request to the Copilot API
	req, err := http.NewRequest(r.Method, fmt.Sprintf("https://api.githubcopilot.com%s", r.URL.Path), r.Body)
	if err != nil {
		http.Error(w, "Failed to create request", http.StatusInternalServerError)
		return
	}
	// Copy all headers except Host and Authorization
	for k, v := range r.Header {
		if k == "Host" || k == "Authorization" {
			continue
		}
		for _, vv := range v {
			req.Header.Add(k, vv)
		}
	}
	req.Header.Set("Authorization", "Bearer "+ct.Token)
	req.Header.Set("x-request-id", uuid.New().String())
	req.Header.Set("vscode-sessionid", r.Header.Get("vscode-sessionid"))
	req.Header.Set("machineid", r.Header.Get("machineid"))
	req.Header.Set("editor-version", "vscode/1.85.1")
	req.Header.Set("editor-plugin-version", "copilot-chat/0.12.2023120701")
	req.Header.Set("openai-organization", "github-copilot")
	req.Header.Set("openai-intent", "conversation-panel")
	req.Header.Set("content-type", "application/json")
	req.Header.Set("user-agent", "GitHubCopilotChat/0.12.2023120701")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		http.Error(w, "Upstream error", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	// Copy all headers
	for k, v := range resp.Header {
		for _, vv := range v {
			w.Header().Add(k, vv)
		}
	}
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
	log.Println("Copilot Request Completed")
}

func handleIndex(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	if path == "/" {
		path = "/index.html"
	}
	data, err := content.ReadFile("public" + path)
	if err != nil {
		log.Printf("404 Not Found: %s", r.URL.Path)
		http.NotFound(w, r)
		return
	}
	http.ServeContent(w, r, path, time.Now(), bytes.NewReader(data))
}
