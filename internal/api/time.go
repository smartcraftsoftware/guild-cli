package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
)

// TimeEntry represents a time entry from the Guild API.
type TimeEntry struct {
	ID              int          `json:"id"`
	Date            string       `json:"date"`
	DurationMinutes int          `json:"duration_minutes"`
	Description     string       `json:"description"`
	EntryType       string       `json:"entry_type"`
	Project         *ProjectInfo `json:"project"`
	StartTime       string       `json:"start_time"`
	EndTime         string       `json:"end_time"`
	CreatedAt       string       `json:"created_at"`
	UpdatedAt       string       `json:"updated_at"`
}

// ProjectInfo is basic project info in API responses.
type ProjectInfo struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// TimeEntryListResponse is the response from GET /api/v1/time_entries.
type TimeEntryListResponse struct {
	TimeEntries []TimeEntry `json:"time_entries"`
}

// TimeEntryListParams are the filter parameters for listing time entries.
type TimeEntryListParams struct {
	From      string
	To        string
	ProjectID string
	Limit     int
}

// ListTimeEntries fetches the current user's time entries.
func (c *Client) ListTimeEntries(params TimeEntryListParams) ([]TimeEntry, error) {
	u := fmt.Sprintf("%s/api/v1/time_entries", c.BaseURL)

	q := url.Values{}
	if params.From != "" {
		q.Set("from", params.From)
	}
	if params.To != "" {
		q.Set("to", params.To)
	}
	if params.ProjectID != "" {
		q.Set("project_id", params.ProjectID)
	}
	if params.Limit > 0 {
		q.Set("limit", strconv.Itoa(params.Limit))
	}
	if len(q) > 0 {
		u += "?" + q.Encode()
	}

	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return nil, fmt.Errorf("creating list time entries request: %w", err)
	}

	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("list time entries: status %d", resp.StatusCode)
	}

	var result TimeEntryListResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decoding time entries: %w", err)
	}

	return result.TimeEntries, nil
}

// CreateTimeEntryParams are the parameters for logging time.
type CreateTimeEntryParams struct {
	ProjectID       int    `json:"project_id"`
	Date            string `json:"date"`
	DurationMinutes int    `json:"duration_minutes"`
	Description     string `json:"description,omitempty"`
	EntryType       string `json:"entry_type,omitempty"`
}

// CreateTimeEntry logs a time entry.
func (c *Client) CreateTimeEntry(params CreateTimeEntryParams) (*TimeEntry, error) {
	u := fmt.Sprintf("%s/api/v1/time_entries", c.BaseURL)

	body := map[string]interface{}{
		"time_entry": params,
	}
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshaling time entry: %w", err)
	}

	req, err := http.NewRequest("POST", u, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("creating time entry request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusCreated {
		var entry TimeEntry
		if err := json.NewDecoder(resp.Body).Decode(&entry); err != nil {
			return nil, fmt.Errorf("decoding created time entry: %w", err)
		}
		return &entry, nil
	}

	if resp.StatusCode == http.StatusUnprocessableEntity {
		var errResp struct {
			Errors []string `json:"errors"`
		}
		json.NewDecoder(resp.Body).Decode(&errResp)
		return nil, fmt.Errorf("validation errors: %s", fmt.Sprint(errResp.Errors))
	}

	return nil, fmt.Errorf("create time entry: status %d", resp.StatusCode)
}

// DeleteTimeEntry deletes a time entry by ID.
func (c *Client) DeleteTimeEntry(id string) error {
	u := fmt.Sprintf("%s/api/v1/time_entries/%s", c.BaseURL, id)

	req, err := http.NewRequest("DELETE", u, nil)
	if err != nil {
		return fmt.Errorf("creating delete time entry request: %w", err)
	}

	resp, err := c.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return fmt.Errorf("time entry not found")
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("delete time entry: status %d", resp.StatusCode)
	}

	return nil
}
