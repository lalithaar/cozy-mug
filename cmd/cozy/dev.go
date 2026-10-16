package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"
)

// runDev implements `cozy dev`: a no-config build -> restart -> browser-refresh
// loop. It builds to a stable binary path, polls the source tree for changes,
// rebuilds and restarts the server on any change (or crash), and the running
// app's dev mode (Config.Dev) makes browsers auto-refresh.
func runDev(args []string) error {
	dir := "."
	port := ""
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-dir", "--dir":
			if i+1 >= len(args) {
				return fmt.Errorf("missing value for %s", args[i])
			}
			dir = args[i+1]
			i++
		case "-port", "--port":
			if i+1 >= len(args) {
				return fmt.Errorf("missing value for %s", args[i])
			}
			port = args[i+1]
			i++
		case "-addr", "--addr":
			if i+1 >= len(args) {
				return fmt.Errorf("missing value for %s", args[i])
			}
			v := args[i+1]
			i++
			v = strings.TrimSpace(v)
			v = strings.TrimPrefix(v, ":")
			port = v
		default:
			return fmt.Errorf("unexpected argument %q", args[i])
		}
	}
	if port == "" {
		port = os.Getenv("PORT")
	}
	if port == "" {
		port = "8080"
	}

	absDir, err := filepath.Abs(dir)
	if err != nil {
		return err
	}
	if _, err := os.Stat(filepath.Join(absDir, "go.mod")); err != nil {
		return fmt.Errorf("no go.mod in %s (run cozy dev from a project directory)", absDir)
	}
	if err := os.MkdirAll(filepath.Join(absDir, ".cozy"), 0o755); err != nil {
		return err
	}

	r := &devRunner{
		dir:  absDir,
		bin:  devBinaryPath(absDir),
		port: port,
	}
	ctx, stopSig := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stopSig()

	lastSnap := snapshotTree(absDir)

	r.buildAndStart()

	ticker := time.NewTicker(300 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			r.stop()
			return nil
		case <-ticker.C:
			snap := snapshotTree(absDir)
			if snapshotsEqual(snap, lastSnap) {
				continue
			}
			lastSnap = snap
			fmt.Println("cozy dev: change detected, rebuilding...")
			if !r.build() {
				fmt.Println("cozy dev: build failed; keeping previous server")
				continue
			}
			r.crashes = 0
			r.stopProcess()
			r.start()
		}
	}
}

func devBinaryPath(dir string) string {
	name := "dev"
	if runtime.GOOS == "windows" {
		name = "dev.exe"
	}
	return filepath.Join(dir, ".cozy", name)
}

type devRunner struct {
	dir     string
	bin     string
	port    string
	mu      sync.Mutex
	cmd     *exec.Cmd
	stopping bool
	crashes int
}

func (r *devRunner) build() bool {
	fmt.Println("cozy dev: building...")
	cmd := exec.Command("go", "build", "-o", r.bin, ".")
	cmd.Dir = r.dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return false
	}
	return true
}

func (r *devRunner) buildAndStart() {
	if !r.build() {
		fmt.Println("cozy dev: initial build failed; watching for changes")
		return
	}
	r.start()
	fmt.Printf("cozy dev: server at http://localhost:%s\n", r.port)
}

func (r *devRunner) start() {
	r.mu.Lock()
	r.stopping = false
	r.mu.Unlock()

	c := exec.Command(r.bin)
	c.Dir = r.dir
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	c.Env = append(os.Environ(), "DEV=1", "PORT="+r.port)
	if err := c.Start(); err != nil {
		fmt.Println("cozy dev: start failed:", err)
		return
	}
	r.mu.Lock()
	r.cmd = c
	r.mu.Unlock()
	go r.monitor(c)
}

func (r *devRunner) monitor(c *exec.Cmd) {
	err := c.Wait()
	r.mu.Lock()
	if r.cmd == c {
		r.cmd = nil
	}
	stopping := r.stopping
	r.mu.Unlock()
	if stopping {
		return
	}
	r.mu.Lock()
	r.crashes++
	crashes := r.crashes
	r.mu.Unlock()
	fmt.Printf("cozy dev: server exited unexpectedly: %v\n", err)
	if crashes > 5 {
		fmt.Println("cozy dev: too many crashes; waiting for a file change")
		return
	}
	time.Sleep(time.Second)
	r.mu.Lock()
	if r.stopping {
		r.mu.Unlock()
		return
	}
	r.mu.Unlock()
	r.start()
}

func (r *devRunner) stopProcess() {
	r.mu.Lock()
	r.stopping = true
	c := r.cmd
	r.cmd = nil
	r.mu.Unlock()
	if c != nil && c.Process != nil {
		_ = c.Process.Kill()
		_ = c.Wait()
	}
}

func (r *devRunner) stop() {
	r.stopProcess()
}

type fileSnapshot map[string]string

func snapshotTree(dir string) fileSnapshot {
	out := fileSnapshot{}
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil
		}
		if d.IsDir() {
			if skipDir(d.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		if !watchedFile(path, d) {
			return nil
		}
		fi, err := d.Info()
		if err != nil {
			return nil
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return nil
		}
		out[rel] = fmt.Sprintf("%d:%d", fi.ModTime().UnixNano(), fi.Size())
		return nil
	})
	if err != nil {
		return out
	}
	return out
}

func skipDir(name string) bool {
	switch name {
	case ".git", ".cozy", "vendor", "node_modules":
		return true
	}
	// skip hidden dirs
	if strings.HasPrefix(name, ".") && name != "." {
		// still allow root; WalkDir root name is "." not passed as d.Name? root handled separately
		return true
	}
	return false
}

func watchedFile(path string, d os.DirEntry) bool {
	base := d.Name()
	ext := strings.ToLower(filepath.Ext(base))
	switch ext {
	case ".go", ".html", ".css", ".js", ".tmpl", ".gotmpl", ".json", ".toml", ".yaml", ".yml":
		return true
	}
	if base == "go.mod" || base == "go.sum" {
		return true
	}
	return false
}

func snapshotsEqual(a, b fileSnapshot) bool {
	if len(a) != len(b) {
		return false
	}
	keys := make([]string, 0, len(a))
	for k := range a {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		if a[k] != b[k] {
			return false
		}
	}
	return true
}
