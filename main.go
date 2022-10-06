// Command opengithub provides a tool to open source code in Github.
package main

import (
	"errors"
	"flag"
	"fmt"
	"golang.design/x/clipboard"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

var (
	file   = flag.String("file", "", "File (with optional line number) to open in github. Defaults to clipboard.")
	root   = flag.String("root", os.Getenv("OPENGITHUB_ROOT"), "Root directory to search for relative paths")
	branch = flag.String("branch", os.Getenv("OPENGITHUB_BRANCH"), "Git branch to use. Defaults to current branch")
	open   = flag.Bool("open", true, "Set to false to disable opening in default browser")
)

func main() {
	flag.Parse()
	if err := run(*file, *root, *branch, *open); err != nil {
		fmt.Printf("❌ Fatal error: %v\n", err)
		os.Exit(1)
	}
}

func run(file string, root string, branch string, open bool) error {
	if file == "" {
		txt, err := readClipboard()
		if err != nil {
			return err
		} else if txt == "" {
			return errors.New("--file and clipboard empty 👻")
		}
		file = txt
		fmt.Printf("Using clipboard text: %s\n", txt)
	}

	if filepath.Ext(file) == "" {
		return fmt.Errorf("clipboard text not a file: '%v'", file)
	}

	file, line, err := splitFileLine(file)
	if err != nil {
		return err
	}

	abs, err := findAbsPath(file, root)
	if err != nil {
		return err
	}

	remote, path, err := findRemotePath(abs)
	if err != nil {
		return err
	}

	if branch == "" {
		fmt.Printf("Using current branch since --branch or $OPENGITHUB_BRANCH not set\n")
		branch, err = currentBranch(abs)
		if err != nil {
			return err
		}
	}

	fmt.Printf("Found remote:%s, branch:%s, path:%s, line=%d\n", remote, branch, path, line)

	uri, err := formatGitURL(remote, branch, path, line)
	if err != nil {
		return err
	}

	fmt.Printf("🎉 %s\n", uri)

	if open {
		if err := exec.Command("open", uri).Run(); err != nil {
			return fmt.Errorf("open url: %w", err)
		}
	}

	return nil
}

// formatGitURL returns the git url to view the path and line.
// It expects remote to be in the format git@github.com:{org}/{repo}.git.
// It returns the url in the format: https://github.com/{org}/{repo}/blob/{branch}/{path}
func formatGitURL(remote string, branch string, path string, line int) (string, error) {
	if !strings.Contains(remote, "git@github.com:") {
		return "", errors.New("only github repos supported")
	}

	resp := strings.Replace(remote, ":", "/", 1)
	resp = strings.Replace(resp, "git@", "https://", 1)
	resp = strings.Replace(resp, ".git", "/"+filepath.Join("blob", branch, path), 1)
	if line != 0 {
		resp += fmt.Sprintf("#L%d", line)
	}

	return resp, nil
}

func findAbsPath(file string, root string) (string, error) {
	if filepath.IsAbs(file) {
		return file, nil
	}

	if root == "" {
		var err error
		root, err = os.Getwd()
		if err != nil {
			return "", err
		}
		fmt.Printf("Using current directory to resolve relative path since --root or $OPENGITHUB_ROOT not set: %s\n", root)
	}

	split := strings.Split(file, string(filepath.Separator))
	for i := 0; i < len(split); i++ {
		if strings.HasSuffix(root, filepath.Join(split[:i+1]...)) {
			continue
		}

		abs, ok, err := findFile(root, split[i:])
		if err != nil {
			return "", fmt.Errorf("cannot find file: %w: %v", err, file)
		} else if !ok {
			return "", fmt.Errorf("cannot find file in root: %v, %v", root, filepath.Join(split[:i]...))
		} else {
			return abs, nil
		}
	}

	return "", errors.New("unexpected error")
}

func findRemotePath(path string) (remote string, relFile string, err error) {
	{ // Get git repo root
		cmd := exec.Command("git", "rev-parse", "--show-toplevel")
		cmd.Dir = filepath.Dir(path)
		out, err := cmd.CombinedOutput()
		if err != nil {
			return "", "", fmt.Errorf("git repo root: %w: %s", err, out)
		}
		repoRoot := strings.TrimSpace(string(out))

		relFile, err = filepath.Rel(repoRoot, path)
		if err != nil {
			return "", "", fmt.Errorf("relative path: %w: %s", err, out)
		}
	}
	{ // Get git remote url
		cmd := exec.Command("git", "config", "--get", "remote.origin.url")
		cmd.Dir = filepath.Dir(path)
		out, err := cmd.CombinedOutput()
		if err != nil {
			return "", "", fmt.Errorf("git config get remote.origin.url: %w: %s", err, out)
		}
		remote = strings.TrimSpace(string(out))
	}

	return remote, relFile, nil
}

func currentBranch(file string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	cmd.Dir = filepath.Dir(file)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git get branch: %w, %s", err, out)
	}

	return strings.TrimSpace(string(out)), nil
}

func findFile(root string, splitFile []string) (string, bool, error) {
	prefix := splitFile[0]
	remaining := splitFile[1:]

	// Check if prefix in root
	if matches, err := filepath.Glob(filepath.Join(root, prefix)); err != nil {
		return "", false, fmt.Errorf("glob %v: %w", filepath.Join(root, prefix), err)
	} else if len(matches) > 1 {
		return "", false, fmt.Errorf("multiple matches: %v", matches)
	} else if len(matches) == 1 && len(remaining) == 0 {
		return matches[0], true, nil
	} else if len(matches) == 1 {
		return findFile(matches[0], remaining)
	}

	// Check each folder in root
	entries, err := os.ReadDir(root)
	if os.IsPermission(err) {
		return "", false, nil // Skip folders we cannot read
	} else if err != nil {
		return "", false, fmt.Errorf("read dir %v: %w", root, err)
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		found, ok, err := findFile(filepath.Join(root, entry.Name()), splitFile)
		if err != nil {
			return "", false, err
		} else if ok {
			return found, ok, nil
		}
	}

	return "", false, nil
}

func splitFileLine(txt string) (string, int, error) {
	split := strings.Split(txt, ":")
	if len(split) > 2 {
		return "", 0, errors.New("cannot parse file line, contains multiple ':'")
	} else if len(split) == 1 {
		return txt, 0, nil
	}

	line, err := strconv.Atoi(split[1])
	if err != nil {
		return "", 0, fmt.Errorf("cannot parse file line: %w", err)
	}

	return split[0], line, nil
}

func readClipboard() (string, error) {
	err := clipboard.Init()
	if err != nil {
		return "", fmt.Errorf("init clipboard: %w", err)
	}

	return string(clipboard.Read(clipboard.FmtText)), nil
}
