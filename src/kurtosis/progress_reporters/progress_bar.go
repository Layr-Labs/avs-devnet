package progress_reporters

import (
	"errors"
	"fmt"
	"os"

	"github.com/kurtosis-tech/kurtosis/api/golang/core/kurtosis_core_rpc_api_bindings"
	"github.com/schollz/progressbar/v3"
)

type KurtosisResponse = *kurtosis_core_rpc_api_bindings.StarlarkRunResponseLine
type KurtosisReporter = chan KurtosisResponse

// type ProgressBarReporter struct {
// 	progressBar  *progressbar.ProgressBar
// 	info         []string
// 	currentStage int
// 	currentStep  int
// 	totalSteps   int
// }

func ReportProgress(reporter KurtosisReporter) error {
	pb := newValidationProgressBar(-1)
	if err := pb.RenderBlank(); err != nil {
		return err
	}
	var description string
	var details []string
	// TODO: clean up this mess
	for line := range reporter {
		if line.GetProgressInfo() != nil {
			// It's a progress info
			progressInfo := line.GetProgressInfo()
			fmt.Fprintln(os.Stderr, "progress: ", progressInfo)
			if len(progressInfo.CurrentStepInfo) == 0 {
				continue
			}
			if progressInfo.CurrentStepInfo[0] == "Starting validation" {
				pb.Set(1)
				pb.AddDetail("")
				pb.Finish()
				pb.Clear()
				pb = newValidationProgressBar(int(progressInfo.TotalSteps))
			}
			if progressInfo.CurrentStepInfo[0] == "Starting execution" {
				pb.Set(1)
				pb.AddDetail("")
				pb.Finish()
				pb.Clear()
				pb = newExecutionProgressBar(int(progressInfo.TotalSteps))
			}

			if len(progressInfo.CurrentStepInfo) > 0 {
				description = progressInfo.CurrentStepInfo[0]
			}
			if len(progressInfo.CurrentStepInfo) > 1 {
				details = progressInfo.CurrentStepInfo[1:]
			} else {
				details = []string{}
			}
			pb.Describe(description)
			for _, detail := range details {
				if err := pb.AddDetail(detail); err != nil {
					return err
				}
			}
			if progressInfo.TotalSteps != 0 {
				if err := pb.Set(int(progressInfo.CurrentStepNumber)); err != nil {
					return err
				}
			}
		}
		if line.GetInstruction() != nil {
			// It's an instruction
			instruction := line.GetInstruction()
			fmt.Fprintln(os.Stderr, instruction.Description)
			detail := instruction.Description
			if len(instruction.Description) > 60 {
				detail = detail[:57] + "..."
			}
			details = append(details, detail)
			pb.Describe(description)
			for _, detail := range details {
				pb.AddDetail(detail)
			}
		}
		if line.GetInfo() != nil {
			pb.Clear()
			// It's an info
			fmt.Println("INFO:", line.GetInfo().InfoMessage)
		}
		if line.GetWarning() != nil {
			pb.Clear()
			// It's a warning
			fmt.Println(line.GetWarning().WarningMessage)
		}
		if line.GetInstructionResult() != nil {
			// It's an instruction result
			result := line.GetInstructionResult()
			fmt.Fprintln(os.Stderr, result)
			// detail := result.SerializedInstructionResult
			// if len(result.SerializedInstructionResult) > 40 {
			// 	detail = detail[:37] + "..."
			// }
			// details = append(details, detail)
		}
		if line.GetError() != nil {
			// It's an error
			pb.Exit()
			var msg string
			if err := line.GetError().GetValidationError(); err != nil {
				msg = err.ErrorMessage
			}
			if err := line.GetError().GetInterpretationError(); err != nil {
				msg = err.ErrorMessage
			}
			if err := line.GetError().GetExecutionError(); err != nil {
				msg = err.ErrorMessage
			}
			return errors.New("error occurred during execution: " + msg)
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

func newValidationProgressBar(max int) *progressbar.ProgressBar {
	pb := progressbar.NewOptions(
		max,
		progressbar.OptionSetMaxDetailRow(1),
		progressbar.OptionShowDescriptionAtLineEnd(),
		progressbar.OptionClearOnFinish(),
		progressbar.OptionSetElapsedTime(true),
		progressbar.OptionSetPredictTime(false),
		// TODO: use Stderr for progress bar
		// progressbar.OptionSetWriter(os.Stderr),
		progressbar.OptionSetWriter(os.Stdout),
		progressbar.OptionShowCount(),
		// progressbar.OptionOnCompletion(func() {
		// 	fmt.Fprint(os.Stderr, "\n")
		// }),
		progressbar.OptionSpinnerType(14),
		progressbar.OptionSetWidth(20),
		progressbar.OptionSetTheme(progressbar.Theme{
			Saucer:        "[green]=[reset]",
			SaucerHead:    "[green]>[reset]",
			SaucerPadding: " ",
			BarStart:      "[",
			BarEnd:        "]",
		}),
		progressbar.OptionEnableColorCodes(true),
	)
	return pb
}

func newExecutionProgressBar(steps int) *progressbar.ProgressBar {
	pb := progressbar.NewOptions(
		steps,
		progressbar.OptionSetMaxDetailRow(1),
		progressbar.OptionShowDescriptionAtLineEnd(),
		progressbar.OptionClearOnFinish(),
		progressbar.OptionSetElapsedTime(true),
		progressbar.OptionSetPredictTime(false),
		// TODO: use Stderr for progress bar
		// progressbar.OptionSetWriter(os.Stderr),
		progressbar.OptionSetWriter(os.Stdout),
		progressbar.OptionShowCount(),
		// progressbar.OptionOnCompletion(func() {
		// 	fmt.Fprint(os.Stderr, "\n")
		// }),
		progressbar.OptionSpinnerType(14),
		progressbar.OptionSetWidth(50),
		progressbar.OptionSetTheme(progressbar.Theme{
			Saucer:        "[green]=[reset]",
			SaucerHead:    "[green]>[reset]",
			SaucerPadding: " ",
			BarStart:      "[",
			BarEnd:        "]",
		}),
		progressbar.OptionEnableColorCodes(true),
	)
	return pb
}
