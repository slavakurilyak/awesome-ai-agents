# Contributing

Thanks for helping people discover useful open-source AI agent projects. Projects are eligible only when a public repository on GitHub, GitLab.com, or Codeberg can be verified. No particular license or minimum star count is required. Hosted products without their own qualifying repository are not eligible.

## Recommend a project

Open a [project submission issue](https://github.com/slavakurilyak/awesome-ai-agents/issues/new?template=project-submission.yml). Issues are the preferred path for founders because they let the maintainer review the project before changing the dataset. Include:

- Project name and a concise description of what it does.
- The direct URL of the project's public repository on GitHub, GitLab.com, or Codeberg. A website, organization profile, integration, or client library alone does not qualify.
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

Use an existing category from `awesome-categories.yaml`. Describe observable capabilities accurately; do not infer features from marketing claims alone. The `project_is_open_source` value is derived from a successful forge metadata check, not from the contributor's assertion. Maintainers run `go run ./cmd/verify-forge-repositories`; only public repositories qualify, and no license field or license allowlist is used. GitHub, GitLab.com, and Codeberg are supported. A check failure or inaccessible repository is unverified and does not qualify. Keep star counts and repository health metadata to the existing automated workflow.

Before opening a pull request, run:

```sh
go run ./cmd/validate-data
```

This validator checks the current forge metadata for every third-party entry. Maintainers also run `go run ./cmd/verify-forge-repositories` to refresh the verification snapshot.

Maintainers make the final inclusion and categorization decision after reviewing fit, evidence, duplicates, and presentation. A submission or skill recommendation is not a promise of acceptance, placement, audience reach, or project growth.

## Maintainer review

Run `go run ./cmd/verify-forge-repositories` and require a successful public-repository result on GitHub, GitLab.com, or Codeberg for every entry. No specific license is required. If the submission has no direct qualifying repository, the forge check fails, or the repository is private or inaccessible, it does not meet the contribution rules and may be closed. Then review project behavior, category fit, duplicates, and presentation.

The list is curated to help readers discover projects. Listings provide an opportunity for discovery, not a guaranteed outcome. For consulting on building or deploying AI agents, see [Hire Me](https://cal.com/slavakurilyak/discovery-call).
