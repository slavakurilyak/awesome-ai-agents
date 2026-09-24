# Contributing

Thanks for helping people discover useful open-source AI agent projects. Projects are eligible only when a public repository on GitHub, GitLab.com, or Codeberg can be verified and its default branch has at least one substantive, non-automated commit within the six months before review. No particular license or minimum star count is required. Hosted products without their own qualifying repository are not eligible.

## Recommend a project

Open a [project submission issue](https://github.com/slavakurilyak/awesome-ai-agents/issues/new?template=project-submission.yml). Issues are the preferred path for founders because they let the maintainer review the project before changing the dataset. Include:

- Project name and a concise description of what it does.
- The direct URL of the project's public repository on GitHub, GitLab.com, or Codeberg. A website, organization profile, integration, or client library alone does not qualify.
- Evidence that the repository's default branch has received a substantive, non-automated commit within the last six months. Maintainers verify this from commit history; repository profile updates, stars, and bot-only dependency or metadata changes do not count.
- How people use or interact with it, and what agent behavior it provides.
- Suggested existing category or categories, if known.
- A clear note when the project is new or has little usage history; stars are not an eligibility requirement.

The optional [Awesome AI Agents Curation skill](skills/awesome-ai-agents-curation/SKILL.md) helps founders prepare evidence-backed submissions and helps maintainers review issues and pull requests. It does not approve or publish entries.

## Submit a pull request

Pull requests are welcome for direct edits. `awesome-agents.json` is the canonical project dataset. Edit that file directly; do not use a separate YAML project list. Keep the existing JSON structure and add a project object with:

```json
{
  "project": "Project name",
  "project_description": "A concise, evidence-backed description of the agent and its useful behavior.",
  "project_is_open_source": true,
  "categories": ["AI Agents"],
  "sources": [
    {
      "source": "github",
      "source_url": "https://github.com/example/project",
      "repository_status": "active",
      "repository_checked_at": "<set by forge verifier>",
      "stars_last_updated": null
    }
  ]
}
```

Use an existing category from `awesome-categories.yaml`. Describe observable capabilities accurately; do not infer features from marketing claims alone. The `project_is_open_source` value is derived from a successful forge metadata check, not from the contributor's assertion. For a single-project change, run these targeted commands from the repository root before opening a pull request, replacing the placeholder with the exact `project` value:

```sh
go run ./cmd/verify-forge-repositories --project "Project name"
go run ./cmd/validate-data --project "Project name"
go run ./cmd/generate-readme
```

Only the named project's forge metadata is checked live; the validator also checks the catalog's local structure. These commands do not make live forge requests for unrelated projects. Only public repositories qualify, and no license field or license allowlist is used. GitHub, GitLab.com, and Codeberg are supported. A check failure or inaccessible repository is unverified and does not qualify. Keep star counts and repository health metadata to the existing automated workflow. To run a full-catalog audit, explicitly pass `--all` to either forge command; full-catalog live checks are not part of routine per-PR validation.

Maintainers make the final inclusion and categorization decision after reviewing fit, evidence, duplicates, and presentation. A submission or skill recommendation is not a promise of acceptance, placement, audience reach, or project growth.

## Maintainer review

For an individual submission, run `go run ./cmd/verify-forge-repositories --project "Project name"` and `go run ./cmd/validate-data --project "Project name"`; do not make live checks for unrelated catalog entries. Require a successful public-repository result on GitHub, GitLab.com, or Codeberg. No specific license is required. Check the repository's default-branch commit history and require at least one substantive, non-automated commit dated within the six months before review. Do not rely on the forge profile's `updated_at` timestamp, stars, or bot-only dependency/metadata changes. If the submission has no direct qualifying repository, the forge check fails, the repository is private or inaccessible, or the activity window is exceeded, it does not meet the contribution rules and may be closed. Then review project behavior, category fit, duplicates, and presentation. Run full-catalog live audits only when specifically requested, using `--all`.

The list is curated to help readers discover projects. Listings provide an opportunity for discovery, not a guaranteed outcome. For consulting on building or deploying AI agents, see [Hire Me](https://cal.com/slavakurilyak/discovery-call).
