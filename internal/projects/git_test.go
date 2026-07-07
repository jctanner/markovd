package projects

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jctanner/markovd/internal/workflowdef"
)

func TestListWorkflowDefinitions(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "standalone.yaml", "entrypoint: main\n")
	writeFile(t, root, "pipelines/dir/meta.yaml", "entrypoint: main\n")
	writeFile(t, root, "pipelines/dir/vars.yaml", "{}\n")
	writeFile(t, root, "pipelines/dir/rules.yaml", "[]\n")
	writeFile(t, root, "pipelines/dir/step_types.yaml", "{}\n")
	writeFile(t, root, "pipelines/dir/workflows/main.yaml", "name: main\nsteps: []\n")

	defs, err := ListWorkflowDefinitions(root)
	if err != nil {
		t.Fatalf("ListWorkflowDefinitions() error: %v", err)
	}
	got := map[string]string{}
	for _, d := range defs {
		got[d.Path] = d.Kind
	}
	if got["standalone.yaml"] != workflowdef.KindFile {
		t.Fatalf("standalone.yaml kind = %q, want file; defs=%v", got["standalone.yaml"], defs)
	}
	if got["pipelines/dir"] != workflowdef.KindDirectory {
		t.Fatalf("pipelines/dir kind = %q, want directory; defs=%v", got["pipelines/dir"], defs)
	}
	if _, ok := got["pipelines/dir/workflows/main.yaml"]; ok {
		t.Fatalf("internal directory workflow file listed separately: %v", defs)
	}
}

func TestListWorkflowDefinitionsUsesMetaYAMLAsDirectoryRoot(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "var/demos/end-to-end/meta.yaml", "entrypoint: main\n")
	writeFile(t, root, "var/demos/end-to-end/workflows/main.yaml", "name: main\nsteps: []\n")
	writeFile(t, root, "var/demos/end-to-end/vars/demo.yaml", "example: true\n")

	defs, err := ListWorkflowDefinitions(root)
	if err != nil {
		t.Fatalf("ListWorkflowDefinitions() error: %v", err)
	}
	got := map[string]string{}
	for _, d := range defs {
		got[d.Path] = d.Kind
	}
	if got["var/demos/end-to-end"] != workflowdef.KindDirectory {
		t.Fatalf("var/demos/end-to-end kind = %q, want directory; defs=%v", got["var/demos/end-to-end"], defs)
	}
	for _, file := range []string{
		"var/demos/end-to-end/meta.yaml",
		"var/demos/end-to-end/workflows/main.yaml",
		"var/demos/end-to-end/vars/demo.yaml",
	} {
		if _, ok := got[file]; ok {
			t.Fatalf("internal directory workflow file %q listed separately: %v", file, defs)
		}
	}
}

func TestReadWorkflowDefinitionDirectory(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "pipeline/meta.yaml", "entrypoint: main\n")
	writeFile(t, root, "pipeline/vars.yaml", "{}\n")
	writeFile(t, root, "pipeline/rules.yaml", "[]\n")
	writeFile(t, root, "pipeline/step_types.yaml", "{}\n")
	writeFile(t, root, "pipeline/workflows/main.yaml", "name: main\nsteps: []\n")

	def, err := ReadWorkflowDefinition(root, "pipeline", workflowdef.KindDirectory)
	if err != nil {
		t.Fatalf("ReadWorkflowDefinition() error: %v", err)
	}
	if def.Kind != workflowdef.KindDirectory {
		t.Fatalf("Kind = %q, want directory", def.Kind)
	}
	if len(def.Files) != 5 {
		t.Fatalf("len(Files) = %d, want 5", len(def.Files))
	}
}

func TestReadWorkflowDefinitionDirectoryWithStepTypesDirectory(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "pipeline/meta.yaml", "entrypoint: main\n")
	writeFile(t, root, "pipeline/vars.yaml", "{}\n")
	writeFile(t, root, "pipeline/rules.yaml", "[]\n")
	writeFile(t, root, "pipeline/step_types/shell.yaml", "echo_local:\n  base: shell_exec\n")
	writeFile(t, root, "pipeline/workflows/main.yaml", "name: main\nsteps: []\n")

	def, err := ReadWorkflowDefinition(root, "pipeline", workflowdef.KindDirectory)
	if err != nil {
		t.Fatalf("ReadWorkflowDefinition() error: %v", err)
	}
	if def.Kind != workflowdef.KindDirectory {
		t.Fatalf("Kind = %q, want directory", def.Kind)
	}
	if len(def.Files) != 5 {
		t.Fatalf("len(Files) = %d, want 5", len(def.Files))
	}
}

func TestReadWorkflowDefinitionRejectsTraversal(t *testing.T) {
	root := t.TempDir()
	if _, err := ReadWorkflowDefinition(root, "../escape.yaml", workflowdef.KindFile); err == nil {
		t.Fatal("ReadWorkflowDefinition() expected traversal error for file")
	}
	if _, err := ReadWorkflowDefinition(root, "../escape", workflowdef.KindDirectory); err == nil {
		t.Fatal("ReadWorkflowDefinition() expected traversal error for directory")
	}
}

func writeFile(t *testing.T, root, rel, content string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("MkdirAll(%s): %v", rel, err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("WriteFile(%s): %v", rel, err)
	}
}
