package main

import (
	"bytes"
	"encoding/json"
	"errors"
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
	var at AccessTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&at); err != nil {
		return AccessTokenResponse{}, err
	}
	if at.AccessToken == "" {
		return AccessTokenResponse{}, errors.New("no access token")
	}
	return at, nil
}

func fetchCopilotToken(accessToken string) (CopilotToken, error) {
	req, _ := http.NewRequest("GET", "https://api.github.com/copilot_internal/v2/token", nil)
	req.Header.Set("authorization", "token "+accessToken)
	req.Header.Set("user-agent", "GitHubCopilotChat/0.12.2023120701")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return CopilotToken{}, err
	}
	defer resp.Body.Close()
	var ct CopilotToken
	if err := json.NewDecoder(resp.Body).Decode(&ct); err != nil {
		return CopilotToken{}, err
	}
	return ct, nil
}
