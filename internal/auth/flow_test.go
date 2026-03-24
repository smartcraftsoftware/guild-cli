package auth

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestStartDeviceFlow(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/auth/device" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != "POST" {
			t.Errorf("unexpected method: %s", r.Method)
		}

		json.NewEncoder(w).Encode(DeviceResponse{
			DeviceCode:      "test-device-code",
			UserCode:        "ABCD1234",
			VerificationURI: "http://example.com/authorize?user_code=ABCD1234",
			ExpiresIn:       600,
		})
	}))
	defer server.Close()

	dr, err := StartDeviceFlow(server.URL)
	if err != nil {
		t.Fatalf("StartDeviceFlow failed: %v", err)
	}

	if dr.DeviceCode != "test-device-code" {
		t.Errorf("DeviceCode = %q, want %q", dr.DeviceCode, "test-device-code")
	}
	if dr.UserCode != "ABCD1234" {
		t.Errorf("UserCode = %q, want %q", dr.UserCode, "ABCD1234")
	}
	if dr.ExpiresIn != 600 {
		t.Errorf("ExpiresIn = %d, want 600", dr.ExpiresIn)
	}
}

func TestPollForToken_ImmediateSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(TokenResponse{
			AccessToken: "guild_test_token_123",
			TokenType:   "Bearer",
			User: UserResponse{
				Name:  "Test User",
				Email: "test@example.com",
			},
		})
	}))
	defer server.Close()

	tr, err := PollForToken(server.URL, "test-code", 10*time.Second)
	if err != nil {
		t.Fatalf("PollForToken failed: %v", err)
	}

	if tr.AccessToken != "guild_test_token_123" {
		t.Errorf("AccessToken = %q, want %q", tr.AccessToken, "guild_test_token_123")
	}
	if tr.User.Name != "Test User" {
		t.Errorf("User.Name = %q, want %q", tr.User.Name, "Test User")
	}
}

func TestPollForToken_PendingThenSuccess(t *testing.T) {
	attempt := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempt++
		if attempt < 3 {
			w.WriteHeader(http.StatusAccepted)
			json.NewEncoder(w).Encode(ErrorResponse{Error: "authorization_pending"})
			return
		}
		json.NewEncoder(w).Encode(TokenResponse{
			AccessToken: "guild_delayed_token",
			TokenType:   "Bearer",
			User:        UserResponse{Name: "Delayed User", Email: "delayed@example.com"},
		})
	}))
	defer server.Close()

	// Override poll interval for testing
	origInterval := PollInterval
	defer func() { /* PollInterval is const, can't restore */ }()
	_ = origInterval

	tr, err := PollForToken(server.URL, "test-code", 30*time.Second)
	if err != nil {
		t.Fatalf("PollForToken failed: %v", err)
	}

	if tr.AccessToken != "guild_delayed_token" {
		t.Errorf("AccessToken = %q, want %q", tr.AccessToken, "guild_delayed_token")
	}
}

func TestPollForToken_Expired(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusGone)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "expired_token"})
	}))
	defer server.Close()

	_, err := PollForToken(server.URL, "test-code", 10*time.Second)
	if err == nil {
		t.Fatal("expected error for expired token, got nil")
	}
}

func TestGetMe_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/auth/me" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test_token" {
			t.Errorf("missing or wrong Authorization header")
		}

		json.NewEncoder(w).Encode(MeResponse{
			User: UserResponse{
				ID:    1,
				Name:  "Test User",
				Email: "test@example.com",
				Teams: []TeamResponse{{ID: 1, Name: "Engineering"}},
			},
		})
	}))
	defer server.Close()

	mr, err := GetMe(server.URL, "test_token")
	if err != nil {
		t.Fatalf("GetMe failed: %v", err)
	}

	if mr.User.Name != "Test User" {
		t.Errorf("User.Name = %q, want %q", mr.User.Name, "Test User")
	}
	if len(mr.User.Teams) != 1 {
		t.Errorf("Teams count = %d, want 1", len(mr.User.Teams))
	}
}

func TestGetMe_Unauthorized(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "Invalid or expired token"})
	}))
	defer server.Close()

	_, err := GetMe(server.URL, "bad_token")
	if err == nil {
		t.Fatal("expected error for unauthorized, got nil")
	}
}
