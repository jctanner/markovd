package api

import (
	"fmt"
	"math"
	"net/url"
	"sort"
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
	Name        string        `yaml:"name"`
	Description string        `yaml:"description"`
	Steps       []diagramStep `yaml:"steps"`
}

type diagramStep struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
	Type        string `yaml:"type"`
	ForEach     string `yaml:"for_each"`
	// Shown in the definition graph\'s details panel.
	ForEachWhen  string   `yaml:"for_each_when"`
	IgnoreErrors bool     `yaml:"ignore_errors"`
	FailedWhen   string   `yaml:"failed_when"`
	Workflow     string   `yaml:"workflow"`
	When         string   `yaml:"when"`
	Rules        []string `yaml:"rules"`
}

type DiagramPosition struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

type DiagramNodeData struct {
	Label          string   `json:"label"`
	Description    string   `json:"description,omitempty"`
	StepType       string   `json:"stepType"`
	Category       string   `json:"category"`
	ForEach        string   `json:"forEach,omitempty"`
	ForEachWhen    string   `json:"forEachWhen,omitempty"`
	IgnoreErrors   bool     `json:"ignoreErrors,omitempty"`
	FailedWhen     string   `json:"failedWhen,omitempty"`
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
	ID           string                 `json:"id"`
	Source       string                 `json:"source"`
	Target       string                 `json:"target"`
	SourceHandle string                 `json:"sourceHandle,omitempty"`
	TargetHandle string                 `json:"targetHandle,omitempty"`
	Type         string                 `json:"type"`
	Animated     bool                   `json:"animated"`
	Style        map[string]interface{} `json:"style,omitempty"`
	Relation     string                 `json:"relation,omitempty"`
	// Data carries routing hints: laneX, the x of the vertical segment for a call or return
	// edge (see assignEdgeLanes).
	Data map[string]interface{} `json:"data,omitempty"`
}

type DiagramResponse struct {
	Nodes []DiagramNode `json:"nodes"`
	Edges []DiagramEdge `json:"edges"`
}

