package session

import "testing"

func TestQueueRemoveLastTakesTheTail(t *testing.T) {
	q := newQueue()
	q.push("a")
	q.push("b")
	q.push("c")

	item, ok := q.removeLast()
	if !ok || item != "c" {
		t.Fatalf("removeLast = %q, %v; want \"c\", true", item, ok)
	}
	if got := q.len(); got != 2 {
		t.Fatalf("len after removeLast = %d; want 2", got)
	}

	first, ok := q.pop()
	if !ok || first != "a" {
		t.Fatalf("pop = %q, %v; want \"a\", true", first, ok)
	}
}

func TestQueueRemoveLastOnEmptyReportsFalse(t *testing.T) {
	q := newQueue()
	if item, ok := q.removeLast(); ok || item != "" {
		t.Fatalf("removeLast on empty = %q, %v; want \"\", false", item, ok)
	}
}

func TestQueueRemoveLastEmptiesTheQueue(t *testing.T) {
	q := newQueue()
	q.push("only")

	if item, ok := q.removeLast(); !ok || item != "only" {
		t.Fatalf("removeLast = %q, %v; want \"only\", true", item, ok)
	}
	if got := q.len(); got != 0 {
		t.Fatalf("len = %d; want 0", got)
	}
	if _, ok := q.removeLast(); ok {
		t.Fatalf("removeLast on drained queue = true; want false")
	}
}
