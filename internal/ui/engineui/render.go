package engineui

import (
	"fmt"
	"strings"
	"time"

	"gogws/internal/engine"
	"gogws/internal/ui/styles"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
)

type WorkerStatus int

const (
	WorkerIdle WorkerStatus = iota
	WorkerRunning
)

type WorkerState struct {
	Status   WorkerStatus
	JobLabel string
	LastLog  string
	Spinner  spinner.Model
}

type CompletedJob struct {
	Label   string
	Success bool
	Error   error
	LastLog string
}

type eventMsg engine.Event

type doneMsg struct{}

type Model struct {
	events    <-chan engine.Event
	workers   []WorkerState
	completed []CompletedJob
	total     int
	done      int
	styles    *styles.Styles
	quitting  bool
}

func NewModel(events <-chan engine.Event, parallelism int, totalJobs int) Model {
	s := styles.Get()
	workers := make([]WorkerState, parallelism)
	for i := range workers {
		sp := spinner.New()
		sp.Spinner = spinner.Dot
		workers[i] = WorkerState{
			Status:  WorkerIdle,
			Spinner: sp,
		}
	}
	return Model{
		events:    events,
		workers:   workers,
		completed: make([]CompletedJob, 0),
		total:     totalJobs,
		done:      0,
		styles:    s,
	}
}

func (m Model) Init() tea.Cmd {
	cmds := make([]tea.Cmd, 0, len(m.workers)+1)
	for i := range m.workers {
		cmds = append(cmds, m.workers[i].Spinner.Tick)
	}
	cmds = append(cmds, m.waitForEvent())
	return tea.Batch(cmds...)
}

func (m Model) waitForEvent() tea.Cmd {
	return func() tea.Msg {
		event, ok := <-m.events
		if !ok {
			return doneMsg{}
		}
		return eventMsg(event)
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			m.quitting = true
			return m, tea.Quit
		}

	case doneMsg:
		m.quitting = true
		return m, tea.Quit

	case eventMsg:
		event := engine.Event(msg)
		workerID := event.GoroutineID

		if workerID >= 0 && workerID < len(m.workers) {
			switch event.Type {
			case engine.EventJobStart:
				m.workers[workerID].Status = WorkerRunning
				m.workers[workerID].JobLabel = event.JobLabel
				m.workers[workerID].LastLog = "starting..."

			case engine.EventLog:
				m.workers[workerID].LastLog = event.Log

			case engine.EventErr:
				m.workers[workerID].LastLog = event.Log

			case engine.EventJobEnd:
				m.completed = append(m.completed, CompletedJob{
					Label:   event.JobLabel,
					Success: event.Success,
					Error:   event.Err,
					LastLog: m.workers[workerID].LastLog,
				})
				m.done++
				m.workers[workerID].Status = WorkerIdle
				m.workers[workerID].JobLabel = ""
				m.workers[workerID].LastLog = ""
			}
		}

		return m, m.waitForEvent()

	case spinner.TickMsg:
		var cmds []tea.Cmd
		for i := range m.workers {
			var cmd tea.Cmd
			m.workers[i].Spinner, cmd = m.workers[i].Spinner.Update(msg)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
		return m, tea.Batch(cmds...)
	}

	return m, nil
}

func (m Model) View() tea.View {
	if m.quitting && m.done == m.total {
		return tea.NewView(m.renderFinalSummary())
	}

	var b strings.Builder

	if len(m.completed) > 0 {
		b.WriteString(m.styles.Muted.Render("─── Completed ───"))
		b.WriteString("\n")
		maxShow := 10
		start := 0
		if len(m.completed) > maxShow {
			start = len(m.completed) - maxShow
			b.WriteString(m.styles.Muted.Render(fmt.Sprintf("  ... and %d more\n", start)))
		}
		for i := start; i < len(m.completed); i++ {
			job := m.completed[i]
			if job.Success {
				b.WriteString(fmt.Sprintf("  %s %s %s\n",
					m.styles.Success.Render(m.styles.IconSuccess),
					job.Label,
					m.styles.Muted.Render(job.LastLog),
				))
			} else {
				errMsg := ""
				if job.Error != nil {
					errMsg = fmt.Sprintf(" - %s", job.Error.Error())
				}
				b.WriteString(fmt.Sprintf("  %s %s%s\n",
					m.styles.Error.Render(m.styles.IconError),
					job.Label,
					m.styles.Error.Render(errMsg),
				))
			}
		}
		b.WriteString("\n")
	}

	b.WriteString(m.styles.Muted.Render("─── Workers ───"))
	b.WriteString("\n")
	for i, w := range m.workers {
		if w.Status == WorkerRunning {
			log := w.LastLog
			if len(log) > 60 {
				log = log[:57] + "..."
			}
			b.WriteString(fmt.Sprintf("  %s [%d] %s %s\n",
				w.Spinner.View(),
				i,
				m.styles.Path.Render(w.JobLabel),
				m.styles.Muted.Render(log),
			))
		} else {
			b.WriteString(fmt.Sprintf("  %s [%d] %s\n",
				m.styles.Muted.Render(m.styles.IconPending),
				i,
				m.styles.Muted.Render("idle"),
			))
		}
	}

	b.WriteString("\n")
	pct := 0
	if m.total > 0 {
		pct = (m.done * 100) / m.total
	}
	b.WriteString(m.styles.Info.Render(fmt.Sprintf("Progress: [%d/%d] %d%%", m.done, m.total, pct)))
	b.WriteString("\n")

	return tea.NewView(b.String())
}

func (m Model) renderFinalSummary() string {
	var b strings.Builder

	successCount := 0
	failedCount := 0
	for _, job := range m.completed {
		if job.Success {
			successCount++
		} else {
			failedCount++
		}
	}

	b.WriteString("\n")
	b.WriteString(m.styles.Title.Render("═══ Summary ═══"))
	b.WriteString("\n\n")

	if successCount > 0 {
		b.WriteString(fmt.Sprintf("  %s %s %d\n",
			m.styles.Success.Render(m.styles.IconSuccess),
			m.styles.Success.Render("Succeeded:"),
			successCount,
		))
	}
	if failedCount > 0 {
		b.WriteString(fmt.Sprintf("  %s %s %d\n",
			m.styles.Error.Render(m.styles.IconError),
			m.styles.Error.Render("Failed:"),
			failedCount,
		))
		b.WriteString("\n")
		b.WriteString(m.styles.Error.Render("  Failed jobs:"))
		b.WriteString("\n")
		for _, job := range m.completed {
			if !job.Success {
				errMsg := ""
				if job.Error != nil {
					errMsg = fmt.Sprintf(": %s", job.Error.Error())
				}
				b.WriteString(fmt.Sprintf("    - %s%s\n", job.Label, m.styles.Muted.Render(errMsg)))
			}
		}
	}

	b.WriteString("\n")
	return b.String()
}

func Run(events <-chan engine.Event, parallelism int, totalJobs int) error {
	p := tea.NewProgram(NewModel(events, parallelism, totalJobs))
	_, err := p.Run()
	return err
}

func RunWithTimeout(events <-chan engine.Event, parallelism int, totalJobs int, timeout time.Duration) error {
	p := tea.NewProgram(NewModel(events, parallelism, totalJobs))

	done := make(chan error, 1)
	go func() {
		_, err := p.Run()
		done <- err
	}()

	select {
	case err := <-done:
		return err
	case <-time.After(timeout):
		p.Quit()
		return fmt.Errorf("UI timeout after %s", timeout)
	}
}
