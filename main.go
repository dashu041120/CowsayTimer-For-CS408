package main

import (
	"fmt"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
)

// ---------------------------------------------------------
// 样式定义 (Lip Gloss)
// ---------------------------------------------------------

var (
	// 色彩系统
	primaryColor   = lipgloss.Color("#7D56F4") // Charm 紫
	secondaryColor = lipgloss.Color("#04B575") // 薄荷绿
	accentColor    = lipgloss.Color("#FF5F87") // 珊瑚粉
	cyanColor      = lipgloss.Color("#00F5D4") // 霓虹青
	subtleColor    = lipgloss.Color("#6272A4") // 灰色
	mutedColor     = lipgloss.Color("#44475A") // 暗底色
	fgColor        = lipgloss.Color("#F8F8F2") // 前景色

	// 顶部 Header 样式
	headerTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#FFFFFF")).
				Background(primaryColor).
				Padding(0, 1)

	kaoyanBadgeStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#1E1E2E")).
				Background(cyanColor).
				Padding(0, 1)

	// 大数字样式
	timerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(secondaryColor)

	// Cowsay 对话框边框样式
	speechBubbleStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(primaryColor).
				Padding(0, 1)

	cowStyle = lipgloss.NewStyle().
			Foreground(primaryColor)

	// 状态样式
	runningStyle = lipgloss.NewStyle().
			Foreground(secondaryColor).
			Bold(true)

	pausedStyle = lipgloss.NewStyle().
			Foreground(accentColor).
			Bold(true)

	// 计圈卡片样式
	lapBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder(), false, false, false, true).
			BorderForeground(primaryColor).
			PaddingLeft(1).
			MarginTop(1)

	lapHeaderStyle = lipgloss.NewStyle().
			Foreground(primaryColor).
			Bold(true)

	lapTextStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#D9D9D9"))

	// 侧边栏任务框样式
	sidebarActiveStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(cyanColor).
				Padding(0, 1)

	sidebarInactiveStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(mutedColor).
				Padding(0, 1)

	sidebarTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(cyanColor)

	fileBadgeStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#1E1E2E")).
			Background(primaryColor).
			Padding(0, 1).
			Bold(true)

	progressFilledStyle = lipgloss.NewStyle().
				Foreground(secondaryColor)

	progressEmptyStyle = lipgloss.NewStyle().
				Foreground(mutedColor)

	taskNormalStyle = lipgloss.NewStyle().
			Foreground(fgColor)

	taskDoneStyle = lipgloss.NewStyle().
			Foreground(subtleColor).
			Strikethrough(true)

	taskCursorStyle = lipgloss.NewStyle().
			Foreground(cyanColor).
			Bold(true)

	checkboxStyle = lipgloss.NewStyle().
			Foreground(secondaryColor).
			Bold(true)

	uncheckedStyle = lipgloss.NewStyle().
			Foreground(subtleColor)

	// 底部帮助快捷键样式
	helpStyle = lipgloss.NewStyle().
			Foreground(subtleColor)

	inputPromptStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(accentColor)
)

// ---------------------------------------------------------
// Markdown 任务数据结构与处理 logic
// ---------------------------------------------------------

type mdLine struct {
	rawText string // 原始行内容
	isTask  bool   // 是否为任务项
	done    bool   // 是否已勾选
	text    string // 任务文本
}

// ---------------------------------------------------------
// Xiaomi MiMo-Code 星空与 Braille 盲文流星引擎数据结构
// ---------------------------------------------------------

type starCell struct {
	isStar     bool
	charIdx    int
	period     float64 // 呼吸周期 (35~80 ticks)
	phase      float64 // 随机相位
	baseBright float64
	amp        float64
}

type mimoMeteor struct {
	startTime time.Time
	startX    float64
	startY    float64
	angle     float64 // 随机偏转角度 (0.18 ~ 0.52 rad)
	speed     float64
	tailLen   float64 // 随机尾巴长度 (20 ~ 36)
	active    bool
}

type activePanel int

const (
	panelMain activePanel = iota
	panelTasks
)

type inputMode int

const (
	inputNone inputMode = iota
	inputAddTask
	inputEditTask
	inputSwitchFile
	inputConfirmDelete
)

type tickMsg time.Time

type particle struct {
	x, y    float64
	vx, vy  float64
	char    string
	color   lipgloss.Color
	life    int
	maxLife int
}

type model struct {
	elapsed          time.Duration
	running          bool
	laps             []string
	lastTick         time.Time
	activePanel      activePanel
	showHelp         bool
	tickCount        int
	starGrid         [][]starCell
	mimoMeteor       *mimoMeteor

	// 100% 任务完成动画状态
	wasCompleted     bool
	celebrating      bool
	celebrationTicks int
	fwParticles      []particle

	// 任务侧边栏状态
	taskFilePath     string
	lines            []mdLine
	taskCursor       int

	// 输入模式状态
	inputMode        inputMode
	inputText        string

	// 终端尺寸
	width            int
	height           int
}

func tickCmd() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// Map a (col, row) within a 2x4 Braille sub-grid to its bit index in U+2800
func brailleBit(col, row int) int {
	if col == 0 {
		if row == 3 {
			return 6
		}
		return row
	}
	if row == 3 {
		return 7
	}
	return 3 + row
}

