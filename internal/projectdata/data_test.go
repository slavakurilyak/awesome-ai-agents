package projectdata

import "testing"

func TestValidateAcceptsZeroStarProjectWithVerifiableLink(t *testing.T) {
	description := "An agent with observable behavior."
	d := Data{Agents: []Project{{
		Project: "New Agent", ProjectDescription: &description, ProjectIsOpenSource: true,
		Categories: []string{"AI Agents"}, Sources: []Source{{Source: "github", SourceURL: "https://github.com/example/new-agent", RepositoryStatus: "active", RepositoryCheckedAt: "2026-09-24T12:00:00Z"}},
	}}}
	if err := Validate(d); err != nil {
		t.Fatalf("Validate() rejected a valid zero-star project: %v", err)
	}
}

func TestValidateRejectsMissingDescription(t *testing.T) {
	d := Data{Agents: []Project{{
		Project: "Missing Description", Categories: []string{"AI Agents"},
		Sources: []Source{{Source: "website", SourceURL: "https://example.com"}},
	}}}
	if err := Validate(d); err == nil {
		t.Fatal("Validate() accepted a project without a description")
	}
}

func TestValidateRejectsUnverifiableSourceURL(t *testing.T) {
	description := "A project description."
	d := Data{Agents: []Project{{
		Project: "Bad Link", ProjectDescription: &description, Categories: []string{"AI Agents"},
		Sources: []Source{{Source: "website", SourceURL: "not a URL"}},
	}}}
	if err := Validate(d); err == nil {
		t.Fatal("Validate() accepted an invalid source URL")
	}
}
