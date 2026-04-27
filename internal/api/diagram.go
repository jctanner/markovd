package api

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

type diagramWorkflowFile struct {
	Entrypoint string           `yaml:"entrypoint"`
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
	Label         string   `json:"label"`
	StepType      string   `json:"stepType"`
	Category      string   `json:"category"`
	ForEach       string   `json:"forEach,omitempty"`
	SubWorkflow   string   `json:"subWorkflow,omitempty"`
	When          string   `json:"when,omitempty"`
	Rules         []string `json:"rules,omitempty"`
	WorkflowGroup string   `json:"workflowGroup"`
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
	ID        string                 `json:"id"`
	Source    string                 `json:"source"`
	Target    string                 `json:"target"`
	Type      string                 `json:"type"`
	Animated  bool                   `json:"animated"`
	Style     map[string]interface{} `json:"style,omitempty"`
}

type DiagramResponse struct {
	Nodes []DiagramNode `json:"nodes"`
	Edges []DiagramEdge `json:"edges"`
}

const (
	nodeW       = 260.0
	nodeH       = 72.0
	nodeGapY    = 60.0
	groupPadX   = 30.0
	groupPadTop = 50.0
	groupPadBot = 20.0
	subWfGapY   = 40.0
)

type diagramIDGen struct {
	counter int
}

func (g *diagramIDGen) next(prefix string) string {
	g.counter++
	return fmt.Sprintf("%s%d", prefix, g.counter)
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

	wfMap := make(map[string]*diagramWorkflow)
	for i := range wf.Workflows {
		wfMap[wf.Workflows[i].Name] = &wf.Workflows[i]
	}

	entry := wf.Entrypoint
	if entry == "" && len(wf.Workflows) > 0 {
		entry = wf.Workflows[0].Name
	}

	idGen := &diagramIDGen{counter: 0}
	rendered := make(map[string]bool)

	var nodes []DiagramNode
	var edges []DiagramEdge

	layoutWorkflow(&nodes, &edges, wfMap, entry, idGen, rendered, 0, 40, "")

	if nodes == nil {
		nodes = []DiagramNode{}
	}
	if edges == nil {
		edges = []DiagramEdge{}
	}

	return &DiagramResponse{Nodes: nodes, Edges: edges}, nil
}

type layoutResult struct {
	nodeIDs []string
	endY    float64
}

func layoutWorkflow(
	nodes *[]DiagramNode,
	edges *[]DiagramEdge,
	wfMap map[string]*diagramWorkflow,
	name string,
	idGen *diagramIDGen,
	rendered map[string]bool,
	startX, startY float64,
	parentGroupID string,
) *layoutResult {
	wf, ok := wfMap[name]
	if !ok || rendered[name] {
		return nil
	}
	rendered[name] = true

	isRoot := parentGroupID == ""
	groupID := ""
	stepOffsetX := startX
	stepOffsetY := startY

	if !isRoot {
		groupID = idGen.next("g")
		stepOffsetX = groupPadX
		stepOffsetY = groupPadTop
	}

	var nodeIDs []string
	y := stepOffsetY

	for _, step := range wf.Steps {
		nodeID := idGen.next("s")
		cat := stepCategory(step)

		node := DiagramNode{
			ID:   nodeID,
			Type: "workflowStep",
			Position: DiagramPosition{
				X: stepOffsetX,
				Y: y,
			},
			Data: DiagramNodeData{
				Label:         step.Name,
				StepType:      step.Type,
				Category:      cat,
				ForEach:       step.ForEach,
				SubWorkflow:   step.Workflow,
				When:          step.When,
				Rules:         step.Rules,
				WorkflowGroup: name,
			},
		}

		if groupID != "" {
			node.ParentID = groupID
			node.Extent = "parent"
		}

		*nodes = append(*nodes, node)
		nodeIDs = append(nodeIDs, nodeID)
		y += nodeH + nodeGapY
	}

	for i := 0; i < len(nodeIDs)-1; i++ {
		*edges = append(*edges, DiagramEdge{
			ID:     fmt.Sprintf("%s->%s", nodeIDs[i], nodeIDs[i+1]),
			Source: nodeIDs[i],
			Target: nodeIDs[i+1],
			Type:   "smoothstep",
		})
	}

	groupEndY := y - nodeGapY + groupPadBot
	if len(wf.Steps) == 0 {
		groupEndY = stepOffsetY + groupPadBot
	}

	if groupID != "" {
		groupNode := DiagramNode{
			ID:   groupID,
			Type: "group",
			Position: DiagramPosition{
				X: startX,
				Y: startY,
			},
			Data: DiagramNodeData{
				Label:         name,
				WorkflowGroup: name,
				Category:      "group",
			},
			Style: map[string]interface{}{
				"width":  nodeW + 2*groupPadX,
				"height": groupEndY,
			},
		}
		if parentGroupID != "" {
			groupNode.ParentID = parentGroupID
			groupNode.Extent = "parent"
		}
		*nodes = append(*nodes, groupNode)
	}

	absoluteEndY := startY + groupEndY
	if isRoot {
		absoluteEndY = y
	}

	subY := absoluteEndY + subWfGapY
	for i, step := range wf.Steps {
		if step.Workflow == "" {
			continue
		}
		childResult := layoutWorkflow(nodes, edges, wfMap, step.Workflow, idGen, rendered, startX, subY, "")
		if childResult != nil && len(childResult.nodeIDs) > 0 && i < len(nodeIDs) {
			*edges = append(*edges, DiagramEdge{
				ID:     fmt.Sprintf("%s-.->%s", nodeIDs[i], childResult.nodeIDs[0]),
				Source: nodeIDs[i],
				Target: childResult.nodeIDs[0],
				Type:   "smoothstep",
				Style: map[string]interface{}{
					"strokeDasharray": "6 3",
					"opacity":         "0.6",
				},
			})
			subY = childResult.endY + subWfGapY
		}
	}

	return &layoutResult{
		nodeIDs: nodeIDs,
		endY:    subY,
	}
}