func (m *model) initStars() {
	if m.width <= 0 || m.height <= 0 {
		return
	}
	m.starGrid = make([][]starCell, m.height)
	density := 0.005 // ~0.5% 密度的星星
	for y := 0; y < m.height; y++ {
		row := make([]starCell, m.width)
		for x := 0; x < m.width; x++ {
			if rand.Float64() < density {
				row[x] = starCell{
					isStar:     true,
					charIdx:    rand.Intn(2),
					period:     35.0 + rand.Float64()*45.0, // 3.5s ~ 8.0s 缓和正弦波呼吸
					phase:      rand.Float64() * 2 * math.Pi,
					baseBright: 0.35 + rand.Float64()*0.2,
					amp:        0.35 + rand.Float64()*0.25,
				}
			}
		}
		m.starGrid[y] = row
	}
}

func (m *model) updateTwinkle() {
	if len(m.starGrid) != m.height || (m.height > 0 && len(m.starGrid[0]) != m.width) {
		m.initStars()
		return
	}
}

func (m *model) checkCompletionTrigger() {
	taskIndices := m.getTaskIndices()
	total := len(taskIndices)
	if total == 0 {
		m.wasCompleted = false
		m.celebrating = false
		m.fwParticles = nil
		return
	}

	completed := 0
	for _, idx := range taskIndices {
		if m.lines[idx].done {
			completed++
		}
	}

	if completed == total {
		if !m.wasCompleted {
			m.wasCompleted = true
			m.celebrating = true
			m.celebrationTicks = 0
			m.spawnFireworks()
		}
	} else {
		m.wasCompleted = false
		m.celebrating = false
		m.fwParticles = nil
	}
}

func (m *model) spawnFireworks() {
	m.fwParticles = nil
	m.launchFountainBatch(50)
}

func (m *model) launchFountainBatch(count int) {
	w := math.Max(40, float64(m.width))
	h := math.Max(15, float64(m.height))

	launchers := []float64{w * 0.20, w * 0.50, w * 0.80}

	symbols := []string{"✨", "⭐", "🌟", "🎆", "🎉", "🎊", "💥", "✦", "★", "🏆", "🥳"}
	colors := []lipgloss.Color{
		lipgloss.Color("#FFD700"), // 金黄
		lipgloss.Color("#FFEA00"), // 亮金
		cyanColor,                 // 霓虹青
		accentColor,               // 珊瑚粉
		lipgloss.Color("#FF2A85"), // 爆紫粉
		lipgloss.Color("#9D4EDD"), // 霓虹紫
		lipgloss.Color("#FFFFFF"), // 亮白
	}

	for i := 0; i < count; i++ {
		lx := launchers[rand.Intn(len(launchers))]
		startY := h - 2.0 // 底部向上喷涌

		// 向上喷射的高初速度 (vy 为负向向上)
		angle := -math.Pi/2.0 + (rand.Float64()-0.5)*0.95 // -90° 向上喷射，扇面散开
		speed := 1.1 + rand.Float64()*1.5                  // 喷涌高速

		m.fwParticles = append(m.fwParticles, particle{
			x:       lx,
			y:       startY,
			vx:      math.Cos(angle) * speed,
			vy:      math.Sin(angle) * speed * 0.55,
			char:    symbols[rand.Intn(len(symbols))],
			color:   colors[rand.Intn(len(colors))],
			life:    35 + rand.Intn(20),
			maxLife: 55,
		})
	}
}

func (m *model) updateCelebration() {
	if !m.celebrating {
		return
	}
	m.celebrationTicks++

	// 持续喷涌 6 波金彩喷泉 (每 6 帧续喷一波，连喷 150+ 粒子)
	if m.celebrationTicks%6 == 0 && m.celebrationTicks <= 36 {
		m.launchFountainBatch(25)
	}

	// 粒子重力与轨迹更新 (抛物线回落)
	for i := range m.fwParticles {
		p := &m.fwParticles[i]
		if p.life > 0 {
			p.x += p.vx
			p.y += p.vy
			p.vy += 0.045 // 重力回落
			p.life--
		}
	}
}

func (m *model) updateMeteors() {
	m.updateTwinkle()
	m.updateCelebration()

	now := time.Now()
	if m.mimoMeteor != nil && m.mimoMeteor.active {
		elapsed := now.Sub(m.mimoMeteor.startTime).Milliseconds()
		if elapsed > 3800 {
			m.mimoMeteor.active = false
		}
	}

	// 概率产生新流星 (每 ~35 ticks 间隔产生)
	if (m.mimoMeteor == nil || !m.mimoMeteor.active) && m.tickCount%35 == 0 {
		maxY := m.height / 3
		if maxY < 1 {
			maxY = 1
		}
		startY := float64(rand.Intn(maxY))
		speed := 0.020 + rand.Float64()*0.015
		// 随机微调偏转角: 0.18 ~ 0.52 弧度 (~10° 到 ~30° 视角)
		angle := 0.18 + rand.Float64()*0.34
		tailLen := 20.0 + rand.Float64()*16.0

		m.mimoMeteor = &mimoMeteor{
			startTime: now,
			startX:    float64(m.width) - rand.Float64()*math.Max(1.0, float64(m.width)*0.2),
			startY:    startY,
			angle:     angle,
			speed:     speed,
			tailLen:   tailLen,
			active:    true,
		}
	}
}

