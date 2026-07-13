package api

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/jctanner/markovd/internal/models"
	"github.com/jctanner/markovd/internal/workflowdef"
	"gopkg.in/yaml.v3"
)

type diagramWorkflowFile struct {
	Entrypoint string            `yaml:"entrypoint"`
	Workflows  []diagramWorkflow `yaml:"workflows"`
}

type diagramWorkflow struct {
	Name  string        `yaml:"name"`
	Steps []diagramStep `yaml:"steps"`
}

type diagramStep struct {
	Name     string   `yaml:"name"`
	Type     string   `yaml:"type"`
	ForEach  string   `yaml:"for_each"`
	Workflow string   `yaml:"workflow"`
	When     string   `yaml:"when"`
	Rules    []string `yaml:"rules"`
}

type DiagramPosition struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

type DiagramNodeData struct {
	Label          string   `json:"label"`
	StepType       string   `json:"stepType"`
	Category       string   `json:"category"`
	ForEach        string   `json:"forEach,omitempty"`
	SubWorkflow    string   `json:"subWorkflow,omitempty"`
	When           string   `json:"when,omitempty"`
	Rules          []string `json:"rules,omitempty"`
	WorkflowGroup  string   `json:"workflowGroup"`
	InvocationPath string   `json:"invocationPath,omitempty"`
	CallerStep     string   `json:"callerStep,omitempty"`
	ReferenceKind  string   `json:"referenceKind,omitempty"`
}

type DiagramNode struct {
	ID       string                 `json:"id"`
	Type     string                 `json:"type"`
	Position DiagramPosition        `json:"position"`
	Data     DiagramNodeData        `json:"data"`
	ParentID string                 `json:"parentId,omitempty"`
	Extent   string                 `json:"extent,omitempty"`
	Style    map[string]interface{} `json:"style,omitempty"`
}

type DiagramEdge struct {
	ID       string                 `json:"id"`
	Source   string                 `json:"source"`
	Target   string                 `json:"target"`
	Type     string                 `json:"type"`
	Animated bool                   `json:"animated"`
	Style    map[string]interface{} `json:"style,omitempty"`
	Relation string                 `json:"relation,omitempty"`
}

type DiagramResponse struct {
	Nodes []DiagramNode `json:"nodes"`
	Edges []DiagramEdge `json:"edges"`
}

const (
	nodeW                 = 260.0
	nodeH                 = 72.0
	nodeGapY              = 60.0
	groupPadX             = 30.0
	groupPadTop           = 72.0
	groupPadBot           = 20.0
	colGap                = 80.0
	groupGapY             = 40.0
	maxDiagramInvocations = 256
	maxDiagramNodes       = 2000
)

type diagramInvocation struct {
	definitionName string
	definition     *diagramWorkflow
	path           string
	callerStep     string
	depth          int
	recursive      bool
	children       map[int]*diagramInvocation
	stepSegments   []string
	groupID        string
	stepIDs        []string
	entryID        string
	exitID         string
	subtreeHeight  float64
	childOffsets   map[int]float64
}

type diagramExpansionBudget struct {
	invocations int
	nodes       int
}

func stepCategory(s diagramStep) string {
	if len(s.Rules) > 0 || s.Type == "gate" || s.Type == "human_gate" {
		return "gate"
	}
	if s.ForEach != "" {
		return "foreach"
	}
	if s.Workflow != "" {
		return "subworkflow"
	}
	if s.When != "" {
		return "conditional"
	}
	return "normal"
}

func generateDiagramFromYAML(yamlContent string) (*DiagramResponse, error) {
	var wf diagramWorkflowFile
	if err := yaml.Unmarshal([]byte(yamlContent), &wf); err != nil {
		return nil, fmt.Errorf("parsing workflow YAML: %w", err)
	}
	return generateDiagram(wf)
}

func generateDiagramFromDefinition(def models.WorkflowDefinition) (*DiagramResponse, error) {
	def, err := workflowdef.Normalize(def.Kind, def.Files)
	if err != nil {
		return nil, err
	}
	if def.Kind == workflowdef.KindFile {
		return generateDiagramFromYAML(def.Files[0].Content)
	}

	var wf diagramWorkflowFile
	for _, f := range def.Files {
		switch f.Path {
		case "meta.yaml":
			var meta struct {
				Entrypoint string `yaml:"entrypoint"`
			}
			if err := yaml.Unmarshal([]byte(f.Content), &meta); err != nil {
				return nil, fmt.Errorf("parsing meta.yaml: %w", err)
			}
			wf.Entrypoint = meta.Entrypoint
		default:
			if len(f.Path) > len("workflows/") && f.Path[:len("workflows/")] == "workflows/" {
				var workflow diagramWorkflow
				if err := yaml.Unmarshal([]byte(f.Content), &workflow); err != nil {
					return nil, fmt.Errorf("parsing %s: %w", f.Path, err)
				}
				wf.Workflows = append(wf.Workflows, workflow)
			}
		}
	}
	return generateDiagram(wf)
}

