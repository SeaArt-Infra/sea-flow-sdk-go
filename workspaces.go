package seaflowsdk

import "context"

// WorkspacesResource groups canvases.
//
// With EndUserID, a workspace belongs to that end user inside the project bound
// to the Production Key. Omitting EndUserID retains the project's shared
// workspace (ADR-0005, ADR-0010).
type WorkspacesResource struct {
	transport *Transport
}

// List lists the caller's workspaces with their canvas counts.
func (r *WorkspacesResource) List(ctx context.Context, options ListOptions) ([]Workspace, error) {
	var result []Workspace
	if err := r.transport.GetJSON(ctx, "/seaflow/workspaces", options.query(), &result); err != nil {
		return nil, err
	}
	return result, nil
}

// Create creates a workspace.
func (r *WorkspacesResource) Create(ctx context.Context, request CreateWorkspaceRequest) (*Workspace, error) {
	if err := requireIdentifier("request.Name", request.Name); err != nil {
		return nil, err
	}
	var result Workspace
	if err := r.transport.PostJSON(ctx, "/seaflow/workspaces", request, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// Get reads one workspace.
func (r *WorkspacesResource) Get(ctx context.Context, workspaceID string) (*Workspace, error) {
	if err := requireIdentifier("workspaceID", workspaceID); err != nil {
		return nil, err
	}
	var result Workspace
	if err := r.transport.GetJSON(ctx, "/seaflow/workspaces/"+urlEscape(workspaceID), nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// Rename renames a workspace.
func (r *WorkspacesResource) Rename(ctx context.Context, workspaceID string, request RenameWorkspaceRequest) (*Workspace, error) {
	if err := requireIdentifier("workspaceID", workspaceID); err != nil {
		return nil, err
	}
	if err := requireIdentifier("request.Name", request.Name); err != nil {
		return nil, err
	}
	var result Workspace
	if err := r.transport.PatchJSON(ctx, "/seaflow/workspaces/"+urlEscape(workspaceID), request, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// Delete deletes a workspace and soft-deletes its canvases. It answers 409 while
// one of those canvases has an active run.
func (r *WorkspacesResource) Delete(ctx context.Context, workspaceID string) error {
	if err := requireIdentifier("workspaceID", workspaceID); err != nil {
		return err
	}
	return r.transport.DeleteJSON(ctx, "/seaflow/workspaces/"+urlEscape(workspaceID), nil, nil)
}

// ListWorkflows lists the canvases in a workspace.
func (r *WorkspacesResource) ListWorkflows(ctx context.Context, workspaceID string, options ListOptions) ([]Workflow, error) {
	if err := requireIdentifier("workspaceID", workspaceID); err != nil {
		return nil, err
	}
	var result []Workflow
	if err := r.transport.GetJSON(ctx, "/seaflow/workspaces/"+urlEscape(workspaceID)+"/workflows", options.query(), &result); err != nil {
		return nil, err
	}
	return result, nil
}

// CreateWorkflow creates a blank draft in a workspace. It is named
// "Untitled workflow" until the first Save.
func (r *WorkspacesResource) CreateWorkflow(ctx context.Context, workspaceID string) (*Workflow, error) {
	if err := requireIdentifier("workspaceID", workspaceID); err != nil {
		return nil, err
	}
	var result Workflow
	if err := r.transport.PostJSON(ctx, "/seaflow/workspaces/"+urlEscape(workspaceID)+"/workflows", nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}
