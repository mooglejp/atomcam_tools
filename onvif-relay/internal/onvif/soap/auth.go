package soap

import (
	"crypto/sha1"
	"crypto/subtle"
	"encoding/base64"
	"encoding/xml"
	"fmt"
	"log"
	"sync"
	"time"
)

var (
	// usedNonces tracks used nonces to prevent replay attacks
	usedNonces sync.Map
	// nonceTTL defines how long nonces are remembered
	nonceTTL = 5 * time.Minute
	// cleanupDone signals the cleanup goroutine to stop
	cleanupDone = make(chan struct{})
	// cleanupOnce ensures cleanupDone is only closed once
	cleanupOnce sync.Once
)

func init() {
	// Start nonce cleanup goroutine
	go cleanupNonces()
}

// StopCleanup stops the nonce cleanup goroutine (call during shutdown)
func StopCleanup() {
	cleanupOnce.Do(func() {
		close(cleanupDone)
	})
}

// Security represents WS-Security header
type Security struct {
	XMLName          xml.Name         `xml:"http://docs.oasis-open.org/wss/2004/01/oasis-200401-wss-wssecurity-secext-1.0.xsd Security"`
	UsernameToken    UsernameToken    `xml:"UsernameToken"`
}

// UsernameToken represents WS-Security UsernameToken
type UsernameToken struct {
	Username string   `xml:"Username"`
	Password Password `xml:"Password"`
	Nonce    string   `xml:"Nonce,omitempty"`
	Created  string   `xml:"http://docs.oasis-open.org/wss/2004/01/oasis-200401-wss-wssecurity-utility-1.0.xsd Created,omitempty"`
}

// Password represents password with type attribute
type Password struct {
	Type  string `xml:"Type,attr"`
	Value string `xml:",chardata"`
}

// ValidateUsernameToken validates WS-UsernameToken authentication.
// security must be parsed from the full SOAP envelope to preserve namespace context.
func ValidateUsernameToken(security *Security, expectedUsername, expectedPassword string) error {
	if security == nil {
		return fmt.Errorf("missing security header")
	}

	// Extract username and password
	username := security.UsernameToken.Username
	password := security.UsernameToken.Password.Value
	nonce := security.UsernameToken.Nonce
	created := security.UsernameToken.Created

	// Check username using constant-time comparison
	if subtle.ConstantTimeCompare([]byte(username), []byte(expectedUsername)) != 1 {
		return fmt.Errorf("invalid credentials")
	}

	// Check password type
	if security.UsernameToken.Password.Type == "" {
		// Plain text password (not recommended)
		log.Printf("WARNING: Plain text password authentication used - consider using PasswordDigest for better security")
		if subtle.ConstantTimeCompare([]byte(password), []byte(expectedPassword)) != 1 {
			return fmt.Errorf("invalid credentials")
		}
	} else if security.UsernameToken.Password.Type == "http://docs.oasis-open.org/wss/2004/01/oasis-200401-wss-username-token-profile-1.0#PasswordDigest" {
		// Password digest
		if nonce == "" || created == "" {
			return fmt.Errorf("nonce and created are required for password digest")
		}

		// Check for nonce reuse (replay attack prevention)
		if nonce != "" {
			if _, exists := usedNonces.LoadOrStore(nonce, time.Now()); exists {
				return fmt.Errorf("nonce already used (replay attack detected)")
			}
		}

		// Validate timestamp (allow 5 minute window)
		createdTime, err := time.Parse(time.RFC3339, created)
		if err != nil {
			return fmt.Errorf("invalid created timestamp: %w", err)
		}

		now := time.Now().UTC()
		if createdTime.Before(now.Add(-5*time.Minute)) || createdTime.After(now.Add(5*time.Minute)) {
			return fmt.Errorf("timestamp out of valid range")
		}

		// Verify password digest
		// PasswordDigest = Base64(SHA1(nonce + created + password))
		nonceBytes, err := base64.StdEncoding.DecodeString(nonce)
		if err != nil {
			return fmt.Errorf("invalid nonce encoding: %w", err)
		}

		hash := sha1.New()
		hash.Write(nonceBytes)
		hash.Write([]byte(created))
		hash.Write([]byte(expectedPassword))
		expectedDigest := base64.StdEncoding.EncodeToString(hash.Sum(nil))

		if subtle.ConstantTimeCompare([]byte(password), []byte(expectedDigest)) != 1 {
			return fmt.Errorf("invalid credentials")
		}
	} else {
		return fmt.Errorf("unsupported password type: %s", security.UsernameToken.Password.Type)
	}

	return nil
}

// cleanupNonces periodically removes expired nonces
func cleanupNonces() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-cleanupDone:
			return
		case <-ticker.C:
			now := time.Now()
			usedNonces.Range(func(key, value interface{}) bool {
				timestamp := value.(time.Time)
				if now.Sub(timestamp) > nonceTTL {
					usedNonces.Delete(key)
				}
				return true
			})
		}
	}
}