func generateDiagram(wf diagramWorkflowFile) (*DiagramResponse, error) {
	wfMap := make(map[string]*diagramWorkflow)
	for i := range wf.Workflows {
		if strings.TrimSpace(wf.Workflows[i].Name) == "" {
			return nil, fmt.Errorf("workflow at index %d has no name", i)
		}
		if _, exists := wfMap[wf.Workflows[i].Name]; exists {
			return nil, fmt.Errorf("duplicate workflow definition %q", wf.Workflows[i].Name)
		}
		wfMap[wf.Workflows[i].Name] = &wf.Workflows[i]
	}

	entry := wf.Entrypoint
	if entry == "" && len(wf.Workflows) > 0 {
		entry = wf.Workflows[0].Name
	}

	if entry == "" {
		return &DiagramResponse{Nodes: []DiagramNode{}, Edges: []DiagramEdge{}}, nil
	}

	budget := &diagramExpansionBudget{}
	root, err := buildDiagramInvocation(wfMap, entry, invocationSegment(entry), "", 0, map[string]bool{}, budget)
	if err != nil {
		return nil, err
	}
	measureDiagramInvocation(root)

	var nodes []DiagramNode
	var edges []DiagramEdge
	layoutDiagramInvocation(root, 0, 0, &nodes, &edges)

	if nodes == nil {
		nodes = []DiagramNode{}
	}
	if edges == nil {
		edges = []DiagramEdge{}
	}
	return &DiagramResponse{Nodes: nodes, Edges: edges}, nil
}

func buildDiagramInvocation(
	wfMap map[string]*diagramWorkflow,
	name string,
	path string,
	callerStep string,
	depth int,
	ancestry map[string]bool,
	budget *diagramExpansionBudget,
) (*diagramInvocation, error) {
	definition := wfMap[name]
	if definition == nil {
		return nil, fmt.Errorf("workflow %q referenced by %q is not defined", name, path)
	}
	budget.invocations++
	if budget.invocations > maxDiagramInvocations {
		return nil, fmt.Errorf("workflow diagram exceeds invocation limit of %d at %q", maxDiagramInvocations, path)
	}
	stepNodeCount := len(definition.Steps)
	if stepNodeCount == 0 || ancestry[name] {
		stepNodeCount = 1
	}
	budget.nodes += 1 + stepNodeCount
	if budget.nodes > maxDiagramNodes {
		return nil, fmt.Errorf("workflow diagram exceeds node limit of %d at %q", maxDiagramNodes, path)
	}

	invocation := &diagramInvocation{
		definitionName: name,
		definition:     definition,
		path:           path,
		callerStep:     callerStep,
		depth:          depth,
		recursive:      ancestry[name],
		children:       make(map[int]*diagramInvocation),
		childOffsets:   make(map[int]float64),
		groupID:        "group:" + path,
	}
	if invocation.recursive {
		invocation.entryID = "reference:" + path
		invocation.exitID = invocation.entryID
		return invocation, nil
	}
	if len(definition.Steps) == 0 {
		invocation.entryID = "empty:" + path
		invocation.exitID = invocation.entryID
		return invocation, nil
	}

	ancestry[name] = true
	defer delete(ancestry, name)
	occurrences := make(map[string]int)
	for i, step := range definition.Steps {
		base := invocationSegment(step.Name)
		occurrences[base]++
		segment := base
		if occurrences[base] > 1 {
			segment = fmt.Sprintf("%s~%d", base, occurrences[base])
		}
		invocation.stepSegments = append(invocation.stepSegments, segment)
		invocation.stepIDs = append(invocation.stepIDs, "step:"+path+"/"+segment)
		if step.Workflow == "" {
			continue
		}
		childPath := path + "/" + segment + "@" + invocationSegment(step.Workflow)
		child, err := buildDiagramInvocation(wfMap, step.Workflow, childPath, step.Name, depth+1, ancestry, budget)
		if err != nil {
			return nil, err
		}
		invocation.children[i] = child
	}
	invocation.entryID = invocation.stepIDs[0]
	last := len(invocation.stepIDs) - 1
	invocation.exitID = invocation.stepIDs[last]
	if child := invocation.children[last]; child != nil {
		invocation.exitID = child.exitID
	}
	return invocation, nil
}

