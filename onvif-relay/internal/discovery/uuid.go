package discovery

import (
	"crypto/rand"
	"crypto/sha1"
	"fmt"
	"time"
)

// generateUUID generates a deterministic UUID v5 based on device name
// This ensures the UUID remains stable across restarts
func generateUUID(deviceName string) string {
	// UUID v5 namespace for DNS (standard namespace)
	namespace := []byte{
		0x6b, 0xa7, 0xb8, 0x10, 0x9d, 0xad, 0x11, 0xd1,
		0x80, 0xb4, 0x00, 0xc0, 0x4f, 0xd4, 0x30, 0xc8,
	}

	// Compute SHA-1 hash of namespace + device name
	h := sha1.New()
	h.Write(namespace)
	h.Write([]byte(deviceName))
	hash := h.Sum(nil)

	// Take first 16 bytes and set version/variant bits
	b := hash[:16]
	b[6] = (b[6] & 0x0f) | 0x50 // Version 5
	b[8] = (b[8] & 0x3f) | 0x80 // Variant 10

	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// generateMessageID generates a unique message ID
func generateMessageID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		// Fallback to timestamp-based ID
		timestamp := time.Now().UnixNano()
		return fmt.Sprintf("%016x-%04x-%04x-%04x-%012x",
			timestamp, timestamp>>32, timestamp>>16, timestamp, timestamp>>48)
	}

	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// generateTimestamp generates an ISO 8601 timestamp
func generateTimestamp() string {
	return time.Now().UTC().Format("2006-01-02T15:04:05.000Z")
}
