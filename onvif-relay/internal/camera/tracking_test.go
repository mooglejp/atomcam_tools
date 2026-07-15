package camera

import (
	"encoding/json"
	"net/http"
	"testing"
)

func TestSetTracking(t *testing.T) {
	tests := []struct {
		name    string
		enabled bool
		want    string
	}{
		{name: "on", enabled: true, want: "property tracking on"},
		{name: "off", enabled: false, want: "property tracking off"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, closeClient := newPTZTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost {
					t.Fatalf("request method = %s, want POST", r.Method)
				}
				if r.URL.Path != "/cgi-bin/cmd.cgi" || r.URL.Query().Get("port") != "socket" {
					t.Fatalf("unexpected request URL: %s", r.URL.String())
				}

				var request CommandRequest
				if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
					t.Fatalf("failed to decode request: %v", err)
				}
				if request.Exec != tt.want {
					t.Fatalf("command = %q, want %q", request.Exec, tt.want)
				}
			})
			defer closeClient()

			if err := client.SetTracking(tt.enabled); err != nil {
				t.Fatalf("SetTracking returned an error: %v", err)
			}
		})
	}
}
