package seaflowsdk

import "context"

// TemplatesResource is the caller's own catalog entries.
type TemplatesResource struct {
	transport *Transport
}

// List lists published catalog entries.
func (r *TemplatesResource) List(ctx context.Context, options ListOptions) ([]Template, error) {
	var result []Template
	if err := r.transport.GetJSON(ctx, "/seaflow/templates", options.query(), &result); err != nil {
		return nil, err
	}
	return result, nil
}

// Publish publishes the caller's own workflow as a new catalog entry. It takes
// effect immediately, with no review step (ADR-0011).
func (r *TemplatesResource) Publish(ctx context.Context, request TemplateSaveRequest) (*TemplateDetail, error) {
	if err := requireIdentifier("request.WorkflowID", request.WorkflowID); err != nil {
		return nil, err
	}
	if err := requireIdentifier("request.Name", request.Name); err != nil {
		return nil, err
	}
	var result TemplateDetail
	if err := r.transport.PostJSON(ctx, "/seaflow/templates", request, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// Get reads one template with its current published version.
func (r *TemplatesResource) Get(ctx context.Context, templateID string) (*TemplateDetail, error) {
	if err := requireIdentifier("templateID", templateID); err != nil {
		return nil, err
	}
	var result TemplateDetail
	if err := r.transport.GetJSON(ctx, "/seaflow/templates/"+urlEscape(templateID), nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// Delete takes the caller's own template out of the catalog. Copies already made
// keep running (ADR-0011).
func (r *TemplatesResource) Delete(ctx context.Context, templateID string) error {
	if err := requireIdentifier("templateID", templateID); err != nil {
		return err
	}
	return r.transport.DeleteJSON(ctx, "/seaflow/templates/"+urlEscape(templateID), nil, nil)
}

// Copy copies a template into a workspace as a new workflow. The copy pins the
// template version it came from.
func (r *TemplatesResource) Copy(ctx context.Context, templateID string, request CopyTemplateRequest) (*Workflow, error) {
	if err := requireIdentifier("templateID", templateID); err != nil {
		return nil, err
	}
	// The Engine refuses a copy with no destination workspace (400 projectId is
	// required), so it is caught here as the caller's mistake it is.
	if err := requireIdentifier("request.WorkspaceID", request.WorkspaceID); err != nil {
		return nil, err
	}
	var result Workflow
	if err := r.transport.PostJSON(ctx, "/seaflow/templates/"+urlEscape(templateID)+"/copies", request, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// PublishVersion ships the next version of the caller's own template.
func (r *TemplatesResource) PublishVersion(ctx context.Context, templateID string, request TemplateSaveRequest) (*TemplateDetail, error) {
	if err := requireIdentifier("templateID", templateID); err != nil {
		return nil, err
	}
	if err := requireIdentifier("request.WorkflowID", request.WorkflowID); err != nil {
		return nil, err
	}
	if err := requireIdentifier("request.Name", request.Name); err != nil {
		return nil, err
	}
	var result TemplateDetail
	if err := r.transport.PostJSON(ctx, "/seaflow/templates/"+urlEscape(templateID)+"/versions", request, &result); err != nil {
		return nil, err
	}
	return &result, nil
}
