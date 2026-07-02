package workflowdef

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jctanner/markovd/internal/models"
)

func TestNormalizeFileDefinition(t *testing.T) {
	def, err := Normalize("", []models.WorkflowDefinitionFile{
		{Path: "pipeline.yaml", Content: "entrypoint: main"},
	})
	if err != nil {
		t.Fatalf("Normalize() error: %v", err)
	}
	if def.Kind != KindFile {
		t.Fatalf("Kind = %q, want %q", def.Kind, KindFile)
	}
	if def.Files[0].Path != "pipeline.yaml" {
		t.Fatalf("Path = %q, want pipeline.yaml", def.Files[0].Path)
	}
}

func TestNormalizeDirectoryDefinition(t *testing.T) {
	_, err := Normalize(KindDirectory, validDirectoryFiles())
	if err != nil {
		t.Fatalf("Normalize() error: %v", err)
	}
}

func TestNormalizeRejectsUnsafePaths(t *testing.T) {
	tests := []string{
		"/abs.yaml",
		"C:\\abs.yaml",
		"../escape.yaml",
		"..\\escape.yaml",
		"nested/../../escape.yaml",
		"workflows/../meta.yaml",
		"",
	}
	for _, path := range tests {
		_, err := Normalize(KindFile, []models.WorkflowDefinitionFile{{Path: path, Content: "x"}})
		if err == nil {
			t.Fatalf("Normalize(%q) expected error", path)
		}
	}
}

func TestNormalizeRejectsDuplicateCleanPaths(t *testing.T) {
	_, err := Normalize(KindDirectory, append(validDirectoryFiles(),
		models.WorkflowDefinitionFile{Path: "./meta.yaml", Content: "duplicate"},
	))
	if err == nil {
		t.Fatal("Normalize() expected duplicate path error")
	}
}

func TestNormalizeRequiresDirectoryFiles(t *testing.T) {
	_, err := Normalize(KindDirectory, []models.WorkflowDefinitionFile{
		{Path: "meta.yaml", Content: "entrypoint: main"},
	})
	if err == nil {
		t.Fatal("Normalize() expected missing required files error")
	}
}

func TestNormalizeRequiresDirectoryWorkflowFile(t *testing.T) {
	_, err := Normalize(KindDirectory, []models.WorkflowDefinitionFile{
		{Path: "meta.yaml", Content: "entrypoint: main"},
		{Path: "vars.yaml", Content: "{}\n"},
		{Path: "rules.yaml", Content: "[]\n"},
		{Path: "step_types.yaml", Content: "{}\n"},
	})
	if err == nil {
		t.Fatal("Normalize() expected missing workflows/*.yaml error")
	}
}

func TestMaterializeDirectory(t *testing.T) {
	def, err := Normalize(KindDirectory, validDirectoryFiles())
	if err != nil {
		t.Fatalf("Normalize() error: %v", err)
	}
	m, err := Materialize(def)
	if err != nil {
		t.Fatalf("Materialize() error: %v", err)
	}
	defer m.Cleanup()

	data, err := os.ReadFile(filepath.Join(m.Path, "workflows", "main.yaml"))
	if err != nil {
		t.Fatalf("reading materialized workflow: %v", err)
	}
	if string(data) == "" {
		t.Fatal("materialized workflow is empty")
	}
}

func validDirectoryFiles() []models.WorkflowDefinitionFile {
	return []models.WorkflowDefinitionFile{
		{Path: "meta.yaml", Content: "entrypoint: main\n"},
		{Path: "vars.yaml", Content: "greeting: hello\n"},
		{Path: "rules.yaml", Content: "[]\n"},
		{Path: "step_types.yaml", Content: "echo_local:\n  base: shell_exec\n"},
		{Path: "workflows/main.yaml", Content: "name: main\nsteps: []\n"},
	}
}
