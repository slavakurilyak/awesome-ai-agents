# Scripts

These Go commands maintain the Awesome AI Agents data and README. Run them from any working directory; each command locates the repository relative to its source file.

## Validate project data

`awesome-agents.json` is the single canonical project dataset. Validate project names, descriptions, categories, and source URLs before submitting a change. For a one-project contribution, pass the exact project name so live forge requests stay scoped to that project.

```shell
go run ./cmd/validate-data --project "Project name"
```

## Verify open-source eligibility

Checks the named project's direct repository against GitHub, GitLab.com, or Codeberg metadata. A successful public repository check is required; no particular license is required.

```shell
go run ./cmd/verify-forge-repositories --project "Project name"
```

The command updates only the named project's verification metadata. It fails on API errors or when that project has no qualifying public repository. `go run ./cmd/validate-data --project "Project name"` checks local catalog structure and live metadata for the named project. Do not run full-catalog live checks for a one-project change.

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

## Per-project workflow

For one new project, run:

```shell
go run ./cmd/verify-forge-repositories --project "Project name"
go run ./cmd/validate-data --project "Project name"
go run ./cmd/generate-readme
```

The two forge commands also support `--all` for an explicit full-catalog audit. They require either `--project "Project name"` or `--all`; no-argument calls fail instead of unexpectedly checking the full catalog. Run the GitHub star updater separately when refreshing catalog-wide metrics.
