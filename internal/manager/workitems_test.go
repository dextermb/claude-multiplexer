package manager

import (
	"testing"

	"github.com/dextermb/claude-multiplexer/internal/workitem"
)

func TestApplyAndClearWorkItem(t *testing.T) {
	var meta Meta
	applyWorkItem(&meta, workitem.Item{
		Provider: "linear",
		Key:      "LIN-1",
		URL:      "https://linear.app/acme/issue/LIN-1",
		Status:   "In Review",
		StatusID: "s3",
	})
	if meta.WorkItemKey != "LIN-1" || meta.WorkItemStatus != "In Review" {
		t.Fatalf("apply did not mirror: %+v", meta)
	}
	if meta.WorkItemSyncedAt.IsZero() {
		t.Fatal("apply did not stamp the sync time")
	}

	view := workItemView(meta)
	if !view.Linked || view.Provider != "linear" || view.Status != "In Review" {
		t.Fatalf("view = %+v", view)
	}

	clearWorkItem(&meta)
	if meta.WorkItemKey != "" || meta.WorkItemStatus != "" || !meta.WorkItemSyncedAt.IsZero() {
		t.Fatalf("clear left data: %+v", meta)
	}
	if workItemView(meta).Linked {
		t.Fatal("a cleared item must not read as linked")
	}
}
