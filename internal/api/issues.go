package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
)

// Issue represents an issue from the Guild API.
type Issue struct {
	ID             int             `json:"id"`
	TeamID         int             `json:"team_id"`
	DisplayNumber  string          `json:"display_number"`
	Title          string          `json:"title"`
	Description    string          `json:"description"`
	IssueType      string          `json:"issue_type"`
	Priority       string          `json:"priority"`
	WorkflowStatus *WorkflowStatus `json:"workflow_status"`
	Assignee       *UserInfo       `json:"assignee"`
	CreatedBy      *UserInfo       `json:"created_by"`
	ProjectID      *int            `json:"project_id"`
	ParentID       *int            `json:"parent_id"`
	Position       int             `json:"position"`
	StartDate      *string         `json:"start_date"`
	DueDate        *string         `json:"due_date"`
	CreatedAt      string          `json:"created_at"`
	UpdatedAt      string          `json:"updated_at"`
}

// WorkflowStatus represents a workflow status.
type WorkflowStatus struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Category string `json:"category"`
}

// UserInfo represents basic user info in API responses.
type UserInfo struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

// IssueListResponse is the response from GET /api/v1/teams/:team_id/issues.
type IssueListResponse struct {
	Issues []Issue `json:"issues"`
}

// IssueListParams are the filter parameters for listing issues.
type IssueListParams struct {
	Status     string
	AssigneeID string
	IssueType  string
	Limit      int
}

// ListIssues fetches issues for a team with optional filters.
func (c *Client) ListIssues(teamID string, params IssueListParams) ([]Issue, error) {
	u := fmt.Sprintf("%s/api/v1/teams/%s/issues", c.BaseURL, teamID)

	q := url.Values{}
	if params.Status != "" {
		q.Set("status", params.Status)
	}
	if params.AssigneeID != "" {
		q.Set("assignee_id", params.AssigneeID)
	}
	if params.IssueType != "" {
		q.Set("issue_type", params.IssueType)
	}
	if params.Limit > 0 {
		q.Set("limit", strconv.Itoa(params.Limit))
	}
	if len(q) > 0 {
		u += "?" + q.Encode()
	}

	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return nil, fmt.Errorf("creating list issues request: %w", err)
	}

	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("list issues: status %d", resp.StatusCode)
	}

	var result IssueListResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decoding issues: %w", err)
	}

	return result.Issues, nil
}

// CreateIssueParams are the parameters for creating an issue.
type CreateIssueParams struct {
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	IssueType   string `json:"issue_type,omitempty"`
	Priority    string `json:"priority,omitempty"`
	AssigneeID  int    `json:"assignee_id,omitempty"`
}

// CreateIssue creates an issue in the given team.
func (c *Client) CreateIssue(teamID string, params CreateIssueParams) (*Issue, error) {
	u := fmt.Sprintf("%s/api/v1/teams/%s/issues", c.BaseURL, teamID)

	body := map[string]interface{}{
		"issue": params,
	}
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshaling issue: %w", err)
	}

	req, err := http.NewRequest("POST", u, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("creating create issue request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusCreated {
		var issue Issue
		if err := json.NewDecoder(resp.Body).Decode(&issue); err != nil {
			return nil, fmt.Errorf("decoding created issue: %w", err)
		}
		return &issue, nil
	}

	if resp.StatusCode == http.StatusUnprocessableEntity {
		var errResp struct {
			Errors []string `json:"errors"`
		}
		json.NewDecoder(resp.Body).Decode(&errResp)
		return nil, fmt.Errorf("validation errors: %s", fmt.Sprint(errResp.Errors))
	}

	return nil, fmt.Errorf("create issue: status %d", resp.StatusCode)
}

// GetIssue fetches a single issue by ID.
func (c *Client) GetIssue(teamID string, issueID string) (*Issue, error) {
	u := fmt.Sprintf("%s/api/v1/teams/%s/issues/%s", c.BaseURL, teamID, issueID)

	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return nil, fmt.Errorf("creating get issue request: %w", err)
	}

	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("issue not found")
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get issue: status %d", resp.StatusCode)
	}

	var issue Issue
	if err := json.NewDecoder(resp.Body).Decode(&issue); err != nil {
		return nil, fmt.Errorf("decoding issue: %w", err)
	}

	return &issue, nil
}
