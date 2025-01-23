package ui

import (
	"fmt"
	"github.com/DBCDK/morph/common"
	"github.com/DBCDK/morph/planner"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/tree"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

var (
	titleStyle = func() lipgloss.Style {

		b := lipgloss.NormalBorder()
		b.Right = "├"
		return lipgloss.NewStyle().BorderStyle(b).Padding(0, 1)
	}()

	infoStyle = func() lipgloss.Style {
		b := lipgloss.HiddenBorder()
		b.Left = "┤"
		return titleStyle.BorderStyle(b)
	}()

	inactiveTabBorder = tabBorderWithBottom("┴", "─", "┴")
	activeTabBorder   = tabBorderWithBottom("┘", " ", "└")
	docStyle          = lipgloss.NewStyle().Padding(0, 0, 0, 0)
	highlightColor    = lipgloss.AdaptiveColor{Light: "#874BFD", Dark: "#7D56F4"}
	inactiveTabStyle  = lipgloss.NewStyle().Border(inactiveTabBorder, true).BorderForeground(highlightColor).Padding(0, 1)
	activeTabStyle    = inactiveTabStyle.Border(activeTabBorder, true)
	windowStyle       = lipgloss.NewStyle().BorderForeground(highlightColor).Padding(0, 0).Align(lipgloss.Left).Border(lipgloss.NormalBorder()).UnsetBorderTop()

	stepStyleWaiting   = lipgloss.NewStyle()
	stepStyleScheduled = lipgloss.NewStyle().Background(lipgloss.Color("#666666"))
	stepStyleBlocked   = lipgloss.NewStyle().Background(lipgloss.Color("#ff6666"))
	stepStyleRunning   = lipgloss.NewStyle().Background(lipgloss.Color("#6666ff"))
	stepStyleDone      = lipgloss.NewStyle().Background(lipgloss.Color("#66cc66"))
	stepStyle          = map[string]lipgloss.Style{
		planner.Waiting:   stepStyleWaiting,
		planner.Scheduled: stepStyleScheduled,
		planner.Blocked:   stepStyleBlocked,
		planner.Running:   stepStyleRunning,
		planner.Done:      stepStyleDone,
	}
)

func tabBorderWithBottom(left, middle, right string) lipgloss.Border {
	border := lipgloss.NormalBorder()
	border.BottomLeft = left
	border.Bottom = middle
	border.BottomRight = right
	return border
}

type model struct {
	ready bool

	width  int
	height int

	viewport        viewport.Model
	viewportContent string
	gotPlan         bool
	plan            planner.Step

	stepStatus map[string]string
	stepLog    map[string]string // FIXME: Is it better for this to be map[string][]string and store each event individually?

	Tabs       []string
	TabContent []string
	activeTab  int
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			return m, tea.Quit

		case "up":

		case "down":

		case "left", "tab":
			m.activeTab = max(m.activeTab-1, 0)
			m.viewport.SetContent(m.TabContent[m.activeTab])
			return m, nil

		case "right", "shift-tab":
			m.activeTab = min(m.activeTab+1, len(m.Tabs)-1)
			m.viewport.SetContent(m.TabContent[m.activeTab])
			return m, nil

		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		headerHeight := lipgloss.Height(m.headerView())
		footerHeight := lipgloss.Height(m.footerView())
		verticalMarginHeight := headerHeight + footerHeight

		if !m.ready {
			m.viewport = viewport.New(msg.Width, msg.Height-verticalMarginHeight)
			m.viewport.YPosition = headerHeight
			m.ready = true
		} else {
			m.viewport.Width = msg.Width
			m.viewport.Height = msg.Height - verticalMarginHeight
		}

	case LogEvent:
		// FIXME: Scroll broken, both manual and automatic
		m.TabContent[1] += msg.Data
		if !strings.HasSuffix(msg.Data, "\n") {
			m.TabContent[1] += "\n"
		}

		if m.activeTab == 1 {
			m.viewport.GotoBottom()
		}

	case common.StepLogEvent:
		m.stepLog[msg.StepId] += msg.Data
		if !strings.HasSuffix(msg.Data, "\n") {
			m.stepLog[msg.StepId] += "\n"
		}

	case common.StepUpdateEvent:
		m.stepStatus[msg.StepId] = msg.State

	case planner.Step:
		m.gotPlan = true
		m.plan = msg
	}

	m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m model) View() string {
	if !m.ready {
		return "\n.. loading .."
	}

	doc := strings.Builder{}
	var tabBar []string

	for i, tab := range m.Tabs {
		var style lipgloss.Style

		isFirst, isLast, isActive := i == 0, i == len(m.Tabs)-1, i == m.activeTab
		if isActive {
			style = activeTabStyle
		} else {
			style = inactiveTabStyle
		}

		border, _, _, _, _ := style.GetBorder()

		if isFirst && isActive {
			border.BottomLeft = "│"
		} else if isFirst && !isActive {
			border.BottomLeft = "├"
		} else if isLast && isActive {
			border.BottomRight = "│"
		} else if isLast && !isActive {
			border.BottomRight = "┤"
		}

		style = style.Border(border)
		tabBar = append(tabBar, style.Render(tab))
	}

	switch m.activeTab {
	case 0:
		if m.gotPlan {
			m.viewportContent = renderStep(m, m.plan).String()
		} else {
			m.viewportContent = "loading plan"
		}
		//m.viewportContent = m.TabContent[1]
	case 1:
		m.viewportContent = m.TabContent[1]
	default:
		m.viewportContent = ":o"
	}

	m.viewport.SetContent(m.viewportContent)

	row := lipgloss.JoinHorizontal(lipgloss.Top, tabBar...)
	doc.WriteString(row)
	doc.WriteString("\n")
	//doc.WriteString(windowStyle.Width((m.width - windowStyle.GetHorizontalFrameSize())).Render(m.TabContent[m.activeTab]))
	doc.WriteString(m.viewport.View())
	doc.WriteString("\n")
	doc.WriteString(m.footerView())

	return docStyle.Render(doc.String())
}

