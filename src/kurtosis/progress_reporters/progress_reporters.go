package progress_reporters

import (
	"errors"
	"strings"

	"github.com/kurtosis-tech/kurtosis/api/golang/core/kurtosis_core_rpc_api_bindings"
)

type KurtosisResponse = *kurtosis_core_rpc_api_bindings.StarlarkRunResponseLine

type State int

const (
	Interpretation State = 0
	Validation     State = iota
	Execution      State = iota
)

type ValidationStep struct {
	CurrentStep int
	TotalSteps  int
	Description string
	Details     []string
}

type ExecutionStep struct {
	CurrentStep            int
	TotalSteps             int
	Description            string
	InstructionDescription *string
	InstructionResult      *string
}

type Reporter interface {
	// Signals the start of the interpretation phase
	ReportInterpretationStart() error

	// Signals the start of the validation phase
	ReportValidationStart(totalSteps int) error

	// Signals a single step of the validation phase
	ReportValidationStep(stepInfo ValidationStep) error

	// Signals the start of the execution phase
	ReportExecutionStart(totalSteps int) error

	// Signals a single step of the execution phase
	// Can be called multiple times for the same step,
	// each time with more information (Description -> InstructionDescription -> InstructionResult)
	ReportExecutionStep(stepInfo ExecutionStep) error

	// Signals an informational message
	ReportInfo(message string) error

	// Signals a warning message
	ReportWarning(message string) error

	// Signals the end of the run
	ReportRunFinished(success bool, output string) error
}

// This function reads the Kurtosis response channel and reports the progress to the reporter
func ReportProgress(reporter Reporter, responseChan chan KurtosisResponse) error {
	state := Interpretation
	var totalSteps uint32
	var currentExecutionStep ExecutionStep
	for line := range responseChan {
		if reporter == nil {
			continue
		}
		if line.GetProgressInfo() != nil {
			// It's a progress info
			progressInfo := line.GetProgressInfo()
			description := progressInfo.CurrentStepInfo[0]

			// We first match against the message for state transitions
			if description == "Interpreting plan - execution will begin shortly" {
				state = Interpretation
				err := reporter.ReportInterpretationStart()
				if err != nil {
					return err
				}
				continue
			} else if description == "Starting validation" {
				// The total step number here is bugged, and shows the amount of execution steps instead.
				// That's why we ignore it and call `ReportValidationStart` later.
				continue
			} else if description == "Starting execution" {
				state = Execution
				totalSteps = progressInfo.TotalSteps
				err := reporter.ReportExecutionStart(int(totalSteps))
				if err != nil {
					return err
				}
				continue
			}

			// We set the total steps here because the "Starting validation" message has the wrong step count.
			if strings.HasPrefix(description, "Validating plan") && state == Interpretation && progressInfo.TotalSteps != 0 {
				state = Validation
				totalSteps = progressInfo.TotalSteps
				err := reporter.ReportValidationStart(int(totalSteps))
				if err != nil {
					return err
				}
			}
			currentStep := progressInfo.CurrentStepNumber
			// Call the corresponding reporter method based on current state
			switch state {
			case Validation:
				step := ValidationStep{
					CurrentStep: int(currentStep),
					TotalSteps:  int(totalSteps),
					Description: description,
					Details:     progressInfo.CurrentStepInfo[1:],
				}
				err := reporter.ReportValidationStep(step)
				if err != nil {
					return err
				}

			case Execution:
				// The current step returned on execution is the step that we're running.
				// We subtract one to make it the number of steps that we've already run instead.
				currentStep -= 1
				currentExecutionStep = ExecutionStep{
					CurrentStep:            int(currentStep),
					TotalSteps:             int(totalSteps),
					Description:            description,
					InstructionDescription: nil,
					InstructionResult:      nil,
				}
				err := reporter.ReportExecutionStep(currentExecutionStep)
				if err != nil {
					return err
				}
			default:
				// do nothing
			}
		} else if line.GetInstruction() != nil {
			// It's an instruction
			instruction := line.GetInstruction()
			if state != Execution { // This should never happen
				panic("Received instruction outside of execution state")
			}
			currentExecutionStep.InstructionDescription = &instruction.Description
			err := reporter.ReportExecutionStep(currentExecutionStep)
			if err != nil {
				return err
			}
		} else if line.GetInfo() != nil {
			err := reporter.ReportInfo(line.GetInfo().InfoMessage)
			if err != nil {
				return err
			}
		} else if line.GetWarning() != nil {
			err := reporter.ReportWarning(line.GetWarning().WarningMessage)
			if err != nil {
				return err
			}
		} else if line.GetInstructionResult() != nil {
			// It's an instruction result
			result := line.GetInstructionResult()
			currentExecutionStep.InstructionResult = &result.SerializedInstructionResult
			err := reporter.ReportExecutionStep(currentExecutionStep)
			if err != nil {
				return err
			}
		} else if line.GetError() != nil {
			// It's an error
			return getKurtosisError(line.GetError())
		} else if line.GetRunFinishedEvent() != nil {
			// It's a run finished event
			event := line.GetRunFinishedEvent()
			err := reporter.ReportRunFinished(event.IsRunSuccessful, event.GetSerializedOutput())
			if err != nil {
				return err
			}
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
