// This program finds pull requests created by the authenticated GitHub user
// since the start of the current month and prints:
//   - Jira ticket IDs extracted from the PR title (format: fix/feat/chore(TICKET-ID):)
//   - the names of the repositories those PRs belong to
//   - links to each PR
//
// Requires the GITHUB_TOKEN environment variable to hold a GitHub personal
// access token (Settings -> Developer settings -> Personal access tokens).
// The "repo" scope is enough (or "public_repo" if you only care about public repos).
//
// Usage:
//
//	export GITHUB_TOKEN=ghp_xxxxxxxx
//	go run main.go
//
// Optional flags:
//
//	go run main.go -org=organization-name
//	go run main.go -since=2026-09-01
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"
)

const githubAPI = "https://api.github.com"

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

type githubUser struct {
	Login string `json:"login"`
}

type prInfo struct {
	JiraID string
	Repo   string
	URL    string
	Number int
	Title  string
}

// Matches titles like: "fix(ABC-123): fixed something", "feat(PROJ-45): added X"
// (case-insensitive, optional space before the colon).
var jiraTitleRegexp = regexp.MustCompile(`(?i)^(?:fix|feat|chore)\(([^)]+)\)\s*:`)

func main() {
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		fmt.Fprintln(os.Stderr, "Error: set the GITHUB_TOKEN environment variable to a GitHub personal access token.")
		os.Exit(1)
	}

	org := flag.String("org", "", "Optional: restrict the search to a specific organization (e.g. -org=my-company)")
	since := flag.String("since", "", "Optional: start date in YYYY-MM-DD format (defaults to the 1st of the current month)")
	flag.Parse()

	startDate := *since
	if startDate == "" {
		now := time.Now()
		startDate = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location()).Format("2006-01-02")
	} else if _, err := time.Parse("2006-01-02", startDate); err != nil {
		fmt.Fprintf(os.Stderr, "Error: -since must be in YYYY-MM-DD format, got %q\n", startDate)
		os.Exit(1)
	}

	client := &http.Client{Timeout: 15 * time.Second}

	username, err := fetchAuthenticatedUsername(client, token)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to log in to GitHub: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Logged in as: %s\n", username)
	fmt.Printf("Searching for PRs created since: %s\n\n", startDate)

	query := fmt.Sprintf("is:pr author:%s created:>=%s", username, startDate)
	if *org != "" {
		query += fmt.Sprintf(" org:%s", *org)
	}

	items, err := searchAllPullRequests(client, token, query)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error while fetching pull requests: %v\n", err)
		os.Exit(1)
	}

	if len(items) == 0 {
		fmt.Println("No pull requests found for the given period.")
		return
	}

	prs := make([]prInfo, 0, len(items))
	repoSet := map[string]struct{}{}
	jiraSet := map[string]struct{}{}

	for _, item := range items {
		repo := repoNameFromURL(item.RepositoryURL)
		jiraID := extractJiraID(item.Title)

		prs = append(prs, prInfo{
			JiraID: jiraID,
			Repo:   repo,
			URL:    item.HTMLURL,
			Number: item.Number,
			Title:  item.Title,
		})
		repoSet[repo] = struct{}{}
		if jiraID != "" {
			jiraSet[jiraID] = struct{}{}
		}
	}

	sort.Slice(prs, func(i, j int) bool {
		if prs[i].Repo != prs[j].Repo {
			return prs[i].Repo < prs[j].Repo
		}
		return prs[i].Number < prs[j].Number
	})

	fmt.Printf("Found %d pull request(s).\n\n", len(prs))

	fmt.Println("=== Repositories ===")
	repos := make([]string, 0, len(repoSet))
	for r := range repoSet {
		repos = append(repos, r)
	}
	sort.Strings(repos)
	for _, r := range repos {
		fmt.Printf("- %s\n", r)
	}

	fmt.Println("\n=== Jira ticket IDs ===")
	jiras := make([]string, 0, len(jiraSet))
	for j := range jiraSet {
		jiras = append(jiras, j)
	}
	sort.Strings(jiras)
	if len(jiras) == 0 {
		fmt.Println("(no ticket IDs found in the PR titles)")
	}
	for _, j := range jiras {
		fmt.Printf("- %s\n", j)
	}

	fmt.Println("\n=== Pull request details ===")
	for _, pr := range prs {
		jira := pr.JiraID
		if jira == "" {
			jira = "(none)"
		}
		fmt.Printf("[%s] %s | %s | %s\n", jira, pr.Repo, pr.URL, pr.Title)
	}
}

func fetchAuthenticatedUsername(client *http.Client, token string) (string, error) {
	req, err := http.NewRequest(http.MethodGet, githubAPI+"/user", nil)
	if err != nil {
		return "", err
	}
	setAuthHeaders(req, token)

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("GitHub API returned status %d: %s", resp.StatusCode, string(body))
	}

	var user githubUser
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return "", err
	}
	return user.Login, nil
}

func searchAllPullRequests(client *http.Client, token, query string) ([]searchItem, error) {
	var all []searchItem
	page := 1
	for {
		items, hasMore, err := searchPullRequestsPage(client, token, query, page)
		if err != nil {
			return nil, err
		}
		all = append(all, items...)
		if !hasMore {
			break
		}
		page++
	}
	return all, nil
}

func searchPullRequestsPage(client *http.Client, token, query string, page int) ([]searchItem, bool, error) {
	req, err := http.NewRequest(http.MethodGet, githubAPI+"/search/issues", nil)
	if err != nil {
		return nil, false, err
	}
	q := url.Values{}
	q.Set("q", query)
	q.Set("per_page", "100")
	q.Set("page", fmt.Sprintf("%d", page))
	req.URL.RawQuery = q.Encode()
	setAuthHeaders(req, token)

	resp, err := client.Do(req)
	if err != nil {
		return nil, false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, false, fmt.Errorf("GitHub API returned status %d: %s", resp.StatusCode, string(body))
	}

	var result searchResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, false, err
	}

	// The Search API caps results at 1000 total (10 pages of 100).
	hasMore := page*100 < result.TotalCount && len(result.Items) > 0
	return result.Items, hasMore, nil
}

func setAuthHeaders(req *http.Request, token string) {
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
}

// repoNameFromURL extracts "owner/repo" from a URL like
// https://api.github.com/repos/owner/repo
func repoNameFromURL(repositoryURL string) string {
	parts := strings.Split(repositoryURL, "/")
	if len(parts) < 2 {
		return repositoryURL
	}
	return parts[len(parts)-2] + "/" + parts[len(parts)-1]
}

func extractJiraID(title string) string {
	matches := jiraTitleRegexp.FindStringSubmatch(strings.TrimSpace(title))
	if len(matches) < 2 {
		return ""
	}
	return matches[1]
}
