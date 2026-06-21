package proxy

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
)

type Handler struct {
	config Config
	client *http.Client
	logger *log.Logger
}

func NewHandler(config Config, logger *log.Logger) http.Handler {
	return &Handler{
		config: config,
		client: &http.Client{Timeout: config.Timeout},
		logger: logger,
	}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet && r.URL.Path == "/health" {
		w.Header().Set("content-type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}` + "\n"))
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, fmt.Sprintf("read request body: %v", err), http.StatusBadGateway)
		return
	}
	_ = r.Body.Close()

	rewrittenBody, mutations := h.rewriteBody(r, body)
	status, err := h.forward(w, r, rewrittenBody)
	if err != nil {
		if status != 0 {
			h.logRequest(r, status, mutations, err)
			return
		}
		http.Error(w, fmt.Sprintf("proxy error: %v", err), http.StatusBadGateway)
		return
	}

	h.logRequest(r, status, mutations, nil)
}

func (h *Handler) logRequest(r *http.Request, status int, mutations []string, err error) {
	marker := ""
	if len(mutations) > 0 {
		marker = " " + strings.Join(mutations, "+")
	}
	if err != nil {
		marker += " copy_error=" + err.Error()
	}
	h.logger.Printf("%s %s -> %d%s", r.Method, r.URL.RequestURI(), status, marker)
}

func (h *Handler) rewriteBody(r *http.Request, body []byte) ([]byte, []string) {
	if len(body) == 0 || !strings.Contains(strings.ToLower(r.Header.Get("content-type")), "application/json") {
		return body, nil
	}

	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return body, nil
	}

	mutations := NormalizePayload(payload, r.URL.Path, h.config)
	if len(mutations) == 0 {
		return body, nil
	}

	encoded, err := json.Marshal(payload)
	if err != nil {
		return body, nil
	}
	return encoded, mutations
}

func (h *Handler) forward(w http.ResponseWriter, r *http.Request, body []byte) (int, error) {
	upstreamURL := h.upstreamURL(r.URL)
	req, err := http.NewRequestWithContext(r.Context(), r.Method, upstreamURL.String(), bytes.NewReader(body))
	if err != nil {
		return 0, err
	}
	copyForwardHeaders(req.Header, r.Header)
	if len(body) > 0 {
		req.ContentLength = int64(len(body))
	}

	resp, err := h.client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	copyResponseHeaders(w.Header(), resp.Header)
	w.WriteHeader(resp.StatusCode)
	copyErr := copyResponseBody(w, resp.Body)
	return resp.StatusCode, copyErr
}

func copyResponseBody(w http.ResponseWriter, body io.Reader) error {
	flusher, ok := w.(http.Flusher)
	if !ok {
		_, err := io.Copy(w, body)
		return err
	}

	buf := make([]byte, 32*1024)
	for {
		n, readErr := body.Read(buf)
		if n > 0 {
			if _, writeErr := w.Write(buf[:n]); writeErr != nil {
				return writeErr
			}
			flusher.Flush()
		}
		if readErr != nil {
			if errors.Is(readErr, io.EOF) {
				return nil
			}
			return readErr
		}
	}
}

func (h *Handler) upstreamURL(requestURL *url.URL) *url.URL {
	upstream := *h.config.UpstreamURL
	basePath := strings.TrimRight(upstream.Path, "/")
	upstream.Path = basePath + requestURL.Path
	upstream.RawQuery = requestURL.RawQuery
	return &upstream
}

func copyForwardHeaders(dst, src http.Header) {
	for key, values := range src {
		if isHopByHopHeader(key) || strings.EqualFold(key, "host") || strings.EqualFold(key, "content-length") {
			continue
		}
		for _, value := range values {
			dst.Add(key, value)
		}
	}
}

func copyResponseHeaders(dst, src http.Header) {
	for key, values := range src {
		if isHopByHopHeader(key) {
			continue
		}
		for _, value := range values {
			dst.Add(key, value)
		}
	}
}

func isHopByHopHeader(name string) bool {
	switch strings.ToLower(name) {
	case "connection", "keep-alive", "proxy-authenticate", "proxy-authorization", "te", "trailer", "transfer-encoding", "upgrade":
		return true
	default:
		return false
	}
}
