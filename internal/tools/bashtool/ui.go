package bashtool

import (
	"encoding/json"
	"fmt"
	"strings"
)

const bashAssistantBlockingBudgetSec = 15 // ASSISTANT_BLOCKING_BUDGET_MS / 1000 (BashTool.tsx)

// MapBashToolResultForMessagesAPI mirrors BashTool.mapToolResultToToolResultBlockParam string path
// (stdout normalize, stderr, interrupted, returnCodeInterpretation, backgroundTaskId + output path).
func MapBashToolResultForMessagesAPI(outJSON []byte) string {
	var o struct {
		Stdout                    string `json:"stdout"`
		Stderr                    string `json:"stderr"`
		Interrupted               bool   `json:"interrupted"`
		BackgroundTaskID          string `json:"backgroundTaskId"`
		BackgroundTaskOutputPath  string `json:"backgroundTaskOutputPath"`
		ReturnCodeInterpretation  string `json:"returnCodeInterpretation"`
		AssistantAutoBackgrounded bool   `json:"assistantAutoBackgrounded"`
		BackgroundedByUser        bool   `json:"backgroundedByUser"`
	}
	if err := json.Unmarshal(outJSON, &o); err != nil {
		return string(outJSON)
	}
	processed := o.Stdout
	if processed != "" {
		processed = strings.TrimLeft(processed, " \t\n\r")
		for strings.HasPrefix(processed, "\n") {
			processed = strings.TrimPrefix(processed, "\n")
			processed = strings.TrimLeft(processed, " \t\n\r")
		}
		processed = strings.TrimRight(processed, " \t\n\r")
		processed = StripEmptyLines(processed)
	}
	errMsg := strings.TrimSpace(o.Stderr)
	if o.Interrupted {
		if errMsg != "" {
			errMsg += "\n"
		}
		errMsg += "<error>Command was aborted before completion</error>"
	}

	bgID := strings.TrimSpace(o.BackgroundTaskID)
	var bgInfo string
	if bgID != "" {
		outPath := strings.TrimSpace(o.BackgroundTaskOutputPath)
		if outPath == "" {
			outPath = BackgroundOutputPath(bgID)
		}
		switch {
		case o.AssistantAutoBackgrounded:
			bgInfo = fmt.Sprintf(
				"Command exceeded the assistant-mode blocking budget (%ds) and was moved to the background with ID: %s. It is still running — you will be notified when it completes. Output is being written to: %s. In assistant mode, delegate long-running work to a subagent or use run_in_background to keep this conversation responsive.",
				bashAssistantBlockingBudgetSec, bgID, outPath,
			)
		case o.BackgroundedByUser:
			bgInfo = fmt.Sprintf(
				"Command was manually backgrounded by user with ID: %s. Output is being written to: %s",
				bgID, outPath,
			)
		default:
			bgInfo = fmt.Sprintf(
				"Command running in background with ID: %s. Output is being written to: %s",
				bgID, outPath,
			)
		}
	}

	interp := strings.TrimSpace(o.ReturnCodeInterpretation)

	parts := []string{}
	if processed != "" {
		parts = append(parts, processed)
	}
	if interp != "" {
		parts = append(parts, interp)
	}
	if errMsg != "" {
		parts = append(parts, errMsg)
	}
	if bgInfo != "" {
		parts = append(parts, bgInfo)
	}
	return strings.Join(parts, "\n")
}
