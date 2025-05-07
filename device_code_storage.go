package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"time"
)

// DeviceCodeStorage represents the persistent storage for device code information
type DeviceCodeStorage struct {
	DeviceCode      string `json:"device_code"`
	UserCode        string `json:"user_code"`
	VerificationURI string `json:"verification_uri"`
	Interval        int    `json:"interval"`
	CreatedAt       int64  `json:"created_at"` // To track when it was created
}

// SaveDeviceCodeToFile saves the device code to a file
func SaveDeviceCodeToFile(filename string, deviceCode DeviceCodeStorage) error {
	// Create directory if it doesn't exist
	dir := filepath.Dir(filename)
	if err := os.MkdirAll(dir, 0755); err != nil {
		log.Printf("Failed to create directory for device code storage: %v", err)
		return err
	}

	// Marshal to JSON
	data, err := json.MarshalIndent(deviceCode, "", "  ")
	if err != nil {
		log.Printf("Failed to marshal device code to JSON: %v", err)
		return err
	}

	// Write to file with secure permissions
	err = ioutil.WriteFile(filename, data, 0600)
	if err != nil {
		log.Printf("Failed to write device code to file: %v", err)
		return err
	}

	log.Printf("Saved device code to %s", filename)
	return nil
}

// LoadDeviceCodeFromFile loads device code from a file
func LoadDeviceCodeFromFile(filename string) (DeviceCodeStorage, error) {
	// Check if file exists
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		log.Printf("Device code file %s does not exist", filename)
		return DeviceCodeStorage{}, err
	}

	// Read file
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		log.Printf("Failed to read device code file: %v", err)
		return DeviceCodeStorage{}, err
	}

	// Unmarshal JSON
	var deviceCode DeviceCodeStorage
	if err := json.Unmarshal(data, &deviceCode); err != nil {
		log.Printf("Failed to unmarshal device code from JSON: %v", err)
		return DeviceCodeStorage{}, err
	}

	log.Printf("Loaded device code from %s", filename)
	return deviceCode, nil
}

// IsDeviceCodeValid checks if the device code is still valid
func IsDeviceCodeValid(deviceCode DeviceCodeStorage) bool {
	// If we don't have a device code, it's not valid
	if deviceCode.DeviceCode == "" {
		return false
	}

	// For simplicity, we'll consider device codes valid for 30 days
	// This is a conservative estimate; in practice, they may be valid for longer
	const deviceCodeValidityDays = 30
	expiryTime := time.Unix(deviceCode.CreatedAt, 0).Add(time.Hour * 24 * deviceCodeValidityDays)

	return time.Now().Before(expiryTime)
}
