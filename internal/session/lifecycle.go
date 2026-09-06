package session

import "time"

func (s *Session) setState(next State) {
	s.mu.Lock()
	prev := s.state
	if prev == next || !prev.Live() {
		s.mu.Unlock()
		return
	}
	s.state = next
	s.mu.Unlock()
	s.afterState(prev, next)
}

func (s *Session) setStateIf(from, next State) {
	s.mu.Lock()
	if s.state != from || from == next {
		s.mu.Unlock()
		return
	}
	s.state = next
	s.mu.Unlock()
	s.afterState(from, next)
}

func (s *Session) markSent() {
	s.mu.Lock()
	s.sent = true
	s.mu.Unlock()
}

func (s *Session) afterState(prev, next State) {
	if next == StateIdle || next == StateWaiting {
		select {
		case s.idleSig <- struct{}{}:
		default:
		}
	}
	s.emit(Event{Kind: KindState, State: next, Prev: prev})
}

func (s *Session) fail(err error) {
	s.mu.Lock()
	if s.err == nil {
		s.err = err
	}
	s.mu.Unlock()
	s.emit(Event{Kind: KindError, Err: err})
	s.setState(StateFailed)
}

func (s *Session) supervise() {
	s.readers.Wait()
	waitErr := s.cmd.Wait()

	s.mu.Lock()
	s.endedAt = time.Now()
	if waitErr != nil && s.err == nil {
		s.err = waitErr
	}
	failed := s.err != nil
	s.mu.Unlock()

	if failed {
		s.setState(StateFailed)
	} else {
		s.setState(StateExited)
	}

	s.cancel()
	<-s.writeDone
	s.tr.Close()
	close(s.events)
	close(s.done)
}

func (s *Session) emit(ev Event) {
	ev.Session = s.cfg.Name
	if ev.At.IsZero() {
		ev.At = time.Now()
	}
	ev.Snapshot = s.Snapshot()
	select {
	case s.events <- ev:
	case <-s.ctx.Done():
	}
}