func (m model) generateBackgroundCanvas() [][]string {
	if m.width <= 0 || m.height <= 0 {
		return nil
	}

	canvas := make([][]string, m.height)
	for y := 0; y < m.height; y++ {
		canvas[y] = make([]string, m.width)
	}

	// 1. 渲染平滑正弦波渐变繁星 (不频繁剧烈闪烁)
	starChars := []string{"✦", "✧"}
	hotChar := "✶"
	t := float64(m.tickCount)

	for y := 0; y < m.height && y < len(m.starGrid); y++ {
		for x := 0; x < m.width && x < len(m.starGrid[y]); x++ {
			cell := m.starGrid[y][x]
			if !cell.isStar {
				continue
			}

			// 计算连续平滑正弦波亮度，平缓渐明渐灭
			b := cell.baseBright + cell.amp*math.Sin((2.0*math.Pi*t)/cell.period+cell.phase)
			if b < 0.15 {
				continue // 暗淡熄灭状态
			}

			var char string
			var color lipgloss.Color

			if b >= 0.88 {
				char = hotChar
				color = lipgloss.Color("#FFFFFF") // Hot spike 爆光
			} else {
				char = starChars[cell.charIdx%len(starChars)]
				if b > 0.70 {
					color = cyanColor
				} else if b > 0.45 {
					color = lipgloss.Color("#EDDCFA")
				} else if b > 0.25 {
					color = lipgloss.Color("#6272A4")
				} else {
					color = lipgloss.Color("#44475A")
				}
			}

			canvas[y][x] = lipgloss.NewStyle().Foreground(color).Render(char)
		}
	}

	// 2. 渲染带随机角度偏转的 2x4 Sub-Pixel Braille 盲文流星
	if m.mimoMeteor != nil && m.mimoMeteor.active {
		elapsed := float64(time.Since(m.mimoMeteor.startTime).Milliseconds())
		if elapsed >= 0 && elapsed <= 3800 {
			meteorAngle := m.mimoMeteor.angle
			meteorTail := m.mimoMeteor.tailLen
			meteorStep := 0.15

			distance := elapsed * m.mimoMeteor.speed
			dx := -math.Cos(meteorAngle)
			dy := math.Sin(meteorAngle)
			headX := m.mimoMeteor.startX + distance*dx
			headY := m.mimoMeteor.startY + distance*dy

			envelope := math.Sin((elapsed / 3800.0) * math.Pi)

			type cellInfo struct {
				dots int
				minT float64
			}
			cellAcc := make(map[string]*cellInfo)

			setDot := func(px, py, t float64) {
				subX := int(math.Floor(px * 2.0))
				subY := int(math.Floor(py * 4.0))
				cx := subX >> 1
				cy := subY >> 2
				if cx < 0 || cx >= m.width || cy < 0 || cy >= m.height {
					return
				}
				bit := brailleBit(subX&1, subY&3)
				key := fmt.Sprintf("%d,%d", cx, cy)
				if existing, ok := cellAcc[key]; ok {
					existing.dots |= (1 << bit)
					if t < existing.minT {
						existing.minT = t
					}
				} else {
					cellAcc[key] = &cellInfo{
						dots: 1 << bit,
						minT: t,
					}
				}
			}

			// 沿尾巴轨迹步进
			for t := 0.0; t <= meteorTail; t += meteorStep {
				setDot(headX-t*dx, headY-t*dy, t)
			}

			// 绘制圆润头部 Core
			headSubX := int(math.Floor(headX * 2.0))
			headSubY := int(math.Floor(headY * 4.0))
			for dsx := -1; dsx <= 1; dsx++ {
				for dsy := -1; dsy <= 1; dsy++ {
					if dsx*dsx+dsy*dsy > 1 {
						continue
					}
					subX := headSubX + dsx
					subY := headSubY + dsy
					cx := subX >> 1
					cy := subY >> 2
					if cx < 0 || cx >= m.width || cy < 0 || cy >= m.height {
						continue
					}
					bit := brailleBit(subX&1, subY&3)
					key := fmt.Sprintf("%d,%d", cx, cy)
					if existing, ok := cellAcc[key]; ok {
						existing.dots |= (1 << bit)
						existing.minT = 0
					} else {
						cellAcc[key] = &cellInfo{
							dots: 1 << bit,
							minT: 0,
						}
					}
				}
			}

			// 将 Braille bitmask 合成并写回 Canvas
			for key, val := range cellAcc {
				var cx, cy int
				fmt.Sscanf(key, "%d,%d", &cx, &cy)
				if cx >= 0 && cx < m.width && cy >= 0 && cy < m.height {
					brailleChar := string(rune(0x2800 + val.dots))

					fade := math.Pow(math.Max(0, 1.0-val.minT/meteorTail), 1.3) * envelope
					if fade < 0.05 {
						continue // 淡入淡出边界隐形
					}

					var color lipgloss.Color
					if fade >= 0.70 && val.minT < 3.0 {
						color = lipgloss.Color("#FFFFFF") // 盛期头部核心亮白
					} else if fade >= 0.45 {
						color = cyanColor // 霓虹青
					} else if fade >= 0.22 {
						color = lipgloss.Color("#80B4FF") // 柔蓝
					} else {
						color = lipgloss.Color("#3A5070") // 渐隐暗蓝
					}

					canvas[cy][cx] = lipgloss.NewStyle().Foreground(color).Bold(true).Render(brailleChar)
				}
			}
		}
	}

	// 3. 渲染 100% 任务完成 Celebration 烟花粒子
	for _, p := range m.fwParticles {
		if p.life > 0 {
			px := int(p.x)
			py := int(p.y)
			if px >= 0 && px < m.width && py >= 0 && py < m.height {
				canvas[py][px] = lipgloss.NewStyle().Foreground(p.color).Bold(true).Render(p.char)
			}
		}
	}

	return canvas
}

