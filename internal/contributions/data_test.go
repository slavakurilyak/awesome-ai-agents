package contributions

import (
	"testing"

	"awesome-ai-agents/internal/projectdata"
)

func TestValidateAcceptedSubmissionAndMaintainerCredit(t *testing.T) {
	catalog := projectdata.Data{Agents: []projectdata.Project{{ID: "example-agent-12345678", Project: "Example Agent"}}}
	data := Data{Projects: []Project{{
		ProjectID: "example-agent-12345678",
		SubmittedBy: []Credit{{
			GitHubID: 42, Login: "founder", EvidenceURL: "https://github.com/slavakurilyak/awesome-ai-agents/issues/10",
			AcceptanceURL: "https://github.com/slavakurilyak/awesome-ai-agents/pull/11", At: "2026-01-02T03:04:05Z",
			Status: "accepted_submission", FounderTeamClaim: true,
		}},
		MaintainedBy: []Credit{{
			GitHubID: 42, Login: "founder", EvidenceURL: "https://github.com/slavakurilyak/awesome-ai-agents/issues/10",
			At: "2026-01-02T03:04:05Z", Status: "self_reported",
		}},
	}}}
	if err := Validate(catalog, data); err != nil {
		t.Fatalf("Validate() rejected valid contribution data: %v", err)
	}
}

func TestValidateRejectsCreditForUnknownProject(t *testing.T) {
	catalog := projectdata.Data{Agents: []projectdata.Project{{ID: "known-12345678", Project: "Known"}}}
	data := Data{Projects: []Project{{ProjectID: "unknown-12345678"}}}
	if err := Validate(catalog, data); err == nil {
		t.Fatal("Validate() accepted a contribution for a missing catalog project")
	}
}

func TestValidateRejectsUntrustedEvidenceScheme(t *testing.T) {
	catalog := projectdata.Data{Agents: []projectdata.Project{{ID: "example-agent-12345678", Project: "Example Agent"}}}
	data := Data{Projects: []Project{{ProjectID: "example-agent-12345678", SubmittedBy: []Credit{{
		GitHubID: 42, Login: "founder", EvidenceURL: "javascript:alert(1)",
		AcceptanceURL: "https://github.com/slavakurilyak/awesome-ai-agents/pull/11", At: "2026-01-02T03:04:05Z", Status: "accepted_submission",
	}}}}}
	if err := Validate(catalog, data); err == nil {
		t.Fatal("Validate() accepted a non-HTTPS evidence URL")
	}
}
