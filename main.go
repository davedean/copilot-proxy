package main

import (
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type DeviceCodeResponse struct {
	DeviceCode      string `json:"device_code"`
	UserCode        string `json:"user_code"`
	VerificationURI string `json:"verification_uri"`
	Interval        int    `json:"interval"`
}

type AccessTokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
}

type CopilotToken struct {
	Token  string    `json:"token"`
	Expiry time.Time `json:"expiry"`
}

type TokenCache struct {
	mu    sync.Mutex
	cache map[string]CopilotToken
}

func NewTokenCache() *TokenCache {
	tc := &TokenCache{cache: make(map[string]CopilotToken)}
	go tc.cleanup()
	return tc
}

func (tc *TokenCache) Set(key string, token CopilotToken) {
	tc.mu.Lock()
	defer tc.mu.Unlock()
	tc.cache[key] = token
}

func (tc *TokenCache) Get(key string) (CopilotToken, bool) {
	tc.mu.Lock()
	defer tc.mu.Unlock()
	token, ok := tc.cache[key]
	if !ok || token.Expiry.Before(time.Now()) {
		delete(tc.cache, key)
		return CopilotToken{}, false
	}
	return token, true
}

func (tc *TokenCache) cleanup() {
	for {
		time.Sleep(1 * time.Minute)
		tc.mu.Lock()
		now := time.Now()
		for k, v := range tc.cache {
			if v.Expiry.Before(now) {
				delete(tc.cache, k)
			}
		}
		tc.mu.Unlock()
	}
}

var upgrader = websocket.Upgrader{}
var tokenCache = NewTokenCache()

func main() {
	http.HandleFunc("/", handleIndex)
	http.HandleFunc("/login", handleLogin)
	http.HandleFunc("/ws/poll", handleWebsocketPoll)
	http.HandleFunc("/chat/completions", handleChatCompletions)
	log.Println("Listening at http://127.0.0.1:8080")
	log.Fatal(http.ListenAndServe("127.0.0.1:8080", nil))
}
