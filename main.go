// tmux-sessionizer
package main

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Scan []Scan
	Dirs []string
}

type Scan struct {
	Path  string
	Depth int
}

func main() {
	if !checkDependencies() {
		os.Exit(1)
	}

	cfg, err := readConfig(resolveTildePath("~/.config/tmux-sessionizer"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "read config: %v\n", err)
		os.Exit(1)
	}

	selected, ok := selectProject(cfg)
	if !ok {
		os.Exit(1)
	}

	sessionName := strings.ReplaceAll(filepath.Base(selected), ".", "-")
	tmuxServer := tmuxServer()
	tmuxClient := os.Getenv("TMUX") != ""

	if !tmuxClient && !tmuxServer {
		tmuxNewSession(sessionName, selected, false)
		return
	}

	if !tmuxHasSession(sessionName) {
		tmuxNewSession(sessionName, selected, true)
	}

	if tmuxClient {
		tmuxSwitchClient(sessionName)
	} else {
		tmuxAttach(sessionName)
	}
}

func readConfig(path string) (Config, error) {
	var cfg Config
	b, err := os.ReadFile(path)
	if err != nil {
		return cfg, err
	}
	if err := yaml.Unmarshal(b, &cfg); err != nil {
		return cfg, err
	}
	return cfg, nil
}

func selectProject(cfg Config) (string, bool) {
	cmd := exec.Command("fzf", "--scheme=path", "--tmux")
	pipe, _ := cmd.StdinPipe()
	out := bytes.NewBuffer(nil)
	cmd.Stdout = out
	cmd.Stderr = os.Stderr

	go func() {
		repos := make(chan string)
		go func() {
			for _, sd := range cfg.Scan {
				if err := findRepos(resolveTildePath(sd.Path), sd.Depth, repos); err != nil {
					fmt.Fprintf(os.Stderr, "error: %v\n", err)
				}
			}
			close(repos)
		}()
		for _, d := range cfg.Dirs {
			pipe.Write([]byte(resolveTildePath(d)))
			pipe.Write([]byte{'\n'})
		}
		for r := range repos {
			pipe.Write([]byte(r))
			pipe.Write([]byte{'\n'})
		}
		pipe.Close()
	}()

	if err := cmd.Run(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return "", false
		}
		fmt.Fprintf(os.Stderr, "fzf: %v\n", err)
		return "", false
	}

	selected := strings.TrimRight(out.String(), "\r\n")

	return selected, true
}

func findRepos(root string, depth int, dirs chan<- string) error {
	rootDepth := strings.Count(root, string(filepath.Separator))
	if err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if !d.IsDir() {
			return nil
		}
		pathDepth := strings.Count(path, string(filepath.Separator)) - rootDepth
		if pathDepth >= depth {
			return fs.SkipDir
		}
		fi, err := os.Stat(filepath.Join(path, ".git"))
		if err != nil {
			return nil
		}
		if fi.IsDir() {
			dirs <- path
		}
		return nil
	}); err != nil {
		return err
	}
	return nil
}

func tmuxServer() bool {
	cmd := exec.Command("pgrep", "-x", "tmux: server")
	cmd.Stderr = os.Stderr
	return cmd.Run() == nil
}

func tmuxNewSession(name, dir string, detached bool) {
	cmd := exec.Command("tmux", "new-session")
	if detached {
		cmd.Args = append(cmd.Args, "-d")
	}
	cmd.Args = append(cmd.Args, "-s", name)
	cmd.Args = append(cmd.Args, "-c", dir)
	if !detached {
		cmd.Stdin = os.Stdin
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Run()
}

func tmuxHasSession(name string) bool {
	return exec.Command("tmux", "has-session", "-t", name).Run() == nil
}

func tmuxAttach(name string) {
	cmd := exec.Command("tmux", "attach", "-t", name)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Run()
}

func tmuxSwitchClient(name string) {
	cmd := exec.Command("tmux", "switch-client", "-t", name)
	cmd.Stderr = os.Stderr
	cmd.Run()
}

func resolveTildePath(p string) string {
	if !strings.HasPrefix(p, "~") {
		return p
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		panic(fmt.Errorf("get user home directory: %w", err))
	}

	resolvedPath := filepath.Join(homeDir, p[1:])

	return filepath.Clean(resolvedPath)
}

func checkDependencies() bool {
	ok := true
	if !checkAvailable("tmux") {
		fmt.Fprintf(os.Stderr, "`tmux` is not found\n")
		ok = false
	}
	if !checkAvailable("fzf") {
		fmt.Fprintf(os.Stderr, "`fzf` is not found\n")
		ok = false
	}
	if !checkAvailable("pgrep") {
		fmt.Fprintf(os.Stderr, "`pgrep` is not found\n")
		ok = false
	}
	return ok
}

func checkAvailable(command string) bool {
	_, err := exec.LookPath(command)
	return err == nil
}
