package api

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

type mermaidWorkflowFile struct {
	Entrypoint string            `yaml:"entrypoint"`
	Workflows  []mermaidWorkflow `yaml:"workflows"`
}

type mermaidWorkflow struct {
	Name  string        `yaml:"name"`
	Steps []mermaidStep `yaml:"steps"`
}

type mermaidStep struct {
	Name     string `yaml:"name"`
	Type     string `yaml:"type"`
	ForEach  string `yaml:"for_each"`
	Workflow string `yaml:"workflow"`
	When     string `yaml:"when"`
	Rules    []string `yaml:"rules"`
}

func generateMermaidFromYAML(yamlContent string) (string, error) {
	var wf mermaidWorkflowFile
	if err := yaml.Unmarshal([]byte(yamlContent), &wf); err != nil {
		return "", fmt.Errorf("parsing workflow YAML: %w", err)
	}

	wfMap := make(map[string]*mermaidWorkflow)
	for i := range wf.Workflows {
		wfMap[wf.Workflows[i].Name] = &wf.Workflows[i]
	}

	entry := wf.Entrypoint
	if entry == "" && len(wf.Workflows) > 0 {
		entry = wf.Workflows[0].Name
	}

	var b strings.Builder
	b.WriteString("flowchart TD\n")

	idGen := &mermaidIDGen{counter: 0}
	rendered := make(map[string]bool)
	renderWorkflow(&b, wfMap, entry, idGen, rendered, "")

	b.WriteString("\n")
	b.WriteString("    classDef gate fill:#fef3c7,stroke:#d97706\n")
	b.WriteString("    classDef foreach fill:#dbeafe,stroke:#3b82f6\n")
	b.WriteString("    classDef subwf fill:#ede9fe,stroke:#7c3aed\n")
	b.WriteString("    classDef conditional fill:#fce7f3,stroke:#db2777,stroke-dasharray:5 5\n")

	return b.String(), nil
}

type mermaidIDGen struct {
	counter int
}

func (g *mermaidIDGen) next(prefix string) string {
	g.counter++
	return fmt.Sprintf("%s%d", prefix, g.counter)
}

func renderWorkflow(b *strings.Builder, wfMap map[string]*mermaidWorkflow, name string, idGen *mermaidIDGen, rendered map[string]bool, indent string) []string {
	wf, ok := wfMap[name]
	if !ok || rendered[name] {
		return nil
	}
	rendered[name] = true

	sgID := idGen.next("wf")
	fmt.Fprintf(b, "%s    subgraph %s[\"%s\"]\n", indent, sgID, name)

	var nodeIDs []string
	for _, step := range wf.Steps {
		nodeID := idGen.next("s")
		label := step.Name
		shape := "[\"%s\"]"
		cssClass := ""

		if step.Rules != nil && len(step.Rules) > 0 {
			shape = "{\"%s\"}"
			cssClass = "gate"
		} else if step.Type == "gate" || step.Type == "human_gate" {
			shape = "{\"%s\"}"
			cssClass = "gate"
		}

		if step.ForEach != "" {
			label = fmt.Sprintf("%s\\n🔄 for_each", step.Name)
			cssClass = "foreach"
		}

		if step.Workflow != "" && step.ForEach == "" {
			label = fmt.Sprintf("%s\\n📎 %s", step.Name, step.Workflow)
			cssClass = "subwf"
		}

		if step.When != "" {
			if cssClass == "" {
				cssClass = "conditional"
			}
			label = fmt.Sprintf("%s\\n❓ when", label)
		}

		if step.Type != "" && step.Type != "gate" && step.Type != "human_gate" {
			label = fmt.Sprintf("%s\\n[%s]", label, step.Type)
		}

		fmt.Fprintf(b, "%s        %s"+shape+"\n", indent, nodeID, label)
		if cssClass != "" {
			fmt.Fprintf(b, "%s        class %s %s\n", indent, nodeID, cssClass)
		}
		nodeIDs = append(nodeIDs, nodeID)
	}

	for i := 0; i < len(nodeIDs)-1; i++ {
		fmt.Fprintf(b, "%s        %s --> %s\n", indent, nodeIDs[i], nodeIDs[i+1])
	}

	fmt.Fprintf(b, "%s    end\n", indent)

	for i, step := range wf.Steps {
		if step.Workflow != "" {
			childIDs := renderWorkflow(b, wfMap, step.Workflow, idGen, rendered, indent)
			if len(childIDs) > 0 && i < len(nodeIDs) {
				fmt.Fprintf(b, "%s    %s -.-> %s\n", indent, nodeIDs[i], childIDs[0])
			}
		}
	}

	return nodeIDs
}
