package proxy

import (
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
)

// Handler is a reverse proxy bound to a specific path prefix.
type Handler struct {
	path        string
	stripPrefix bool
	proxy       *httputil.ReverseProxy
}

// New creates a Handler that forwards requests to target.
// path is the URL prefix this handler is registered for (e.g. "/api/").
// target is the backend base URL (e.g. "http://192.168.1.100:9000").
// When stripPrefix is true, the path prefix is removed before forwarding.
func New(path, target string, stripPrefix bool) (*Handler, error) {
	targetURL, err := url.Parse(target)
	if err != nil {
		return nil, err
	}

	targetBase := strings.TrimRight(targetURL.Path, "/")
	prefix := strings.TrimRight(path, "/")

	rp := &httputil.ReverseProxy{
		Director: func(r *http.Request) {
			r.URL.Scheme = targetURL.Scheme
			r.URL.Host = targetURL.Host
			r.Host = targetURL.Host

			reqPath := r.URL.Path
			if stripPrefix {
				reqPath = strings.TrimPrefix(reqPath, prefix)
				if reqPath == "" || reqPath[0] != '/' {
					reqPath = "/" + reqPath
				}
			}
			r.URL.Path = targetBase + reqPath
			r.URL.RawPath = ""

			// X-Forwarded-For
			if clientIP, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
				if prior, ok := r.Header["X-Forwarded-For"]; ok {
					clientIP = strings.Join(prior, ", ") + ", " + clientIP
				}
				r.Header.Set("X-Forwarded-For", clientIP)
			}
			r.Header.Set("X-Forwarded-Host", r.Host)
			r.Header.Set("X-Forwarded-Proto", "http")
		},
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			log.Printf("[proxy] %s %s → %s: %v", r.Method, r.URL.Path, target, err)
			http.Error(w, "Bad Gateway", http.StatusBadGateway)
		},
	}

	return &Handler{
		path:        path,
		stripPrefix: stripPrefix,
		proxy:       rp,
	}, nil
}

// ServeHTTP implements http.Handler.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.proxy.ServeHTTP(w, r)
}