func overlayBackgroundOnLine(uiLine string, bgRow []string) string {
	if len(bgRow) == 0 {
		return uiLine
	}

	var sb strings.Builder
	col := 0
	skipNextSpace := false

	for i := 0; i < len(uiLine); {
		if uiLine[i] == '\x1b' {
			j := i
			for j < len(uiLine) && !isAnsiEndChar(uiLine[j]) {
				j++
			}
			if j < len(uiLine) {
				j++
			}
			sb.WriteString(uiLine[i:j])
			i = j
			continue
		}

		r, size := utf8.DecodeRuneInString(uiLine[i:])
		i += size

		if skipNextSpace {
			skipNextSpace = false
			w := runewidth.RuneWidth(r)
			if w < 1 {
				w = 1
			}
			col += w
			continue
		}

		if r == ' ' && col < len(bgRow) && bgRow[col] != "" {
			bgElem := bgRow[col]
			bgW := lipgloss.Width(bgElem)
			if bgW <= 1 {
				sb.WriteString(bgElem)
			} else if bgW == 2 {
				// 检查下一个字符是否也是空格，避免挤变形卡片右边框
				if i < len(uiLine) && uiLine[i] == ' ' {
					sb.WriteString(bgElem)
					skipNextSpace = true
				} else {
					sb.WriteString(string(r))
				}
			} else {
				sb.WriteString(string(r))
			}
		} else {
			sb.WriteString(string(r))
		}

		w := runewidth.RuneWidth(r)
		if w < 1 {
			w = 1
		}
		col += w
	}

	return sb.String()
}

func isAnsiEndChar(c byte) bool {
	return (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z')
}

// ---------------------------------------------------------
// MD 文件读写与自动补充 checkbox 逻辑
// ---------------------------------------------------------

func getCandidateFiles() []string {
	now := time.Now()
	mmdd := now.Format("01-02.md")
	yyyymmdd := now.Format("2006-01-02.md")
	return []string{mmdd, yyyymmdd, "tasks.md"}
}

func resolveTaskFile(specifiedPath string) string {
	if specifiedPath != "" {
		return specifiedPath
	}
	candidates := getCandidateFiles()
	for _, file := range candidates {
		if _, err := os.Stat(file); err == nil {
			return file
		}
	}
	// 默认自动创建 MM-DD.md
	defaultFile := candidates[0]
	createDefaultMD(defaultFile)
	return defaultFile
}

func createDefaultMD(filename string) {
	nowStr := time.Now().Format("01-02")
	content := fmt.Sprintf("# %s 当日计划\n\n- [ ] 英语单词背诵 & 复习\n- [ ] 数学真题刷题 & 错题总结\n- [ ] 政治章节练习\n- [ ] 专业课知识点整理\n", nowStr)
	_ = os.WriteFile(filename, []byte(content), 0644)
}

func loadMDFile(filepath string) ([]mdLine, error) {
	data, err := os.ReadFile(filepath)
	if err != nil {
		return nil, err
	}

	content := strings.TrimRight(string(data), "\r\n")
	if content == "" {
		return nil, nil
	}
	rawLines := strings.Split(content, "\n")
	var lines []mdLine
	modified := false

	checkboxReg := regexp.MustCompile(`^\s*[-*+]\s*\[([ xX])\]\s*(.*)$`)
	listPrefixReg := regexp.MustCompile(`^\s*(?:[-*+]|\d+\.)\s*(.*)$`)

	for _, line := range rawLines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			lines = append(lines, mdLine{
				rawText: line,
				isTask:  false,
			})
			continue
		}

		// 匹配已有 checkbox
		if loc := checkboxReg.FindStringSubmatch(line); len(loc) == 3 {
			done := strings.ToLower(loc[1]) == "x"
			text := strings.TrimSpace(loc[2])
			lines = append(lines, mdLine{
				rawText: line,
				isTask:  true,
				done:    done,
				text:    text,
			})
			continue
		}

		// 非 header、非空行，但没有 checkbox -> 自动加入 checkbox!
		var taskText string
		if loc := listPrefixReg.FindStringSubmatch(trimmed); len(loc) == 2 {
			taskText = strings.TrimSpace(loc[1])
		} else {
			taskText = trimmed
		}

		if taskText != "" {
			lines = append(lines, mdLine{
				rawText: "- [ ] " + taskText,
				isTask:  true,
				done:    false,
				text:    taskText,
			})
			modified = true
		} else {
			lines = append(lines, mdLine{
				rawText: line,
				isTask:  false,
			})
		}
	}

	if modified {
		_ = saveMDFile(filepath, lines)
	}

	return lines, nil
}

func saveMDFile(filepath string, lines []mdLine) error {
	var sb strings.Builder
	for i, l := range lines {
		if i > 0 {
			sb.WriteString("\n")
		}
		if l.isTask {
			checkStr := "[ ]"
			if l.done {
				checkStr = "[x]"
			}
			sb.WriteString(fmt.Sprintf("- %s %s", checkStr, l.text))
		} else {
			sb.WriteString(l.rawText)
		}
	}
	sb.WriteString("\n")
	return os.WriteFile(filepath, []byte(sb.String()), 0644)
}

func initialModel(specifiedFile string) model {
	targetFile := resolveTaskFile(specifiedFile)
	lines, _ := loadMDFile(targetFile)

	m := model{
		elapsed:      0,
		running:      false,
		laps:         []string{},
		activePanel:  panelMain,
		showHelp:     false,
		taskFilePath: targetFile,
		lines:        lines,
		taskCursor:   0,
		inputMode:    inputNone,
		inputText:    "",
		width:        80,
		height:       24,
	}
	m.initStars()
	m.checkCompletionTrigger()
	return m
}

