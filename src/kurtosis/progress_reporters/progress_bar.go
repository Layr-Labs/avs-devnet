package progress_reporters

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/kurtosis-tech/kurtosis/api/golang/core/kurtosis_core_rpc_api_bindings"
	"github.com/schollz/progressbar/v3"
	"golang.org/x/term"
)

type KurtosisResponse = *kurtosis_core_rpc_api_bindings.StarlarkRunResponseLine

type State int

const (
	Interpretation State = 0
	Validation     State = iota
	Execution      State = iota
)

func ReportProgress(reporter chan KurtosisResponse) error {
	var maxWidth int

	pb := newProgressBar(-1)
	state := Interpretation
	var totalSteps uint32
	var currentStep uint32
	for line := range reporter {
		if line.GetProgressInfo() != nil {
			// It's a progress info
			progressInfo := line.GetProgressInfo()
			description := progressInfo.CurrentStepInfo[0]
			if description == "Interpreting plan - execution will begin shortly" {
				state = Interpretation
				clearBar(pb)
				pb = newProgressBar(-1)
				_ = pb.RenderBlank()
				pb.Describe("Interpreting plan...")
				continue
			} else if description == "Starting validation" {
				// NOTE: the total step number here is bugged, and shows the amount of execution steps instead
				clearBar(pb)
				pb = newProgressBar(-1)
				_ = pb.RenderBlank()
				pb.Describe("Starting validation...")
				state = Validation
				continue
			} else if description == "Starting execution" {
				state = Execution
				totalSteps = progressInfo.TotalSteps
				clearBar(pb)
				pb = newProgressBar(int(totalSteps))
				_ = pb.RenderBlank()
				pb.Describe("Starting execution...")
				continue
			} else if strings.HasPrefix(description, "Validating plan") && totalSteps == 0 && progressInfo.TotalSteps != 0 {
				state = Validation
				totalSteps = progressInfo.TotalSteps
				clearBar(pb)
				pb = newProgressBar(int(totalSteps))
				_ = pb.RenderBlank()
				pb.Describe(description)
			}
			currentStep = progressInfo.CurrentStepNumber
			if len(progressInfo.CurrentStepInfo) > 1 {
				details := strings.Join(progressInfo.CurrentStepInfo[1:], ", ")
				maxWidth = termWidth()
				if len(details) > maxWidth {
					details = details[:maxWidth-3] + "..."
				}
				_ = pb.AddDetail(details)
			}
			if state == Execution {
				// The current step returned on execution is the step that we're running.
				// We subtract one to make it the number of steps that we've already run instead.
				currentStep -= 1
			}
			_ = pb.Set(int(currentStep))
		}
		if line.GetInstruction() != nil {
			// It's an instruction
			instruction := line.GetInstruction()
			if state != Execution { // This should never happen
				panic("Received instruction outside of execution state")
			}
			detail := instruction.Description
			maxWidth = termWidth()
			if len(instruction.Description) > maxWidth {
				detail = detail[:maxWidth-3] + "..."
			}
			_ = pb.AddDetail(detail)
		}
		if line.GetInfo() != nil {
			_ = pb.Clear()
			fmt.Println(line.GetInfo().InfoMessage)
		}
		if line.GetWarning() != nil {
			_ = pb.Clear()
			fmt.Println(line.GetWarning().WarningMessage)
		}
		if line.GetInstructionResult() != nil {
			// It's an instruction result
			// TODO: implement verbosity levels and print this only on verbose
			// result := line.GetInstructionResult()
			// _ = pb.Clear()
			// fmt.Println(result.SerializedInstructionResult)
			currentStep += 1
			_ = pb.Set(int(currentStep))
			continue
		}
		if line.GetError() != nil {
			// It's an error
			return getKurtosisError(line.GetError())
		}
		if line.GetRunFinishedEvent() != nil {
			// It's a run finished event
			pb.Close()
			event := line.GetRunFinishedEvent()
			if event.IsRunSuccessful {
				fmt.Println("Run finished successfully with output:")
			} else {
				fmt.Println("Run failed with output:")
			}
			fmt.Println(event.SerializedOutput)
			break
		}
	}
	return nil
}

func getKurtosisError(starlarkError *kurtosis_core_rpc_api_bindings.StarlarkError) error {
	var msg string
	if err := starlarkError.GetValidationError(); err != nil {
		msg = err.ErrorMessage
	}
	if err := starlarkError.GetInterpretationError(); err != nil {
		msg = err.ErrorMessage
	}
	if err := starlarkError.GetExecutionError(); err != nil {
		msg = err.ErrorMessage
	}
	return errors.New("error occurred during execution: " + msg)
}

func termWidth() int {
	width, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		return 80
	}
	return width
}

// Ends and clears the progress bar
func clearBar(pb *progressbar.ProgressBar) {
	// Ignore any errors
	_ = pb.Set(1)
	_ = pb.AddDetail("")
	_ = pb.Finish()
	_ = pb.Clear()
}

func newProgressBar(steps int) *progressbar.ProgressBar {
	pb := progressbar.NewOptions(
		steps,
		progressbar.OptionSetMaxDetailRow(1),
		progressbar.OptionShowDescriptionAtLineEnd(),
		progressbar.OptionClearOnFinish(),
		progressbar.OptionSetElapsedTime(true),
		progressbar.OptionSetPredictTime(false),
		progressbar.OptionSetWriter(os.Stdout),
		progressbar.OptionShowCount(),
		progressbar.OptionSpinnerType(14),
		progressbar.OptionFullWidth(),
		progressbar.OptionSetTheme(progressbar.Theme{
			Saucer:        "[green]=[reset]",
			SaucerHead:    "[green]>[reset]",
			SaucerPadding: " ",
			BarStart:      "[",
			BarEnd:        "]",
		}),
		progressbar.OptionEnableColorCodes(true),
	)
	if steps != -1 {
		go func() {
			ticker := time.NewTicker(1 * time.Second)
			defer ticker.Stop()
			for range ticker.C {
				if pb.IsFinished() {
					return
				}
				_ = pb.RenderBlank()
			}
		}()
	}
	return pb
}
