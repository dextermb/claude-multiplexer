package session

import (
	"os"
	"path/filepath"
	"sync"
)

type Transcript struct {
	mu   sync.Mutex
	file *os.File
}

func OpenTranscript(path string) (*Transcript, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, err
	}
	return &Transcript{file: file}, nil
}

func (t *Transcript) WriteLine(line []byte) error {
	if t == nil {
		return nil
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	if _, err := t.file.Write(line); err != nil {
		return err
	}
	_, err := t.file.Write([]byte{'\n'})
	return err
}

func (t *Transcript) Close() error {
	if t == nil {
		return nil
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.file.Close()
}