func (m model) Init() tea.Cmd {
	return tickCmd()
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.initStars()
		return m, nil

	case tea.KeyMsg:
		// 处理文本输入模式
		if m.inputMode != inputNone {
			switch msg.String() {
			case "y", "Y":
				if m.inputMode == inputConfirmDelete {
					taskIndices := m.getTaskIndices()
					if len(taskIndices) > 0 && m.taskCursor < len(taskIndices) {
						targetIdx := taskIndices[m.taskCursor]
						m.lines = append(m.lines[:targetIdx], m.lines[targetIdx+1:]...)
						_ = saveMDFile(m.taskFilePath, m.lines)
						if m.taskCursor >= len(m.getTaskIndices()) && m.taskCursor > 0 {
							m.taskCursor--
						}
					}
					m.inputMode = inputNone
					m.inputText = ""
					return m, nil
				}

			case "n", "N":
				if m.inputMode == inputConfirmDelete {
					m.inputMode = inputNone
					m.inputText = ""
					return m, nil
				}

			case "enter":
				if m.inputMode == inputConfirmDelete {
					taskIndices := m.getTaskIndices()
					if len(taskIndices) > 0 && m.taskCursor < len(taskIndices) {
						targetIdx := taskIndices[m.taskCursor]
						m.lines = append(m.lines[:targetIdx], m.lines[targetIdx+1:]...)
						_ = saveMDFile(m.taskFilePath, m.lines)
						if m.taskCursor >= len(m.getTaskIndices()) && m.taskCursor > 0 {
							m.taskCursor--
						}
					}
				} else if m.inputMode == inputAddTask {
					text := strings.TrimSpace(m.inputText)
					if text != "" {
						m.lines = append(m.lines, mdLine{
							rawText: "- [ ] " + text,
							isTask:  true,
							done:    false,
							text:    text,
						})
						_ = saveMDFile(m.taskFilePath, m.lines)
						taskIndices := m.getTaskIndices()
						if len(taskIndices) > 0 {
							m.taskCursor = len(taskIndices) - 1
						}
					}
				} else if m.inputMode == inputEditTask {
					text := strings.TrimSpace(m.inputText)
					taskIndices := m.getTaskIndices()
					if text != "" && len(taskIndices) > 0 && m.taskCursor < len(taskIndices) {
						targetIdx := taskIndices[m.taskCursor]
						m.lines[targetIdx].text = text
						_ = saveMDFile(m.taskFilePath, m.lines)
					}
				} else if m.inputMode == inputSwitchFile {
					fileName := strings.TrimSpace(m.inputText)
					if fileName != "" {
						if !strings.HasSuffix(fileName, ".md") {
							fileName += ".md"
						}
						if _, err := os.Stat(fileName); os.IsNotExist(err) {
							createDefaultMD(fileName)
						}
						m.taskFilePath = fileName
						lines, _ := loadMDFile(fileName)
						m.lines = lines
						m.taskCursor = 0
					}
				}
				m.inputMode = inputNone
				m.inputText = ""
				m.checkCompletionTrigger()
				return m, nil

			case "esc":
				m.inputMode = inputNone
				m.inputText = ""
				return m, nil

			case "backspace":
				runes := []rune(m.inputText)
				if len(runes) > 0 {
					m.inputText = string(runes[:len(runes)-1])
				}
				return m, nil

			default:
				if len(msg.Runes) > 0 {
					m.inputText += string(msg.Runes)
				} else if len(msg.String()) == 1 {
					m.inputText += msg.String()
				}
				return m, nil
			}
		}

		// 正常按键模式
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit

		case "h", "?":
			m.showHelp = !m.showHelp
			return m, nil

		case "esc":
			if m.showHelp {
				m.showHelp = false
				return m, nil
			}

		case "tab":
			if m.activePanel == panelMain {
				m.activePanel = panelTasks
			} else {
				m.activePanel = panelMain
			}
			return m, nil

		case "a":
			m.activePanel = panelTasks
			m.inputMode = inputAddTask
			m.inputText = ""
			return m, nil

		case "e":
			m.activePanel = panelTasks
			taskIndices := m.getTaskIndices()
			if len(taskIndices) > 0 && m.taskCursor < len(taskIndices) {
				targetIdx := taskIndices[m.taskCursor]
				m.inputMode = inputEditTask
				m.inputText = m.lines[targetIdx].text
			}
			return m, nil

		case "n":
			m.inputMode = inputSwitchFile
			m.inputText = ""
			return m, nil

		case "d":
			if m.activePanel == panelTasks {
				taskIndices := m.getTaskIndices()
				if len(taskIndices) > 0 && m.taskCursor < len(taskIndices) {
					m.inputMode = inputConfirmDelete
					m.inputText = ""
				}
			}
			return m, nil

		case "j", "down":
			if m.activePanel == panelTasks {
				taskCount := len(m.getTaskIndices())
				if taskCount > 0 && m.taskCursor < taskCount-1 {
					m.taskCursor++
				}
			}
			return m, nil

		case "k", "up":
			if m.activePanel == panelTasks {
				if m.taskCursor > 0 {
					m.taskCursor--
				}
			}
			return m, nil

		case " ", "x":
			if m.activePanel == panelTasks {
				taskIndices := m.getTaskIndices()
				if len(taskIndices) > 0 && m.taskCursor < len(taskIndices) {
					targetIdx := taskIndices[m.taskCursor]
					m.lines[targetIdx].done = !m.lines[targetIdx].done
					_ = saveMDFile(m.taskFilePath, m.lines)
					m.checkCompletionTrigger()
				}
			} else if msg.String() == " " || msg.String() == "s" {
				m.running = !m.running
				if m.running {
					m.lastTick = time.Now()
				}
			}
			return m, nil

		case "s":
			m.running = !m.running
			if m.running {
				m.lastTick = time.Now()
			}
			return m, nil

		case "r":
			if m.activePanel == panelMain {
				m.running = false
				m.elapsed = 0
				m.laps = nil
			}
			return m, nil

		case "l", "enter":
			if m.activePanel == panelMain {
				if m.running || m.elapsed > 0 {
					lapNum := len(m.laps) + 1
					lapStr := fmt.Sprintf("Lap %02d   %s", lapNum, formatStandardDuration(m.elapsed))
					m.laps = append(m.laps, lapStr)
				}
			} else if m.activePanel == panelTasks {
				taskIndices := m.getTaskIndices()
				if len(taskIndices) > 0 && m.taskCursor < len(taskIndices) {
					targetIdx := taskIndices[m.taskCursor]
					m.lines[targetIdx].done = !m.lines[targetIdx].done
					_ = saveMDFile(m.taskFilePath, m.lines)
					m.checkCompletionTrigger()
				}
			}
			return m, nil
		}

	case tickMsg:
		m.tickCount++
		now := time.Time(msg)
		if m.running {
			m.elapsed += now.Sub(m.lastTick)
		}
		m.lastTick = now
		m.updateMeteors()
		return m, tickCmd()
	}

	return m, nil
}

