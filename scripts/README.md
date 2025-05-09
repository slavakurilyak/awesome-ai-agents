# Scripts

This directory contains utility scripts for the Awesome AI Agents project.

## Available Scripts

### `01-generate-json.py`
This script generates the `awesome-agents.json` file from the `awesome-agents.yaml` and `awesome-categories.yaml` files.

**Usage:**
```shell
python scripts/01-generate-json.py
````

### `02-update-github-stars.py`

This script updates the GitHub stars count for each project listed in `awesome-agents.json`. It requires a GitHub Personal Access Token with `public_repo` scope, which should be set as an environment variable `GITHUB_TOKEN`.

__Usage:__

```shell
# Ensure GITHUB_TOKEN environment variable is set
# export GITHUB_TOKEN="your_github_pat"
python scripts/02-update-github-stars.py
```

### `03-generate-readme.py`

This script generates the main `README.md` file for the project using the `README.template.md` and the data from `awesome-agents.json`.

__Usage:__

```shell
python scripts/03-generate-readme.py
```

## Workflow

To update all data and regenerate the main project README, run the scripts in the following order:

1. __Generate JSON data:__
   ```shell
   python scripts/01-generate-json.py
   ```
2. __Update GitHub stars (ensure `GITHUB_TOKEN` is set):__
   ```shell
   # export GITHUB_TOKEN="your_github_pat" # Uncomment and set if not already set
   python scripts/02-update-github-stars.py
   ```
3. __Generate README:__
   ```shell
   python scripts/03-generate-readme.py
   ```
