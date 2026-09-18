# KUP Helper

This is simple-to-use CLI script for IT Professionals in Poland that have right for Koszty Uzystkania Przychodu,
a.k.a. Poland Copyright Tax Credit.

## Disclaimer

**This is not production-ready code**. It is meant to work locally, and help putting together KUP report by automating mundane task of querying GitHub.
Nothing is collected or stored on third-party services. Despite using best GitHub practices while connecting to GitHub,
please remember that it is still the most unstable, and unreliable Git platform in the market. Therefore, I will not be
surprised if PAT used for authorization might leak (i.e. trough server-side logs). **Please rotate your secrets regularly**.

**AI Usage warning:** While trying keep best practices, AI was used during development, as I am lazy piece of developer. You were warned.

## Prerequisites

- Go 1.27 or later
- A GitHub personal access token with the `repo` scope (or `public_repo` if you only need public repositories)

## Installation

Clone the repository and build the binary:

```sh
git clone https://github.com/mark00s/kup-helper.git
cd kup-helper
go build -o kup-helper ./cmd/kup-helper
```

Or skip the build step and run it directly:

```sh
go run ./cmd/kup-helper
```

## Authentication

Export your GitHub personal access token before running the tool:

```sh
export GITHUB_TOKEN=ghp_xxxxxxxx
```

## Usage

```sh
go run ./cmd/kup-helper [flags]
```

| Flag     | Description                                           | Default                                        |
| -------- | ----------------------------------------------------- | ---------------------------------------------- |
| `-org`   | Restrict the search to a specific GitHub organization | (none, searches everywhere your token can see) |
| `-since` | Start date in `YYYY-MM-DD` format                     | 1st of the current month                       |

Example:

```sh
go run ./cmd/kup-helper -org=my-company -since=2026-01-01
```

### Example output

```text
Logged in as: mark00s
Searching for PRs created since: 2026-09-01

Found 2 pull request(s).

=== Repositories ===
- my-company/backend-service
- my-company/frontend-app

=== Jira ticket IDs ===
- PROJ-123
- PROJ-456

=== Pull request details ===
[PROJ-123] my-company/backend-service | https://github.com/my-company/backend-service/pull/42 | fix(PROJ-123): correct pagination bug
[PROJ-456] my-company/frontend-app | https://github.com/my-company/frontend-app/pull/17 | feat(PROJ-456): add dark mode toggle
```

## Project structure

```text
cmd/kup-helper/     entry point: flag parsing and orchestration
internal/github/    minimal GitHub REST API client (auth, PR search with pagination)
internal/report/    Jira ID extraction, PR aggregation, report printing
```
