package progress_reporters

import (
	"fmt"
	"os"

	"github.com/kurtosis-tech/kurtosis/api/golang/core/kurtosis_core_rpc_api_bindings"
	"github.com/schollz/progressbar/v3"
)

type KurtosisReporter = chan *kurtosis_core_rpc_api_bindings.StarlarkRunResponseLine

func ReportProgress(reporter KurtosisReporter) error {
	pb := newValidationProgressBar()
	if err := pb.RenderBlank(); err != nil {
		return err
	}
	for line := range reporter {
		if line.GetProgressInfo() != nil {
			// It's a progress info
			progressInfo := line.GetProgressInfo()
			if len(progressInfo.CurrentStepInfo) == 0 {
				continue
			}
			if progressInfo.CurrentStepInfo[0] == "Starting validation" {
				continue
			}
			fmt.Fprintln(os.Stderr, "progress: ", progressInfo)

			pb.Describe(progressInfo.CurrentStepInfo[0])
			detail := ""
			if len(progressInfo.CurrentStepInfo) > 1 {
				detail = progressInfo.CurrentStepInfo[1]
			}
			if err := pb.AddDetail(detail); err != nil {
				return err
			}
			if progressInfo.TotalSteps != 0 {
				if progressInfo.CurrentStepNumber >= progressInfo.TotalSteps {
					pb.Close()
					pb = newExecutionProgressBar(-1)
					if err := pb.RenderBlank(); err != nil {
						return err
					}
				} else {
					pb.ChangeMax(int(progressInfo.TotalSteps))
					if err := pb.Set(int(progressInfo.CurrentStepNumber)); err != nil {
						return err
					}
				}
			}
		}
		if line.GetInstruction() != nil {
			// It's an instruction
			fmt.Fprintln(os.Stderr, line.GetInstruction().Description)
		}
		if line.GetInfo() != nil {
			// It's an info
			fmt.Fprintln(os.Stderr, "INFO:", line.GetInfo().InfoMessage)
		}
		if line.GetWarning() != nil {
			// It's a warning
			fmt.Fprintln(os.Stderr, line.GetWarning().WarningMessage)
		}
		if line.GetInstructionResult() != nil {
			// It's an instruction result
			fmt.Fprintln(os.Stderr, line.GetInstructionResult())
		}
		if line.GetError() != nil {
			// It's an error
			// pb.Exit()
			fmt.Fprintln(os.Stderr, line.GetError())
		}
		if line.GetRunFinishedEvent() != nil {
			// It's a run finished event
			// pb.Close()
			fmt.Fprintln(os.Stderr, line.GetRunFinishedEvent())
		}
	}
	return nil
}

func newValidationProgressBar() *progressbar.ProgressBar {
	pb := progressbar.NewOptions(
		-1,
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
		progressbar.OptionSetMaxDetailRow(2),
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
