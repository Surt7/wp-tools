package audit

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"SetSecrets/internal/dotnet"
)

// RepoAuditResult holds the security audit status of a single git repository.
type RepoAuditResult struct {
	Name          string
	Path          string
	EnvIgnored    bool
	EnvTracked    bool
	ConfigIgnored bool
	ConfigTracked bool
	Status        string // "SAFE", "WARNING", "DANGER"
	Issues        []string
}

// FindGitRoot traverses upwards from startPath to find the enclosing .git directory.
func FindGitRoot(startPath string) (string, error) {
	fi, err := os.Stat(startPath)
	if err != nil {
		return "", err
	}
	dir := startPath
	if !fi.IsDir() {
		dir = filepath.Dir(startPath)
	}
	for {
		gitPath := filepath.Join(dir, ".git")
		if _, err := os.Stat(gitPath); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", fmt.Errorf("no git repository found for %s", startPath)
}

// IsIgnored checks whether a specific file path is ignored by git rules in repoRoot.
func IsIgnored(repoRoot string, file string) bool {
	cmd := exec.Command("git", "-C", repoRoot, "check-ignore", "-q", file)
	return cmd.Run() == nil
}

// IsTracked checks whether a file is currently tracked (in git index or commit) in repoRoot.
func IsTracked(repoRoot string, file string) bool {
	cmd := exec.Command("git", "-C", repoRoot, "ls-files", file)
	out, err := cmd.Output()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(out)) != ""
}

// AddToGitIgnore appends a pattern to the repository's .gitignore file.
func AddToGitIgnore(repoRoot string, pattern string) error {
	gitignorePath := filepath.Join(repoRoot, ".gitignore")
	f, err := os.OpenFile(gitignorePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = fmt.Fprintf(f, "\n# Added by wp-tools audit\n%s\n", pattern)
	return err
}

// AuditRepo inspects a repository for potential secret leaks or missing ignore rules.
func AuditRepo(repoRoot string) RepoAuditResult {
	res := RepoAuditResult{
		Name:          filepath.Base(repoRoot),
		Path:          repoRoot,
		EnvIgnored:    IsIgnored(repoRoot, ".env"),
		EnvTracked:    IsTracked(repoRoot, ".env"),
		ConfigIgnored: IsIgnored(repoRoot, "wp-config.json"),
		ConfigTracked: IsTracked(repoRoot, "wp-config.json"),
		Status:        "SAFE",
	}

	if res.EnvTracked {
		res.Status = "DANGER"
		res.Issues = append(res.Issues, ".env is TRACKED in git index! High leak risk.")
	} else if !res.EnvIgnored {
		res.Status = "WARNING"
		res.Issues = append(res.Issues, ".env is NOT ignored in .gitignore.")
	}

	if !res.ConfigIgnored && !res.ConfigTracked {
		// Only warn if wp-config.json is not ignored and not explicitly tracked as part of tooling
		if res.Name != "wp-tools" {
			if res.Status == "SAFE" {
				res.Status = "WARNING"
			}
			res.Issues = append(res.Issues, "wp-config.json is NOT ignored in .gitignore.")
		}
	}

	return res
}

// DiscoverRepositories finds all unique git repositories referenced in configuration and workspace.
func DiscoverRepositories(cfg dotnet.Config) []string {
	seen := make(map[string]bool)
	var repos []string

	addRepo := func(p string) {
		if p == "" {
			return
		}
		root, err := FindGitRoot(p)
		if err == nil {
			cleanRoot := filepath.Clean(root)
			if !seen[cleanRoot] {
				seen[cleanRoot] = true
				repos = append(repos, cleanRoot)
			}
		}
	}

	// 1. Current working directory
	if cwd, err := os.Getwd(); err == nil {
		addRepo(cwd)
	}

	// 2. Executable directory
	if exePath, err := os.Executable(); err == nil {
		addRepo(filepath.Dir(exePath))
	}

	// 3. All configured targets
	for _, envTargets := range cfg.Environments {
		for _, targetPath := range envTargets {
			addRepo(targetPath)
		}
	}

	return repos
}
