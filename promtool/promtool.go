// Package promtool provides functions that run promtool tests over given
// directories.
package promtool

import (
	"bytes"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
)

// PromtoolError is returned to the called when an underlying call to promtool failed.
type PromtoolError struct {
	Executable    string
	Operation     string
	FileName      string
	Stdout        string
	Stderr        string
	OriginalError error
}

func (p PromtoolError) Error() string {
	return fmt.Sprintf("%s %s %s failed (%s).\nstdout: %s\nstderr: %s\n", p.Executable, p.Operation, p.FileName, p.OriginalError, p.Stdout, p.Stderr)
}

// Promtool is a struct for manipulating the promtool executable.
type Promtool struct {
	Executable string
}

// New tries to find and return a usable promtool
func New() (*Promtool, error) {
	p := Promtool{}
	for _, promtools := range []string{"promtool.exe", "promtool"} {
		if path, err := exec.LookPath(promtools); err == nil {
			p.Executable = path
			return &p, nil
		}
	}
	return nil, errors.New("promtool not found in path")
}

// Check takes the path to a prometheus rule file and runs promtool check on it.
func (p *Promtool) Check(file string) (string, error) {
	return p.execute("check", file, "")
}

// Test takes the path to a prometheus test file and runs promtool test in the specified working directory
// If workdir is empty, it runs it in the current working directory
func (p *Promtool) Test(file, workdir string) (string, error) {
	return p.execute("test", file, workdir)
}

// execute invokes promtool and passes errors back
func (p *Promtool) execute(op, path, workdir string) (string, error) {
	var stdout, stderr bytes.Buffer
	c := exec.Command(p.Executable, op, "rules", path)
	c.Dir = workdir
	c.Stdout = &stdout
	c.Stderr = &stderr
	if err := c.Run(); err != nil {
		return "", PromtoolError{
			Executable:    p.Executable,
			Operation:     op,
			FileName:      path,
			Stdout:        stdout.String(),
			Stderr:        stderr.String(),
			OriginalError: err,
		}
	}
	return stdout.String(), nil
}

// CheckDirectory validates that dir contains some yaml files, then calls check on each one.
func (p *Promtool) CheckDirectory(dir string) error {
	return p.executeDirectory("check", dir, "")
}

// TestDirectory Takes a directory to test and a working directory for promtool.
// If workdir is empty, it runs it in the current working directory.
func (p *Promtool) TestDirectory(dir, workdir string) error {
	return p.executeDirectory("test", dir, workdir)
}

func (p *Promtool) executeDirectory(op, dir, workdir string) error {
	paths, err := filepath.Glob(filepath.Join(workdir, dir, "*.yaml"))
	if err != nil {
		return err
	}
	if len(paths) == 0 {
		return errors.New("No .yaml files found in directory " + dir)
	}
	for _, path := range paths {
		switch op {
		case "check":
			if _, err := p.Check(path); err != nil {
				return err
			}
		case "test":
			if _, err := p.Test(path, workdir); err != nil {
				return err
			}
		default:
			return errors.New("Invalid operation " + op)
		}
	}
	return nil
}
