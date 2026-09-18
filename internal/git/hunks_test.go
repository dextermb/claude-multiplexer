package git

import "testing"

func TestHunksSplitsMultipleHunks(t *testing.T) {
	diff := `diff --git a/foo.go b/foo.go
index 111..222 100644
--- a/foo.go
+++ b/foo.go
@@ -1,3 +1,4 @@ func A()
 a
+b
 c
@@ -10,2 +11,3 @@ func B()
 d
+e
`
	hunks := Hunks(diff)
	if len(hunks) != 2 {
		t.Fatalf("want 2 hunks, got %d", len(hunks))
	}
	if hunks[0].NewStart != 1 || hunks[0].NewCount != 4 {
		t.Errorf("hunk 0 range = %d,%d want 1,4", hunks[0].NewStart, hunks[0].NewCount)
	}
	if hunks[0].NewEnd() != 4 {
		t.Errorf("hunk 0 end = %d want 4", hunks[0].NewEnd())
	}
	if hunks[1].NewStart != 11 || hunks[1].NewCount != 3 {
		t.Errorf("hunk 1 range = %d,%d want 11,3", hunks[1].NewStart, hunks[1].NewCount)
	}
	if len(hunks[0].Body) != 3 {
		t.Errorf("hunk 0 body = %d lines want 3", len(hunks[0].Body))
	}
}

func TestHunksNoHunkForRename(t *testing.T) {
	diff := `diff --git a/old.go b/new.go
similarity index 100%
rename from old.go
rename to new.go
`
	if hunks := Hunks(diff); len(hunks) != 0 {
		t.Fatalf("want 0 hunks for a pure rename, got %d", len(hunks))
	}
}

func TestHunksBinaryHasNoHunk(t *testing.T) {
	diff := "diff --git a/logo.png b/logo.png\nBinary files a/logo.png and b/logo.png differ\n"
	if hunks := Hunks(diff); len(hunks) != 0 {
		t.Fatalf("want 0 hunks for a binary file, got %d", len(hunks))
	}
}

func TestHunksPureDeletionHasZeroCount(t *testing.T) {
	diff := `diff --git a/foo.go b/foo.go
--- a/foo.go
+++ b/foo.go
@@ -5,3 +4,0 @@ func A()
-gone one
-gone two
-gone three
`
	hunks := Hunks(diff)
	if len(hunks) != 1 {
		t.Fatalf("want 1 hunk, got %d", len(hunks))
	}
	if hunks[0].NewCount != 0 {
		t.Errorf("count = %d want 0 for a pure deletion", hunks[0].NewCount)
	}
	if hunks[0].NewEnd() != hunks[0].NewStart {
		t.Errorf("end = %d want the anchor start %d", hunks[0].NewEnd(), hunks[0].NewStart)
	}
}

func TestHunksSingleLineHeaderCountsAsOne(t *testing.T) {
	diff := "@@ -0,0 +1 @@\n+only line\n"
	hunks := Hunks(diff)
	if len(hunks) != 1 {
		t.Fatalf("want 1 hunk, got %d", len(hunks))
	}
	if hunks[0].NewStart != 1 || hunks[0].NewCount != 1 {
		t.Errorf("range = %d,%d want 1,1", hunks[0].NewStart, hunks[0].NewCount)
	}
}