func (m model) headerView() string {
	title := titleStyle.Render("morph: plan")
	line := strings.Repeat("─", max(0, m.viewport.Width-lipgloss.Width(title)))
	return lipgloss.JoinHorizontal(lipgloss.Center, title, line)
}

func (m model) footerView() string {
	info := infoStyle.Render(fmt.Sprintf("%3.f%%", m.viewport.ScrollPercent()*100))
	line := strings.Repeat("─", max(0, m.viewport.Width-lipgloss.Width(info)))
	return lipgloss.JoinHorizontal(lipgloss.Center, line, info)
}

func renderStep(m model, step planner.Step) *tree.Tree {
	allowCollapse := false

	stepStatus := m.stepStatus[step.Id]
	if stepStatus == "" {
		stepStatus = planner.Waiting
	}

	style, styleFound := stepStyle[stepStatus]

	t := tree.Root(fmt.Sprintf("%s", step.Description))

	// FIXME: Colors don't change properly
	// FIXME: ^ delay exit to have everything update at the end
	// FIXME: add latest step log next to the step description
	// FIXME: fix step status mismatch for e.g. push

	childSteps := CountChildSteps(step)
	childStepsDone := CountChildStepsDone(m, step)

	if styleFound {

		if len(step.Steps) == 0 {
			t = tree.Root(fmt.Sprintf("%*s %s", 9, stepStatus, style.Render(step.Description)))
		} else {
			t = tree.Root(fmt.Sprintf("%*s %s (%d/%d)", 9, stepStatus, style.Render(step.Description), childStepsDone, childSteps))
		}
	}

	if allowCollapse && childSteps == childStepsDone {
		// intentionally left blank
	} else {
		for _, subStep := range step.Steps {
			t.Child(renderStep(m, subStep))
		}
	}

	return t
}

func DoTea() *tea.Program {
	p := tea.NewProgram(
		model{
			viewportContent: "",
			gotPlan:         false,
			Tabs:            []string{"plan", "logs"},
			TabContent:      []string{"creating plan", ""},
			stepLog:         make(map[string]string),
			stepStatus:      make(map[string]string),
		},
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	return p
}
