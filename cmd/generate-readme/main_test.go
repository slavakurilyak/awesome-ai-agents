package main

import (
	"strings"
	"testing"

	"awesome-ai-agents/internal/contributions"
	"awesome-ai-agents/internal/projectdata"
)

func TestRenderProvenanceShowsRolesOwnerAndEscapedEvidence(t *testing.T) {
	project := projectdata.Project{
		ID: "example-agent-12345678",
		Sources: []projectdata.Source{{
			SourceURL:       "https://github.com/old-owner/example-agent",
			RepositoryOwner: "CurrentOrg", RepositoryOwnerURL: "https://github.com/CurrentOrg",
		}},
	}
	credit := contributions.Project{
		ProjectID: project.ID,
		SubmittedBy: []contributions.Credit{{
			Login: "founder", EvidenceURL: "https://github.com/slavakurilyak/awesome-ai-agents/issues/10?x=1&y=2",
			AcceptanceURL: "https://github.com/slavakurilyak/awesome-ai-agents/pull/11", FounderTeamClaim: true,
		}},
		MaintainedBy: []contributions.Credit{{
			Login: "maintainer", EvidenceURL: "https://github.com/example/example-agent/blob/main/MAINTAINERS.md", Status: "self_reported",
		}},
	}
	got := renderProvenance(project, credit)
	for _, want := range []string{"Submitted by:", "@founder", "self-reported founder/team", "Maintained by:", "@maintainer", "Repository owner:", "@CurrentOrg", "accepted PR", "&amp;y=2"} {
		if !strings.Contains(got, want) {
			t.Errorf("renderProvenance() missing %q in %s", want, got)
		}
	}
	if strings.Contains(got, "@old-owner") || strings.Contains(strings.ToLower(got), "catalogued by") {
		t.Fatalf("renderProvenance() used stale owner or added cataloguer credit: %s", got)
	}
}

func TestRenderProvenanceShowsOwnerWithoutInferringMaintainer(t *testing.T) {
	project := projectdata.Project{Sources: []projectdata.Source{{SourceURL: "https://github.com/Founder/project"}}}
	got := renderProvenance(project, contributions.Project{})
	for _, want := range []string{"Repository owner:", "Submitted by:", "Maintained by:", "not yet recovered from repository history", "not yet verified"} {
		if !strings.Contains(got, want) {
			t.Errorf("renderProvenance() missing explicit unknown state %q: %s", want, got)
		}
	}
	roles := strings.SplitN(got, "Repository owner:", 2)[0]
	if strings.Contains(roles, "@Founder") {
		t.Fatalf("renderProvenance() inferred a contributor role from repository ownership: %s", got)
	}
}

func TestHeadingSlugMatchesGitHubAnchors(t *testing.T) {
	for heading, want := range map[string]string{
		"Cal.ai":                   "calai",
		"GPT Researcher by Tavily": "gpt-researcher-by-tavily",
		"Agent-007":                "agent-007",
		"📈 Star Growth":            "-star-growth",
		"Upload-Post MCP":          "upload-post-mcp",
	} {
		if got := headingSlug(heading); got != want {
			t.Errorf("headingSlug(%q) = %q, want %q", heading, got, want)
		}
	}
}

func TestRenderCategoryLegendLinksProjectsUnderEachCategory(t *testing.T) {
	d := projectdata.Data{Agents: []projectdata.Project{
		{Project: "Today", Categories: []string{"AI Agents"}},
		{Project: "Zed <Agent>", Categories: []string{"AI Agents", "Terminal-Friendly"}},
		{Project: "Empty Category Owner", Categories: []string{"Unlisted"}},
	}}
	categories := []categoryInfo{{"AI Agents", "🤖"}, {"Terminal-Friendly", "💻"}, {"Unused", "❓"}}
	got := renderCategoryLegend(d, categories, []string{"Today"})
	for _, want := range []string{
		"<summary>🤖 AI Agents (2)</summary>",
		"<summary>💻 Terminal-Friendly (1)</summary>",
		"<summary>Unlisted (1)</summary>",
		`<a href="#today-1">Today</a>`,
		`<a href="#zed-agent">Zed &lt;Agent&gt;</a>`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("legend missing %q in:\n%s", want, got)
		}
	}
	if strings.Contains(got, "Unused") {
		t.Errorf("legend listed an empty category:\n%s", got)
	}
	if strings.Index(got, "AI Agents") > strings.Index(got, "Terminal-Friendly") {
		t.Error("categories are not in yaml order")
	}
}
