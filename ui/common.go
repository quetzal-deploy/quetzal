package ui

import (
	"github.com/DBCDK/morph/planner"
	tea "github.com/charmbracelet/bubbletea"
)

type LogWriter struct {
	Program *tea.Program
}

type LogEvent struct {
	Data string
}

func (w LogWriter) Write(inputBytes []byte) (n int, err error) {
	msg := string(inputBytes)
	if msg == "" {

	}

	w.Program.Send(LogEvent{Data: msg})

	return len(inputBytes), nil
}

func CountChildSteps(step planner.Step) int {
	children := 0
	for _, child := range step.Steps {
		children += 1 + CountChildSteps(child)
	}

	return children
}

func CountChildStepsDone(m model, step planner.Step) int {
	childrenDone := 0
	for _, child := range step.Steps {
		if m.stepStatus[child.Id] == "done" {
			childrenDone += 1
		}

		childrenDone += CountChildStepsDone(m, child)
	}

	return childrenDone
}
