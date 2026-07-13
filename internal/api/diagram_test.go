package api

import (
	"fmt"
	"strings"
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

func TestGenerateDiagramFromFileDefinition(t *testing.T) {
	diagram, err := generateDiagramFromDefinition(models.WorkflowDefinition{
		Kind: workflowdef.KindFile,
		Files: []models.WorkflowDefinitionFile{{
			Path: "workflow.yaml",
			Content: `entrypoint: main
workflows:
  - name: main
    steps:
      - name: child
        workflow: child
      - name: done
        type: shell_exec
  - name: child
    steps: []
`,
		}},
	})
	if err != nil {
		t.Fatalf("generateDiagramFromDefinition() error: %v", err)
	}
	assertEdge(t, diagram, "call", "step:main/child", "empty:main/child@child")
	assertEdge(t, diagram, "return", "empty:main/child@child", "step:main/done")
}

func TestGenerateDiagramExpandsRepeatedWorkflowByCallSite(t *testing.T) {
	diagram, err := generateDiagram(diagramWorkflowFile{
		Entrypoint: "main",
		Workflows: []diagramWorkflow{
			{Name: "main", Steps: []diagramStep{
				{Name: "call", Workflow: "child"},
				{Name: "between", Type: "shell_exec"},
				{Name: "call", Workflow: "child"},
				{Name: "after", Type: "shell_exec"},
			}},
			{Name: "child", Steps: []diagramStep{{Name: "done", Type: "shell_exec"}}},
		},
	})
	if err != nil {
		t.Fatalf("generateDiagram() error: %v", err)
	}

	childGroups := nodesMatching(diagram, func(node DiagramNode) bool {
		return node.Data.Category == "group" && node.Data.Label == "child"
	})
	if len(childGroups) != 2 {
		t.Fatalf("child groups = %d, want 2: %#v", len(childGroups), childGroups)
	}
	if childGroups[0].ID == childGroups[1].ID || childGroups[0].Data.InvocationPath == childGroups[1].Data.InvocationPath {
		t.Fatalf("repeated child groups do not have unique identities: %#v", childGroups)
	}

	firstCall := "step:main/call"
	between := "step:main/between"
	secondCall := "step:main/call~2"
	after := "step:main/after"
	firstChildExit := "step:main/call@child/done"
	secondChildExit := "step:main/call~2@child/done"
	assertEdge(t, diagram, "call", firstCall, firstChildExit)
	assertEdge(t, diagram, "return", firstChildExit, between)
	assertEdge(t, diagram, "sequence", between, secondCall)
	assertEdge(t, diagram, "call", secondCall, secondChildExit)
	assertEdge(t, diagram, "return", secondChildExit, after)
	assertNoEdge(t, diagram, firstCall, between)
	assertNoEdge(t, diagram, secondCall, after)
}

func TestGenerateDiagramPropagatesFinalNestedCallExit(t *testing.T) {
	diagram, err := generateDiagram(diagramWorkflowFile{
		Entrypoint: "main",
		Workflows: []diagramWorkflow{
			{Name: "main", Steps: []diagramStep{{Name: "parent", Workflow: "parent"}, {Name: "after", Type: "shell_exec"}}},
			{Name: "parent", Steps: []diagramStep{{Name: "leaf", Workflow: "leaf"}}},
			{Name: "leaf", Steps: []diagramStep{{Name: "done", Type: "shell_exec"}}},
		},
	})
	if err != nil {
		t.Fatalf("generateDiagram() error: %v", err)
	}

	parentStep := "step:main/parent"
	parentEntry := "step:main/parent@parent/leaf"
	leafExit := "step:main/parent@parent/leaf@leaf/done"
	after := "step:main/after"
	assertEdge(t, diagram, "call", parentStep, parentEntry)
	assertEdge(t, diagram, "call", parentEntry, leafExit)
	assertEdge(t, diagram, "return", leafExit, after)
	assertNoEdge(t, diagram, parentStep, after)
}

func TestGenerateDiagramForEachUsesOneTemplateAndJoin(t *testing.T) {
	diagram, err := generateDiagram(diagramWorkflowFile{
		Entrypoint: "main",
		Workflows: []diagramWorkflow{
			{Name: "main", Steps: []diagramStep{
				{Name: "fan-out", ForEach: "items", Workflow: "worker"},
				{Name: "joined", Type: "shell_exec"},
			}},
			{Name: "worker", Steps: []diagramStep{{Name: "work", Type: "shell_exec"}}},
		},
	})
	if err != nil {
		t.Fatalf("generateDiagram() error: %v", err)
	}

	workerGroups := nodesMatching(diagram, func(node DiagramNode) bool {
		return node.Data.Category == "group" && node.Data.Label == "worker"
	})
	if len(workerGroups) != 1 {
		t.Fatalf("worker groups = %d, want one static template", len(workerGroups))
	}
	assertEdge(t, diagram, "call", "step:main/fan-out", "step:main/fan-out@worker/work")
	assertEdge(t, diagram, "return", "step:main/fan-out@worker/work", "step:main/joined")
}

func TestGenerateDiagramConnectsEmptyAndRecursiveWorkflows(t *testing.T) {
	diagram, err := generateDiagram(diagramWorkflowFile{
		Entrypoint: "main",
		Workflows: []diagramWorkflow{
			{Name: "main", Steps: []diagramStep{
				{Name: "empty", Workflow: "empty"},
				{Name: "again", Workflow: "main"},
				{Name: "done", Type: "shell_exec"},
			}},
			{Name: "empty", Steps: []diagramStep{}},
		},
	})
	if err != nil {
		t.Fatalf("generateDiagram() error: %v", err)
	}

	emptyID := "empty:main/empty@empty"
	recursiveID := "reference:main/again@main"
	assertNode(t, diagram, emptyID, "workflowReference", "empty")
	assertNode(t, diagram, recursiveID, "workflowReference", "recursive")
	assertEdge(t, diagram, "call", "step:main/empty", emptyID)
	assertEdge(t, diagram, "return", emptyID, "step:main/again")
	assertEdge(t, diagram, "call", "step:main/again", recursiveID)
	assertEdge(t, diagram, "return", recursiveID, "step:main/done")
}

func TestGenerateDiagramStopsIndirectRecursion(t *testing.T) {
	diagram, err := generateDiagram(diagramWorkflowFile{
		Entrypoint: "a",
		Workflows: []diagramWorkflow{
			{Name: "a", Steps: []diagramStep{{Name: "to-b", Workflow: "b"}}},
			{Name: "b", Steps: []diagramStep{{Name: "to-a", Workflow: "a"}}},
		},
	})
	if err != nil {
		t.Fatalf("generateDiagram() error: %v", err)
	}
	assertNode(t, diagram, "reference:a/to-b@b/to-a@a", "workflowReference", "recursive")
	assertEdge(t, diagram, "call", "step:a/to-b@b/to-a", "reference:a/to-b@b/to-a@a")
}

func TestGenerateDiagramRejectsUnresolvedWorkflow(t *testing.T) {
	_, err := generateDiagram(diagramWorkflowFile{
		Entrypoint: "main",
		Workflows: []diagramWorkflow{{
			Name:  "main",
			Steps: []diagramStep{{Name: "missing", Workflow: "not-defined"}},
		}},
	})
	if err == nil || !strings.Contains(err.Error(), `workflow "not-defined"`) {
		t.Fatalf("generateDiagram() error = %v, want unresolved workflow error", err)
	}
}

func TestGenerateDiagramUsesStableIDsForDuplicateStepNames(t *testing.T) {
	wf := diagramWorkflowFile{
		Entrypoint: "main",
		Workflows: []diagramWorkflow{{
			Name: "main",
			Steps: []diagramStep{
				{Name: "same", Type: "shell_exec"},
				{Name: "same", Type: "shell_exec"},
				{Name: "same~2", Type: "shell_exec"},
			},
		}},
	}
	first, err := generateDiagram(wf)
	if err != nil {
		t.Fatalf("generateDiagram() first error: %v", err)
	}
	second, err := generateDiagram(wf)
	if err != nil {
		t.Fatalf("generateDiagram() second error: %v", err)
	}
	for _, id := range []string{"step:main/same", "step:main/same~2", "step:main/same%7E2"} {
		assertNode(t, first, id, "workflowStep", "normal")
		assertNode(t, second, id, "workflowStep", "normal")
	}
}

func TestGenerateDiagramEnforcesInvocationLimit(t *testing.T) {
	steps := make([]diagramStep, maxDiagramInvocations)
	for i := range steps {
		steps[i] = diagramStep{Name: "call-" + strings.Repeat("x", i%3), Workflow: "child"}
	}
	_, err := generateDiagram(diagramWorkflowFile{
		Entrypoint: "main",
		Workflows: []diagramWorkflow{
			{Name: "main", Steps: steps},
			{Name: "child", Steps: []diagramStep{{Name: "done"}}},
		},
	})
	if err == nil || !strings.Contains(err.Error(), "invocation limit") {
		t.Fatalf("generateDiagram() error = %v, want invocation limit error", err)
	}
}

func TestGenerateDiagramEnforcesNodeLimit(t *testing.T) {
	steps := make([]diagramStep, maxDiagramNodes)
	for i := range steps {
		steps[i] = diagramStep{Name: fmt.Sprintf("step-%d", i), Type: "shell_exec"}
	}
	_, err := generateDiagram(diagramWorkflowFile{
		Entrypoint: "main",
		Workflows:  []diagramWorkflow{{Name: "main", Steps: steps}},
	})
	if err == nil || !strings.Contains(err.Error(), "node limit") {
		t.Fatalf("generateDiagram() error = %v, want node limit error", err)
	}
}

func nodesMatching(diagram *DiagramResponse, predicate func(DiagramNode) bool) []DiagramNode {
	var result []DiagramNode
	for _, node := range diagram.Nodes {
		if predicate(node) {
			result = append(result, node)
		}
	}
	return result
}

func assertNode(t *testing.T, diagram *DiagramResponse, id, nodeType, category string) {
	t.Helper()
	for _, node := range diagram.Nodes {
		if node.ID == id {
			if node.Type != nodeType || node.Data.Category != category {
				t.Fatalf("node %q = type %q category %q, want type %q category %q", id, node.Type, node.Data.Category, nodeType, category)
			}
			return
		}
	}
	t.Fatalf("missing node %q", id)
}

func assertEdge(t *testing.T, diagram *DiagramResponse, relation, source, target string) {
	t.Helper()
	for _, edge := range diagram.Edges {
		if edge.Relation == relation && edge.Source == source && edge.Target == target {
			return
		}
	}
	t.Fatalf("missing %s edge %q -> %q; edges=%#v", relation, source, target, diagram.Edges)
}

func assertNoEdge(t *testing.T, diagram *DiagramResponse, source, target string) {
	t.Helper()
	for _, edge := range diagram.Edges {
		if edge.Source == source && edge.Target == target {
			t.Fatalf("unexpected edge %q -> %q: %#v", source, target, edge)
		}
	}
}
