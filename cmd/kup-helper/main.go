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
//	go run ./cmd/kup-helper
//
// Optional flags:
//
//	go run ./cmd/kup-helper -org=organization-name
//	go run ./cmd/kup-helper -since=2026-09-01
package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/mark00s/kup-helper/internal/github"
	"github.com/mark00s/kup-helper/internal/report"
)

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

	client := github.NewClient(token, 15*time.Second)

	username, err := client.AuthenticatedUsername()
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

	items, err := client.SearchPullRequests(query)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error while fetching pull requests: %v\n", err)
		os.Exit(1)
	}

	if len(items) == 0 {
		fmt.Println("No pull requests found for the given period.")
		return
	}

	prs := report.Build(items)
	report.Print(os.Stdout, prs)
}
