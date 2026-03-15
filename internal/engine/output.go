package engine

import (
	"fmt"
	"strings"

	"gogws/internal/ui/cli"
)

type OutputMode int

const (
	OutputModeStacked OutputMode = iota
	OutputModeVerbose
	OutputModeStream
)

type OutputHandler struct {
	Mode     OutputMode
	Renderer *cli.Renderer
}

func NewOutputHandler(renderer *cli.Renderer, verbose bool) *OutputHandler {
	var mode OutputMode
	if verbose {
		mode = OutputModeVerbose
	}
	mode = OutputModeStacked
	return &OutputHandler{
		Mode:     mode,
		Renderer: renderer,
	}
}

func (h *OutputHandler) RenderResult(result Result) {
	if h.Mode == OutputModeVerbose {
		h.renderVerboseResult(result)
	}
}

func (h *OutputHandler) renderVerboseResult(result Result) {
	if result.IsSkipped() {
		fmt.Println(h.Renderer.RenderWarning(fmt.Sprintf("%s: skipped (%s)", result.Command.RepoName, result.SkipReason)))
	} else if result.IsFailure() {
		errMsg := result.Stderr
		if errMsg == "" && result.Error != nil {
			errMsg = result.Error.Error()
		}
		fmt.Println(h.Renderer.RenderError(fmt.Sprintf("%s: %s", result.Command.RepoName, strings.TrimSpace(errMsg))))
	} else {
		fmt.Println(h.Renderer.RenderSuccess(result.Command.RepoName))
	}
}

func (h *OutputHandler) RenderSummary(execResult *ExecuteResult, actionName string) {
	fmt.Println()

	if h.Mode == OutputModeStacked {
		h.renderStackedSummary(execResult, actionName)
	} else {
		h.renderSimpleSummary(execResult, actionName)
	}
}

func (h *OutputHandler) renderStackedSummary(execResult *ExecuteResult, actionName string) {
	succeeded := execResult.Succeeded()
	failed := execResult.Failed()
	skipped := execResult.Skipped()

	if len(succeeded) > 0 {
		if len(succeeded) <= 5 {
			names := make([]string, len(succeeded))
			for i, r := range succeeded {
				names[i] = r.Command.RepoName
			}
			fmt.Println(h.Renderer.RenderSuccess(fmt.Sprintf("%s: %s", actionName, strings.Join(names, ", "))))
		} else {
			fmt.Println(h.Renderer.RenderSuccess(fmt.Sprintf("%s %d repositories successfully", actionName, len(succeeded))))
		}
	}

	if len(skipped) > 0 {
		if len(skipped) <= 3 {
			for _, r := range skipped {
				fmt.Println(h.Renderer.RenderWarning(fmt.Sprintf("%s: skipped (%s)", r.Command.RepoName, r.SkipReason)))
			}
		} else {
			fmt.Println(h.Renderer.RenderWarning(fmt.Sprintf("Skipped %d repositories", len(skipped))))
		}
	}

	if len(failed) > 0 {
		fmt.Println(h.Renderer.RenderError(fmt.Sprintf("Failed (%d):", len(failed))))
		for _, r := range failed {
			errMsg := r.Stderr
			if errMsg == "" && r.Error != nil {
				errMsg = r.Error.Error()
			}
			errMsg = strings.TrimSpace(errMsg)
			if errMsg == "" {
				errMsg = "unknown error"
			}
			fmt.Println(h.Renderer.RenderError(fmt.Sprintf("  %s: %s", r.Command.RepoName, errMsg)))
		}
	}

	if execResult.Stopped {
		fmt.Println(h.Renderer.RenderWarning(fmt.Sprintf("Execution stopped: %s", execResult.StopReason)))
	}
}

func (h *OutputHandler) renderSimpleSummary(execResult *ExecuteResult, actionName string) {
	successCount := execResult.SuccessCount()
	failedCount := execResult.FailedCount()
	skippedCount := execResult.SkippedCount()

	var parts []string
	if successCount > 0 {
		parts = append(parts, fmt.Sprintf("%d succeeded", successCount))
	}
	if failedCount > 0 {
		parts = append(parts, fmt.Sprintf("%d failed", failedCount))
	}
	if skippedCount > 0 {
		parts = append(parts, fmt.Sprintf("%d skipped", skippedCount))
	}

	summary := strings.Join(parts, ", ")
	if failedCount > 0 {
		fmt.Println(h.Renderer.RenderWarning(fmt.Sprintf("%s: %s", actionName, summary)))
	} else {
		fmt.Println(h.Renderer.RenderSuccess(fmt.Sprintf("%s: %s", actionName, summary)))
	}
}

func (h *OutputHandler) RenderProgress(current, total int, repoName string) {
	percentage := float64(current) / float64(total) * 100
	fmt.Printf("\r[%d/%d] %.0f%% - %s", current, total, percentage, repoName)
}

func (h *OutputHandler) ClearProgress() {
	fmt.Printf("\r%s\r", strings.Repeat(" ", 80))
}
