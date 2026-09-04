package protocol

import (
	"bufio"
	"encoding/json"
	"errors"
	"io"
)

const MaxLineBytes = 16 << 20

var ErrNotJSON = errors.New("protocol: line is not a JSON object")

type envelope struct {
	Type      Type            `json:"type"`
	Subtype   string          `json:"subtype"`
	SessionID string          `json:"session_id"`
	Message   json.RawMessage `json:"message"`
	Event     json.RawMessage `json:"event"`
	IsReplay  bool            `json:"isReplay"`
}

type streamInner struct {
	Type  string `json:"type"`
	Index int    `json:"index"`
	Delta struct {
		Type        string `json:"type"`
		Text        string `json:"text"`
		Thinking    string `json:"thinking"`
		PartialJSON string `json:"partial_json"`
	} `json:"delta"`
}

func Decode(line []byte) (Event, error) {
	ev := Event{Raw: line}
	var env envelope
	if err := json.Unmarshal(line, &env); err != nil {
		return ev, ErrNotJSON
	}
	ev.Type = env.Type
	ev.Subtype = env.Subtype
	ev.SessionID = env.SessionID
	ev.IsReplay = env.IsReplay

	switch env.Type {
	case TypeSystem:
		switch env.Subtype {
		case SubtypeInit:
			var init Init
			if err := json.Unmarshal(line, &init); err != nil {
				return ev, nil
			}
			ev.Init = &init
		case SubtypeTaskStarted, SubtypeTaskUpdated, SubtypeTaskNotification:
			var task Task
			if err := json.Unmarshal(line, &task); err != nil {
				return ev, nil
			}
			ev.Task = &task
		}
	case TypeAssistant, TypeUser:
		if len(env.Message) == 0 {
			return ev, nil
		}
		var msg Message
		if err := json.Unmarshal(env.Message, &msg); err != nil {
			return ev, nil
		}
		ev.Message = &msg
	case TypeResult:
		var res Result
		if err := json.Unmarshal(line, &res); err != nil {
			return ev, nil
		}
		ev.Result = &res
	case TypeStreamEvent:
		if len(env.Event) == 0 {
			return ev, nil
		}
		var inner streamInner
		if err := json.Unmarshal(env.Event, &inner); err != nil {
			return ev, nil
		}
		ev.Delta = &Delta{
			Kind:        inner.Type,
			Index:       inner.Index,
			Text:        inner.Delta.Text,
			Thinking:    inner.Delta.Thinking,
			PartialJSON: inner.Delta.PartialJSON,
		}
	}
	return ev, nil
}

type Reader struct {
	br  *bufio.Reader
	max int
}

func NewReader(r io.Reader) *Reader {
	return &Reader{br: bufio.NewReaderSize(r, 64<<10), max: MaxLineBytes}
}

func (r *Reader) SetMaxLine(n int) {
	if n > 0 {
		r.max = n
	}
}

func (r *Reader) Next() (Event, error) {
	for {
		line, truncated, err := r.readLine()
		if err != nil {
			return Event{}, err
		}
		if len(trimSpace(line)) == 0 {
			continue
		}
		ev, decodeErr := Decode(line)
		ev.Truncated = truncated
		if truncated {
			return ev, ErrNotJSON
		}
		return ev, decodeErr
	}
}

func (r *Reader) readLine() ([]byte, bool, error) {
	var (
		buf       []byte
		truncated bool
	)
	for {
		chunk, err := r.br.ReadSlice('\n')
		if len(chunk) > 0 {
			if len(buf)+len(chunk) > r.max {
				truncated = true
				room := r.max - len(buf)
				if room > 0 {
					buf = append(buf, chunk[:room]...)
				}
			} else {
				buf = append(buf, chunk...)
			}
		}
		if errors.Is(err, bufio.ErrBufferFull) {
			continue
		}
		if err != nil {
			if len(buf) == 0 {
				return nil, false, err
			}
			return dropNewline(buf), truncated, nil
		}
		return dropNewline(buf), truncated, nil
	}
}

func dropNewline(b []byte) []byte {
	for len(b) > 0 && (b[len(b)-1] == '\n' || b[len(b)-1] == '\r') {
		b = b[:len(b)-1]
	}
	return b
}
