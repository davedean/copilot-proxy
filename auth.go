package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
)

var (
	clientID = "Iv1.b507a08c87ecfe98"
)

func requestDeviceCode() (DeviceCodeResponse, error) {
	body := map[string]string{
		"client_id": clientID,
		"scope":     "read:user",
	}
	b, _ := json.Marshal(body)
	req, _ := http.NewRequest("POST", "https://github.com/login/device/code", bytes.NewReader(b))
	req.Header.Set("accept", "application/json")
	req.Header.Set("content-type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return DeviceCodeResponse{}, err
	}
	defer resp.Body.Close()
	var dc DeviceCodeResponse
	if err := json.NewDecoder(resp.Body).Decode(&dc); err != nil {
		return DeviceCodeResponse{}, err
	}
	return dc, nil
}

func pollAccessToken(deviceCode string) (AccessTokenResponse, error) {
	log.Println("Polling access token")
	body := map[string]string{
		"client_id":   clientID,
		"device_code": deviceCode,
		"grant_type":  "urn:ietf:params:oauth:grant-type:device_code",
	}
	b, _ := json.Marshal(body)
	req, _ := http.NewRequest("POST", "https://github.com/login/oauth/access_token", bytes.NewReader(b))
	req.Header.Set("accept", "application/json")
	req.Header.Set("content-type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return AccessTokenResponse{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		log.Printf("%d: Failed to get access token: %s", resp.StatusCode, string(body))
		return AccessTokenResponse{}, errors.New("Failed to get access token")
	}
	var at AccessTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&at); err != nil {
		return AccessTokenResponse{}, err
	}
	if at.AccessToken == "" {
		log.Println("No access token found")
		return AccessTokenResponse{}, errors.New("no access token")
	}
	log.Printf("Got access token, %s", at.AccessToken)
	return at, nil
}

func fetchCopilotToken(accessToken string) (CopilotToken, error) {
	req, err := http.NewRequest("GET", "https://api.github.com/copilot_internal/v2/token", nil)
	if err != nil {
		log.Printf("Failed to create request: %v", err)
		return CopilotToken{}, err
	}
	req.Header.Set("authorization", "token "+accessToken)
	req.Header.Set("user-agent", "GitHubCopilotChat/0.12.2023120701")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("HTTP request failed: %v", err)
		return CopilotToken{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		var errResp struct {
			Message string `json:"message"`
		}
		if decodeErr := json.NewDecoder(resp.Body).Decode(&errResp); decodeErr != nil {
			log.Printf("Failed to decode error response: %v", decodeErr)
		}
		if errResp.Message != "" {
			log.Printf("API error: %s", errResp.Message)
			return CopilotToken{}, errors.New(errResp.Message)
		}
		log.Printf("Failed to get copilot token: status %d", resp.StatusCode)
		return CopilotToken{}, errors.New("failed to get copilot token")
	}
	var ct CopilotToken
	if err := json.NewDecoder(resp.Body).Decode(&ct); err != nil {
		log.Printf("Failed to decode copilot token: %v", err)
		return CopilotToken{}, err
	}
	return ct, nil
}
