package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ollama/ollama/auth"
	"github.com/ollama/ollama/envconfig"
	"github.com/ollama/ollama/openai"
)

const (
	defaultCloudProxyBaseURL       = "https://ollama.com:443"
	defaultCloudProxySigningHost   = "ollama.com"
	cloudProxyBaseURLEnv           = "OLLAMA_CLOUD_BASE_URL"
	cloudWebSearchOrchestrationKey = "cloud_web_search_orchestration"
	cloudProxyClientVersionHeader  = "X-Ollama-Client-Version"

	// maxDecompressedBodySize limits the size of a decompressed request body
	maxDecompressedBodySize = 20 << 20
)

var (
	cloudProxyBaseURL     = defaultCloudProxyBaseURL
	cloudProxySigningHost = defaultCloudProxySigningHost
	cloudProxySignRequest = signCloudProxyRequest
	cloudProxySigninURL   = signinURL
)

var hopByHopHeaders = map[string]struct{}{
	"connection":          {},
	"content-length":      {},
	"proxy-connection":    {},
	"keep-alive":          {},
	"proxy-authenticate":  {},
	"proxy-authorization": {},
	"te":                  {},
	"trailer":             {},
	"transfer-encoding":   {},
	"upgrade":             {},
}

func init() {
	baseURL, signingHost, overridden, err := resolveCloudProxyBaseURL(envconfig.Var(cloudProxyBaseURLEnv), mode)
	if err != nil {
		slog.Warn("ignoring cloud base URL override", "env", cloudProxyBaseURLEnv, "error", err)
		return
	}

	cloudProxyBaseURL = baseURL
	cloudProxySigningHost = signingHost

	if overridden {
		slog.Info("cloud base URL override enabled", "env", cloudProxyBaseURLEnv, "url", cloudProxyBaseURL, "mode", mode)
	}
}

func readRequestBody(r *http.Request) ([]byte, error) {
	if r.Body == nil {
		return nil, nil
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}

	r.Body = io.NopCloser(bytes.NewReader(body))
	return body, nil
}

func extractModelField(body []byte) (string, bool) {
	if len(body) == 0 {
		return "", false
	}

	var payload map[string]json.RawMessage
	if err := json.Unmarshal(body, &payload); err != nil {
		return "", false
	}

	raw, ok := payload["model"]
	if !ok {
		return "", false
	}

	var model string
	if err := json.Unmarshal(raw, &model); err != nil {
		return "", false
	}

	model = strings.TrimSpace(model)
	return model, model != ""
}

