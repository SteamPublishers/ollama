package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/ollama/ollama/envconfig"
)

// buildLocalAuthRouter wires a route behind the local-auth gate so the
// middleware can be exercised end to end. The gate captures the current
// LocalAuthHash at build time, so callers that set LOCAL_AUTH must rebuild.
func buildLocalAuthRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(allowedLocalAuth())
	r.GET("/ping", func(c *gin.Context) {
		c.String(http.StatusOK, "pong")
	})
	return r
}

func TestAllowedLocalAuthDisabled(t *testing.T) {
	// LOCAL_AUTH unset -> gate passes every request through.
	r := buildLocalAuthRouter()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 when gate disabled, got %d", w.Code)
	}
}

func TestAllowedLocalAuthAcceptsValidHeader(t *testing.T) {
	t.Setenv("LOCAL_AUTH", "foo")
	r := buildLocalAuthRouter()

	want := localAuthScheme + " " + envconfig.LocalAuthHash()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	req.Header.Set("Authorization", want)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for valid Internal header, got %d", w.Code)
	}
}

func TestAllowedLocalAuthRejectsMissingHeader(t *testing.T) {
	t.Setenv("LOCAL_AUTH", "foo")
	r := buildLocalAuthRouter()

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for missing header, got %d", w.Code)
	}
}

func TestAllowedLocalAuthRejectsWrongHeader(t *testing.T) {
	t.Setenv("LOCAL_AUTH", "foo")
	r := buildLocalAuthRouter()

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	req.Header.Set("Authorization", "Internal deadbeef")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for wrong hash, got %d", w.Code)
	}
}
