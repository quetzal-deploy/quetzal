package ui

import (
	"fmt"
	"maps"
	"slices"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/tree"

	"github.com/DBCDK/morph/events"
	"github.com/DBCDK/morph/planner"
	"github.com/DBCDK/morph/steps"
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

	docStyle              = lipgloss.NewStyle().Padding(0, 0, 0, 0)
	highlightColor        = lipgloss.AdaptiveColor{Light: "#874BFD", Dark: "#7D56F4"}
	windowStyle           = lipgloss.NewStyle().BorderForeground(highlightColor).Padding(0, 0).Align(lipgloss.Left).Border(lipgloss.NormalBorder()).UnsetBorderTop()
	menuAccentColor       = lipgloss.AdaptiveColor{Light: "#FF0000", Dark: "#FF0000"}
	menuBorderStyle       = lipgloss.NewStyle().PaddingBottom(1).PaddingLeft(1).PaddingRight(1)
	inactiveMenuItemStyle = lipgloss.NewStyle().Bold(true)
	activeMenuItemStyle   = inactiveMenuItemStyle.Foreground(menuAccentColor)

	stepStyleWaiting   = lipgloss.NewStyle()
	stepStyleScheduled = lipgloss.NewStyle().Background(lipgloss.Color("#666666"))
	stepStyleBlocked   = lipgloss.NewStyle().Background(lipgloss.Color("#ffff66"))
	stepStyleRunning   = lipgloss.NewStyle().Background(lipgloss.Color("#6666ff"))
	stepStyleDone      = lipgloss.NewStyle().Background(lipgloss.Color("#66cc66"))
	stepStyleFailed    = lipgloss.NewStyle().Background(lipgloss.Color("#ff6666"))
	stepStyle          = map[string]lipgloss.Style{
		planner.Waiting: stepStyleWaiting,
		planner.Queued:  stepStyleScheduled,
		planner.Blocked: stepStyleBlocked,
		planner.Running: stepStyleRunning,
		planner.Done:    stepStyleDone,
		planner.Failed:  stepStyleFailed,
	}
)

type MenuItem struct {
	name    string
	keybind string
}

