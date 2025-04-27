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
	Token  string `json:"token"`
	Expiry int64  `json:"expires_at"`
}
type TokenCache struct {
	mu       sync.Mutex
	cache    map[string]CopilotToken
	timer    *time.Timer
	stopChan chan struct{}
}

func NewTokenCache() *TokenCache {
	tc := &TokenCache{
		cache:    make(map[string]CopilotToken),
		stopChan: make(chan struct{}),
	}
	tc.scheduleCleanup()
	return tc
}

func (tc *TokenCache) Set(key string, token CopilotToken) {
	tc.mu.Lock()
	tc.cache[key] = token
	tc.mu.Unlock()
	tc.scheduleCleanup()
}

func (tc *TokenCache) Get(key string) (CopilotToken, bool) {
	tc.mu.Lock()
	defer tc.mu.Unlock()
	token, ok := tc.cache[key]
	if !ok || int64(token.Expiry) <= time.Now().Unix() {
		delete(tc.cache, key)
		return CopilotToken{}, false
	}
	return token, true
}

func (tc *TokenCache) scheduleCleanup() {
	tc.mu.Lock()
	defer tc.mu.Unlock()

	if tc.timer != nil {
		tc.timer.Stop()
	}

	var earliest int64 = 0
	for _, v := range tc.cache {
		if earliest == 0 || v.Expiry < earliest {
			earliest = v.Expiry
		}
	}

	if earliest == 0 {
		return
	}

	duration := time.Until(time.Unix(earliest, 0))
	if duration <= 0 {
		duration = time.Second
	}

	tc.timer = time.AfterFunc(duration, func() {
		tc.cleanup()
	})
}

func (tc *TokenCache) cleanup() {
	tc.mu.Lock()
	now := time.Now().Unix()
	for k, v := range tc.cache {
		if v.Expiry <= now {
			delete(tc.cache, k)
		}
	}
	tc.mu.Unlock()
	tc.scheduleCleanup()
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
