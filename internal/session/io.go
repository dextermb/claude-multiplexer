package session

import (
	"errors"
	"io"

	"github.com/dextermb/claude-multiplexer/internal/protocol"
)

func (s *Session) closeStdin() {
	s.closeIn.Do(func() {
		if s.stdin != nil {
			_ = s.stdin.Close()
		}
	})
}

func (s *Session) readStdout(r io.ReadCloser) {
	defer s.readers.Done()
	defer r.Close()
	reader := protocol.NewReader(r)
	for {
		ev, err := reader.Next()
		if errors.Is(err, io.EOF) {
			return
		}
		if err != nil && !errors.Is(err, protocol.ErrNotJSON) {
			s.fail(err)
			return
		}
		if ev.Type != protocol.TypeStreamEvent {
			_ = s.tr.WriteLine(ev.Raw)
		}
		if errors.Is(err, protocol.ErrNotJSON) {
			s.emit(Event{Kind: KindError, Line: string(ev.Raw), Err: err})
			continue
		}
		s.apply(ev)
		s.emit(Event{Kind: KindProtocol, Protocol: ev})
	}
}

func (s *Session) readStderr(r io.ReadCloser) {
	defer s.readers.Done()
	defer r.Close()
	reader := protocol.NewReader(r)
	for {
		ev, err := reader.Next()
		if err != nil && !errors.Is(err, protocol.ErrNotJSON) {
			return
		}
		line := string(ev.Raw)
		s.stderr.add(line)
		s.emit(Event{Kind: KindStderr, Line: line})
	}
}

func (s *Session) writeLoop() {
	defer close(s.writeDone)
	for {
		select {
		case <-s.ctx.Done():
			return
		case <-s.q.sig:
		}
		for {
			if err := s.waitIdle(); err != nil {
				return
			}
			text, ok := s.q.pop()
			if !ok {
				break
			}
			s.setState(StateBusy)
			s.markSent()
			if err := s.writer.SendUser(s.ClaudeSessionID(), text); err != nil {
				s.fail(err)
				return
			}
		}
	}
}

func (s *Session) waitIdle() error {
	for {
		s.mu.Lock()
		state, sent := s.state, s.sent
		s.mu.Unlock()
		switch {
		case state == StateIdle || state == StateWaiting:
			return nil
		case state == StateStarting && !sent:
			// Claude Code emits init only after it reads input; see docs/protocol.md.
			return nil
		case !state.Live():
			return ErrNotLive
		}
		select {
		case <-s.idleSig:
		case <-s.ctx.Done():
			return s.ctx.Err()
		}
	}
}
