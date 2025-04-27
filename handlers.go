package main

import (
	"bytes"
	"context"
	"embed"
	"encoding/json"
	"net/http"
	"time"
)

//go:embed public/*
var content embed.FS

func handleLogin(w http.ResponseWriter, r *http.Request) {
	dc, err := requestDeviceCode()
	if err != nil {
		http.Error(w, "Failed to get device code", http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(dc)
}

func handleWebsocketPoll(w http.ResponseWriter, r *http.Request) {
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

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	ticker := time.NewTicker(time.Duration(req.Interval) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			conn.WriteJSON(map[string]string{"error": "timeout"})
			return
		case <-ticker.C:
			at, err := pollAccessToken(req.DeviceCode)
			if err == nil && at.AccessToken != "" {
				ct, err := fetchCopilotToken(at.AccessToken)
				if err == nil {
					tokenCache.Set(at.AccessToken, ct)
					conn.WriteJSON(map[string]string{"access_token": at.AccessToken})
					return
				}
			}
		}
		if _, _, err := conn.NextReader(); err != nil {
			return
		}
	}
}

func handleChatCompletions(w http.ResponseWriter, r *http.Request) {
	auth := r.Header.Get("Authorization")
	if len(auth) < 8 || auth[:7] != "Bearer " {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	accessToken := auth[7:]
	ct, ok := tokenCache.Get(accessToken)
	if !ok {
		var err error
		ct, err = fetchCopilotToken(accessToken)
		if err != nil {
			http.Error(w, "Failed to fetch copilot token", http.StatusUnauthorized)
			return
		}
		tokenCache.Set(accessToken, ct)
	}
	// Forward request to Copilot API using ct.Token as needed
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ct)
}

func handleIndex(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	if path == "/" {
		path = "/index.html"
	}
	data, err := content.ReadFile("public" + path)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	http.ServeContent(w, r, path, time.Now(), bytes.NewReader(data))
}
