package progress_reporters

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/schollz/progressbar/v3"
	"golang.org/x/term"
)

var _ Reporter = (*ProgressBarReporter)(nil)

// A reporter that reports progress via a progress bar
type ProgressBarReporter struct {
	pb *progressbar.ProgressBar
}

func NewProgressBarReporter() *ProgressBarReporter {
	return &ProgressBarReporter{}
}

func (r *ProgressBarReporter) ReportInterpretationStart() error {
	changeProgressBar(r.pb, -1, "Interpreting plan...")
	return nil
}

func (r *ProgressBarReporter) ReportValidationStart(totalSteps int) error {
	changeProgressBar(r.pb, totalSteps, "Starting validation...")
	return nil
}

func (r *ProgressBarReporter) ReportValidationStep(stepInfo ValidationStep) error {
	_ = r.pb.Set(int(stepInfo.CurrentStep))
	r.pb.Describe(stepInfo.Description)
	details := strings.Join(stepInfo.Details, ", ")
	maxWidth := termWidth()
	if len(details) > maxWidth {
		details = details[:maxWidth-3] + "..."
	}
	_ = r.pb.AddDetail(details)
	return nil
}

func (r *ProgressBarReporter) ReportExecutionStart(totalSteps int) error {
	changeProgressBar(r.pb, totalSteps, "Starting execution...")
	return nil
}

func (r *ProgressBarReporter) ReportExecutionStep(stepInfo ExecutionStep) error {
	_ = r.pb.Set(int(stepInfo.CurrentStep))
	r.pb.Describe(stepInfo.Description)
	if stepInfo.InstructionDescription != nil {
		_ = r.pb.AddDetail(*stepInfo.InstructionDescription)
	}
	// TODO: implement verbosity levels and print this only on verbose
	// if stepInfo.InstructionResult != nil {
	// 	fmt.Println(*stepInfo.InstructionResult)
	// }
	return nil
}

func (r *ProgressBarReporter) ReportInfo(message string) error {
	_ = r.pb.Clear()
	fmt.Println(message)
	_ = r.pb.RenderBlank()
	return nil
}

func (r *ProgressBarReporter) ReportWarning(message string) error {
	_ = r.pb.Clear()
	fmt.Println(message)
	_ = r.pb.RenderBlank()
	return nil
}

func (r *ProgressBarReporter) ReportRunFinished(success bool, output string) error {
	if success {
		fmt.Println("Run finished successfully with output:")
	} else {
		fmt.Println("Run failed with output:")
	}
	fmt.Println(output)
	return nil
}

func changeProgressBar(pb *progressbar.ProgressBar, max int, message string) {
	clearBar(pb)
	pb = newProgressBar(max)
	_ = pb.RenderBlank()
	pb.Describe(message)
}

func termWidth() int {
	width, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		return 80
	}
	return width
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

// Ends and clears the progress bar
func clearBar(pb *progressbar.ProgressBar) {
	// Ignore any errors
	_ = pb.Set(1)
	_ = pb.AddDetail("")
	_ = pb.Finish()
	_ = pb.Clear()
}
