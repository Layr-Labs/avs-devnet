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

// This function reads the Kurtosis response channel and reports the progress to the reporter.
func ReportProgress(reporter Reporter, responseChan chan KurtosisResponse) error {
	state := Interpretation
	var totalSteps uint32
	var currentExecutionStep ExecutionStep
	for line := range responseChan {
		switch {
		case line.GetProgressInfo() != nil:
			// It's a progress info
			progressInfo := line.GetProgressInfo()
			description := progressInfo.GetCurrentStepInfo()[0]

			// We first match against the message for state transitions
			switch description {
			case "Interpreting plan - execution will begin shortly":
				state = Interpretation
				err := reporter.ReportInterpretationStart()
				if err != nil {
					return err
				}
				continue
			case "Starting validation":
				// The total step number here is bugged, and shows the amount of execution steps instead.
				// That's why we ignore it and call `ReportValidationStart` later.
				continue
			case "Starting execution":
				state = Execution
				totalSteps = progressInfo.GetTotalSteps()
				err := reporter.ReportExecutionStart(int(totalSteps))
				if err != nil {
					return err
				}
				continue
			default:
			}

			// We set the total steps here because the "Starting validation" message has the wrong step count.
			isValidating := strings.HasPrefix(description, "Validating plan")
			if isValidating && state == Interpretation && progressInfo.GetTotalSteps() != 0 {
				state = Validation
				totalSteps = progressInfo.GetTotalSteps()
				err := reporter.ReportValidationStart(int(totalSteps))
				if err != nil {
					return err
				}
			}
			currentStep := progressInfo.GetCurrentStepNumber()
			// Call the corresponding reporter method based on current state
			switch state {
			case Validation:
				step := ValidationStep{
					CurrentStep: int(currentStep),
					TotalSteps:  int(totalSteps),
					Description: description,
					Details:     progressInfo.GetCurrentStepInfo()[1:],
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
			case Interpretation:
				// do nothing
			}
		case line.GetInstruction() != nil:
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
		case line.GetInfo() != nil:
			err := reporter.ReportInfo(line.GetInfo().GetInfoMessage())
			if err != nil {
				return err
			}
		case line.GetWarning() != nil:
			err := reporter.ReportWarning(line.GetWarning().GetWarningMessage())
			if err != nil {
				return err
			}
		case line.GetInstructionResult() != nil:
			// It's an instruction result
			result := line.GetInstructionResult()
			currentExecutionStep.InstructionResult = &result.SerializedInstructionResult
			err := reporter.ReportExecutionStep(currentExecutionStep)
			if err != nil {
				return err
			}
		case line.GetError() != nil:
			// It's an error
			return getKurtosisError(line.GetError())
		case line.GetRunFinishedEvent() != nil:
			// It's a run finished event
			event := line.GetRunFinishedEvent()
			err := reporter.ReportRunFinished(event.GetIsRunSuccessful(), event.GetSerializedOutput())
			if err != nil {
				return err
			}
			return nil
		}
	}
	return nil
}

func getKurtosisError(starlarkError *kurtosis_core_rpc_api_bindings.StarlarkError) error {
	var msg string
	if err := starlarkError.GetValidationError(); err != nil {
		msg = err.GetErrorMessage()
	}
	if err := starlarkError.GetInterpretationError(); err != nil {
		msg = err.GetErrorMessage()
	}
	if err := starlarkError.GetExecutionError(); err != nil {
		msg = err.GetErrorMessage()
	}
	return errors.New("error occurred during execution: " + msg)
}
