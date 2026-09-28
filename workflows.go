package seaflowsdk

import "context"

// WorkflowsResource is the canvas lifecycle: draft, save, publish, run.
type WorkflowsResource struct {
	transport *Transport
}

// List lists the workflows visible to the caller: their own drafts and published
// workflows, plus published ones shared with their production line.
func (r *WorkflowsResource) List(ctx context.Context, options ListOptions) ([]Workflow, error) {
	var result []Workflow
	if err := r.transport.GetJSON(ctx, "/seaflow/workflows", options.query(), &result); err != nil {
		return nil, err
	}
	return result, nil
}

// Create creates a blank draft in the caller's default workspace.
func (r *WorkflowsResource) Create(ctx context.Context) (*Workflow, error) {
	var result Workflow
	if err := r.transport.PostJSON(ctx, "/seaflow/workflows", nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// Get reads one workflow. IsOwner=false on the result means read-only.
func (r *WorkflowsResource) Get(ctx context.Context, workflowID string) (*Workflow, error) {
	if err := requireIdentifier("workflowID", workflowID); err != nil {
		return nil, err
	}
	var result Workflow
	if err := r.transport.GetJSON(ctx, "/seaflow/workflows/"+urlEscape(workflowID), nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// Save saves the draft's name and graph. Owner only. The Engine validates the
// graph and never accepts run results here.
//
// Both fields are replaced by what the request carries, so a blank Name blanks
// the workflow's name on the Engine rather than leaving it alone. The name is
// therefore required here and refused locally; read the draft with Get first
// when only the graph is changing.
func (r *WorkflowsResource) Save(ctx context.Context, workflowID string, request SaveWorkflowRequest) (*Workflow, error) {
	if err := requireIdentifier("workflowID", workflowID); err != nil {
		return nil, err
	}
	if err := requireIdentifier("request.Name", request.Name); err != nil {
		return nil, err
	}
	var result Workflow
	if err := r.transport.PatchJSON(ctx, "/seaflow/workflows/"+urlEscape(workflowID), request, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// Delete soft-deletes a workflow. It answers 409 while the workflow has an
// active or stopping run.
func (r *WorkflowsResource) Delete(ctx context.Context, workflowID string) error {
	if err := requireIdentifier("workflowID", workflowID); err != nil {
		return err
	}
	return r.transport.DeleteJSON(ctx, "/seaflow/workflows/"+urlEscape(workflowID), nil, nil)
}

// Publish publishes the saved draft: it mints an immutable version and moves the
// definition's pointer to it (ADR-0007). The result's PublishedVersionID names
// the version a new end-user run will execute.
func (r *WorkflowsResource) Publish(ctx context.Context, workflowID string) (*Workflow, error) {
	if err := requireIdentifier("workflowID", workflowID); err != nil {
		return nil, err
	}
	var result Workflow
	if err := r.transport.PostJSON(ctx, "/seaflow/workflows/"+urlEscape(workflowID)+"/publish", nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// PinCover pins a completed node run's image as the workflow's library cover.
func (r *WorkflowsResource) PinCover(ctx context.Context, workflowID string, request PinCoverRequest) (*Workflow, error) {
	if err := requireIdentifier("workflowID", workflowID); err != nil {
		return nil, err
	}
	if err := requireIdentifier("request.NodeRunID", request.NodeRunID); err != nil {
		return nil, err
	}
	var result Workflow
	if err := r.transport.PutJSON(ctx, "/seaflow/workflows/"+urlEscape(workflowID)+"/cover", request, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// ListRuns lists the caller's own runs of this workflow, newest first. Runs are
// private to their runner.
func (r *WorkflowsResource) ListRuns(ctx context.Context, workflowID string, options ListOptions) ([]Run, error) {
	if err := requireIdentifier("workflowID", workflowID); err != nil {
		return nil, err
	}
	var result []Run
	if err := r.transport.GetJSON(ctx, "/seaflow/workflows/"+urlEscape(workflowID)+"/runs", options.query(), &result); err != nil {
		return nil, err
	}
	return result, nil
}

// CreateRun starts a run and returns as soon as the Engine has scheduled it.
// The snapshot and node records are written in one transaction, so the caller
// does not have to stay connected.
//
// One active run per runner per workflow: a second attempt answers 409, which
// IsConflict reports.
func (r *WorkflowsResource) CreateRun(ctx context.Context, workflowID string, request CreateRunRequest) (*Run, error) {
	if err := requireIdentifier("workflowID", workflowID); err != nil {
		return nil, err
	}
	var result Run
	if err := r.transport.PostJSON(ctx, "/seaflow/workflows/"+urlEscape(workflowID)+"/runs", request, &result); err != nil {
		return nil, err
	}
	return &result, nil
}
