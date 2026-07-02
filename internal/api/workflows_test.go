package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jctanner/markovd/internal/models"
	"github.com/jctanner/markovd/internal/workflowdef"
)

func TestWorkflowDefinitionFromLegacyPayload(t *testing.T) {
	def, err := workflowDefinitionFromPayload("", nil, "entrypoint: main\n")
	if err != nil {
		t.Fatalf("workflowDefinitionFromPayload() error: %v", err)
	}
	if def.Kind != workflowdef.KindFile {
		t.Fatalf("Kind = %q, want file", def.Kind)
	}
	if len(def.Files) != 1 || def.Files[0].Path != "workflow.yaml" {
		t.Fatalf("Files = %#v, want one workflow.yaml file", def.Files)
	}
}

func TestWorkflowDefinitionFromDirectoryPayload(t *testing.T) {
	def, err := workflowDefinitionFromPayload(workflowdef.KindDirectory, validAPIDirectoryFiles(), "")
	if err != nil {
		t.Fatalf("workflowDefinitionFromPayload() error: %v", err)
	}
	if def.Kind != workflowdef.KindDirectory {
		t.Fatalf("Kind = %q, want directory", def.Kind)
	}
	if def.Files[0].Path != "meta.yaml" {
		t.Fatalf("Files were not normalized and sorted: %#v", def.Files)
	}
}

func TestHandleValidateWorkflowInvalidJSONReturnsBooleanValid(t *testing.T) {
	server := &Server{}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/workflows/validate", strings.NewReader("{"))
	rec := httptest.NewRecorder()

	server.handleValidateWorkflow(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
	var body struct {
		Valid any    `json:"valid"`
		Error string `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	valid, ok := body.Valid.(bool)
	if !ok || valid {
		t.Fatalf("valid = %#v, want boolean false", body.Valid)
	}
}

func validAPIDirectoryFiles() []models.WorkflowDefinitionFile {
	return []models.WorkflowDefinitionFile{
		{Path: "vars.yaml", Content: "{}\n"},
		{Path: "meta.yaml", Content: "entrypoint: main\n"},
		{Path: "rules.yaml", Content: "[]\n"},
		{Path: "step_types.yaml", Content: "{}\n"},
		{Path: "workflows/main.yaml", Content: "name: main\nsteps: []\n"},
	}
}
