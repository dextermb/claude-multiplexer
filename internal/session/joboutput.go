package session

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// MaxJobOutput is the most output that a read returns. A longer file gives its
// last MaxJobOutput bytes, so a job that prints for hours cannot fill memory.
const MaxJobOutput = 256 << 10

var (
	// ErrNoOutputPath means the job has not named its output file yet.
	ErrNoOutputPath = errors.New("the job has no output file yet")
	// ErrBadOutputPath means the path is not one Claude Code writes a job to.
	ErrBadOutputPath = errors.New("the output path is not a job output file")
)

// ReadOutput returns the output that the job wrote, up to MaxJobOutput bytes
// from the end. A file that is absent or empty returns an empty string and no
// error, because a job that has printed nothing yet is not a failure. See
// docs/sessions.md.
func ReadOutput(job Job) (string, error) {
	if job.OutputPath == "" {
		return "", ErrNoOutputPath
	}
	if !validOutputPath(job.OutputPath, job.ID) {
		return "", ErrBadOutputPath
	}
	file, err := os.Open(job.OutputPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", nil
		}
		return "", err
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return "", err
	}
	if size := info.Size(); size > MaxJobOutput {
		if _, err := file.Seek(size-MaxJobOutput, io.SeekStart); err != nil {
			return "", err
		}
	}
	body, err := io.ReadAll(io.LimitReader(file, MaxJobOutput))
	if err != nil {
		return "", err
	}
	return string(body), nil
}

// validOutputPath holds a path that came off the child stream to the shape
// Claude Code writes a job output to; see docs/sessions.md.
func validOutputPath(path, id string) bool {
	if !filepath.IsAbs(path) || id == "" {
		return false
	}
	clean := filepath.Clean(path)
	if filepath.Base(clean) != id+".output" {
		return false
	}
	return filepath.Base(filepath.Dir(clean)) == "tasks"
}

// OutputLines splits job output into lines, without a trailing empty one.
func OutputLines(body string) []string {
	body = strings.TrimRight(body, "\n")
	if body == "" {
		return nil
	}
	return strings.Split(body, "\n")
}
