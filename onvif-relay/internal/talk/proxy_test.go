package talk

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestClearStreamingDeadlinesAllowsSlowBody(t *testing.T) {
	handlerErr := make(chan error, 1)
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		clearStreamingDeadlines(w, "test-camera")
		body, err := io.ReadAll(r.Body)
		if err != nil {
			handlerErr <- err
			http.Error(w, err.Error(), http.StatusGatewayTimeout)
			return
		}
		if string(body) != "ab" {
			handlerErr <- fmt.Errorf("unexpected body: %q", string(body))
			http.Error(w, "unexpected body", http.StatusBadRequest)
			return
		}
		handlerErr <- nil
		w.WriteHeader(http.StatusNoContent)
	}))
	server.Config.ReadTimeout = 100 * time.Millisecond
	server.Config.WriteTimeout = 100 * time.Millisecond
	server.Start()
	defer server.Close()

	pr, pw := io.Pipe()
	req, err := http.NewRequest(http.MethodPost, server.URL, pr)
	if err != nil {
		t.Fatal(err)
	}

	responseErr := make(chan error, 1)
	go func() {
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			responseErr <- err
			return
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusNoContent {
			responseErr <- fmt.Errorf("unexpected status: %s", resp.Status)
			return
		}
		responseErr <- nil
	}()

	if _, err := pw.Write([]byte("a")); err != nil {
		t.Fatal(err)
	}
	time.Sleep(250 * time.Millisecond)
	if _, err := pw.Write([]byte("b")); err != nil {
		t.Fatal(err)
	}
	if err := pw.Close(); err != nil {
		t.Fatal(err)
	}

	select {
	case err := <-handlerErr:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("handler timed out")
	}

	select {
	case err := <-responseErr:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("client timed out")
	}
}
