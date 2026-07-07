package workflowdef

import (
	"os"
	"path/filepath"
	"strings"
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

func TestNormalizeDirectoryAllowsStepTypesDirectory(t *testing.T) {
	def, err := Normalize(KindDirectory, []models.WorkflowDefinitionFile{
		{Path: "meta.yaml", Content: "entrypoint: main\n"},
		{Path: "vars.yaml", Content: "{}\n"},
		{Path: "rules.yaml", Content: "[]\n"},
		{Path: "step_types/shell.yaml", Content: "echo_local:\n  base: shell_exec\n"},
		{Path: "workflows/main.yaml", Content: "name: main\nsteps: []\n"},
	})
	if err != nil {
		t.Fatalf("Normalize() error: %v", err)
	}
	if def.Kind != KindDirectory {
		t.Fatalf("Kind = %q, want %q", def.Kind, KindDirectory)
	}
}

func TestNormalizeDirectoryRequiresStepTypesSource(t *testing.T) {
	_, err := Normalize(KindDirectory, []models.WorkflowDefinitionFile{
		{Path: "meta.yaml", Content: "entrypoint: main\n"},
		{Path: "vars.yaml", Content: "{}\n"},
		{Path: "rules.yaml", Content: "[]\n"},
		{Path: "workflows/main.yaml", Content: "name: main\nsteps: []\n"},
	})
	if err == nil {
		t.Fatal("Normalize() expected missing step types source error")
	}
	if !strings.Contains(err.Error(), "step_types.yaml") {
		t.Fatalf("error = %q, want step_types.yaml", err)
	}
}

func TestRuntimeCompatibleDefinitionSynthesizesStepTypesYAML(t *testing.T) {
	def, err := RuntimeCompatibleDefinition(models.WorkflowDefinition{
		Kind: KindDirectory,
		Files: []models.WorkflowDefinitionFile{
			{Path: "meta.yaml", Content: "entrypoint: main\n"},
			{Path: "vars.yaml", Content: "{}\n"},
			{Path: "rules.yaml", Content: "[]\n"},
			{Path: "step_types/shell.yaml", Content: "echo_local:\n  base: shell_exec\n"},
			{Path: "step_types/http.yaml", Content: "http_get:\n  base: http_request\n"},
			{Path: "workflows/main.yaml", Content: "name: main\nsteps: []\n"},
		},
	})
	if err != nil {
		t.Fatalf("RuntimeCompatibleDefinition() error: %v", err)
	}
	files := map[string]string{}
	for _, f := range def.Files {
		files[f.Path] = f.Content
	}
	if _, ok := files["step_types/shell.yaml"]; ok {
		t.Fatalf("runtime definition kept step_types/shell.yaml: %#v", files)
	}
	if _, ok := files["step_types/http.yaml"]; ok {
		t.Fatalf("runtime definition kept step_types/http.yaml: %#v", files)
	}
	stepTypes, ok := files["step_types.yaml"]
	if !ok {
		t.Fatalf("runtime definition missing step_types.yaml: %#v", files)
	}
	for _, want := range []string{"echo_local:", "http_get:"} {
		if !strings.Contains(stepTypes, want) {
			t.Fatalf("step_types.yaml = %q, want %q", stepTypes, want)
		}
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
