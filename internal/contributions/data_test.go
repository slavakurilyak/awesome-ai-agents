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

func TestRequireSubmitterRejectsProjectWithoutCredit(t *testing.T) {
	catalog := projectdata.Data{Agents: []projectdata.Project{
		{ID: "credited-11111111", Project: "Credited"},
		{ID: "bare-22222222", Project: "Bare"},
	}}
	ledger := Data{Projects: []Project{{ProjectID: "credited-11111111", SubmittedBy: []Credit{{Login: "someone"}}}}}
	if err := RequireSubmitter(catalog, ledger, "credited"); err != nil {
		t.Fatalf("RequireSubmitter rejected a credited project: %v", err)
	}
	if err := RequireSubmitter(catalog, ledger, "Bare"); err == nil {
		t.Fatal("RequireSubmitter accepted a project without a submitter record")
	}
	if err := RequireSubmitter(catalog, ledger, "Missing"); err == nil {
		t.Fatal("RequireSubmitter accepted a project that is not in the catalog")
	}
}