func hasWebSearchTool(path string, body []byte) bool {
	if len(body) == 0 {
		return false
	}

	if path == "/v1/responses" {
		var payload struct {
			Tools []openai.ResponsesTool `json:"tools"`
		}
		return json.Unmarshal(body, &payload) == nil && openai.HasWebSearchTool(payload.Tools)
	}

	if path != "/v1/messages" {
		return false
	}

	var payload struct {
		Tools []struct {
			Type string `json:"type"`
		} `json:"tools"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return false
	}

	for _, tool := range payload.Tools {
		if strings.HasPrefix(strings.TrimSpace(tool.Type), "web_search") {
			return true
		}
	}

	return false
}

func writeCloudUnauthorized(c *gin.Context) {
	signinURL, err := cloudProxySigninURL()
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized", "signin_url": signinURL})
}

func signCloudProxyRequest(ctx context.Context, req *http.Request) error {
	if !strings.EqualFold(req.URL.Hostname(), cloudProxySigningHost) {
		return nil
	}

	ts := strconv.FormatInt(time.Now().Unix(), 10)
	challenge := buildCloudSignatureChallenge(req, ts)
	signature, err := auth.Sign(ctx, []byte(challenge))
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", signature)
	return nil
}

func buildCloudSignatureChallenge(req *http.Request, ts string) string {
	query := req.URL.Query()
	query.Set("ts", ts)
	req.URL.RawQuery = query.Encode()

	return fmt.Sprintf("%s,%s", req.Method, req.URL.RequestURI())
}

func resolveCloudProxyBaseURL(rawOverride string, runMode string) (baseURL string, signingHost string, overridden bool, err error) {
	baseURL = defaultCloudProxyBaseURL
	signingHost = defaultCloudProxySigningHost

	rawOverride = strings.TrimSpace(rawOverride)
	if rawOverride == "" {
		return baseURL, signingHost, false, nil
	}

	u, err := url.Parse(rawOverride)
	if err != nil {
		return "", "", false, fmt.Errorf("invalid URL: %w", err)
	}
	if u.Scheme == "" || u.Host == "" {
		return "", "", false, fmt.Errorf("invalid URL: scheme and host are required")
	}
	if u.User != nil {
		return "", "", false, fmt.Errorf("invalid URL: userinfo is not allowed")
	}
	if u.Path != "" && u.Path != "/" {
		return "", "", false, fmt.Errorf("invalid URL: path is not allowed")
	}
	if u.RawQuery != "" || u.Fragment != "" {
		return "", "", false, fmt.Errorf("invalid URL: query and fragment are not allowed")
	}

	host := u.Hostname()
	if host == "" {
		return "", "", false, fmt.Errorf("invalid URL: host is required")
	}

	loopback := isLoopbackHost(host)
	if runMode == gin.ReleaseMode && !loopback {
		return "", "", false, fmt.Errorf("non-loopback cloud override is not allowed in release mode")
	}
	if !loopback && !strings.EqualFold(u.Scheme, "https") {
		return "", "", false, fmt.Errorf("non-loopback cloud override must use https")
	}

	u.Path = ""
	u.RawPath = ""
	u.RawQuery = ""
	u.Fragment = ""

	return u.String(), strings.ToLower(host), true, nil
}

func isLoopbackHost(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}

	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func copyProxyRequestHeaders(dst, src http.Header) {
	connectionTokens := connectionHeaderTokens(src)
	for key, values := range src {
		if isHopByHopHeader(key) || isConnectionTokenHeader(key, connectionTokens) {
			continue
		}

		dst.Del(key)
		for _, value := range values {
			dst.Add(key, value)
		}
	}
}

func copyProxyResponseHeaders(dst, src http.Header) {
	connectionTokens := connectionHeaderTokens(src)
	for key, values := range src {
		if isHopByHopHeader(key) || isConnectionTokenHeader(key, connectionTokens) {
			continue
		}

		dst.Del(key)
		for _, value := range values {
			dst.Add(key, value)
		}
	}
}

func copyProxyResponseBody(dst http.ResponseWriter, src io.Reader) error {
	flusher, canFlush := dst.(http.Flusher)
	buf := make([]byte, 32*1024)

	for {
		n, err := src.Read(buf)
		if n > 0 {
			if _, writeErr := dst.Write(buf[:n]); writeErr != nil {
				return writeErr
			}
			if canFlush {
				// TODO(drifkin): Consider conditional flushing so non-streaming
				// responses don't flush every write and can optimize throughput.
				flusher.Flush()
			}
		}

		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
	}
}

type jsonlFramingResponseWriter struct {
	http.ResponseWriter
	pending []byte
}

func (w *jsonlFramingResponseWriter) Flush() {
	if flusher, ok := w.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

func (w *jsonlFramingResponseWriter) Write(p []byte) (int, error) {
	w.pending = append(w.pending, p...)
	if err := w.flushCompleteLines(); err != nil {
		return len(p), err
	}
	return len(p), nil
}

func (w *jsonlFramingResponseWriter) FlushPending() error {
	trailing := bytes.TrimSpace(w.pending)
	w.pending = nil
	if len(trailing) == 0 {
		return nil
	}

	_, err := w.ResponseWriter.Write(trailing)
	return err
}

func (w *jsonlFramingResponseWriter) flushCompleteLines() error {
	for {
		newline := bytes.IndexByte(w.pending, '\n')
		if newline < 0 {
			return nil
		}

		line := bytes.TrimSpace(w.pending[:newline])
		w.pending = w.pending[newline+1:]
		if len(line) == 0 {
			continue
		}

		if _, err := w.ResponseWriter.Write(line); err != nil {
			return err
		}
	}
}

func isHopByHopHeader(name string) bool {
	_, ok := hopByHopHeaders[strings.ToLower(name)]
	return ok
}

func connectionHeaderTokens(header http.Header) map[string]struct{} {
	tokens := map[string]struct{}{}
	for _, raw := range header.Values("Connection") {
		for _, token := range strings.Split(raw, ",") {
			token = strings.TrimSpace(strings.ToLower(token))
			if token == "" {
				continue
			}
			tokens[token] = struct{}{}
		}
	}
	return tokens
}

func isConnectionTokenHeader(name string, tokens map[string]struct{}) bool {
	if len(tokens) == 0 {
		return false
	}
	_, ok := tokens[strings.ToLower(name)]
	return ok
}
