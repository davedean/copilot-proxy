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

	// Make a copy of the cache to avoid holding the lock during file I/O
	cacheCopy := make(map[string]CopilotToken)
	for k, v := range tc.cache {
		cacheCopy[k] = v
	}

	// Get the filename before unlocking
	filename := tc.filename
	tc.mu.Unlock()

	// Save to file if filename is set
	if filename != "" {
		go func() {
			if err := SaveTokensToFile(filename, cacheCopy); err != nil {
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

	// Make a copy of the cache to avoid holding the lock during file I/O
	cacheCopy := make(map[string]CopilotToken)

	for k, v := range tc.cache {
		if v.Expiry <= now {
			delete(tc.cache, k)
			tokensRemoved = true
		} else {
			cacheCopy[k] = v
		}
	}

	// Get the filename before unlocking
	filename := tc.filename
	tc.mu.Unlock()

	// If tokens were removed and we have a filename, save the updated cache
	if tokensRemoved && filename != "" {
		go func() {
			if err := SaveTokensToFile(filename, cacheCopy); err != nil {
				log.Printf("Failed to save tokens to file after cleanup: %v", err)
			}
		}()
	}

	tc.scheduleCleanup()
}

var upgrader = websocket.Upgrader{}
var tokenCache *TokenCache

// Default storage file paths
const defaultTokenFile = "copilot_tokens.json"
const defaultDeviceCodeFile = "device_code.json"

// Global variable for device code storage
var deviceCodeStorage DeviceCodeStorage

func main() {
	// Ensure logs go to stderr
	log.SetOutput(os.Stderr)

	listenAddr := "127.0.0.1:8080"
	tokenFile := defaultTokenFile
	deviceCodeFile := defaultDeviceCodeFile
	tokenOnly := false

	if len(os.Args) > 1 {
		for i, arg := range os.Args {
			if arg == "-listen" && i+1 < len(os.Args) {
				listenAddr = os.Args[i+1]
			}
			if arg == "-token-file" && i+1 < len(os.Args) {
				tokenFile = os.Args[i+1]
			}
			if arg == "-device-code-file" && i+1 < len(os.Args) {
				deviceCodeFile = os.Args[i+1]
			}
			if arg == "-token-only" {
				tokenOnly = true
				log.Println("Running in CLI-only mode. No web server will be started.")
			}
		}
	}

	// Initialize token cache with persistent storage (for current session only)
	tokenCache = NewTokenCache(tokenFile)
	log.Printf("Using token storage file: %s", tokenFile)

	// Load device code from file if it exists
	var err error
	deviceCodeStorage, err = LoadDeviceCodeFromFile(deviceCodeFile)
	if err == nil && IsDeviceCodeValid(deviceCodeStorage) {
		log.Printf("Loaded valid device code from %s", deviceCodeFile)
	} else {
		log.Printf("No valid device code found or error loading from %s", deviceCodeFile)
		deviceCodeStorage = DeviceCodeStorage{}
	}

	// Check if we have any valid tokens in the cache and output them
	validTokenFound := false
	var validAccessToken string

	tokenCache.mu.Lock()
	for accessToken, token := range tokenCache.cache {
		// Check if the token is still valid
		if time.Until(time.Unix(token.Expiry, 0)) > tokenExpiryBuffer {
			validTokenFound = true
			validAccessToken = accessToken
			fmt.Printf("%s\n", accessToken)
			log.Printf("Found valid access token: %s", accessToken)
			log.Printf("You can use this as 'Authorization: Bearer %s' for /chat/completions", accessToken)

			// Try to fetch a Copilot token to verify it's still valid
			ct, err := fetchCopilotToken(accessToken)
			if err == nil {
				log.Printf("Verified token is valid with GitHub Copilot API")

				if !tokenOnly {
					// Update the token in the cache with the latest expiry in a separate goroutine
					go func(accessToken string, copilotToken CopilotToken) {
						tokenCache.Set(accessToken, copilotToken)
					}(accessToken, ct)
				}
			} else {
				log.Printf("Warning: Could not verify token with GitHub Copilot API: %v", err)
			}
		}
	}
	tokenCache.mu.Unlock()

	// If we have a valid token but no valid device code, try to obtain a device code
	if validTokenFound && !IsDeviceCodeValid(deviceCodeStorage) && !tokenOnly {
		log.Println("Valid token found but no valid device code. Attempting to obtain device code.")

		// Request a new device code
		dc, err := requestDeviceCode()
		if err == nil {
			// Store the device code
			deviceCodeStorage = DeviceCodeStorage{
				DeviceCode:      dc.DeviceCode,
				UserCode:        dc.UserCode,
				VerificationURI: dc.VerificationURI,
				Interval:        dc.Interval,
				CreatedAt:       time.Now().Unix(),
			}

			// Save the device code to file
			if deviceCodeFile != "" {
				if err := SaveDeviceCodeToFile(deviceCodeFile, deviceCodeStorage); err != nil {
					log.Printf("Failed to save device code to file: %v", err)
				} else {
					log.Printf("Saved device code to %s for future use", deviceCodeFile)
				}
			}

			log.Printf("Device code obtained. User code: %s, Verification URI: %s",
				dc.UserCode, dc.VerificationURI)
			log.Printf("Please visit %s and enter code %s to authenticate this device",
				dc.VerificationURI, dc.UserCode)

			// Start polling for access token in the background
			go func() {
				ticker := time.NewTicker(time.Duration(dc.Interval) * time.Second)
				defer ticker.Stop()

				timeout := time.After(10 * time.Minute)

				for {
					select {
					case <-timeout:
						log.Println("Timed out waiting for device code authentication")
						return
					case <-ticker.C:
						at, err := pollAccessToken(dc.DeviceCode)
						if err == nil && at.AccessToken != "" {
							log.Println("Successfully authenticated with device code")

							// If the access token is different from the one we already have,
							// fetch a new Copilot token and store it
							if at.AccessToken != validAccessToken {
								ct, err := fetchCopilotToken(at.AccessToken)
								if err == nil {
									// Store the token in the cache in a separate goroutine
									go func(accessToken string, copilotToken CopilotToken) {
										tokenCache.Set(accessToken, copilotToken)
										log.Printf("New access token obtained and stored")
									}(at.AccessToken, ct)
								}
							}

							return
						}
					}
				}
			}()
		} else {
			log.Printf("Failed to request device code: %v", err)
		}
	}

	if !validTokenFound {
		log.Printf("No valid tokens found in cache. Please authenticate via the web interface.")

		if tokenOnly {
			log.Printf("To authenticate, run without the -token-only flag and visit http://%s", listenAddr)
			os.Exit(1)
		}
	}

	if tokenOnly {
		// In CLI-only mode, exit after displaying the token
		log.Printf("Token found and displayed. Exiting CLI-only mode.")
		os.Exit(0)
	} else {
		http.HandleFunc("/", handleIndex)
		http.HandleFunc("/login", handleLogin)
		http.HandleFunc("/ws/poll", handleWebsocketPoll)
		http.HandleFunc("/chat/completions", handleGitHubProxy)
		http.HandleFunc("/models", handleGitHubProxy)
		log.Printf("Listening at http://%s\n", listenAddr)
		log.Fatal(http.ListenAndServe(listenAddr, nil))
	}
}
