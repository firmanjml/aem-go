package progress

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type Tracker struct {
	slot      string
	stage     string
	label     string
	total     int64
	current   int64
	lastPrint time.Time
	done      bool
	mu        sync.Mutex
}

type stageState struct {
	label   string
	total   int64
	current int64
	done    bool
}

type slotState struct {
	name   string
	stages map[string]stageState
}

type manager struct {
	mu            sync.Mutex
	order         []string
	slots         map[string]*slotState
	renderedLines int
}

var defaultManager = &manager{
	slots: make(map[string]*slotState),
}

func New(label string, total int64) *Tracker {
	slot, stage := inferSlotAndStage(label)
	return defaultManager.newTracker(slot, stage, label, total)
}

func (m *manager) newTracker(slot, stage, label string, total int64) *Tracker {
	m.mu.Lock()
	defer m.mu.Unlock()

	state, exists := m.slots[slot]
	if !exists {
		state = &slotState{
			name:   slot,
			stages: make(map[string]stageState),
		}
		m.slots[slot] = state
		m.order = append(m.order, slot)
	}

	state.stages[stage] = stageState{
		label:   label,
		total:   total,
		current: 0,
		done:    false,
	}

	tracker := &Tracker{
		slot:  slot,
		stage: stage,
		label: label,
		total: total,
	}

	m.renderLocked()
	return tracker
}

func (t *Tracker) Add(delta int64) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.done {
		return
	}

	t.current += delta
	if !t.shouldRender() {
		return
	}

	defaultManager.updateTracker(t)
}

func (t *Tracker) Finish() {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.done {
		return
	}

	if t.total > 0 {
		t.current = t.total
	}
	t.done = true
	defaultManager.updateTracker(t)
}

func (t *Tracker) shouldRender() bool {
	return time.Since(t.lastPrint) >= 120*time.Millisecond || (t.total > 0 && t.current == t.total)
}

func (m *manager) updateTracker(t *Tracker) {
	m.mu.Lock()
	defer m.mu.Unlock()

	slotState := m.slots[t.slot]
	if slotState == nil {
		return
	}

	slotState.stages[t.stage] = stageState{
		label:   t.label,
		total:   t.total,
		current: t.current,
		done:    t.done,
	}
	t.lastPrint = time.Now()
	m.renderLocked()
}

func (m *manager) renderLocked() {
	if len(m.order) == 0 {
		return
	}

	if m.renderedLines > 0 {
		fmt.Fprintf(os.Stdout, "\033[%dA", m.renderedLines)
	}

	for _, slot := range m.order {
		slotState := m.slots[slot]
		if slotState == nil {
			continue
		}
		fmt.Fprintf(os.Stdout, "\r%s\033[K\n", renderSlot(slotState))
	}

	m.renderedLines = len(m.order)
}

func renderSlot(slot *slotState) string {
	parts := []string{slot.name}

	parts = append(parts, "download "+renderStageOrPending(slot.stages, "download"))
	parts = append(parts, "extract "+renderStageOrPending(slot.stages, "extract"))

	if stage, ok := slot.stages["other"]; ok {
		parts = append(parts, renderStage(stage))
	}

	return strings.Join(parts, " | ")
}

func renderStageOrPending(stages map[string]stageState, key string) string {
	stage, ok := stages[key]
	if !ok {
		return "pending"
	}
	return renderStage(stage)
}

func renderStage(stage stageState) string {
	if stage.total <= 0 {
		return humanizeBytes(stage.current)
	}

	percent := int64(0)
	if stage.total > 0 {
		percent = (stage.current * 100) / stage.total
	}

	barWidth := 16
	filled := int((percent * int64(barWidth)) / 100)
	if filled > barWidth {
		filled = barWidth
	}
	bar := strings.Repeat("#", filled) + strings.Repeat("-", barWidth-filled)

	return fmt.Sprintf("[%s] %3d%% (%s/%s)", bar, percent, humanizeBytes(stage.current), humanizeBytes(stage.total))
}

type Writer struct {
	reader  io.Reader
	tracker *Tracker
}

func NewWriter(reader io.Reader, tracker *Tracker) *Writer {
	return &Writer{
		reader:  reader,
		tracker: tracker,
	}
}

func (w *Writer) Read(p []byte) (int, error) {
	n, err := w.reader.Read(p)
	if n > 0 {
		w.tracker.Add(int64(n))
	}
	return n, err
}

func inferSlotAndStage(label string) (string, string) {
	lower := strings.ToLower(label)

	stage := "other"
	switch {
	case strings.HasPrefix(lower, "downloading "):
		stage = "download"
	case strings.HasPrefix(lower, "extracting "):
		stage = "extract"
	}

	slot := "default"
	switch {
	case strings.Contains(lower, "node"):
		slot = "node"
	case strings.Contains(lower, "zulu"), strings.Contains(lower, "jdk"), strings.Contains(lower, "java"):
		slot = "java"
	case strings.Contains(lower, "android"), strings.Contains(lower, "sdk"), strings.Contains(lower, "commandlinetools"):
		slot = "android"
	default:
		base := filepath.Base(lower)
		base = strings.TrimSpace(base)
		if base != "" {
			slot = base
		}
	}

	return slot, stage
}

func humanizeBytes(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}

	div, exp := int64(unit), 0
	for value := n / unit; value >= unit; value /= unit {
		div *= unit
		exp++
	}

	suffixes := []string{"KB", "MB", "GB", "TB"}
	return fmt.Sprintf("%.1f %s", float64(n)/float64(div), suffixes[exp])
}
