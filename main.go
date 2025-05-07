package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// tokenExpiryBuffer is how far before actual expiry we consider a CopilotToken expired to avoid races
const tokenExpiryBuffer = 10 * time.Second

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
	filename string
}

func NewTokenCache(filename string) *TokenCache {
	tc := &TokenCache{
		cache:    make(map[string]CopilotToken),
		stopChan: make(chan struct{}),
		filename: filename,
	}

	// Load tokens from file if it exists
	if filename != "" {
		loadedTokens, err := LoadTokensFromFile(filename)
		if err == nil && len(loadedTokens) > 0 {
			tc.cache = loadedTokens
			log.Printf("Loaded %d tokens from storage", len(loadedTokens))
		}
	}

	tc.scheduleCleanup()
	return tc
}

func (tc *TokenCache) Set(key string, token CopilotToken) {
	tc.mu.Lock()
	tc.cache[key] = token
	tc.mu.Unlock()

	// Save to file if filename is set
	if tc.filename != "" {
		go func() {
			tc.mu.Lock()
			defer tc.mu.Unlock()
			if err := SaveTokensToFile(tc.filename, tc.cache); err != nil {
				log.Printf("Failed to save tokens to file: %v", err)
			}
		}()
	}

	// schedule a cleanup taking into account our expiry buffer
	tc.scheduleCleanup()
}

func (tc *TokenCache) Get(key string) (CopilotToken, bool) {
	tc.mu.Lock()
	defer tc.mu.Unlock()
	token, ok := tc.cache[key]
	// treat tokens as expired tokenExpiryBuffer before their actual expiry to avoid races
	if !ok || time.Until(time.Unix(token.Expiry, 0)) <= tokenExpiryBuffer {
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

	var earliestEvict time.Time
	for _, v := range tc.cache {
		// schedule eviction tokenExpiryBuffer before actual expiry
		evict := time.Unix(v.Expiry, 0).Add(-tokenExpiryBuffer)
		if earliestEvict.IsZero() || evict.Before(earliestEvict) {
			earliestEvict = evict
		}
	}
	if earliestEvict.IsZero() {
		return
	}
	// compute duration until eviction time
	duration := time.Until(earliestEvict)
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
	tokensRemoved := false

	for k, v := range tc.cache {
		if v.Expiry <= now {
			delete(tc.cache, k)
			tokensRemoved = true
		}
	}
	tc.mu.Unlock()

	// If tokens were removed and we have a filename, save the updated cache
	if tokensRemoved && tc.filename != "" {
		go func() {
			tc.mu.Lock()
			defer tc.mu.Unlock()
			if err := SaveTokensToFile(tc.filename, tc.cache); err != nil {
				log.Printf("Failed to save tokens to file after cleanup: %v", err)
			}
		}()
	}

	tc.scheduleCleanup()
}

var upgrader = websocket.Upgrader{}
var tokenCache *TokenCache

// Default token storage file path
const defaultTokenFile = "copilot_tokens.json"

func main() {
	listenAddr := "127.0.0.1:8080"
	tokenFile := defaultTokenFile
	cliOnly := false

	if len(os.Args) > 1 {
		for i, arg := range os.Args {
			if arg == "-listen" && i+1 < len(os.Args) {
				listenAddr = os.Args[i+1]
			}
			if arg == "-token-file" && i+1 < len(os.Args) {
				tokenFile = os.Args[i+1]
			}
			if arg == "-cli-only" {
				cliOnly = true
				fmt.Println("Running in CLI-only mode. No web server will be started.")
			}
		}
	}

	// Initialize token cache with persistent storage
	tokenCache = NewTokenCache(tokenFile)
	log.Printf("Using token storage file: %s", tokenFile)

	// Check if we have any valid tokens in the cache and output them
	validTokenFound := false
	tokenCache.mu.Lock()
	for accessToken, token := range tokenCache.cache {
		// Check if the token is still valid
		if time.Until(time.Unix(token.Expiry, 0)) > tokenExpiryBuffer {
			validTokenFound = true
			log.Printf("Found valid access token: %s", accessToken)
			log.Printf("You can use this as 'Authorization: Bearer %s' for /chat/completions", accessToken)

			// Try to fetch a Copilot token to verify it's still valid
			ct, err := fetchCopilotToken(accessToken)
			if err == nil {
				log.Printf("Verified token is valid with GitHub Copilot API")

				if !cliOnly {
					// Update the token in the cache with the latest expiry
					tokenCache.Set(accessToken, ct)
				}
			} else {
				log.Printf("Warning: Could not verify token with GitHub Copilot API: %v", err)
			}
		}
		fmt.Println("HONK")
		break
	}
	tokenCache.mu.Unlock()

	if !validTokenFound {
		log.Printf("No valid tokens found in cache. Please authenticate via the web interface.")

		if cliOnly {
			log.Printf("To authenticate, run without the -cli-only flag and visit http://%s", listenAddr)
			os.Exit(1)
		}
	}

	if cliOnly {
		// In CLI-only mode, exit after displaying the token
		log.Printf("Token found and displayed. Exiting CLI-only mode.")
		os.Exit(0)
	}

	// Only start the web server if not in CLI-only mode or if no valid tokens were found
	http.HandleFunc("/", handleIndex)
	http.HandleFunc("/login", handleLogin)
	http.HandleFunc("/ws/poll", handleWebsocketPoll)
	http.HandleFunc("/chat/completions", handleGitHubProxy)
	http.HandleFunc("/models", handleGitHubProxy)
	log.Printf("Listening at http://%s\n", listenAddr)
	log.Fatal(http.ListenAndServe(listenAddr, nil))

}
