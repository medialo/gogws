package engineui

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/medialo/gogws/internal/engine"
	"github.com/medialo/gogws/internal/ui/styles"

	"charm.land/bubbles/v2/progress"
	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
)

const (
	maxShownCompleted = 10
	progressBarWidth  = 20
)

type WorkerStatus int

const (
	WorkerIdle WorkerStatus = iota
	WorkerRunning
)

type WorkerState struct {
	Status   WorkerStatus
	JobId    string
	Phase    string
	LastLog  string
	Progress *engine.Progress
}

type CompletedJob struct {
	Label    any
	Success  bool
	Error    error
	LastLog  string
	rendered string
}

type doneMsg struct{}

type Model struct {
	events    <-chan engine.Event
	cancel    context.CancelFunc
	workers   []WorkerState
	spinner   spinner.Model
	bar       progress.Model
	recent    []CompletedJob
	failed    []CompletedJob
	succeeded int
	total     int
	done      int
	styles    *styles.Styles
	quitting  bool
}

func NewModel(events <-chan engine.Event, cancel context.CancelFunc, maxParallel int, totalJobs int) Model {
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	return Model{
		events:  events,
		cancel:  cancel,
		workers: make([]WorkerState, maxParallel),
		spinner: sp,
		bar:     progress.New(progress.WithWidth(progressBarWidth), progress.WithoutPercentage(), progress.WithFillCharacters(progress.DefaultFullCharFullBlock, progress.DefaultEmptyCharBlock)),
		recent:  make([]CompletedJob, 0, maxShownCompleted),
		total:   totalJobs,
		styles:  styles.Get(),
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, m.waitForEvent())
}

func (m Model) waitForEvent() tea.Cmd {
	return func() tea.Msg {
		event, ok := <-m.events
		if !ok {
			return doneMsg{}
		}
		return event
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			m.quitting = true
			if m.cancel != nil {
				m.cancel()
			}
			return m, tea.Quit
		}

	case doneMsg:
		m.quitting = true
		return m, tea.Quit

	case engine.Event:
		m.applyEvent(msg)
		return m, m.waitForEvent()

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m *Model) applyEvent(event engine.Event) {
	workerID := event.WorkerID()
	if workerID < 0 || workerID >= len(m.workers) {
		return
	}
	worker := &m.workers[workerID]

	switch e := event.(type) {
	case engine.JobStarted:
		*worker = WorkerState{Status: WorkerRunning, JobId: e.Job, LastLog: "starting..."}
	case engine.JobPhase:
		worker.Phase = e.Phase
		if e.Phase != "" {
			worker.LastLog = ""
		}
	case engine.JobLog:
		worker.LastLog = e.Line
	case engine.JobProgress:
		progress := e.Progress
		worker.Progress = &progress
	case engine.JobEnded:
		m.addCompleted(CompletedJob{
			Label:   e.Job,
			Success: e.Success,
			Error:   e.Err,
			LastLog: worker.LastLog,
		})
		*worker = WorkerState{Status: WorkerIdle}
	}
}

func (m *Model) addCompleted(job CompletedJob) {
	m.done++
	if job.Success {
		m.succeeded++
	} else {
		m.failed = append(m.failed, job)
	}
	job.rendered = m.renderCompleted(job)
	if len(m.recent) == maxShownCompleted {
		copy(m.recent, m.recent[1:])
		m.recent = m.recent[:maxShownCompleted-1]
	}
	m.recent = append(m.recent, job)
}

func (m Model) renderCompleted(job CompletedJob) string {
	if job.Success {
		return fmt.Sprintf("  %s %s %s\n",
			m.styles.Success.Render(m.styles.IconSuccess),
			job.Label,
			m.styles.Muted.Render(job.LastLog),
		)
	}
	errMsg := ""
	if job.Error != nil {
		errMsg = fmt.Sprintf(" - %s", job.Error.Error())
	}
	return fmt.Sprintf("  %s %s%s\n",
		m.styles.Error.Render(m.styles.IconError),
		job.Label,
		m.styles.Error.Render(errMsg),
	)
}

func (m Model) View() tea.View {
	var b strings.Builder

	if m.quitting && m.done == m.total {
		b.WriteString(m.renderFinalSummary())
		return tea.NewView(b.String())
	}

	if len(m.recent) > 0 {
		b.WriteString(m.styles.Muted.Render("─── Completed ───"))
		b.WriteString("\n")
		if hidden := m.done - len(m.recent); hidden > 0 {
			b.WriteString(m.styles.Muted.Render(fmt.Sprintf("  ... and %d more\n", hidden)))
		}
		for _, job := range m.recent {
			b.WriteString(job.rendered)
		}
		b.WriteString("\n")
	}

	spinnerView := m.spinner.View()
	b.WriteString(m.styles.Muted.Render("─── Workers ───"))
	b.WriteString("\n")
	for i, w := range m.workers {
		if w.Status == WorkerRunning {
			log := w.LastLog
			if len(log) > 60 {
				log = log[:57] + "..."
			}
			if w.Progress != nil {
				b.WriteString(fmt.Sprintf("  %s [%d] %s %s %s %s\n",
					spinnerView,
					i,
					m.styles.Path.Render(w.JobId),
					m.styles.Muted.Render(w.Progress.Phase),
					m.bar.ViewAs(float64(w.Progress.Percent)/100),
					m.styles.Info.Render(fmt.Sprintf("%3d%% (%d/%d)", w.Progress.Percent, w.Progress.Current, w.Progress.Total)),
				))
				continue
			}
			if w.Phase != "" {
				b.WriteString(fmt.Sprintf("  %s [%d] %s %s\n",
					spinnerView,
					i,
					m.styles.Path.Render(w.JobId),
					m.styles.Warning.Render("⚙ "+w.Phase),
				))
				if log != "" {
					b.WriteString(fmt.Sprintf("        %s\n", m.styles.Muted.Render(log)))
				}
				continue
			}
			b.WriteString(fmt.Sprintf("  %s [%d] %s %s\n",
				spinnerView,
				i,
				m.styles.Path.Render(w.JobId),
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

	b.WriteString("\n")
	b.WriteString(m.styles.Title.Render("═══ Summary ═══"))
	b.WriteString("\n\n")

	if m.succeeded > 0 {
		b.WriteString(fmt.Sprintf("  %s %s %d\n",
			m.styles.Success.Render(m.styles.IconSuccess),
			m.styles.Success.Render("Succeeded:"),
			m.succeeded,
		))
	}
	if len(m.failed) > 0 {
		b.WriteString(fmt.Sprintf("  %s %s %d\n",
			m.styles.Error.Render(m.styles.IconError),
			m.styles.Error.Render("Failed:"),
			len(m.failed),
		))
		b.WriteString("\n")
		b.WriteString(m.styles.Error.Render("  Failed jobs:"))
		b.WriteString("\n")
		for _, job := range m.failed {
			errMsg := ""
			if job.Error != nil {
				errMsg = fmt.Sprintf(": %s", job.Error.Error())
			}
			b.WriteString(fmt.Sprintf("    - %s%s\n", job.Label, m.styles.Muted.Render(errMsg)))
		}
	}

	b.WriteString("\n")
	return b.String()
}

func Run(eventsCh <-chan engine.Event, cancel context.CancelFunc, maxParallel int, totalJobs int) error {
	slog.Debug("Starting UI", "maxParallel", maxParallel, "totalJobs", totalJobs)
	model := NewModel(eventsCh, cancel, maxParallel, totalJobs)

	p := tea.NewProgram(model)
	_, err := p.Run()
	for range eventsCh {
	}
	return err
}
