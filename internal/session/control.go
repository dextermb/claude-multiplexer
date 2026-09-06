package session

import "fmt"

func (s *Session) Send(text string) error {
	if s.cmd == nil {
		return ErrNotStarted
	}
	if !s.State().Live() {
		return ErrNotLive
	}
	s.q.push(text)
	return nil
}

func (s *Session) Interrupt() error {
	if s.cmd == nil {
		return ErrNotStarted
	}
	if s.State() != StateBusy {
		return nil
	}
	s.mu.Lock()
	s.interruptSeq++
	n := s.interruptSeq
	s.mu.Unlock()
	return s.writer.SendInterrupt(fmt.Sprintf("int-%d", n))
}

func (s *Session) DiscardQueued() {
	s.q.clear()
}

func (s *Session) controlID(kind string) string {
	s.mu.Lock()
	s.controlSeq++
	n := s.controlSeq
	s.mu.Unlock()
	return fmt.Sprintf("%s-%d", kind, n)
}

// SetModel switches the model of the running child; see docs/protocol.md.
func (s *Session) SetModel(model string) error {
	if s.cmd == nil {
		return ErrNotStarted
	}
	if !s.State().Live() {
		return ErrNotLive
	}
	if err := s.writer.SetModel(s.controlID("model"), model); err != nil {
		return err
	}
	s.mu.Lock()
	s.model = model
	s.mu.Unlock()
	return nil
}

// SetPermissionMode switches the permission mode of the running child; see
// docs/protocol.md.
func (s *Session) SetPermissionMode(mode string) error {
	if s.cmd == nil {
		return ErrNotStarted
	}
	if !s.State().Live() {
		return ErrNotLive
	}
	if err := s.writer.SetPermissionMode(s.controlID("mode"), mode); err != nil {
		return err
	}
	s.mu.Lock()
	s.permissionMode = mode
	s.mu.Unlock()
	return nil
}

// SetTitle changes the display title of the session. The title is local
// metadata and does not touch the child, so it works whatever the state.
func (s *Session) SetTitle(title string) {
	s.mu.Lock()
	s.title = title
	state := s.state
	s.mu.Unlock()
	s.emit(Event{Kind: KindState, State: state, Prev: state})
}
