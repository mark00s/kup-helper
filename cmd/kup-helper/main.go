// This program finds pull requests created by the authenticated GitHub user
// since the start of the current month and prints:
//   - Jira ticket IDs extracted from the PR title (format: fix/feat/chore(TICKET-ID):)
//   - the names of the repositories those PRs belong to
//   - links to each PR
//
// Requires GITHUB_TOKEN (environment variable or .env file) to hold a GitHub
// personal access token (Settings -> Developer settings -> Personal access tokens).
// The "repo" scope is enough (or "public_repo" if you only care about public repos).
//
// Usage:
//
//	export GITHUB_TOKEN=ghp_xxxxxxxx
//	go run ./cmd/kup-helper
//
// Optional flags:
//
//	go run ./cmd/kup-helper -orgs=organization-name,other-org
//	go run ./cmd/kup-helper -since=2026-09-01
//	go run ./cmd/kup-helper -day=20
//	go run ./cmd/kup-helper -no-jira
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/mark00s/kup-helper/internal/config"
	"github.com/mark00s/kup-helper/internal/github"
	"github.com/mark00s/kup-helper/internal/jira"
	"github.com/mark00s/kup-helper/internal/report"
)

func main() {
	cfg := config.Load()
	for _, w := range cfg.Warnings {
		fmt.Fprintf(os.Stderr, "Warning: %s\n", w)
	}
	token := cfg.Token
	if token == "" {
		fmt.Fprintln(os.Stderr, "Error: set GITHUB_TOKEN (environment variable or .env file) to a GitHub personal access token.")
		os.Exit(1)
	}

	orgs := flag.String("orgs", cfg.Orgs, "Optional: restrict the search to one or more comma-separated organizations (e.g. -orgs=my-company,other-org); defaults to KUP_ORGS")
	since := flag.String("since", "", "Optional: start date in YYYY-MM-DD format (defaults to the 1st of the current month)")
	day := flag.Int("day", 0, "Optional: day of the month the report is submitted (1-31); searches from that day of the previous month until today. Cannot be combined with -since")
	noJira := flag.Bool("no-jira", false, "Optional: skip fetching ticket titles from Jira Cloud (needs JIRA_HOSTNAME, JIRA_EMAIL and JIRA_API_TOKEN)")
	flag.Parse()

	if *day != 0 && (*day < 1 || *day > 31) {
		fmt.Fprintf(os.Stderr, "Error: -day must be between 1 and 31, got %d\n", *day)
		os.Exit(1)
	}
	if *day != 0 && *since != "" {
		fmt.Fprintln(os.Stderr, "Error: -day and -since cannot be used together.")
		os.Exit(1)
	}

	startDate := *since
	if *day != 0 {
		startDate = startFromDayOfMonth(time.Now(), *day).Format("2006-01-02")
	} else if startDate == "" {
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
	for _, o := range strings.Split(*orgs, ",") {
		if o = strings.TrimSpace(o); o != "" {
			query += " org:" + o
		}
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
	hostname := strings.TrimSpace(cfg.JiraHostname)

	var titles map[string]string
	if !*noJira && hostname != "" && cfg.JiraEmail != "" && cfg.JiraAPIToken != "" {
		jc := jira.NewClient(hostname, cfg.JiraEmail, cfg.JiraAPIToken, 15*time.Second)
		titles, err = jc.Summaries(context.Background(), report.UniqueJiraIDs(prs))
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Jira lookup failed, continuing without ticket titles: %v\n", err)
			titles = nil
		}
	}

	report.Print(os.Stdout, prs, hostname, titles)
}
