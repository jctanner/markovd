package api

import (
	"testing"

	"github.com/jctanner/markovd/internal/models"
	"github.com/jctanner/markovd/internal/workflowdef"
)

func TestGenerateDiagramFromDirectoryDefinition(t *testing.T) {
	diagram, err := generateDiagramFromDefinition(models.WorkflowDefinition{
		Kind: workflowdef.KindDirectory,
		Files: []models.WorkflowDefinitionFile{
			{Path: "meta.yaml", Content: "entrypoint: main\n"},
			{Path: "vars.yaml", Content: "{}\n"},
			{Path: "rules.yaml", Content: "[]\n"},
			{Path: "step_types.yaml", Content: "{}\n"},
			{Path: "workflows/main.yaml", Content: "name: main\nsteps:\n  - name: call-child\n    workflow: child\n"},
			{Path: "workflows/child.yaml", Content: "name: child\nsteps:\n  - name: done\n    type: shell_exec\n"},
		},
	})
	if err != nil {
		t.Fatalf("generateDiagramFromDefinition() error: %v", err)
	}

	groups := map[string]bool{}
	for _, node := range diagram.Nodes {
		if node.Data.Category == "group" {
			groups[node.Data.Label] = true
		}
	}
	for _, want := range []string{"main", "child"} {
		if !groups[want] {
			t.Fatalf("missing workflow group %q in nodes: %#v", want, diagram.Nodes)
		}
	}
}
