package projectdata

import (
	"encoding/json"
	"testing"
)

func TestValidateAcceptsZeroStarProjectWithVerifiableLink(t *testing.T) {
	description := "An agent with observable behavior."
	d := Data{Agents: []Project{{
		ID:      "new-agent-12345678",
		Project: "New Agent", ProjectDescription: &description, HasPublicRepository: true,
		Categories: []string{"AI Agents"}, Sources: []Source{{Source: "github", SourceURL: "https://github.com/example/new-agent", RepositoryStatus: "active", RepositoryCheckedAt: "2026-09-24T12:00:00Z"}},
	}}}
	if err := Validate(d); err != nil {
		t.Fatalf("Validate() rejected a valid zero-star project: %v", err)
	}
}

func TestValidateRejectsMissingDescription(t *testing.T) {
	d := Data{Agents: []Project{{
		ID:      "missing-description-12345678",
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
		ID:      "bad-link-12345678",
		Project: "Bad Link", ProjectDescription: &description, Categories: []string{"AI Agents"},
		Sources: []Source{{Source: "website", SourceURL: "not a URL"}},
	}}}
	if err := Validate(d); err == nil {
		t.Fatal("Validate() accepted an invalid source URL")
	}
}

func TestNewProjectIDIsStableAcrossRepositoryOwnerChanges(t *testing.T) {
	p := Project{Project: "Example Agent", Sources: []Source{{SourceURL: "https://github.com/Founder/example"}}}
	if !EnsureProjectID(&p) {
		t.Fatal("expected an ID to be assigned")
	}
	id := p.ID
	p.Sources[0].SourceURL = "https://github.com/Company/example"
	if EnsureProjectID(&p) || p.ID != id {
		t.Fatal("an existing project ID should be retained after an owner transfer")
	}
}

func TestProjectUnmarshalAcceptsLegacyPublicRepositoryKey(t *testing.T) {
	var legacy, current Project
	if err := json.Unmarshal([]byte(`{"project":"A","project_is_open_source":true}`), &legacy); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal([]byte(`{"project":"B","has_public_repository":true}`), &current); err != nil {
		t.Fatal(err)
	}
	if !legacy.HasPublicRepository || !current.HasPublicRepository {
		t.Fatalf("legacy=%v current=%v, want both true", legacy.HasPublicRepository, current.HasPublicRepository)
	}
}