func (m model) getTaskIndices() []int {
	var indices []int
	for idx, l := range m.lines {
		if l.isTask {
			indices = append(indices, idx)
		}
	}
	return indices
}

// ---------------------------------------------------------
// View 渲染
// ---------------------------------------------------------

func (m model) View() string {
	rawUI := m.renderBaseUI()
	uiLines := strings.Split(rawUI, "\n")
	bgCanvas := m.generateBackgroundCanvas()

	if len(bgCanvas) == 0 {
		return rawUI
	}

	var finalLines []string
	for y := 0; y < len(uiLines) && y < m.height; y++ {
		var bgRow []string
		if y < len(bgCanvas) {
			bgRow = bgCanvas[y]
		}
		finalLines = append(finalLines, overlayBackgroundOnLine(uiLines[y], bgRow))
	}

	return strings.Join(finalLines, "\n")
}

func (m model) renderBaseUI() string {
	var doc strings.Builder

	// 1. 顶部 Header
	header := m.renderHeader()
	doc.WriteString(header + "\n")

	isWide := m.width >= 85

	// 2. 帮助弹窗模式或主界面
	if m.showHelp {
		doc.WriteString(lipgloss.Place(m.width, m.height-4, lipgloss.Center, lipgloss.Center, m.renderHelpModal()) + "\n")
	} else {
		mainView := m.renderMainPanel(isWide)
		sidebarView := m.renderSidebarPanel(isWide)

		var body string
		if isWide {
			body = lipgloss.JoinHorizontal(lipgloss.Top, mainView, " ", sidebarView)
		} else {
			body = lipgloss.JoinVertical(lipgloss.Left, mainView, sidebarView)
		}
		doc.WriteString(body + "\n")
	}

	// 3. 底部快捷图标栏 / 输入提示
	if m.inputMode != inputNone {
		var prompt string
		if m.inputMode == inputAddTask {
			prompt = fmt.Sprintf("✏️  添加新任务: %s█ (Enter 确认, Esc 取消)", m.inputText)
		} else if m.inputMode == inputEditTask {
			prompt = fmt.Sprintf("✏️  编辑任务: %s█ (Enter 确认, Esc 取消)", m.inputText)
		} else if m.inputMode == inputConfirmDelete {
			taskText := ""
			taskIndices := m.getTaskIndices()
			if len(taskIndices) > 0 && m.taskCursor < len(taskIndices) {
				taskText = m.lines[taskIndices[m.taskCursor]].text
			}
			prompt = fmt.Sprintf("⚠️  确认删除任务 [%s] 吗? (y/Enter 确认, n/Esc 取消)", taskText)
		} else if m.inputMode == inputSwitchFile {
			prompt = fmt.Sprintf("📁  切换/新建 MD 文件: %s█ (Enter 确认, Esc 取消)", m.inputText)
		}
		doc.WriteString(inputPromptStyle.Render(prompt))
	} else {
		iconBar := "⏱️ [Space]  │  🏁 [L]  │  🔄 [R]  │  ✏️ [a/e]  │  ❓ [h]"
		doc.WriteString(helpStyle.Render(iconBar))
	}

	return doc.String()
}

func (m model) renderHelpModal() string {
	var sb strings.Builder
	sb.WriteString(lipgloss.NewStyle().Bold(true).Foreground(cyanColor).Render("⌨️  快捷键与帮助指南") + "\n\n")

	items := []struct {
		key  string
		desc string
	}{
		{"Tab", "切换焦点 (秒表窗口 ↔ 当日安排)"},
		{"Space / S", "启动 / 暂停秒表"},
		{"L / Enter", "秒表计圈 (Lap)"},
		{"R", "重置秒表"},
		{"j/k, ↓/↑", "上下选择安排列表任务"},
		{"Space / X", "勾选 / 取消勾选选中任务"},
		{"a", "在当日安排中添加新任务"},
		{"e", "编辑选中的任务内容"},
		{"d", "删除选中的任务"},
		{"n", "新建或切换 MD 计划文件"},
		{"h / ?", "显示 / 隐藏此帮助窗口"},
		{"q / Ctrl+C", "退出程序"},
	}

	for _, item := range items {
		k := lipgloss.NewStyle().Bold(true).Foreground(primaryColor).Render(fmt.Sprintf("%-12s", item.key))
		d := lipgloss.NewStyle().Foreground(fgColor).Render(item.desc)
		sb.WriteString(fmt.Sprintf("  %s  %s\n", k, d))
	}

	modalStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(cyanColor).
		Padding(1, 2).
		Width(56)

	return modalStyle.Render(sb.String())
}

