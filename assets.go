package seaflowsdk

import "context"

// AssetsResource is the terminal output of the caller's runs, as the asset
// library lists it.
type AssetsResource struct {
	transport *Transport
}

// List lists the produced assets of the caller identity, newest first.
//
// Ownership follows the credential, not the end user: with the Production Key the
// list spans every end user of that project (the Engine records a workspace's
// creator as the project). Only runs are separated per end user. The result
// therefore never contains another project's assets.
func (r *AssetsResource) List(ctx context.Context, options ListAssetsOptions) ([]Asset, error) {
	query := QueryParams{
		"workspaceId": options.WorkspaceID,
		"workflowId":  options.WorkflowID,
		"limit":       options.Limit,
		"offset":      options.Offset,
	}
	var result []Asset
	if err := r.transport.GetJSON(ctx, "/seaflow/assets", query, &result); err != nil {
		return nil, err
	}
	return result, nil
}