func invocationSegment(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		value = "step"
	}
	escaped := url.PathEscape(value)
	return strings.NewReplacer("~", "%7E", "@", "%40").Replace(escaped)
}

func invocationGroupHeight(invocation *diagramInvocation) float64 {
	n := len(invocation.definition.Steps)
	if invocation.recursive || n == 0 {
		n = 1
	}
	return groupPadTop + float64(n)*nodeH + float64(n-1)*nodeGapY + groupPadBot
}

func measureDiagramInvocation(invocation *diagramInvocation) float64 {
	height := invocationGroupHeight(invocation)
	cursor := 0.0
	for i := range invocation.definition.Steps {
		child := invocation.children[i]
		if child == nil {
			continue
		}
		childHeight := measureDiagramInvocation(child)
		desired := groupPadTop + float64(i)*(nodeH+nodeGapY)
		if cursor > desired {
			desired = cursor
		}
		invocation.childOffsets[i] = desired
		cursor = desired + childHeight + groupGapY
		if cursor-groupGapY > height {
			height = cursor - groupGapY
		}
	}
	invocation.subtreeHeight = height
	return height
}

func layoutDiagramInvocation(invocation *diagramInvocation, x, y float64, nodes *[]DiagramNode, edges *[]DiagramEdge) {
	groupHeight := invocationGroupHeight(invocation)
	*nodes = append(*nodes, DiagramNode{
		ID:       invocation.groupID,
		Type:     "group",
		Position: DiagramPosition{X: x, Y: y},
		Data: DiagramNodeData{
			Label:          invocation.definitionName,
			WorkflowGroup:  invocation.definitionName,
			Category:       "group",
			InvocationPath: invocation.path,
			CallerStep:     invocation.callerStep,
		},
		Style: map[string]interface{}{"width": nodeW + 2*groupPadX, "height": groupHeight},
	})

	if invocation.recursive || len(invocation.definition.Steps) == 0 {
		kind := "empty"
		label := "Empty workflow"
		if invocation.recursive {
			kind = "recursive"
			label = "Recursive reference"
		}
		*nodes = append(*nodes, DiagramNode{
			ID:       invocation.entryID,
			Type:     "workflowReference",
			Position: DiagramPosition{X: groupPadX, Y: groupPadTop},
			Data: DiagramNodeData{
				Label:          label,
				Category:       kind,
				WorkflowGroup:  invocation.definitionName,
				InvocationPath: invocation.path,
				CallerStep:     invocation.callerStep,
				ReferenceKind:  kind,
			},
			ParentID: invocation.groupID,
			Extent:   "parent",
		})
		return
	}

	for i, step := range invocation.definition.Steps {
		*nodes = append(*nodes, DiagramNode{
			ID:       invocation.stepIDs[i],
			Type:     "workflowStep",
			Position: DiagramPosition{X: groupPadX, Y: groupPadTop + float64(i)*(nodeH+nodeGapY)},
			Data: DiagramNodeData{
				Label:          step.Name,
				StepType:       step.Type,
				Category:       stepCategory(step),
				ForEach:        step.ForEach,
				SubWorkflow:    step.Workflow,
				When:           step.When,
				Rules:          step.Rules,
				WorkflowGroup:  invocation.definitionName,
				InvocationPath: invocation.path,
				CallerStep:     invocation.callerStep,
			},
			ParentID: invocation.groupID,
			Extent:   "parent",
		})
	}

	for i := range invocation.definition.Steps {
		child := invocation.children[i]
		if child != nil {
			appendDiagramEdge(edges, "call", invocation.stepIDs[i], child.entryID)
			if i+1 < len(invocation.stepIDs) {
				appendDiagramEdge(edges, "return", child.exitID, invocation.stepIDs[i+1])
			}
		} else if i+1 < len(invocation.stepIDs) {
			appendDiagramEdge(edges, "sequence", invocation.stepIDs[i], invocation.stepIDs[i+1])
		}
	}

	colPitch := nodeW + 2*groupPadX + colGap
	for i := range invocation.definition.Steps {
		child := invocation.children[i]
		if child == nil {
			continue
		}
		layoutDiagramInvocation(child, x+colPitch, y+invocation.childOffsets[i], nodes, edges)
	}
}

func appendDiagramEdge(edges *[]DiagramEdge, relation, source, target string) {
	*edges = append(*edges, DiagramEdge{
		ID:       fmt.Sprintf("edge:%s:%s->%s", relation, source, target),
		Source:   source,
		Target:   target,
		Type:     "smoothstep",
		Relation: relation,
	})
}