func (m model) renderHeader() string {
	title := headerTitleStyle.Render("⏱  COWSAY TIMER")

	// 计算考研倒计时
	now := time.Now()
	target := time.Date(2026, 12, 19, 0, 0, 0, 0, now.Location())
	var countdownStr string
	if now.After(target) {
		countdownStr = "🎯 2026考研已结束"
	} else {
		diff := target.Sub(now)
		days := int(diff.Hours()) / 24
		hours := int(diff.Hours()) % 24
		mins := int(diff.Minutes()) % 60
		secs := int(diff.Seconds()) % 60
		countdownStr = fmt.Sprintf("🎓 距2026考研(12-19): %d天 %02d时%02d分%02d秒", days, hours, mins, secs)
	}

	kaoyanBadge := kaoyanBadgeStyle.Render(countdownStr)

	if m.width < 65 {
		return lipgloss.JoinHorizontal(lipgloss.Center, title, " ", kaoyanBadge)
	}

	availWidth := m.width - 2
	titleLen := lipgloss.Width(title)
	badgeLen := lipgloss.Width(kaoyanBadge)
	spaceWidth := availWidth - titleLen - badgeLen
	if spaceWidth < 1 {
		spaceWidth = 1
	}

	return lipgloss.JoinHorizontal(lipgloss.Center, title, strings.Repeat(" ", spaceWidth), kaoyanBadge)
}

func (m model) renderMainPanel(isWide bool) string {
	var sb strings.Builder

	// 大字时间
	timeStr := formatStandardDuration(m.elapsed)
	asciiTime := renderASCIIText(timeStr)
	styledTime := timerStyle.Render(asciiTime)

	// Cowsay 对话框
	speechBubble := speechBubbleStyle.Render(styledTime)

	// 奶牛
	cowStr := "       \\   ^__^\n" +
		"        \\  (oo)\\_______\n" +
		"           (__)\\       )\\/\\\n" +
		"               ||----w |\n" +
		"               ||     ||"

	if m.wasCompleted {
		cowStr = "       \\   👑  🎉\n" +
			"        \\  (≧▽≦)\\_______\n" +
			"           (__)\\       )\\/\\  🏆 100%\n" +
			"               ||----w |\n" +
			"               ||     ||"
	}
	cow := cowStyle.Render(cowStr)

	if isWide {
		sb.WriteString(speechBubble + "\n" + cow + "\n\n")
	} else {
		// 窄屏下 Speech Bubble 与 Cowsay 组合
		topCombined := lipgloss.JoinHorizontal(lipgloss.Center, speechBubble, "  ", cow)
		sb.WriteString(topCombined + "\n")
	}

	// 状态
	statusText := pausedStyle.Render("● PAUSED")
	if m.running {
		statusText = runningStyle.Render("● RUNNING")
	}
	focusIndicator := ""
	if m.activePanel == panelMain {
		focusIndicator = lipgloss.NewStyle().Foreground(secondaryColor).Bold(true).Render(" [主窗口焦点]")
	}
	sb.WriteString("Status: " + statusText + focusIndicator + "\n")

	if m.wasCompleted {
		celebrationBanner := lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFD700")).
			Render("🏆 恭喜！今日计划全部完成！🎉")
		sb.WriteString(celebrationBanner + "\n")
	}

	// 计圈记录 (宽屏显示)
	if isWide && len(m.laps) > 0 {
		var lapContent strings.Builder
		lapContent.WriteString(lapHeaderStyle.Render("Laps History") + "\n")

		start := 0
		maxLaps := 3
		if len(m.laps) > maxLaps {
			start = len(m.laps) - maxLaps
		}

		for _, lap := range m.laps[start:] {
			lapContent.WriteString(lapTextStyle.Render(lap) + "\n")
		}

		sb.WriteString(lapBoxStyle.Render(strings.TrimSpace(lapContent.String())) + "\n")
	}

	cardStyle := sidebarInactiveStyle
	if m.activePanel == panelMain {
		cardStyle = sidebarActiveStyle
	}

	bubbleWidth := lipgloss.Width(speechBubble)
	mainWidth := bubbleWidth + 4
	if mainWidth < 48 {
		mainWidth = 48
	}
	if !isWide {
		mainWidth = m.width - 4
		if mainWidth < 38 {
			mainWidth = 38
		}
	}

	mainHeight := m.height - 5
	if !isWide {
		mainHeight = 9
	}

	return cardStyle.Width(mainWidth).Height(mainHeight).Render(sb.String())
}

