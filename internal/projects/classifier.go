package projects

import (
	"bytes"
	"io"
	"strings"

	"gopkg.in/yaml.v3"
)

type standaloneWorkflowClassification struct {
	IsWorkflow bool
	Reason     string
}

const (
	classificationValid                = "valid"
	classificationEmpty                = "empty"
	classificationInvalidYAML          = "invalid_yaml"
	classificationMultipleDocuments    = "multiple_documents"
	classificationRootNotMapping       = "root_not_mapping"
	classificationDuplicateRequiredKey = "duplicate_required_key"
	classificationMissingEntrypoint    = "missing_entrypoint"
	classificationInvalidEntrypoint    = "invalid_entrypoint"
	classificationMissingWorkflows     = "missing_workflows"
	classificationInvalidWorkflows     = "invalid_workflows"
	classificationMissingWorkflowShape = "missing_workflow_shape"
	classificationEntrypointNotFound   = "entrypoint_not_found"
)

func classifyStandaloneWorkflowYAML(content []byte) standaloneWorkflowClassification {
	decoder := yaml.NewDecoder(bytes.NewReader(content))
	var document yaml.Node
	if err := decoder.Decode(&document); err != nil {
		if err == io.EOF {
			return standaloneWorkflowClassification{Reason: classificationEmpty}
		}
		return standaloneWorkflowClassification{Reason: classificationInvalidYAML}
	}

	var extra yaml.Node
	if err := decoder.Decode(&extra); err == nil {
		return standaloneWorkflowClassification{Reason: classificationMultipleDocuments}
	} else if err != io.EOF {
		return standaloneWorkflowClassification{Reason: classificationInvalidYAML}
	}

	if len(document.Content) != 1 || document.Content[0].Kind != yaml.MappingNode {
		return standaloneWorkflowClassification{Reason: classificationRootNotMapping}
	}
	root := document.Content[0]

	entrypointNode, duplicate := uniqueMappingValue(root, "entrypoint")
	if duplicate {
		return standaloneWorkflowClassification{Reason: classificationDuplicateRequiredKey}
	}
	if entrypointNode == nil {
		return standaloneWorkflowClassification{Reason: classificationMissingEntrypoint}
	}
	if entrypointNode.Kind != yaml.ScalarNode || strings.TrimSpace(entrypointNode.Value) == "" {
		return standaloneWorkflowClassification{Reason: classificationInvalidEntrypoint}
	}
	entrypoint := strings.TrimSpace(entrypointNode.Value)

	workflowsNode, duplicate := uniqueMappingValue(root, "workflows")
	if duplicate {
		return standaloneWorkflowClassification{Reason: classificationDuplicateRequiredKey}
	}
	if workflowsNode == nil {
		return standaloneWorkflowClassification{Reason: classificationMissingWorkflows}
	}
	if workflowsNode.Kind != yaml.SequenceNode || len(workflowsNode.Content) == 0 {
		return standaloneWorkflowClassification{Reason: classificationInvalidWorkflows}
	}

	hasWorkflowShape := false
	hasEntrypoint := false
	for _, workflowNode := range workflowsNode.Content {
		if workflowNode.Kind != yaml.MappingNode {
			continue
		}
		nameNode, duplicateName := uniqueMappingValue(workflowNode, "name")
		stepsNode, duplicateSteps := uniqueMappingValue(workflowNode, "steps")
		if duplicateName || duplicateSteps {
			return standaloneWorkflowClassification{Reason: classificationDuplicateRequiredKey}
		}
		if nameNode == nil || nameNode.Kind != yaml.ScalarNode || strings.TrimSpace(nameNode.Value) == "" {
			continue
		}
		if stepsNode != nil && stepsNode.Kind == yaml.SequenceNode {
			hasWorkflowShape = true
			if strings.TrimSpace(nameNode.Value) == entrypoint {
				hasEntrypoint = true
			}
		}
	}

	if !hasWorkflowShape {
		return standaloneWorkflowClassification{Reason: classificationMissingWorkflowShape}
	}
	if !hasEntrypoint {
		return standaloneWorkflowClassification{Reason: classificationEntrypointNotFound}
	}
	return standaloneWorkflowClassification{IsWorkflow: true, Reason: classificationValid}
}

func uniqueMappingValue(mapping *yaml.Node, key string) (*yaml.Node, bool) {
	var value *yaml.Node
	for i := 0; i+1 < len(mapping.Content); i += 2 {
		if mapping.Content[i].Value != key {
			continue
		}
		if value != nil {
			return nil, true
		}
		value = mapping.Content[i+1]
	}
	return value, false
}
