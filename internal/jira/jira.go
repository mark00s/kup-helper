// Package jira is a minimal Jira Cloud REST API client that looks up issue
// summaries by key.
package jira

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

// Client talks to a single Jira Cloud site using email + API token (Basic auth).
type Client struct {
	baseURL string
	email   string
	token   string
	http    *http.Client
}

// NewClient returns a client for https://<hostname>, e.g. "mycompany.atlassian.net".
func NewClient(hostname, email, token string, timeout time.Duration) *Client {
	return &Client{
		baseURL: "https://" + hostname,
		email:   email,
		token:   token,
		http:    &http.Client{Timeout: timeout},
	}
}

// Summaries returns the summary (title) of each given issue key. Keys that do
// not exist, or that the user cannot see, are left out of the result. Any
// other failure (bad credentials, network, server error) aborts the lookup.
func (c *Client) Summaries(ctx context.Context, keys []string) (map[string]string, error) {
	out := make(map[string]string, len(keys))
	for _, key := range keys {
		summary, found, err := c.summary(ctx, key)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", key, err)
		}
		if found {
			out[key] = summary
		}
	}
	return out, nil
}

func (c *Client) summary(ctx context.Context, key string) (string, bool, error) {
	u := fmt.Sprintf("%s/rest/api/3/issue/%s?fields=summary", c.baseURL, url.PathEscape(key))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return "", false, err
	}
	req.SetBasicAuth(c.email, c.token)
	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return "", false, err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
	case http.StatusNotFound:
		// Jira answers 404 both for missing issues and for issues the user cannot see.
		return "", false, nil
	case http.StatusUnauthorized:
		return "", false, fmt.Errorf("unauthorized; check JIRA_EMAIL and JIRA_API_TOKEN")
	default:
		return "", false, fmt.Errorf("unexpected status %s", resp.Status)
	}

	var body struct {
		Fields struct {
			Summary string `json:"summary"`
		} `json:"fields"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return "", false, fmt.Errorf("decoding response: %w", err)
	}
	return body.Fields.Summary, true, nil
}
