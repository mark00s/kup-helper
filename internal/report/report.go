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

// Matches titles like: "fix(ABC-123): fixed something", "feat(PROJ-45): added X"
// (case-insensitive, optional space before the colon).
var jiraTitleRegexp = regexp.MustCompile(`(?i)^(?:fix|feat|chore)\(([^)]+)\)\s*:`)

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

// ExtractJiraID pulls the Jira ticket ID out of a conventional-commit-style
// PR title, or returns "" if none is found.
func ExtractJiraID(title string) string {
	matches := jiraTitleRegexp.FindStringSubmatch(strings.TrimSpace(title))
	if len(matches) < 2 {
		return ""
	}
	return matches[1]
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

func sortedKeys(set map[string]struct{}) []string {
	keys := make([]string, 0, len(set))
	for k := range set {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// Print writes the full kup-helper report (repositories, Jira ticket IDs,
// and per-PR details) to w.
func Print(w io.Writer, prs []PR) {
	fmt.Fprintf(w, "Found %d pull request(s).\n\n", len(prs))

	fmt.Fprintln(w, "=== Repositories ===")
	for _, r := range UniqueRepos(prs) {
		fmt.Fprintf(w, "- %s\n", r)
	}

	fmt.Fprintln(w, "\n=== Jira ticket IDs ===")
	jiras := UniqueJiraIDs(prs)
	if len(jiras) == 0 {
		fmt.Fprintln(w, "(no ticket IDs found in the PR titles)")
	}
	for _, j := range jiras {
		fmt.Fprintf(w, "- %s\n", j)
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
