package projects

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
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

func RemoveClone(destPath string) error {
	return os.RemoveAll(destPath)
}