const (
	nodeW                 = 260.0
	nodeH                 = 72.0
	nodeGapY              = 36.0
	groupPadX             = 30.0
	groupPadTop           = 90.0 // name, "via <caller>" and a one-line description
	groupPadBot           = 20.0
	colGap                = 80.0 // minimum; a gap widens to fit its edge lanes
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

	l := &diagramLayout{top: map[string]float64{}, column: map[string]int{}}
	layoutDiagramInvocation(root, 0, 0, l)
	assignEdgeLanes(l)
	nodes, edges := l.nodes, l.edges

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
		// A templated name (`submit-{{ test.arm }}`) is resolved only when the step runs, so the
		// diagram shows the call without expanding it.
		if step.Workflow == "" || strings.Contains(step.Workflow, "{{") || strings.Contains(step.Workflow, "{%") {
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
	// The invocation's own last step, even when it calls another workflow: that call returns
	// into it (see layoutDiagramInvocation), so every call and return edge stays between
	// adjacent columns (ADR-0006).
	invocation.exitID = invocation.stepIDs[last]
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
		// Align the child's first step with its caller, so the call edge is a straight line.
		desired := float64(i) * (nodeH + nodeGapY)
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

// diagramLayout collects nodes and edges, and where each node sits, so edges can be routed
// once everything is placed: groups get their x only after the gaps between columns are sized.
type diagramLayout struct {
	nodes  []DiagramNode
	edges  []DiagramEdge
	groups []groupColumn
	top    map[string]float64 // absolute y of each step or reference node
	column map[string]int     // column (call depth) of each step or reference node
}

type groupColumn struct {
	node   int
	column int
}

func layoutDiagramInvocation(invocation *diagramInvocation, column int, y float64, l *diagramLayout) {
	groupHeight := invocationGroupHeight(invocation)
	l.groups = append(l.groups, groupColumn{node: len(l.nodes), column: column})
	l.nodes = append(l.nodes, DiagramNode{
		ID:       invocation.groupID,
		Type:     "group",
		Position: DiagramPosition{X: 0, Y: y},
		Data: DiagramNodeData{
			Label:          invocation.definitionName,
			Description:    invocation.definition.Description,
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
		l.top[invocation.entryID] = y + groupPadTop
		l.column[invocation.entryID] = column
		l.nodes = append(l.nodes, DiagramNode{
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
		stepY := groupPadTop + float64(i)*(nodeH+nodeGapY)
		l.top[invocation.stepIDs[i]] = y + stepY
		l.column[invocation.stepIDs[i]] = column
		l.nodes = append(l.nodes, DiagramNode{
			ID:       invocation.stepIDs[i],
			Type:     "workflowStep",
			Position: DiagramPosition{X: groupPadX, Y: stepY},
			Data: DiagramNodeData{
				Label:          step.Name,
				Description:    step.Description,
				StepType:       step.Type,
				Category:       stepCategory(step),
				ForEach:        step.ForEach,
				ForEachWhen:    step.ForEachWhen,
				IgnoreErrors:   step.IgnoreErrors,
				FailedWhen:     step.FailedWhen,
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
			appendDiagramEdge(&l.edges, "call", invocation.stepIDs[i], child.entryID)
			// The child returns to the next step, or, when the call is this workflow's last
			// step, to the calling step itself (it completes when the child does).
			returnTo := invocation.stepIDs[i]
			if i+1 < len(invocation.stepIDs) {
				returnTo = invocation.stepIDs[i+1]
			}
			appendDiagramEdge(&l.edges, "return", child.exitID, returnTo)
		} else if i+1 < len(invocation.stepIDs) {
			appendDiagramEdge(&l.edges, "sequence", invocation.stepIDs[i], invocation.stepIDs[i+1])
		}
	}

	for i := range invocation.definition.Steps {
		child := invocation.children[i]
		if child == nil {
			continue
		}
		layoutDiagramInvocation(child, column+1, y+invocation.childOffsets[i], l)
	}
}

// Where call and return edges meet a step card, as a fraction of its height. The UI places the
// handles at the same heights (SemanticHandles), so a call and a return at one card don't share
// a point.
const (
	callHandleFrac   = 0.35
	returnHandleFrac = 0.65
	laneSpacing      = 16.0
	laneMargin       = 24.0
)

type laneEdge struct {
	edge   int
	gap    int     // the gap between column gap and gap+1
	lo, hi float64 // vertical extent of the edge's middle segment
	leftY  float64 // where it meets the left column
	rightY float64 // where it meets the right column
	lane   int
}

// assignEdgeLanes routes every call and return edge through its own lane in the gap it crosses,
// so no two edges share a vertical segment where their extents overlap, and orders the lanes in
// each gap to cut crossings. It returns the number of lanes per gap.
func assignEdgeLanes(l *diagramLayout) map[int]int {
	byGap := map[int][]*laneEdge{}
	for i, e := range l.edges {
		if e.Relation != "call" && e.Relation != "return" {
			continue
		}
		srcCol, tgtCol := l.column[e.Source], l.column[e.Target]
		frac := callHandleFrac
		if e.Relation == "return" {
			frac = returnHandleFrac
		}
		srcY := l.top[e.Source] + frac*nodeH
		tgtY := l.top[e.Target] + frac*nodeH
		le := &laneEdge{edge: i, lo: math.Min(srcY, tgtY), hi: math.Max(srcY, tgtY)}
		if srcCol <= tgtCol {
			le.gap, le.leftY, le.rightY = srcCol, srcY, tgtY
		} else {
			le.gap, le.leftY, le.rightY = tgtCol, tgtY, srcY
		}
		byGap[le.gap] = append(byGap[le.gap], le)
	}

	lanes := map[int]int{}
	for gap, list := range byGap {
		// Interval partitioning: longest-first keeps long edges on few lanes.
		sort.SliceStable(list, func(a, b int) bool {
			if list[a].lo != list[b].lo {
				return list[a].lo < list[b].lo
			}
			return list[a].hi-list[a].lo > list[b].hi-list[b].lo
		})
		var laneEnds []float64
		for _, le := range list {
			placed := false
			for k, end := range laneEnds {
				if le.lo > end+laneSpacing {
					le.lane, laneEnds[k], placed = k, le.hi, true
					break
				}
			}
			if !placed {
				le.lane = len(laneEnds)
				laneEnds = append(laneEnds, le.hi)
			}
		}
		order := bestLaneOrder(list, len(laneEnds))
		for _, le := range list {
			le.lane = order[le.lane]
		}
		lanes[gap] = len(laneEnds)
	}

	widths := map[int]float64{}
	for gap, n := range lanes {
		widths[gap] = gapWidth(n)
	}
	columnX := func(col int) float64 {
		x := 0.0
		for c := 0; c < col; c++ {
			w, ok := widths[c]
			if !ok {
				w = colGap
			}
			x += nodeW + 2*groupPadX + w
		}
		return x
	}
	for _, g := range l.groups {
		l.nodes[g.node].Position.X = columnX(g.column)
	}
	for gap, list := range byGap {
		start := columnX(gap) + nodeW + 2*groupPadX
		first := start + (widths[gap]-float64(lanes[gap]-1)*laneSpacing)/2
		for _, le := range list {
			e := &l.edges[le.edge]
			e.Type = "lane"
			e.Data = map[string]interface{}{"laneX": first + float64(le.lane)*laneSpacing}
		}
	}
	return lanes
}

func gapWidth(lanes int) float64 {
	return math.Max(colGap, 2*laneMargin+float64(lanes-1)*laneSpacing)
}

// bestLaneOrder maps each lane to a position, left to right, with the fewest crossings. A
// crossing is a horizontal segment passing another edge's vertical one: an edge's right-hand
// horizontal crosses every lane to its right whose extent contains it, and its left-hand
// horizontal every lane to its left. Small gaps are searched exhaustively.
func bestLaneOrder(list []*laneEdge, n int) []int {
	identity := make([]int, n)
	for i := range identity {
		identity[i] = i
	}
	if n <= 1 || n > 7 {
		return identity
	}
	crossings := func(pos []int) int {
		count := 0
		for _, a := range list {
			for _, b := range list {
				if a == b || pos[a.lane] >= pos[b.lane] {
					continue
				}
				// a is left of b.
				if a.rightY > b.lo && a.rightY < b.hi {
					count++
				}
				if b.leftY > a.lo && b.leftY < a.hi {
					count++
				}
			}
		}
		return count
	}
	best := append([]int(nil), identity...)
	bestCount := crossings(best)
	perm := append([]int(nil), identity...)
	var permute func(k int)
	permute = func(k int) {
		if k == len(perm) {
			if c := crossings(perm); c < bestCount {
				bestCount = c
				copy(best, perm)
			}
			return
		}
		for i := k; i < len(perm); i++ {
			perm[k], perm[i] = perm[i], perm[k]
			permute(k + 1)
			perm[k], perm[i] = perm[i], perm[k]
		}
	}
	permute(0)
	return best
}

func appendDiagramEdge(edges *[]DiagramEdge, relation, source, target string) {
	sourceHandle := "sequence-source"
	targetHandle := "sequence-target"
	switch relation {
	case "call":
		sourceHandle = "call-source"
		targetHandle = "call-target"
	case "return":
		sourceHandle = "return-source"
		targetHandle = "return-target"
	}
	*edges = append(*edges, DiagramEdge{
		ID:           fmt.Sprintf("edge:%s:%s->%s", relation, source, target),
		Source:       source,
		Target:       target,
		SourceHandle: sourceHandle,
		TargetHandle: targetHandle,
		Type:         "smoothstep",
		Relation:     relation,
	})
}
