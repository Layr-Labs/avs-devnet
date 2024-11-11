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
	ReportInterpretationStart() error

	ReportValidationStart(totalSteps int) error
	ReportValidationStep(stepInfo ValidationStep) error

	ReportExecutionStart(totalSteps int) error
	ReportExecutionStep(stepInfo ExecutionStep) error

	ReportInfo(message string) error
	ReportWarning(message string) error
	ReportRunFinished(success bool, output string) error
}

func ReportProgress(reporter Reporter, responseChan chan KurtosisResponse) error {
	state := Interpretation
	var totalSteps uint32
	var currentExecutionStep ExecutionStep
	for line := range responseChan {
		if line.GetProgressInfo() != nil {
			// It's a progress info
			progressInfo := line.GetProgressInfo()
			description := progressInfo.CurrentStepInfo[0]
			if description == "Interpreting plan - execution will begin shortly" {
				state = Interpretation
				if err := reporter.ReportInterpretationStart(); err != nil {
					return err
				}
				continue
			} else if description == "Starting validation" {
				// NOTE: the total step number here is bugged, and shows the amount of execution steps instead
				continue
			} else if description == "Starting execution" {
				state = Execution
				totalSteps = progressInfo.TotalSteps
				if err := reporter.ReportExecutionStart(int(totalSteps)); err != nil {
					return err
				}
				continue
			} else if strings.HasPrefix(description, "Validating plan") && totalSteps == 0 && progressInfo.TotalSteps != 0 {
				state = Validation
				totalSteps = progressInfo.TotalSteps
				if err := reporter.ReportValidationStart(int(totalSteps)); err != nil {
					return err
				}
			}
			currentStep := progressInfo.CurrentStepNumber
			if state == Validation {
				step := ValidationStep{
					CurrentStep: int(currentStep),
					TotalSteps:  int(totalSteps),
					Description: description,
					Details:     progressInfo.CurrentStepInfo[1:],
				}
				if err := reporter.ReportValidationStep(step); err != nil {
					return err
				}
			} else if state == Execution {
				currentStep -= 1
				currentExecutionStep = ExecutionStep{
					CurrentStep:            int(currentStep),
					TotalSteps:             int(totalSteps),
					Description:            description,
					InstructionDescription: nil,
					InstructionResult:      nil,
				}
				if err := reporter.ReportExecutionStep(currentExecutionStep); err != nil {
					return err
				}
			} else {
				panic("Unknown state")
			}
		}
		if line.GetInstruction() != nil {
			// It's an instruction
			instruction := line.GetInstruction()
			if state != Execution { // This should never happen
				panic("Received instruction outside of execution state")
			}
			currentExecutionStep.InstructionDescription = &instruction.Description
			if err := reporter.ReportExecutionStep(currentExecutionStep); err != nil {
				return err
			}
		}
		if line.GetInfo() != nil {
			if err := reporter.ReportInfo(line.GetInfo().InfoMessage); err != nil {
				return err
			}
		}
		if line.GetWarning() != nil {
			if err := reporter.ReportWarning(line.GetWarning().WarningMessage); err != nil {
				return err
			}
		}
		if line.GetInstructionResult() != nil {
			// It's an instruction result
			result := line.GetInstructionResult()
			currentExecutionStep.InstructionResult = &result.SerializedInstructionResult
			if err := reporter.ReportExecutionStep(currentExecutionStep); err != nil {
				return err
			}
			continue
		}
		if line.GetError() != nil {
			// It's an error
			return getKurtosisError(line.GetError())
		}
		if line.GetRunFinishedEvent() != nil {
			// It's a run finished event
			event := line.GetRunFinishedEvent()
			if err := reporter.ReportRunFinished(event.IsRunSuccessful, event.GetSerializedOutput()); err != nil {
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