type model struct {
	planner *planner.Planner
	ready   bool

	width  int
	height int

	viewport        viewport.Model
	viewportContent string
	gotPlan         bool
	plan            steps.Step

	stepStatus map[string]string
	stepLog    map[string]string // FIXME: Is it better for this to be map[string][]string and store each event individually?
	queue      []events.StepStatus
	steps      map[string]steps.Step

	Tabs       []MenuItem
	TabContent []string
	activeTab  int

	tabKeyBinds map[string]int
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) ChangeTab(tabIndex int) (tea.Model, tea.Cmd) {
	if m.activeTab != tabIndex {
		m.activeTab = tabIndex
		m.viewport.SetContent(m.TabContent[m.activeTab])
	}

	return m, nil
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

		case "left", "shift+tab":
			tabIndex := max(m.activeTab-1, 0)
			return m.ChangeTab(tabIndex)

		case "right", "tab":
			tabIndex := min(m.activeTab+1, len(m.Tabs)-1)
			return m.ChangeTab(tabIndex)

		default:
			// try matching on default tabKeyBinds
			if tabIndex, ok := m.tabKeyBinds[msg.String()]; ok {
				return m.ChangeTab(tabIndex)
			}

			// try matching numbers to tabIndex
			if number, err := strconv.Atoi(msg.String()); err == nil {
				tabIndex := number - 1
				if 0 <= tabIndex && tabIndex < len(m.Tabs) {
					return m.ChangeTab(tabIndex)
				}
			}
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

	case events.Log:
		// FIXME: Scroll broken, both manual and automatic
		m.TabContent[1] += msg.Data
		if !strings.HasSuffix(msg.Data, "\n") {
			m.TabContent[1] += "\n"
		}

		if m.activeTab == 1 {
			m.viewport.GotoBottom()
		}

	case events.RegisterStep:
		m.steps[msg.Step.Id] = msg.Step

	case events.StepLog:
		m.stepLog[msg.StepId] += msg.Data
		if !strings.HasSuffix(msg.Data, "\n") {
			m.stepLog[msg.StepId] += "\n"
		}

	case events.StepUpdate:
		m.stepStatus[msg.StepId] = msg.State

	case events.QueueStatus:
		m.queue = msg.Queue

	case steps.Step:
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

		isActive := i == m.activeTab
		if isActive {
			style = activeMenuItemStyle
		} else {
			style = inactiveMenuItemStyle
		}

		tabBar = append(tabBar, menuBorderStyle.Render(tab.RenderWithBaseStyle(style)))
	}

	switch m.activeTab {
	case 0:
		if m.gotPlan {
			m.viewportContent = renderPlan(m, m.plan).String()
		} else {
			m.viewportContent = "loading plan"
		}
	case 1:
		m.viewportContent = m.TabContent[1]

	case 2:
		m.viewportContent = renderStepsInState(m, planner.Running)

	case 3:
		m.viewportContent = renderQueue(m)

	case 4:
		m.viewportContent = renderStepsInState(m, planner.Done)

	case 5:
		m.viewportContent = renderStepsInState(m, planner.Failed)

	case 6:
		m.viewportContent = renderStepsInState(m, "")

	default:
		m.viewportContent = "configure me mr programmer"
	}

	m.viewport.SetContent(m.viewportContent)

	tabsView := lipgloss.JoinHorizontal(lipgloss.Top, tabBar...)

	menuView := m.menuView()

	line := strings.Repeat(" ", max(0, m.viewport.Width-lipgloss.Width(tabsView)-lipgloss.Width(menuView)))

	header := lipgloss.JoinHorizontal(lipgloss.Center, tabsView, line, menuView)

	doc.WriteString(header)
	doc.WriteString("\n")
	//doc.WriteString(windowStyle.Width((m.width - windowStyle.GetHorizontalFrameSize())).Render(m.TabContent[m.activeTab]))
	doc.WriteString(m.viewport.View())
	doc.WriteString("\n")
	doc.WriteString(m.footerView())

	return docStyle.Render(doc.String())
}

func (m model) menuView() string {
	menuItems := []MenuItem{
		{
			name:    "pause",
			keybind: "p",
		},
		{
			name:    "quit",
			keybind: "q",
		},
	}

	menu := make([]string, 0)

	for _, menuItem := range menuItems {
		menu = append(menu, menuBorderStyle.Render(menuItem.RenderWithBaseStyle(inactiveMenuItemStyle)))
	}

	return lipgloss.JoinHorizontal(lipgloss.Top, menu...)
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

func (m MenuItem) Render() string {
	return m.RenderWithBaseStyle(lipgloss.NewStyle())
}

func (m MenuItem) RenderWithBaseStyle(baseStyle lipgloss.Style) string {
	r := strings.Builder{}

	style := baseStyle.Foreground(menuAccentColor)

	before, after, found := strings.Cut(m.name, m.keybind)

	if found {
		r.WriteString(baseStyle.Render(before))
		r.WriteString(style.Render(m.keybind))
		r.WriteString(baseStyle.Render(after))

	} else {
		r.WriteString(baseStyle.Render(m.name + " ("))
		r.WriteString(style.Render(m.keybind))
		r.WriteString(baseStyle.Render(")"))
	}

	return r.String()
}

func renderPlan(m model, step steps.Step) *tree.Tree {
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
	} else {
		t = tree.Root(fmt.Sprintf("BUG: missing style for stepStatus = '%s'", stepStatus))
	}

	if allowCollapse && childSteps == childStepsDone {
		// intentionally left blank
	} else {
		for _, subStep := range step.Steps {
			t.Child(renderPlan(m, subStep))
		}
	}

	return t
}

