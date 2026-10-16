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

## Configuration

Settings are resolved in this order (highest priority first):

1. CLI flags (e.g. `-org`)
2. Environment variables
3. `.env` in the current directory
4. `.env` in the user config directory (`~/Library/Application Support/kup-helper/.env` on macOS, `~/.config/kup-helper/.env` on Linux)

| Variable       | Description                                   |
| -------------- | --------------------------------------------- |
| `GITHUB_TOKEN` | GitHub personal access token (required)       |
| `KUP_ORG`      | Default value for the `-org` flag (optional)  |

Create a `.env` file (see [.env.example](.env.example)) and restrict its permissions:

```sh
cp .env.example .env
chmod 600 .env
```

Or just export the variable; it overrides the file:

```sh
export GITHUB_TOKEN=ghp_xxxxxxxx
```

## Usage

```sh
go run ./cmd/kup-helper [flags]
```

| Flag     | Description                                           | Default                                        |
| -------- | ----------------------------------------------------- | ---------------------------------------------- |
| `-org`   | Restrict the search to a specific GitHub organization | `KUP_ORG`, else none (searches everywhere your token can see) |
| `-since` | Start date in `YYYY-MM-DD` format                     | 1st of the current month                       |
| `-day`   | Day of the month the report is submitted (1-31). Searches from that day of the previous month until today; shorter months are clamped to their last day. Cannot be combined with `-since` | (none) |

Example:

```sh
go run ./cmd/kup-helper -org=my-company -since=2026-01-01
go run ./cmd/kup-helper -day=20   # run on 2026-10-15 -> searches since 2026-09-20
```

### Example output

```text
Logged in as: mark00s
Searching for PRs created since: 2026-09-01

Found 3 pull request(s).

=== Repositories ===
- my-company/backend-service
- my-company/frontend-app

=== Jira ticket IDs ===
- PROJ-123
- PROJ-456

=== PRs without Jira ticket ID (validate manually) ===
- my-company/frontend-app | https://github.com/my-company/frontend-app/pull/19 | update dependencies

=== Pull request details ===
[PROJ-123] my-company/backend-service | https://github.com/my-company/backend-service/pull/42 | fix(PROJ-123): correct pagination bug
[PROJ-456] my-company/frontend-app | https://github.com/my-company/frontend-app/pull/17 | feat(PROJ-456): add dark mode toggle
[(none)] my-company/frontend-app | https://github.com/my-company/frontend-app/pull/19 | update dependencies
```

### Jira ticket ID detection

The first Jira ticket ID (uppercase project key that may contain digits and underscores, dash, number, e.g. `JP-2137` or `MY_PROJ-12`) found anywhere in a PR title is used,
so `fix(JP-2137): ...`, `[JP-2137] ...` and `JP-2137 ...` all work. PRs whose titles contain no ticket ID are
listed separately under "PRs without Jira ticket ID" so they can be validated manually.

## Project structure

```text
cmd/kup-helper/     entry point: flag parsing and orchestration
internal/config/    settings from environment variables and .env files
internal/github/    minimal GitHub REST API client (auth, PR search with pagination)
internal/report/    Jira ID extraction, PR aggregation, report printing
```
