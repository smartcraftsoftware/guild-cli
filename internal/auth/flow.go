package auth

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

// DeviceResponse is returned by POST /api/v1/auth/device
type DeviceResponse struct {
	DeviceCode      string `json:"device_code"`
	UserCode        string `json:"user_code"`
	VerificationURI string `json:"verification_uri"`
	ExpiresIn       int    `json:"expires_in"`
}

// TokenResponse is returned by POST /api/v1/auth/device/token on success
type TokenResponse struct {
	AccessToken string       `json:"access_token"`
	TokenType   string       `json:"token_type"`
	User        UserResponse `json:"user"`
}

// UserResponse is the user info from token exchange or /me
type UserResponse struct {
	ID    int            `json:"id"`
	Name  string         `json:"name"`
	Email string         `json:"email"`
	Teams []TeamResponse `json:"teams,omitempty"`
}

// TeamResponse is a team in the user info
type TeamResponse struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// MeResponse is returned by GET /api/v1/auth/me
type MeResponse struct {
	User UserResponse `json:"user"`
}

// ErrorResponse is returned on error
type ErrorResponse struct {
	Error string `json:"error"`
}

// PollInterval is how often the CLI polls for the token
const PollInterval = 5 * time.Second

// StartDeviceFlow calls POST /api/v1/auth/device to initiate the auth flow.
func StartDeviceFlow(serverURL string) (*DeviceResponse, error) {
	url := strings.TrimRight(serverURL, "/") + "/api/v1/auth/device"

	resp, err := http.Post(url, "application/json", nil)
	if err != nil {
		return nil, fmt.Errorf("starting device flow: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("device flow returned status %d", resp.StatusCode)
	}

	var dr DeviceResponse
	if err := json.NewDecoder(resp.Body).Decode(&dr); err != nil {
		return nil, fmt.Errorf("decoding device response: %w", err)
	}

	return &dr, nil
}

// PollForToken polls POST /api/v1/auth/device/token until authorized, expired, or timeout.
func PollForToken(serverURL, deviceCode string, timeout time.Duration) (*TokenResponse, error) {
	url := strings.TrimRight(serverURL, "/") + "/api/v1/auth/device/token"
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		resp, err := http.PostForm(url, map[string][]string{
			"device_code": {deviceCode},
		})
		if err != nil {
			// Network error — wait and retry
			time.Sleep(PollInterval)
			continue
		}

		switch resp.StatusCode {
		case http.StatusOK:
			defer resp.Body.Close()
			var tr TokenResponse
			if err := json.NewDecoder(resp.Body).Decode(&tr); err != nil {
				resp.Body.Close()
				return nil, fmt.Errorf("decoding token response: %w", err)
			}
			return &tr, nil

		case http.StatusAccepted:
			// Still pending — wait and retry
			resp.Body.Close()
			time.Sleep(PollInterval)
			continue

		case http.StatusGone:
			resp.Body.Close()
			return nil, fmt.Errorf("authorization expired or already used")

		default:
			var er ErrorResponse
			json.NewDecoder(resp.Body).Decode(&er)
			resp.Body.Close()
			return nil, fmt.Errorf("token poll error (status %d): %s", resp.StatusCode, er.Error)
		}
	}

	return nil, fmt.Errorf("timed out waiting for authorization (%.0fs)", timeout.Seconds())
}

// GetMe calls GET /api/v1/auth/me with the given token.
func GetMe(serverURL, token string) (*MeResponse, error) {
	url := strings.TrimRight(serverURL, "/") + "/api/v1/auth/me"

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating me request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("calling /auth/me: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return nil, fmt.Errorf("invalid or expired token")
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("/auth/me returned status %d", resp.StatusCode)
	}

	var mr MeResponse
	if err := json.NewDecoder(resp.Body).Decode(&mr); err != nil {
		return nil, fmt.Errorf("decoding me response: %w", err)
	}

	return &mr, nil
}

// OpenBrowser opens the given URL in the system's default browser.
func OpenBrowser(url string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", url)
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}

	return cmd.Start()
}