func (m model) renderSidebarPanel(isWide bool) string {
	var sb strings.Builder

	// 标题
	title := sidebarTitleStyle.Render("📋 当日安排")
	fileBadge := fileBadgeStyle.Render(filepath.Base(m.taskFilePath))
	sb.WriteString(lipgloss.JoinHorizontal(lipgloss.Center, title, " ", fileBadge) + "\n")

	// 计算完成度
	taskIndices := m.getTaskIndices()
	totalTasks := len(taskIndices)
	completedTasks := 0
	for _, idx := range taskIndices {
		if m.lines[idx].done {
			completedTasks++
		}
	}

	// 进度条
	if totalTasks > 0 {
		percent := float64(completedTasks) / float64(totalTasks)
		barWidth := 15
		if m.width > 90 {
			barWidth = 20
		}
		filledLen := int(percent * float64(barWidth))
		if filledLen > barWidth {
			filledLen = barWidth
		}
		emptyLen := barWidth - filledLen

		barFilled := progressFilledStyle.Render(strings.Repeat("█", filledLen))
		barEmpty := progressEmptyStyle.Render(strings.Repeat("░", emptyLen))
		progressText := fmt.Sprintf(" %d/%d (%d%%)", completedTasks, totalTasks, int(percent*100))

		sb.WriteString("进度: [" + barFilled + barEmpty + "]" + progressText + "\n")
	} else {
		sb.WriteString(uncheckedStyle.Render("暂无任务安排 (按 a 添加)") + "\n")
	}

	// 渲染任务列表
	if totalTasks > 0 {
		taskItemCursor := 0
		for _, line := range m.lines {
			if !line.isTask {
				continue
			}

			isFocused := (m.activePanel == panelTasks) && (taskItemCursor == m.taskCursor)
			prefix := "  "
			if isFocused {
				prefix = taskCursorStyle.Render("➜ ")
			}

			checkIcon := uncheckedStyle.Render("[ ] ")
			if line.done {
				checkIcon = checkboxStyle.Render("[✓] ")
			}

			textStyle := taskNormalStyle
			if line.done {
				textStyle = taskDoneStyle
			}

			itemStr := prefix + checkIcon + textStyle.Render(line.text)
			sb.WriteString(itemStr + "\n")

			taskItemCursor++
		}
	}

	cardStyle := sidebarInactiveStyle
	if m.activePanel == panelTasks {
		cardStyle = sidebarActiveStyle
	}

	sidebarWidth := 38
	if isWide {
		sidebarWidth = m.width - 48
		if sidebarWidth < 38 {
			sidebarWidth = 38
		}
		if sidebarWidth > 60 {
			sidebarWidth = 60
		}
	} else {
		sidebarWidth = m.width - 4
		if sidebarWidth < 38 {
			sidebarWidth = 38
		}
	}

	targetHeight := m.height - 5
	if !isWide {
		// 窄屏模式下: 留出 15 行给 Header + 上卡片(9) + 底部 + 安全边界
		targetHeight = m.height - 15
		if targetHeight < 5 {
			targetHeight = 5
		}
	} else {
		if targetHeight < 10 {
			targetHeight = 10
		}
	}

	return cardStyle.Width(sidebarWidth).Height(targetHeight).Render(sb.String())
}

// ---------------------------------------------------------
// 时间与 ASCII Rendering
// ---------------------------------------------------------

func formatStandardDuration(d time.Duration) string {
	h := d / time.Hour
	d -= h * time.Hour
	m := d / time.Minute
	d -= m * time.Minute
	s := d / time.Second

	if h > 0 {
		return fmt.Sprintf("%02d:%02d:%02d", h, m, s)
	}
	return fmt.Sprintf("%02d:%02d", m, s)
}

var asciiGlyphs = map[rune][]string{
	'0': {
		" ██████╗ ",
		"██╔═████╗",
		"██║██╔██║",
		"████╔╝██║",
		"╚██████╔╝",
		" ╚═════╝ ",
	},
	'1': {
		"  ██╗    ",
		" ███║    ",
		" ╚██║    ",
		"  ██║    ",
		"  ██║    ",
		"  ╚═╝    ",
	},
	'2': {
		"██████╗  ",
		"╚════██╗ ",
		" █████╔╝ ",
		"██╔═══╝  ",
		"███████╗ ",
		"╚══════╝ ",
	},
	'3': {
		"██████╗  ",
		"╚════██╗ ",
		" █████╔╝ ",
		" ╚═══██╗ ",
		"██████╔╝ ",
		"╚═════╝  ",
	},
	'4': {
		"██╗  ██╗ ",
		"██║  ██║ ",
		"███████║ ",
		"╚════██║ ",
		"     ██║ ",
		"     ╚═╝ ",
	},
	'5': {
		"███████╗ ",
		"██╔════╝ ",
		"███████╗ ",
		"╚════██║ ",
		"███████║ ",
		"╚══════╝ ",
	},
	'6': {
		" ██████╗ ",
		"██╔════╝ ",
		"███████╗ ",
		"██╔═══██╗",
		"╚██████╔╝",
		" ╚═════╝ ",
	},
	'7': {
		"███████╗ ",
		"╚════██║ ",
		"    ██╔╝ ",
		"   ██╔╝  ",
		"  ██║    ",
		"  ╚═╝    ",
	},
	'8': {
		" █████╗  ",
		"██╔══██╗ ",
		"╚█████╔╝ ",
		"██╔══██╗ ",
		"╚█████╔╝ ",
		" ╚════╝  ",
	},
	'9': {
		" █████╗  ",
		"██╔══██╗ ",
		"╚██████║ ",
		" ╚═══██║ ",
		" █████╔╝ ",
		" ╚════╝  ",
	},
	':': {
		"   ",
		"██╗",
		"╚═╝",
		"██╗",
		"╚═╝",
		"   ",
	},
}

func renderASCIIText(text string) string {
	lines := make([]string, 6)
	for _, ch := range text {
		glyph, ok := asciiGlyphs[ch]
		if !ok {
			continue
		}
		for i := 0; i < 6; i++ {
			lines[i] += glyph[i] + " "
		}
	}
	return strings.Join(lines, "\n")
}

func main() {
	var specifiedFile string
	if len(os.Args) > 1 {
		specifiedFile = os.Args[1]
	}

	p := tea.NewProgram(initialModel(specifiedFile), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}
