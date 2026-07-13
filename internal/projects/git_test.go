package projects

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/jctanner/markovd/internal/workflowdef"
)

func TestListWorkflowDefinitions(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "standalone.yaml", "entrypoint: main\nworkflows:\n  - name: main\n    steps: []\n")
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

func TestClassifyStandaloneWorkflowYAML(t *testing.T) {
	tests := []struct {
		name       string
		content    string
		want       bool
		wantReason string
	}{
		{
			name:       "minimal workflow",
			content:    "entrypoint: main\nworkflows:\n  - name: main\n    steps: []\n",
			want:       true,
			wantReason: classificationValid,
		},
		{
			name:       "optional and unknown sections",
			content:    "entrypoint: main\nvars: {}\nrules: []\nstep_types: {}\nfuture: true\nworkflows:\n  - name: main\n    steps: []\n",
			want:       true,
			wantReason: classificationValid,
		},
		{name: "empty", content: "", wantReason: classificationEmpty},
		{name: "malformed", content: "entrypoint: [\n", wantReason: classificationInvalidYAML},
		{name: "multiple documents", content: "entrypoint: main\nworkflows: []\n---\nother: value\n", wantReason: classificationMultipleDocuments},
		{name: "sequence root", content: "- entrypoint\n- workflows\n", wantReason: classificationRootNotMapping},
		{name: "missing entrypoint", content: "workflows:\n  - name: main\n    steps: []\n", wantReason: classificationMissingEntrypoint},
		{name: "empty entrypoint", content: "entrypoint: ''\nworkflows:\n  - name: main\n    steps: []\n", wantReason: classificationInvalidEntrypoint},
		{name: "mapping entrypoint", content: "entrypoint: {name: main}\nworkflows:\n  - name: main\n    steps: []\n", wantReason: classificationInvalidEntrypoint},
		{name: "missing workflows", content: "entrypoint: main\n", wantReason: classificationMissingWorkflows},
		{name: "empty workflows", content: "entrypoint: main\nworkflows: []\n", wantReason: classificationInvalidWorkflows},
		{name: "mapping workflows", content: "entrypoint: main\nworkflows: {}\n", wantReason: classificationInvalidWorkflows},
		{name: "workflow fragment", content: "name: main\nsteps: []\n", wantReason: classificationMissingEntrypoint},
		{name: "workflow missing steps", content: "entrypoint: main\nworkflows:\n  - name: main\n", wantReason: classificationMissingWorkflowShape},
		{name: "steps wrong kind", content: "entrypoint: main\nworkflows:\n  - name: main\n    steps: {}\n", wantReason: classificationMissingWorkflowShape},
		{name: "entrypoint mismatch", content: "entrypoint: other\nworkflows:\n  - name: main\n    steps: []\n", wantReason: classificationEntrypointNotFound},
		{name: "entrypoint matches malformed workflow", content: "entrypoint: main\nworkflows:\n  - name: main\n  - name: helper\n    steps: []\n", wantReason: classificationEntrypointNotFound},
		{name: "duplicate entrypoint", content: "entrypoint: main\nentrypoint: other\nworkflows:\n  - name: main\n    steps: []\n", wantReason: classificationDuplicateRequiredKey},
		{name: "duplicate workflows", content: "entrypoint: main\nworkflows: []\nworkflows:\n  - name: main\n    steps: []\n", wantReason: classificationDuplicateRequiredKey},
		{name: "duplicate workflow name", content: "entrypoint: main\nworkflows:\n  - name: main\n    name: other\n    steps: []\n", wantReason: classificationDuplicateRequiredKey},
		{name: "kubernetes manifest", content: "apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: example\ndata: {}\n", wantReason: classificationMissingEntrypoint},
		{name: "ci workflow", content: "name: CI\non: [push]\njobs:\n  test:\n    runs-on: ubuntu-latest\n", wantReason: classificationMissingEntrypoint},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := classifyStandaloneWorkflowYAML([]byte(tt.content))
			if got.IsWorkflow != tt.want || got.Reason != tt.wantReason {
				t.Fatalf("classifyStandaloneWorkflowYAML() = %+v, want workflow=%v reason=%q", got, tt.want, tt.wantReason)
			}
		})
	}
}

func TestListWorkflowDefinitionsFiltersMixedRepository(t *testing.T) {
	root := t.TempDir()
	valid := "entrypoint: main\nworkflows:\n  - name: main\n    steps: []\n"
	writeFile(t, root, "a-config.yaml", "apiVersion: v1\nkind: ConfigMap\n")
	writeFile(t, root, "pipelines/zeta.yml", valid)
	writeFile(t, root, "pipelines/alpha.yaml", "future: true\n"+valid)
	writeFile(t, root, "pipelines/fragment.yaml", "name: helper\nsteps: []\n")
	writeFile(t, root, "pipelines/malformed.yml", "entrypoint: [\n")
	writeFile(t, root, "pipelines/multiple.yaml", valid+"---\nother: document\n")
	writeFile(t, root, "pipelines/directory/meta.yaml", "entrypoint: main\n")
	writeFile(t, root, "pipelines/directory/workflows/main.yaml", "name: main\nsteps: []\n")

	defs, err := ListWorkflowDefinitions(root)
	if err != nil {
		t.Fatalf("ListWorkflowDefinitions() error: %v", err)
	}
	var paths []string
	for _, definition := range defs {
		paths = append(paths, definition.Path)
	}
	want := []string{"pipelines/alpha.yaml", "pipelines/directory", "pipelines/zeta.yml"}
	if !reflect.DeepEqual(paths, want) {
		t.Fatalf("definition paths = %v, want %v", paths, want)
	}
}

func TestListWorkflowDefinitionsReturnsFilesystemErrors(t *testing.T) {
	_, err := ListWorkflowDefinitions(filepath.Join(t.TempDir(), "missing"))
	if err == nil {
		t.Fatal("ListWorkflowDefinitions() expected filesystem error")
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
