// Package report turns raw GitHub pull requests into the kup-helper report:
// Jira ticket IDs, repositories touched, and per-PR details.
package report

import (
	"fmt"
	"io"
	"regexp"
	"sort"
	"strings"

	"github.com/mark00s/kup-helper/internal/github"
)

// Matches a Jira ticket ID anywhere in a title, e.g. "JP-2137" in
// "fix(JP-2137): something" or "[JP-2137] something".
var jiraTitleRegexp = regexp.MustCompile(`\b[A-Z][A-Z_0-9]+-\d+\b`)

// PR is a pull request enriched with its extracted Jira ticket ID and
// repository name.
type PR struct {
	JiraID string
	Repo   string
	URL    string
	Number int
	Title  string
}

// Build converts raw GitHub pull requests into PRs, sorted by repository
// then PR number.
func Build(prs []github.PullRequest) []PR {
	result := make([]PR, 0, len(prs))
	for _, item := range prs {
		result = append(result, PR{
			JiraID: ExtractJiraID(item.Title),
			Repo:   repoNameFromURL(item.RepositoryURL),
			URL:    item.HTMLURL,
			Number: item.Number,
			Title:  item.Title,
		})
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Repo != result[j].Repo {
			return result[i].Repo < result[j].Repo
		}
		return result[i].Number < result[j].Number
	})
	return result
}

// ExtractJiraID returns the first Jira ticket ID found in the PR title, or ""
// if none is found.
func ExtractJiraID(title string) string {
	return jiraTitleRegexp.FindString(title)
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

// UniqueRepos returns the sorted, deduplicated set of repositories touched
// by the given PRs.
func UniqueRepos(prs []PR) []string {
	set := map[string]struct{}{}
	for _, pr := range prs {
		set[pr.Repo] = struct{}{}
	}
	return sortedKeys(set)
}

// UniqueJiraIDs returns the sorted, deduplicated set of non-empty Jira
// ticket IDs found in the given PRs.
func UniqueJiraIDs(prs []PR) []string {
	set := map[string]struct{}{}
	for _, pr := range prs {
		if pr.JiraID != "" {
			set[pr.JiraID] = struct{}{}
		}
	}
	return sortedKeys(set)
}

// WithoutJiraID returns the PRs whose titles contain no Jira ticket ID.
func WithoutJiraID(prs []PR) []PR {
	var out []PR
	for _, pr := range prs {
		if pr.JiraID == "" {
			out = append(out, pr)
		}
	}
	return out
}

func sortedKeys(set map[string]struct{}) []string {
	keys := make([]string, 0, len(set))
	for k := range set {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// Print writes the full kup-helper report (Jira ticket IDs, repositories,
// valid PR URLs, and per-PR details) to w. When hostname is set (e.g. "mycompany.atlassian.net"),
// each Jira ticket ID is followed by its https://<hostname>/browse/<ID> link.
// When titles is non-nil, each ticket line ends with its Jira title, or
// "(not found)" if the ticket is missing from titles.
func Print(w io.Writer, prs []PR, hostname string, titles map[string]string) {
	fmt.Fprintf(w, "Found %d pull request(s).\n\n", len(prs))

	fmt.Fprintln(w, "=== Jira ticket IDs ===")
	jiras := UniqueJiraIDs(prs)
	if len(jiras) == 0 {
		fmt.Fprintln(w, "(no ticket IDs found in the PR titles)")
	}
	for _, j := range jiras {
		line := "- " + j
		if hostname != "" {
			line += fmt.Sprintf(" | https://%s/browse/%s", hostname, j)
		}
		if titles != nil {
			title, ok := titles[j]
			if !ok {
				title = "(not found)"
			}
			line += " | " + title
		}
		fmt.Fprintln(w, line)
	}

	fmt.Fprintln(w, "\n=== Repositories ===")
	for _, r := range UniqueRepos(prs) {
		fmt.Fprintf(w, "- %s\n", r)
	}

	fmt.Fprintln(w, "\n=== Valid PR URLs ===")
	valid := 0
	for _, pr := range prs {
		if pr.JiraID != "" {
			fmt.Fprintf(w, "- %s\n", pr.URL)
			valid++
		}
	}
	if valid == 0 {
		fmt.Fprintln(w, "(none)")
	}

	fmt.Fprintln(w, "\n=== PRs without Jira ticket ID (validate manually) ===")
	missing := WithoutJiraID(prs)
	if len(missing) == 0 {
		fmt.Fprintln(w, "(none)")
	}
	for _, pr := range missing {
		fmt.Fprintf(w, "- %s | %s | %s\n", pr.Repo, pr.URL, pr.Title)
	}

	fmt.Fprintln(w, "\n=== Pull request details ===")
	for _, pr := range prs {
		jira := pr.JiraID
		if jira == "" {
			jira = "(none)"
		}
		fmt.Fprintf(w, "[%s] %s | %s | %s\n", jira, pr.Repo, pr.URL, pr.Title)
	}
}
