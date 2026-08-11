package progress

import "testing"

func TestRenderPercentStageOmitsByteCounts(t *testing.T) {
	got := renderStage(stageState{total: 100, current: 25, percent: true})
	const want = "[####------------]  25%"
	if got != want {
		t.Fatalf("renderStage() = %q, want %q", got, want)
	}
}

func TestRenderSlotUsesRemovalStage(t *testing.T) {
	slot := &slotState{
		name: "android",
		stages: map[string]stageState{
			"remove": {total: 100, current: 25, percent: true},
		},
	}
	const want = "android | remove [####------------]  25%"
	if got := renderSlot(slot); got != want {
		t.Fatalf("renderSlot() = %q, want %q", got, want)
	}
}
