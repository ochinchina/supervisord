package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// basicAuthStatus sends one request through an httpBasicAuth configured with
// user/password and returns the response status. A nil creds sends no
// Authorization header.
func basicAuthStatus(t *testing.T, user, password string, creds *[2]string) int {
	t.Helper()
	ok := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	req := httptest.NewRequest(http.MethodGet, "/RPC2", nil)
	if creds != nil {
		req.SetBasicAuth(creds[0], creds[1])
	}
	rec := httptest.NewRecorder()
	newHTTPBasicAuth(user, password, ok).ServeHTTP(rec, req)
	if rec.Code == http.StatusUnauthorized && rec.Header().Get("WWW-Authenticate") == "" {
		t.Errorf("401 without WWW-Authenticate header")
	}
	return rec.Code
}

func TestBasicAuth(t *testing.T) {
	// {SHA} value from Python supervisor's docs: SHA-1 of "thepassword".
	const sha = "{SHA}82ab876d1387bfafe46cc1c8a2ef074eae50cb1d"

	tests := []struct {
		name     string
		user     string
		password string
		creds    *[2]string
		want     int
	}{
		{"no auth configured", "", "", nil, http.StatusOK},
		{"no password configured", "test1", "", nil, http.StatusOK},
		{"missing header", "test1", "thepassword", nil, http.StatusUnauthorized},
		{"plaintext ok", "test1", "thepassword", &[2]string{"test1", "thepassword"}, http.StatusOK},
		{"plaintext wrong password", "test1", "thepassword", &[2]string{"test1", "wrong"}, http.StatusUnauthorized},
		{"plaintext wrong user", "test1", "thepassword", &[2]string{"test2", "thepassword"}, http.StatusUnauthorized},
		{"plaintext prefix only", "test1", "thepassword", &[2]string{"test1", "thepass"}, http.StatusUnauthorized},
		{"sha ok", "test1", sha, &[2]string{"test1", "thepassword"}, http.StatusOK},
		{"sha wrong password", "test1", sha, &[2]string{"test1", "wrong"}, http.StatusUnauthorized},
		{"sha wrong user", "test1", sha, &[2]string{"test2", "thepassword"}, http.StatusUnauthorized},
		{"sha value sent as password", "test1", sha, &[2]string{"test1", sha}, http.StatusUnauthorized},
	}
	for _, tt := range tests {
		if got := basicAuthStatus(t, tt.user, tt.password, tt.creds); got != tt.want {
			t.Errorf("%s: status = %d, want %d", tt.name, got, tt.want)
		}
	}
}