func renderStepById(m model, stepId string) string {
	if step, ok := m.steps[stepId]; ok {
		return renderStep(m, step)
	} else {
		return "unknown step: " + stepId
	}
}

func renderStep(m model, step steps.Step) string {

	r := strings.Builder{}

	r.WriteString(fmt.Sprintf("# %s: %s\n\n", step.Action.Name(), step.Description))
	r.WriteString("id: " + step.Id + "\n\n")

	if len(step.Labels) == 0 {
		r.WriteString("labels: none\n")
	} else {
		r.WriteString("labels:\n")
		labelKeys := slices.Sorted(maps.Keys(step.Labels))
		for _, key := range labelKeys {
			r.WriteString("- " + key + "=" + step.Labels[key] + "\n")
		}
	}
	r.WriteString("\n")

	if len(step.DependsOn) == 0 {
		r.WriteString("dependencies: none\n")
	} else {
		r.WriteString("dependencies:\n")
		for _, subStep := range step.DependsOn {
			r.WriteString("- " + subStep + "\n")
		}
	}
	r.WriteString("\n")

	r.WriteString("\n")

	return r.String()
}

func renderQueue(m model) string {
	r := strings.Builder{}

	if len(m.queue) == 0 {
		r.WriteString("queue empty\n")
	}

	for _, stepStatus := range m.queue {
		r.WriteString(renderStep(m, stepStatus.Step))

		r.WriteString("Blocked by:\n")

		for _, blockingStep := range stepStatus.BlockedBy {
			r.WriteString("* " + blockingStep + "\n")
		}

		r.WriteString("\n")
	}

	render, err := glamour.Render(r.String(), "dark")
	if err != nil {
		return err.Error()
	}

	return render
}

func renderStepsInState(m model, state string) string {
	renderAllSteps := state == ""
	r := strings.Builder{}

	stepIds := slices.Sorted(maps.Keys(m.stepStatus))
	matchingIds := make([]string, 0)

	if renderAllSteps {
		matchingIds = stepIds
	} else {
		for _, stepId := range stepIds {
			if m.stepStatus[stepId] == state {
				matchingIds = append(matchingIds, stepId)
			}
		}
	}

	if len(matchingIds) == 0 {
		r.WriteString("nothing matching state=" + state + "\n")
	}

	for _, stepId := range matchingIds {
		r.WriteString(renderStepById(m, stepId))
		if renderAllSteps {
			r.WriteString("state: " + m.stepStatus[stepId] + "\n")
		}

		r.WriteString("\n")
	}

	render, err := glamour.Render(r.String(), "dark")
	if err != nil {
		return err.Error()
	}

	return render
}

func DoTea(eventChan chan events.EventWithId) *tea.Program {
	tabs := []MenuItem{
		{
			name:    "plan",
			keybind: "p",
		},
		{
			name:    "logs",
			keybind: "l",
		},
		{
			name:    "running",
			keybind: "r",
		},
		{
			name:    "queue",
			keybind: "u",
		},
		{
			name:    "done",
			keybind: "d",
		},
		{
			name:    "failed",
			keybind: "f",
		},
		{
			name:    "all",
			keybind: "a",
		},
	}

	tabKeyBinds := make(map[string]int)
	tabContent := make([]string, 0)

	for tabIndex, menuItem := range tabs {
		tabKeyBinds[menuItem.keybind] = tabIndex

		tabContent = append(tabContent, "")
	}

	p := tea.NewProgram(
		model{
			viewportContent: "",
			gotPlan:         false,
			Tabs:            tabs,
			TabContent:      tabContent,
			tabKeyBinds:     tabKeyBinds,
			stepLog:         make(map[string]string),
			stepStatus:      make(map[string]string),
			steps:           make(map[string]steps.Step),
		},
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	go func() {
		for {
			event := <-eventChan
			p.Send(event.Event) // ignore the ID
		}
	}()

	return p
}
