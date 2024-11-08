package progress_reporters

import (
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

type CurrentProgress struct {
	Description string
	Details     []string
	CurrentStep uint32
	TotalSteps  uint32
}

func newProgress() CurrentProgress {
	return CurrentProgress{
		Description: "",
		Details:     []string{},
		CurrentStep: 0,
		TotalSteps:  0,
	}
}

func ReportProgress(reporter KurtosisReporter) error {
	pb := newValidationProgressBar(-1)
	if err := pb.RenderBlank(); err != nil {
		return err
	}
	cp := newProgress()
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
				cp.Description = progressInfo.CurrentStepInfo[0]
			}
			if len(progressInfo.CurrentStepInfo) > 1 {
				cp.Details = progressInfo.CurrentStepInfo[1:]
			} else {
				cp.Details = []string{}
			}
			if progressInfo.TotalSteps != 0 {
				cp.TotalSteps = progressInfo.TotalSteps
				cp.CurrentStep = progressInfo.CurrentStepNumber
				if err := pb.Set(int(cp.CurrentStep)); err != nil {
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
			cp.Details = append(cp.Details, detail)
		}
		if line.GetInfo() != nil {
			pb.Clear()
			// It's an info
			fmt.Println("INFO:", line.GetInfo().InfoMessage)
			continue
		}
		if line.GetWarning() != nil {
			pb.Clear()
			// It's a warning
			fmt.Println(line.GetWarning().WarningMessage)
			continue
		}
		if line.GetInstructionResult() != nil {
			// It's an instruction result
			result := line.GetInstructionResult()
			fmt.Fprintln(os.Stderr, result)
			// detail := result.SerializedInstructionResult
			// if len(result.SerializedInstructionResult) > 40 {
			// 	detail = detail[:37] + "..."
			// }
			// cp.Details = append(cp.Details, detail)
			continue
		}
		if line.GetError() != nil {
			// It's an error
			pb.Exit()
			fmt.Println(line.GetError().Error)
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

		pb.Describe(cp.Description)
		for _, detail := range cp.Details {
			if err := pb.AddDetail(detail); err != nil {
				return err
			}
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
