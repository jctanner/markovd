package api

import (
	"fmt"

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
	ID       string                 `json:"id"`
	Source   string                 `json:"source"`
	Target   string                 `json:"target"`
	Type     string                 `json:"type"`
	Animated bool                   `json:"animated"`
	Style    map[string]interface{} `json:"style,omitempty"`
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
	colGap      = 80.0
	groupGapY   = 40.0
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

	type wfPlacement struct {
		name       string
		column     int
		callerWf   string
		callerStep int
	}
	var placements []wfPlacement
	rendered := make(map[string]bool)

	var collect func(name string, col int, caller string, stepIdx int)
	collect = func(name string, col int, caller string, stepIdx int) {
		if _, ok := wfMap[name]; !ok || rendered[name] {
			return
		}
		rendered[name] = true
		placements = append(placements, wfPlacement{name, col, caller, stepIdx})
		for i, step := range wfMap[name].Steps {
			if step.Workflow != "" {
				collect(step.Workflow, col+1, name, i)
			}
		}
	}
	collect(entry, 0, "", 0)

	colPitch := nodeW + 2*groupPadX + colGap
	stepPitch := nodeH + nodeGapY

	type placedInfo struct {
		groupID string
		nodeIDs []string
		y       float64
	}
	placed := make(map[string]*placedInfo)
	colNextY := make(map[int]float64)
	idGen := &diagramIDGen{}

	var nodes []DiagramNode
	var edges []DiagramEdge

	for _, p := range placements {
		wfDef := wfMap[p.name]
		nSteps := len(wfDef.Steps)
		h := groupPadTop + groupPadBot
		if nSteps > 0 {
			h = groupPadTop + float64(nSteps)*nodeH + float64(nSteps-1)*nodeGapY + groupPadBot
		}

		var desiredY float64
		if parent := placed[p.callerWf]; parent != nil {
			desiredY = parent.y + groupPadTop + float64(p.callerStep)*stepPitch
		}
		if colNextY[p.column] > desiredY {
			desiredY = colNextY[p.column]
		}
		colNextY[p.column] = desiredY + h + groupGapY

		x := float64(p.column) * colPitch
		groupID := idGen.next("g")
		nodes = append(nodes, DiagramNode{
			ID:       groupID,
			Type:     "group",
			Position: DiagramPosition{X: x, Y: desiredY},
			Data:     DiagramNodeData{Label: p.name, WorkflowGroup: p.name, Category: "group"},
			Style:    map[string]interface{}{"width": nodeW + 2*groupPadX, "height": h},
		})

		var nodeIDs []string
		for i, step := range wfDef.Steps {
			nid := idGen.next("s")
			nodes = append(nodes, DiagramNode{
				ID:       nid,
				Type:     "workflowStep",
				Position: DiagramPosition{X: groupPadX, Y: groupPadTop + float64(i)*stepPitch},
				Data: DiagramNodeData{
					Label:         step.Name,
					StepType:      step.Type,
					Category:      stepCategory(step),
					ForEach:       step.ForEach,
					SubWorkflow:   step.Workflow,
					When:          step.When,
					Rules:         step.Rules,
					WorkflowGroup: p.name,
				},
				ParentID: groupID,
				Extent:   "parent",
			})
			nodeIDs = append(nodeIDs, nid)
		}

		for i := 0; i < len(nodeIDs)-1; i++ {
			edges = append(edges, DiagramEdge{
				ID:     fmt.Sprintf("%s->%s", nodeIDs[i], nodeIDs[i+1]),
				Source: nodeIDs[i],
				Target: nodeIDs[i+1],
				Type:   "smoothstep",
			})
		}

		placed[p.name] = &placedInfo{groupID: groupID, nodeIDs: nodeIDs, y: desiredY}
	}

	for _, p := range placements {
		wfDef := wfMap[p.name]
		info := placed[p.name]
		if info == nil {
			continue
		}
		for i, step := range wfDef.Steps {
			if step.Workflow == "" || i >= len(info.nodeIDs) {
				continue
			}
			target := placed[step.Workflow]
			if target == nil || len(target.nodeIDs) == 0 {
				continue
			}
			e := DiagramEdge{
				ID:     fmt.Sprintf("%s-.->%s", info.nodeIDs[i], target.nodeIDs[0]),
				Source: info.nodeIDs[i],
				Target: target.nodeIDs[0],
				Type:   "smoothstep",
				Style:  map[string]interface{}{"strokeDasharray": "6 3", "opacity": 0.6},
			}
			if step.Workflow == p.name {
				e.Animated = true
				e.Style["opacity"] = 0.4
			}
			edges = append(edges, e)
		}
	}

	if nodes == nil {
		nodes = []DiagramNode{}
	}
	if edges == nil {
		edges = []DiagramEdge{}
	}
	return &DiagramResponse{Nodes: nodes, Edges: edges}, nil
}
