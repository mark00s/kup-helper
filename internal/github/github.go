// Package github is a minimal client for the parts of the GitHub REST API
// that kup-helper needs: authenticating the current user and searching
// pull requests.
package github

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

const apiBaseURL = "https://api.github.com"

// Client is a small authenticated wrapper around http.Client for the
// GitHub REST API.
type Client struct {
	httpClient *http.Client
	token      string
}

// NewClient creates a GitHub API client authenticated with the given
// personal access token.
func NewClient(token string, timeout time.Duration) *Client {
	return &Client{
		httpClient: &http.Client{Timeout: timeout},
		token:      token,
	}
}

// PullRequest is the subset of a GitHub search result item that
// kup-helper cares about.
type PullRequest struct {
	Title         string
	HTMLURL       string
	RepositoryURL string
	Number        int
	CreatedAt     string
}

// searchResponse is the response shape from the GitHub Search API (/search/issues).
type searchResponse struct {
	TotalCount int          `json:"total_count"`
	Items      []searchItem `json:"items"`
}

type searchItem struct {
	Title         string `json:"title"`
	HTMLURL       string `json:"html_url"`
	RepositoryURL string `json:"repository_url"`
	Number        int    `json:"number"`
	CreatedAt     string `json:"created_at"`
}

type user struct {
	Login string `json:"login"`
}

// AuthenticatedUsername returns the login of the user the client's token
// belongs to.
func (c *Client) AuthenticatedUsername() (string, error) {
	req, err := http.NewRequest(http.MethodGet, apiBaseURL+"/user", nil)
	if err != nil {
		return "", err
	}
	c.setAuthHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("GitHub API returned status %d: %s", resp.StatusCode, string(body))
	}

	var u user
	if err := json.NewDecoder(resp.Body).Decode(&u); err != nil {
		return "", err
	}
	return u.Login, nil
}

// SearchPullRequests runs the given GitHub search query against
// /search/issues and returns every matching pull request, following
// pagination until exhausted (the Search API caps results at 1000 total).
func (c *Client) SearchPullRequests(query string) ([]PullRequest, error) {
	var all []PullRequest
	page := 1
	for {
		items, totalCount, err := c.searchPullRequestsPage(query, page)
		if err != nil {
			return nil, err
		}
		for _, item := range items {
			all = append(all, PullRequest{
				Title:         item.Title,
				HTMLURL:       item.HTMLURL,
				RepositoryURL: item.RepositoryURL,
				Number:        item.Number,
				CreatedAt:     item.CreatedAt,
			})
		}
		hasMore := page*100 < totalCount && len(items) > 0
		if !hasMore {
			break
		}
		page++
	}
	return all, nil
}

func (c *Client) searchPullRequestsPage(query string, page int) ([]searchItem, int, error) {
	req, err := http.NewRequest(http.MethodGet, apiBaseURL+"/search/issues", nil)
	if err != nil {
		return nil, 0, err
	}
	q := url.Values{}
	q.Set("q", query)
	q.Set("per_page", "100")
	q.Set("page", fmt.Sprintf("%d", page))
	req.URL.RawQuery = q.Encode()
	c.setAuthHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, 0, fmt.Errorf("GitHub API returned status %d: %s", resp.StatusCode, string(body))
	}

	var result searchResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, 0, err
	}
	return result.Items, result.TotalCount, nil
}

func (c *Client) setAuthHeaders(req *http.Request) {
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
}
