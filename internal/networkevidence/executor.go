package networkevidence

import "context"

type ReconcilingExecutor struct {
	Inner Executor
}

func (executor ReconcilingExecutor) Execute(ctx context.Context, request ExecutionRequest) (ExecutionResult, error) {
	result, err := executor.Inner.Execute(ctx, request)
	if err != nil {
		return result, err
	}
	result.Observations = ReconcileObservations(result.Observations)
	for index, observation := range result.Observations {
		if validationErr := observation.Validate(); validationErr != nil {
			return ExecutionResult{Limitations: result.Limitations}, validationErr
		}
		result.Observations[index] = observation
	}
	return result, nil
}
