package api

import (
	"fmt"
	"math"
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

func TestGenerateDiagramAssignsSemanticEdgeHandles(t *testing.T) {
	diagram, err := generateDiagram(diagramWorkflowFile{
		Entrypoint: "main",
		Workflows: []diagramWorkflow{
			{Name: "main", Steps: []diagramStep{
				{Name: "before", Type: "shell_exec"},
				{Name: "child", Workflow: "child"},
				{Name: "after", Type: "shell_exec"},
			}},
			{Name: "child", Steps: []diagramStep{{Name: "work", Type: "shell_exec"}}},
		},
	})
	if err != nil {
		t.Fatalf("generateDiagram() error: %v", err)
	}

	want := map[string][2]string{
		"sequence": {"sequence-source", "sequence-target"},
		"call":     {"call-source", "call-target"},
		"return":   {"return-source", "return-target"},
	}
	for _, edge := range diagram.Edges {
		handles, ok := want[edge.Relation]
		if !ok {
			t.Fatalf("unexpected relation %q", edge.Relation)
		}
		if edge.SourceHandle != handles[0] || edge.TargetHandle != handles[1] {
			t.Errorf("%s handles = (%q, %q), want (%q, %q)", edge.Relation, edge.SourceHandle, edge.TargetHandle, handles[0], handles[1])
		}
	}
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

func TestGenerateDiagramReturnsStayBetweenAdjacentColumns(t *testing.T) {
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
	// leaf returns into the step that called it (parent's last step); parent returns from that
	// step to main's next step. No return skips a column (ADR-0006).
	assertEdge(t, diagram, "return", leafExit, parentEntry)
	assertEdge(t, diagram, "return", parentEntry, after)
	assertNoEdge(t, diagram, parentStep, after)
	assertNoEdge(t, diagram, leafExit, after)
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

func templatedDiagram(t *testing.T, mainYAML string) *DiagramResponse {
	t.Helper()
	diagram, err := generateDiagramFromDefinition(models.WorkflowDefinition{
		Kind: workflowdef.KindDirectory,
		Files: []models.WorkflowDefinitionFile{
			{Path: "meta.yaml", Content: "entrypoint: main\n"},
			{Path: "vars.yaml", Content: "{}\n"},
			{Path: "rules.yaml", Content: "[]\n"},
			{Path: "step_types.yaml", Content: "{}\n"},
			{Path: "workflows/main.yaml", Content: mainYAML},
			{Path: "workflows/a.yaml", Content: "name: submit-bash\nsteps:\n  - name: b\n    type: shell_exec\n"},
			{Path: "workflows/b.yaml", Content: "name: submit-workflow\nsteps:\n  - name: w\n    type: shell_exec\n"},
			{Path: "workflows/c.yaml", Content: "name: other\nsteps:\n  - name: o\n    type: shell_exec\n"},
		},
	})
	if err != nil {
		t.Fatalf("generateDiagramFromDefinition() error: %v", err)
	}
	return diagram
}

func nodeByLabel(d *DiagramResponse, label string) *DiagramNodeData {
	for i := range d.Nodes {
		if d.Nodes[i].Data.Label == label {
			return &d.Nodes[i].Data
		}
	}
	return nil
}

func TestTemplatedCallExpandsPatternMatchesAsAlternatives(t *testing.T) {
	d := templatedDiagram(t, "name: main\nsteps:\n  - name: submit\n    description: Submit the arm's job\n    workflow: \"submit-{{ test.arm }}\"\n  - name: after\n    type: shell_exec\n")
	submit := nodeByLabel(d, "submit")
	if submit == nil || submit.SubWorkflow != "submit-{{ test.arm }}" || submit.CandidatesFrom != "pattern" ||
		strings.Join(submit.WorkflowNames, ",") != "submit-bash,submit-workflow" {
		t.Fatalf("submit = %#v", submit)
	}
	for _, name := range []string{"submit-bash", "submit-workflow"} {
		g := nodeByLabel(d, name)
		if g == nil || g.AlternativeOf != "submit-{{ test.arm }}" {
			t.Fatalf("group %s = %#v", name, g)
		}
	}
	if nodeByLabel(d, "other") != nil {
		t.Fatal("a workflow that doesn't fit the template was drawn")
	}
	assertEdge(t, d, "call", "step:main/submit", "step:main/submit@submit-bash/b")
	assertEdge(t, d, "return", "step:main/submit@submit-workflow/w", "step:main/after")
	for _, e := range d.Edges {
		if e.Relation != "sequence" && e.Data["alternative"] != true {
			t.Fatalf("edge %s not marked alternative: %#v", e.ID, e.Data)
		}
		if e.Relation != "sequence" && e.Data["laneX"] == nil {
			t.Fatalf("edge %s lost its lane: %#v", e.ID, e.Data)
		}
	}
}

func TestTemplatedCallUsesWorkflowNames(t *testing.T) {
	d := templatedDiagram(t, "name: main\nsteps:\n  - name: submit\n    workflow: \"{{ target }}\"\n    workflow_names: [other, submit-bash]\n")
	submit := nodeByLabel(d, "submit")
	if submit == nil || submit.CandidatesFrom != "workflow_names" || strings.Join(submit.WorkflowNames, ",") != "other,submit-bash" {
		t.Fatalf("submit = %#v", submit)
	}
	if nodeByLabel(d, "submit-workflow") != nil {
		t.Fatal("an unlisted workflow was drawn")
	}
}

func TestFullyDynamicTemplateWithoutNamesIsUnresolved(t *testing.T) {
	d := templatedDiagram(t, "name: main\nsteps:\n  - name: submit\n    workflow: \"{{ target }}\"\n  - name: after\n    type: shell_exec\n")
	submit := nodeByLabel(d, "submit")
	if submit == nil || len(submit.WorkflowNames) != 0 || submit.CandidatesFrom != "pattern" {
		t.Fatalf("submit = %#v", submit)
	}
	for _, name := range []string{"submit-bash", "submit-workflow", "other"} {
		if nodeByLabel(d, name) != nil {
			t.Fatalf("%s drawn for a template with no fixed text", name)
		}
	}
	assertEdge(t, d, "sequence", "step:main/submit", "step:main/after")
}

func TestGenerateDiagramCarriesDescriptionsAndStepDetails(t *testing.T) {
	diagram, err := generateDiagramFromDefinition(models.WorkflowDefinition{
		Kind: workflowdef.KindDirectory,
		Files: []models.WorkflowDefinitionFile{
			{Path: "meta.yaml", Content: "entrypoint: main\n"},
			{Path: "vars.yaml", Content: "{}\n"},
			{Path: "rules.yaml", Content: "[]\n"},
			{Path: "step_types.yaml", Content: "{}\n"},
			{Path: "workflows/main.yaml", Content: `name: main
description: The whole benchmark
steps:
  - name: each
    description: One per item
    for_each: items
    for_each_when: "item.on"
    as: item
    type: shell_exec
    ignore_errors: true
    failed_when: "result.rc != 0"
`},
		},
	})
	if err != nil {
		t.Fatalf("generateDiagramFromDefinition() error: %v", err)
	}
	var group, step *DiagramNodeData
	for i := range diagram.Nodes {
		switch diagram.Nodes[i].Data.Label {
		case "main":
			group = &diagram.Nodes[i].Data
		case "each":
			step = &diagram.Nodes[i].Data
		}
	}
	if group == nil || group.Description != "The whole benchmark" {
		t.Fatalf("group = %#v", group)
	}
	if step == nil || step.Description != "One per item" || step.ForEachWhen != "item.on" || !step.IgnoreErrors || step.FailedWhen != "result.rc != 0" {
		t.Fatalf("step = %#v", step)
	}
}

// strat-workflow's shape: a long caller with two calls, the second nested three deep.
func benchmarkShapedDiagram(t *testing.T) *DiagramResponse {
	t.Helper()
	steps := func(names ...string) []diagramStep {
		out := make([]diagramStep, len(names))
		for i, n := range names {
			out[i] = diagramStep{Name: n, Type: "shell_exec"}
		}
		return out
	}
	main := steps("a", "b", "c", "d", "e", "wipe", "rounds", "summarize", "save", "review")
	main[5] = diagramStep{Name: "wipe", Workflow: "wipe"}
	main[6] = diagramStep{Name: "rounds", Workflow: "round", ForEach: "x"}
	round := []diagramStep{{Name: "tests", Workflow: "variants", ForEach: "x"}}
	variants := []diagramStep{{Name: "variants", Workflow: "run-test", ForEach: "x"}}
	diagram, err := generateDiagram(diagramWorkflowFile{
		Entrypoint: "main",
		Workflows: []diagramWorkflow{
			{Name: "main", Steps: main},
			{Name: "wipe", Steps: steps("w1", "w2", "w3", "w4")},
			{Name: "round", Steps: round},
			{Name: "variants", Steps: variants},
			{Name: "run-test", Steps: steps("r1", "r2", "r3", "r4", "r5", "r6", "r7", "r8")},
		},
	})
	if err != nil {
		t.Fatalf("generateDiagram() error: %v", err)
	}
	return diagram
}

func TestDiagramLanesKeepEdgesApart(t *testing.T) {
	diagram := benchmarkShapedDiagram(t)
	pos := absolutePositions(diagram)
	type seg struct {
		x, lo, hi float64
		id        string
	}
	var segs []seg
	for _, e := range diagram.Edges {
		if e.Relation == "sequence" {
			continue
		}
		if e.Type != "lane" {
			t.Fatalf("%s edge %s has type %q, want lane", e.Relation, e.ID, e.Type)
		}
		x, ok := e.Data["laneX"].(float64)
		if !ok {
			t.Fatalf("edge %s has no laneX: %#v", e.ID, e.Data)
		}
		frac := callHandleFrac
		if e.Relation == "return" {
			frac = returnHandleFrac
		}
		y1, y2 := pos[e.Source].Y+frac*nodeH, pos[e.Target].Y+frac*nodeH
		// The lane lies between the two columns, clear of both groups.
		left, right := math.Min(pos[e.Source].X, pos[e.Target].X), math.Max(pos[e.Source].X, pos[e.Target].X)
		if x <= left+nodeW+groupPadX || x >= right-groupPadX {
			t.Fatalf("edge %s lane x=%v is not in the gap between %v and %v", e.ID, x, left+nodeW+groupPadX, right-groupPadX)
		}
		segs = append(segs, seg{x: x, lo: math.Min(y1, y2), hi: math.Max(y1, y2), id: e.ID})
	}
	for i := range segs {
		for j := i + 1; j < len(segs); j++ {
			a, b := segs[i], segs[j]
			if a.x == b.x && a.lo <= b.hi && b.lo <= a.hi && a.hi > a.lo && b.hi > b.lo {
				t.Fatalf("edges share a vertical segment at x=%v: %s [%v,%v] and %s [%v,%v]", a.x, a.id, a.lo, a.hi, b.id, b.lo, b.hi)
			}
		}
	}
}

func TestDiagramAlignsChildEntryWithCaller(t *testing.T) {
	diagram := benchmarkShapedDiagram(t)
	pos := absolutePositions(diagram)
	// Each child entry lines up with its caller unless an earlier group in the same column is in
	// the way: here wipe's group pushes round below the rounds step.
	blocked := map[string]bool{"edge:call:step:main/rounds->step:main/rounds@round/tests": true}
	for _, e := range diagram.Edges {
		if e.Relation != "call" {
			continue
		}
		aligned := pos[e.Source].Y == pos[e.Target].Y
		if aligned == blocked[e.ID] {
			t.Errorf("call %s: caller at y=%v, child entry at y=%v (blocked=%v)", e.ID, pos[e.Source].Y, pos[e.Target].Y, blocked[e.ID])
		}
	}
	// Groups in one column must not overlap.
	type box struct{ x, top, bottom float64 }
	var groups []box
	for _, n := range diagram.Nodes {
		if n.Type == "group" {
			h := n.Style["height"].(float64)
			groups = append(groups, box{n.Position.X, n.Position.Y, n.Position.Y + h})
		}
	}
	for i := range groups {
		for j := i + 1; j < len(groups); j++ {
			a, b := groups[i], groups[j]
			if a.x == b.x && a.top < b.bottom && b.top < a.bottom {
				t.Fatalf("groups overlap: %#v and %#v", a, b)
			}
		}
	}
}

// absolutePositions resolves step positions (relative to their group) to canvas positions.
func absolutePositions(d *DiagramResponse) map[string]DiagramPosition {
	groups := map[string]DiagramPosition{}
	for _, n := range d.Nodes {
		if n.Type == "group" {
			groups[n.ID] = n.Position
		}
	}
	out := map[string]DiagramPosition{}
	for _, n := range d.Nodes {
		p := n.Position
		if g, ok := groups[n.ParentID]; ok {
			p = DiagramPosition{X: g.X + p.X, Y: g.Y + p.Y}
		}
		out[n.ID] = p
	}
	return out
}
