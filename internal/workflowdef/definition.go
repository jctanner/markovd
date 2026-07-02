package workflowdef

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jctanner/markovd/internal/models"
)

const (
	KindFile      = "file"
	KindDirectory = "directory"
)

var requiredDirectoryFiles = []string{
	"meta.yaml",
	"vars.yaml",
	"rules.yaml",
	"step_types.yaml",
}

func FromLegacyYAML(yamlContent string) models.WorkflowDefinition {
	return models.WorkflowDefinition{
		Kind: KindFile,
		Files: []models.WorkflowDefinitionFile{
			{Path: "workflow.yaml", Content: yamlContent},
		},
	}
}

func Normalize(kind string, files []models.WorkflowDefinitionFile) (models.WorkflowDefinition, error) {
	kind = strings.TrimSpace(kind)
	if kind == "" {
		kind = KindFile
	}
	if kind != KindFile && kind != KindDirectory {
		return models.WorkflowDefinition{}, fmt.Errorf("definition_kind must be %q or %q", KindFile, KindDirectory)
	}
	if len(files) == 0 {
		return models.WorkflowDefinition{}, errors.New("workflow definition requires at least one file")
	}

	seen := map[string]bool{}
	normalized := make([]models.WorkflowDefinitionFile, 0, len(files))
	for _, f := range files {
		path, err := CleanRelativePath(f.Path)
		if err != nil {
			return models.WorkflowDefinition{}, err
		}
		if seen[path] {
			return models.WorkflowDefinition{}, fmt.Errorf("duplicate workflow definition file path: %s", path)
		}
		seen[path] = true
		normalized = append(normalized, models.WorkflowDefinitionFile{
			Path:    path,
			Content: f.Content,
		})
	}
	sort.Slice(normalized, func(i, j int) bool {
		return normalized[i].Path < normalized[j].Path
	})

	def := models.WorkflowDefinition{Kind: kind, Files: normalized}
	if err := validateShape(def, seen); err != nil {
		return models.WorkflowDefinition{}, err
	}
	return def, nil
}

func CleanRelativePath(path string) (string, error) {
	path = strings.TrimSpace(strings.ReplaceAll(filepath.ToSlash(path), "\\", "/"))
	if path == "" || path == "." {
		return "", errors.New("workflow definition file path cannot be empty")
	}
	if filepath.IsAbs(path) || strings.HasPrefix(path, "/") || looksLikeWindowsAbs(path) {
		return "", fmt.Errorf("workflow definition file path must be relative: %s", path)
	}
	for _, part := range strings.Split(path, "/") {
		if part == ".." {
			return "", fmt.Errorf("workflow definition file path escapes root: %s", path)
		}
	}
	cleaned := filepath.ToSlash(filepath.Clean(path))
	if cleaned == "." || cleaned == "" {
		return "", errors.New("workflow definition file path cannot be empty")
	}
	if strings.HasSuffix(cleaned, "/") {
		return "", fmt.Errorf("workflow definition file path cannot be a directory: %s", path)
	}
	return cleaned, nil
}

func looksLikeWindowsAbs(path string) bool {
	return len(path) >= 3 && path[1] == ':' && path[2] == '/'
}

func validateShape(def models.WorkflowDefinition, seen map[string]bool) error {
	switch def.Kind {
	case KindFile:
		if len(def.Files) != 1 {
			return errors.New("file workflow definitions must contain exactly one YAML file")
		}
		if !isYAMLPath(def.Files[0].Path) {
			return fmt.Errorf("file workflow definition must use a .yaml or .yml file: %s", def.Files[0].Path)
		}
	case KindDirectory:
		for _, required := range requiredDirectoryFiles {
			if !seen[required] {
				return fmt.Errorf("missing required directory workflow file: %s", required)
			}
		}
		hasWorkflow := false
		for _, f := range def.Files {
			if strings.HasPrefix(f.Path, "workflows/") && isYAMLPath(f.Path) {
				hasWorkflow = true
				break
			}
		}
		if !hasWorkflow {
			return errors.New("directory workflow definitions must include at least one workflows/*.yaml file")
		}
	}
	return nil
}

func isYAMLPath(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".yaml" || ext == ".yml"
}

func MarshalFiles(files []models.WorkflowDefinitionFile) (string, error) {
	data, err := json.Marshal(files)
	if err != nil {
		return "", fmt.Errorf("marshalling workflow definition files: %w", err)
	}
	return string(data), nil
}

func UnmarshalFiles(raw string) ([]models.WorkflowDefinitionFile, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}
	var files []models.WorkflowDefinitionFile
	if err := json.Unmarshal([]byte(raw), &files); err != nil {
		return nil, fmt.Errorf("unmarshalling workflow definition files: %w", err)
	}
	return files, nil
}

func LegacyYAML(def models.WorkflowDefinition) string {
	if def.Kind == KindFile && len(def.Files) == 1 {
		return def.Files[0].Content
	}
	return ""
}

type Materialized struct {
	Path    string
	Cleanup func()
}

func Materialize(def models.WorkflowDefinition) (*Materialized, error) {
	def, err := Normalize(def.Kind, def.Files)
	if err != nil {
		return nil, err
	}
	switch def.Kind {
	case KindFile:
		tmpFile, err := os.CreateTemp("", "markov-workflow-*.yaml")
		if err != nil {
			return nil, fmt.Errorf("creating workflow temp file: %w", err)
		}
		if _, err := tmpFile.WriteString(def.Files[0].Content); err != nil {
			name := tmpFile.Name()
			tmpFile.Close()
			os.Remove(name)
			return nil, fmt.Errorf("writing workflow temp file: %w", err)
		}
		if err := tmpFile.Close(); err != nil {
			name := tmpFile.Name()
			os.Remove(name)
			return nil, fmt.Errorf("closing workflow temp file: %w", err)
		}
		name := tmpFile.Name()
		return &Materialized{
			Path: name,
			Cleanup: func() {
				_ = os.Remove(name)
			},
		}, nil
	case KindDirectory:
		tmpDir, err := os.MkdirTemp("", "markov-workflow-*")
		if err != nil {
			return nil, fmt.Errorf("creating workflow temp directory: %w", err)
		}
		for _, f := range def.Files {
			target := filepath.Join(tmpDir, filepath.FromSlash(f.Path))
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				os.RemoveAll(tmpDir)
				return nil, fmt.Errorf("creating workflow temp directory path %s: %w", f.Path, err)
			}
			if err := os.WriteFile(target, []byte(f.Content), 0644); err != nil {
				os.RemoveAll(tmpDir)
				return nil, fmt.Errorf("writing workflow temp file %s: %w", f.Path, err)
			}
		}
		return &Materialized{
			Path: tmpDir,
			Cleanup: func() {
				_ = os.RemoveAll(tmpDir)
			},
		}, nil
	default:
		return nil, fmt.Errorf("unsupported workflow definition kind: %s", def.Kind)
	}
}

func ValidateWithMarkov(ctx context.Context, markovBin string, def models.WorkflowDefinition) error {
	if strings.TrimSpace(markovBin) == "" {
		markovBin = "markov"
	}
	m, err := Materialize(def)
	if err != nil {
		return err
	}
	defer m.Cleanup()

	cmd := exec.CommandContext(ctx, markovBin, "validate", m.Path)
	output, err := cmd.CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(output))
		if msg == "" {
			msg = err.Error()
		}
		return fmt.Errorf("markov validate failed: %s", msg)
	}
	return nil
}
