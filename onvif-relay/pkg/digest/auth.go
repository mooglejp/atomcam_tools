package digest

import (
	"crypto/md5"
	"crypto/rand"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// Transport implements HTTP Digest authentication
type Transport struct {
	Username  string
	Password  string
	Transport http.RoundTripper
	mu        sync.Mutex
	challenges map[string]*challenge
	nc        uint32 // nonce counter (atomic)
	done      chan struct{} // shutdown signal for cleanup goroutine
	closeOnce sync.Once      // ensures done is only closed once
}

type challenge struct {
	Realm     string
	Nonce     string
	Opaque    string
	Algorithm string
	Qop       string
	timestamp time.Time // Added for TTL cleanup
}

// NewTransport creates a new Digest transport
func NewTransport(username, password string) *Transport {
	t := &Transport{
		Username:   username,
		Password:   password,
		Transport:  http.DefaultTransport,
		challenges: make(map[string]*challenge),
		done:       make(chan struct{}),
	}
	// Start cleanup goroutine to remove stale challenges
	go t.cleanupChallenges()
	return t
}

// Close stops the cleanup goroutine (call during shutdown)
func (t *Transport) Close() {
	t.closeOnce.Do(func() {
		close(t.done)
	})
}

// RoundTrip implements http.RoundTripper
func (t *Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Clone request for the first attempt (to preserve GetBody for retry)
	req1 := cloneRequest(req)

	// Try request without auth first
	resp, err := t.Transport.RoundTrip(req1)
	if err != nil {
		return nil, err
	}

	// If not 401, return response
	if resp.StatusCode != http.StatusUnauthorized {
		return resp, nil
	}

	// Parse WWW-Authenticate header before closing body
	wwwAuth := resp.Header.Get("WWW-Authenticate")

	// Close initial response body
	resp.Body.Close()

	if wwwAuth == "" {
		return nil, fmt.Errorf("401 response without WWW-Authenticate header")
	}

	chal, err := parseChallenge(wwwAuth)
	if err != nil {
		return resp, fmt.Errorf("failed to parse WWW-Authenticate: %w", err)
	}

	// Store challenge for this URL with timestamp
	chal.timestamp = time.Now()
	t.mu.Lock()
	t.challenges[req.URL.String()] = chal
	t.mu.Unlock()

	// Clone request and add Authorization header
	req2 := cloneRequest(req)
	auth := t.buildAuthorization(req2, chal)
	req2.Header.Set("Authorization", auth)

	// Retry with auth
	return t.Transport.RoundTrip(req2)
}

// parseChallenge parses WWW-Authenticate header
func parseChallenge(header string) (*challenge, error) {
	if !strings.HasPrefix(header, "Digest ") {
		return nil, fmt.Errorf("not a Digest challenge")
	}

	parts := strings.Split(header[7:], ",")
	chal := &challenge{
		Algorithm: "MD5", // default
	}

	for _, part := range parts {
		kv := strings.SplitN(strings.TrimSpace(part), "=", 2)
		if len(kv) != 2 {
			continue
		}

		key := strings.TrimSpace(kv[0])
		value := strings.Trim(strings.TrimSpace(kv[1]), "\"")

		switch key {
		case "realm":
			chal.Realm = value
		case "nonce":
			chal.Nonce = value
		case "opaque":
			chal.Opaque = value
		case "algorithm":
			chal.Algorithm = value
		case "qop":
			chal.Qop = value
		}
	}

	return chal, nil
}

// buildAuthorization builds Authorization header
func (t *Transport) buildAuthorization(req *http.Request, chal *challenge) string {
	// HA1 = MD5(username:realm:password)
	ha1 := md5Hash(fmt.Sprintf("%s:%s:%s", t.Username, chal.Realm, t.Password))

	// HA2 = MD5(method:uri)
	ha2 := md5Hash(fmt.Sprintf("%s:%s", req.Method, req.URL.RequestURI()))

	// Response = MD5(HA1:nonce:HA2)
	var response string
	var nc string
	var cnonce string

	if chal.Qop == "" {
		response = md5Hash(fmt.Sprintf("%s:%s:%s", ha1, chal.Nonce, ha2))
	} else {
		// With qop (quality of protection)
		// Increment nonce counter atomically
		ncValue := atomic.AddUint32(&t.nc, 1)
		nc = fmt.Sprintf("%08x", ncValue)

		// Generate random cnonce
		cnonce = generateCnonce()

		response = md5Hash(fmt.Sprintf("%s:%s:%s:%s:%s:%s", ha1, chal.Nonce, nc, cnonce, chal.Qop, ha2))
	}

	// Build Authorization header
	auth := fmt.Sprintf(`Digest username="%s", realm="%s", nonce="%s", uri="%s", response="%s"`,
		t.Username, chal.Realm, chal.Nonce, req.URL.RequestURI(), response)

	if chal.Opaque != "" {
		auth += fmt.Sprintf(`, opaque="%s"`, chal.Opaque)
	}

	if chal.Algorithm != "" && chal.Algorithm != "MD5" {
		auth += fmt.Sprintf(`, algorithm="%s"`, chal.Algorithm)
	}

	if chal.Qop != "" && nc != "" && cnonce != "" {
		auth += fmt.Sprintf(`, qop=%s, nc=%s, cnonce="%s"`, chal.Qop, nc, cnonce)
	}

	return auth
}

// md5Hash returns MD5 hash of input string
func md5Hash(input string) string {
	h := md5.New()
	io.WriteString(h, input)
	return fmt.Sprintf("%x", h.Sum(nil))
}

// cloneRequest creates a shallow copy of the request
func cloneRequest(req *http.Request) *http.Request {
	req2 := new(http.Request)
	*req2 = *req
	req2.Header = make(http.Header, len(req.Header))
	for k, v := range req.Header {
		req2.Header[k] = v
	}
	// Clone the body using GetBody if available
	if req.GetBody != nil {
		req2.Body, _ = req.GetBody()
	}
	return req2
}

// generateCnonce generates a random client nonce
func generateCnonce() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		// Fallback to less secure but still functional method
		return fmt.Sprintf("%x", time.Now().UnixNano())
	}
	return fmt.Sprintf("%x", b)
}

// cleanupChallenges periodically removes stale challenges (TTL: 5 minutes)
func (t *Transport) cleanupChallenges() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	const challengeTTL = 5 * time.Minute

	for {
		select {
		case <-t.done:
			return
		case <-ticker.C:
			t.mu.Lock()
			now := time.Now()
			for url, chal := range t.challenges {
				if now.Sub(chal.timestamp) > challengeTTL {
					delete(t.challenges, url)
				}
			}
			t.mu.Unlock()
		}
	}
}
