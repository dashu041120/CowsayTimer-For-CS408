package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestAutoAddCheckbox(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test_tasks.md")

	initialContent := `# 08-06 安排

- 英语单词 100个
- [ ] 数学真题
- [x] 政治复习
专业课图表绘制
`
	if err := os.WriteFile(testFile, []byte(initialContent), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	lines, err := loadMDFile(testFile)
	if err != nil {
		t.Fatalf("failed to load md file: %v", err)
	}

	if len(lines) != 6 {
		t.Fatalf("expected 6 lines, got %d", len(lines))
	}

	// Line 2 ("- 英语单词 100个") should be converted to task with done = false
	if !lines[2].isTask || lines[2].done || lines[2].text != "英语单词 100个" {
		t.Errorf("line 2 mismatch: %+v", lines[2])
	}

	// Line 3 ("- [ ] 数学真题") task done = false
	if !lines[3].isTask || lines[3].done || lines[3].text != "数学真题" {
		t.Errorf("line 3 mismatch: %+v", lines[3])
	}

	// Line 4 ("- [x] 政治复习") task done = true
	if !lines[4].isTask || !lines[4].done || lines[4].text != "政治复习" {
		t.Errorf("line 4 mismatch: %+v", lines[4])
	}

	// Line 5 ("专业课图表绘制") should be auto converted to task with done = false
	if !lines[5].isTask || lines[5].done || lines[5].text != "专业课图表绘制" {
		t.Errorf("line 5 mismatch: %+v", lines[5])
	}

	// Check saved file content
	savedData, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("failed to read saved file: %v", err)
	}
	savedStr := string(savedData)

	if !strings.Contains(savedStr, "- [ ] 英语单词 100个") {
		t.Errorf("saved content missing auto checkbox for line 2: %s", savedStr)
	}
	if !strings.Contains(savedStr, "- [ ] 专业课图表绘制") {
		t.Errorf("saved content missing auto checkbox for line 5: %s", savedStr)
	}
}

func TestToggleTaskAndSave(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "toggle_test.md")

	initialContent := `- [ ] 任务A
- [ ] 任务B
`
	_ = os.WriteFile(testFile, []byte(initialContent), 0644)

	lines, err := loadMDFile(testFile)
	if err != nil {
		t.Fatalf("failed to load md file: %v", err)
	}

	// Toggle first task
	lines[0].done = true
	if err := saveMDFile(testFile, lines); err != nil {
		t.Fatalf("failed to save md file: %v", err)
	}

	savedData, _ := os.ReadFile(testFile)
	savedStr := string(savedData)

	if !strings.Contains(savedStr, "- [x] 任务A") {
		t.Errorf("expected '- [x] 任务A' in saved content, got: %s", savedStr)
	}
}

func TestResolveTaskFile(t *testing.T) {
	tempDir := t.TempDir()
	origWd, _ := os.Getwd()
	_ = os.Chdir(tempDir)
	defer func() { _ = os.Chdir(origWd) }()

	mmdd := time.Now().Format("01-02.md")

	resolved := resolveTaskFile("")
	if resolved != mmdd {
		t.Errorf("expected resolved file to be %s, got %s", mmdd, resolved)
	}

	// Verify file was created
	if _, err := os.Stat(mmdd); os.IsNotExist(err) {
		t.Errorf("expected %s to be created automatically", mmdd)
	}
}

func TestASCIIGlyphs(t *testing.T) {
	formatted := formatStandardDuration(45*time.Second + 4*time.Minute)
	if formatted != "04:45" {
		t.Errorf("expected '04:45', got '%s'", formatted)
	}

	rendered := renderASCIIText("0123456789:")
	lines := strings.Split(rendered, "\n")
	if len(lines) != 6 {
		t.Fatalf("expected 6 lines in ascii rendering, got %d", len(lines))
	}
}

func TestFullWindowCanvas(t *testing.T) {
	m := initialModel("")
	m.width = 100
	m.height = 30
	m.initStars()

	if len(m.starGrid) == 0 {
		t.Fatalf("expected starGrid to be initialized")
	}

	canvas := m.generateBackgroundCanvas()
	if len(canvas) != 30 || len(canvas[0]) != 100 {
		t.Fatalf("expected canvas size 30x100, got %dx%d", len(canvas), len(canvas[0]))
	}

	bgRow := make([]string, 10)
	bgRow[6] = "★"
	result := overlayBackgroundOnLine("Hello     ", bgRow)
	if !strings.Contains(result, "★") {
		t.Errorf("expected overlay to inject star ★ into space, got: %s", result)
	}
}

func TestCompletionCelebrationAnimation(t *testing.T) {
	m := initialModel("")
	m.width = 100
	m.height = 30

	for i := range m.lines {
		if m.lines[i].isTask {
			m.lines[i].done = true
		}
	}

	m.checkCompletionTrigger()

	if !m.wasCompleted {
		t.Errorf("expected wasCompleted to be true when all tasks are done")
	}
	if !m.celebrating {
		t.Errorf("expected celebrating to be true when all tasks are done")
	}
	if len(m.fwParticles) == 0 {
		t.Errorf("expected fireworks particles to be spawned")
	}
}
