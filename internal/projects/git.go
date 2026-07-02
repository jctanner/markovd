package projects

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/jctanner/markovd/internal/models"
	"github.com/jctanner/markovd/internal/workflowdef"
)

func CloneOrPull(url, branch, destPath string) error {
	if _, err := os.Stat(filepath.Join(destPath, ".git")); os.IsNotExist(err) {
		_, err := git.PlainClone(destPath, false, &git.CloneOptions{
			URL:           url,
			ReferenceName: plumbing.NewBranchReferenceName(branch),
			SingleBranch:  true,
		})
		if err != nil {
			return fmt.Errorf("cloning repository: %w", err)
		}
		return nil
	}

	repo, err := git.PlainOpen(destPath)
	if err != nil {
		return fmt.Errorf("opening repository: %w", err)
	}

	wt, err := repo.Worktree()
	if err != nil {
		return fmt.Errorf("getting worktree: %w", err)
	}

	err = wt.Pull(&git.PullOptions{
		ReferenceName: plumbing.NewBranchReferenceName(branch),
		Force:         true,
	})
	if err != nil && err != git.NoErrAlreadyUpToDate {
		return fmt.Errorf("pulling repository: %w", err)
	}

	return nil
}

func ListYAMLFiles(repoPath string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(repoPath, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && d.Name() == ".git" {
			return filepath.SkipDir
		}
		if d.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if ext == ".yaml" || ext == ".yml" {
			rel, err := filepath.Rel(repoPath, path)
			if err != nil {
				return err
			}
			files = append(files, rel)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("listing YAML files: %w", err)
	}
	return files, nil
}

type WorkflowDefinitionEntry struct {
	Path string `json:"path"`
	Kind string `json:"kind"`
	Name string `json:"name"`
}

func ListWorkflowDefinitions(repoPath string) ([]WorkflowDefinitionEntry, error) {
	var dirs []string
	err := filepath.WalkDir(repoPath, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			return nil
		}
		if d.Name() == ".git" {
			return filepath.SkipDir
		}
		if isWorkflowDirectory(path) {
			rel, err := filepath.Rel(repoPath, path)
			if err != nil {
				return err
			}
			if rel == "." {
				rel = ""
			}
			dirs = append(dirs, filepath.ToSlash(rel))
			return filepath.SkipDir
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("listing workflow directories: %w", err)
	}

	yamlFiles, err := ListYAMLFiles(repoPath)
	if err != nil {
		return nil, err
	}
	var entries []WorkflowDefinitionEntry
	for _, dir := range dirs {
		path := dir
		if path == "" {
			path = "."
		}
		entries = append(entries, WorkflowDefinitionEntry{
			Path: path,
			Kind: workflowdef.KindDirectory,
			Name: DeriveWorkflowName(path),
		})
	}
	for _, file := range yamlFiles {
		if insideWorkflowDirectory(file, dirs) {
			continue
		}
		entries = append(entries, WorkflowDefinitionEntry{
			Path: file,
			Kind: workflowdef.KindFile,
			Name: DeriveWorkflowName(file),
		})
	}
	return entries, nil
}

func isWorkflowDirectory(path string) bool {
	for _, name := range []string{"meta.yaml", "vars.yaml", "rules.yaml", "step_types.yaml"} {
		if _, err := os.Stat(filepath.Join(path, name)); err != nil {
			return false
		}
	}
	info, err := os.Stat(filepath.Join(path, "workflows"))
	return err == nil && info.IsDir()
}

func insideWorkflowDirectory(file string, dirs []string) bool {
	file = filepath.ToSlash(file)
	for _, dir := range dirs {
		if dir == "" || dir == "." {
			return true
		}
		prefix := strings.TrimSuffix(filepath.ToSlash(dir), "/") + "/"
		if strings.HasPrefix(file, prefix) {
			return true
		}
	}
	return false
}

func ReadFile(repoPath, filePath string) (string, error) {
	cleaned := filepath.Clean(filePath)
	if strings.HasPrefix(cleaned, "..") || filepath.IsAbs(cleaned) {
		return "", fmt.Errorf("invalid file path: %s", filePath)
	}

	data, err := os.ReadFile(filepath.Join(repoPath, cleaned))
	if err != nil {
		return "", fmt.Errorf("reading file: %w", err)
	}
	return string(data), nil
}

func ReadWorkflowDefinition(repoPath, rootPath, kind string) (models.WorkflowDefinition, error) {
	switch kind {
	case workflowdef.KindFile:
		content, err := ReadFile(repoPath, rootPath)
		if err != nil {
			return models.WorkflowDefinition{}, err
		}
		return workflowdef.Normalize(kind, []models.WorkflowDefinitionFile{{Path: filepath.Base(rootPath), Content: content}})
	case workflowdef.KindDirectory:
		cleaned := filepath.Clean(rootPath)
		if cleaned == "." {
			cleaned = ""
		}
		if strings.HasPrefix(cleaned, "..") || filepath.IsAbs(cleaned) {
			return models.WorkflowDefinition{}, fmt.Errorf("invalid directory path: %s", rootPath)
		}
		root := filepath.Join(repoPath, cleaned)
		var files []models.WorkflowDefinitionFile
		err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}
			ext := strings.ToLower(filepath.Ext(path))
			if ext != ".yaml" && ext != ".yml" {
				return nil
			}
			rel, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			files = append(files, models.WorkflowDefinitionFile{
				Path:    filepath.ToSlash(rel),
				Content: string(data),
			})
			return nil
		})
		if err != nil {
			return models.WorkflowDefinition{}, fmt.Errorf("reading workflow directory: %w", err)
		}
		return workflowdef.Normalize(kind, files)
	default:
		return models.WorkflowDefinition{}, fmt.Errorf("unsupported workflow definition kind: %s", kind)
	}
}

func DeriveWorkflowName(path string) string {
	name := strings.TrimSuffix(path, filepath.Ext(path))
	name = strings.Trim(name, ".")
	name = strings.Trim(name, "/")
	if name == "" {
		name = "workflow"
	}
	name = strings.ReplaceAll(name, "/", "-")
	name = strings.ReplaceAll(name, "\\", "-")
	return name
}

func RemoveClone(destPath string) error {
	return os.RemoveAll(destPath)
}
