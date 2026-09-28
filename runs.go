package seaflowsdk

import "context"

// RunsResource reads and stops executions.
type RunsResource struct {
	transport *Transport
}

// Get reads a run with every node record it created. Runner only.
func (r *RunsResource) Get(ctx context.Context, runID string) (*RunDetail, error) {
	if err := requireIdentifier("runID", runID); err != nil {
		return nil, err
	}
	var result RunDetail
	if err := r.transport.GetJSON(ctx, "/seaflow/runs/"+urlEscape(runID), nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// Stop stops future execution: nodes that have not started are not scheduled.
// A task already submitted to the model gateway runs to a terminal state and may
// still incur cost, but its result no longer triggers downstream nodes.
// Idempotent.
func (r *RunsResource) Stop(ctx context.Context, runID string) (*Run, error) {
	if err := requireIdentifier("runID", runID); err != nil {
		return nil, err
	}
	var result Run
	if err := r.transport.PatchJSON(ctx, "/seaflow/runs/"+urlEscape(runID), nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}
