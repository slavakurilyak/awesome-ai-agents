# Scripts

These Go commands maintain the Awesome AI Agents data and README. Run them from any working directory; each command locates the repository relative to its source file.

## Validate project data

`awesome-agents.json` is the single canonical project dataset. Validate project names, descriptions, categories, and source URLs before submitting a change.

```shell
go run ./cmd/validate-data
```

## Verify open-source eligibility

Checks each project's direct repository against GitHub, GitLab.com, or Codeberg metadata. A successful public repository check is required; no particular license is required.

```shell
go run ./cmd/verify-forge-repositories
```

The command fails closed on API errors and does not write partial verification results. Projects without a qualifying public repository are marked ineligible and must be removed before validation. `go run ./cmd/validate-data` independently performs live metadata checks and rejects any third-party project without a qualifying repository.

## Update GitHub stars

Fetches current repository stars and archived status for GitHub sources, updates `awesome-agents.json`, and appends or refreshes the UTC daily record in `github-stars-history.json`. `GITHUB_TOKEN` is optional and may be set to increase API limits.

```shell
GITHUB_TOKEN="your_github_pat" go run ./cmd/update-github-stars
```

## Generate README

Uses `awesome-agents.json`, `awesome-categories.yaml`, `github-stars-history.json`, and `README.template.md` to generate `README.md`, including top starred and seven-day rising projects.

```shell
go run ./cmd/generate-readme
```

## Full workflow

```shell
go run ./cmd/verify-forge-repositories
go run ./cmd/validate-data
go run ./cmd/update-github-stars
go run ./cmd/generate-readme
```
