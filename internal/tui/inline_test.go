package tui

import "testing"

func spanKinds(spans []inlineSpan) []spanKind {
	kinds := make([]spanKind, len(spans))
	for i, sp := range spans {
		kinds[i] = sp.kind
	}
	return kinds
}

func TestParseInlinePlain(t *testing.T) {
	spans := parseInline("just plain text")
	if len(spans) != 1 || spans[0].kind != spanPlain || spans[0].text != "just plain text" {
		t.Fatalf("got %+v", spans)
	}
}

func TestParseInlineItalic(t *testing.T) {
	spans := parseInline("an _italic_ word")
	want := []inlineSpan{
		{"an ", spanPlain},
		{"italic", spanItalic},
		{" word", spanPlain},
	}
	if len(spans) != len(want) {
		t.Fatalf("got %+v", spans)
	}
	for i := range want {
		if spans[i] != want[i] {
			t.Fatalf("span %d: got %+v want %+v", i, spans[i], want[i])
		}
	}
}

func TestParseInlineStarItalic(t *testing.T) {
	spans := parseInline("an *italic* word")
	want := []spanKind{spanPlain, spanItalic, spanPlain}
	got := spanKinds(spans)
	if len(got) != len(want) {
		t.Fatalf("got %+v", spans)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("kinds got %v want %v", got, want)
		}
	}
}

func TestParseInlineBold(t *testing.T) {
	spans := parseInline("a **bold** word")
	if len(spans) != 3 || spans[1].kind != spanBold || spans[1].text != "bold" {
		t.Fatalf("got %+v", spans)
	}
}

func TestParseInlineCode(t *testing.T) {
	spans := parseInline("run `go test` now")
	if len(spans) != 3 || spans[1].kind != spanCode || spans[1].text != "go test" {
		t.Fatalf("got %+v", spans)
	}
}

func TestParseInlineSnakeCaseStaysPlain(t *testing.T) {
	spans := parseInline("edit some_var_name here")
	if len(spans) != 1 || spans[0].kind != spanPlain {
		t.Fatalf("snake_case must stay plain, got %+v", spans)
	}
}

func TestParseInlineUnclosedStaysPlain(t *testing.T) {
	spans := parseInline("a _lonely mark")
	if len(spans) != 1 || spans[0].kind != spanPlain {
		t.Fatalf("unclosed mark must stay plain, got %+v", spans)
	}
}

func TestParseInlineMix(t *testing.T) {
	spans := parseInline("_i_ and **b** and `c`")
	want := []spanKind{spanItalic, spanPlain, spanBold, spanPlain, spanCode}
	got := spanKinds(spans)
	if len(got) != len(want) {
		t.Fatalf("got %+v", spans)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("kinds got %v want %v", got, want)
		}
	}
}

func TestParseInlineMarkerStaysPlain(t *testing.T) {
	spans := parseInline("› a plain prompt")
	if len(spans) != 1 || spans[0].kind != spanPlain {
		t.Fatalf("got %+v", spans)
	}
}
