package seaflowsdk

import "context"

// AssetsResource is the terminal output of the caller's runs, as the asset
// library lists it.
type AssetsResource struct {
	transport *Transport
}

// List lists the produced assets of the caller identity, newest first.
//
// With EndUserID, the list includes only that end user's outputs inside the
// project bound to the Production Key. The result never contains another
// project's or end user's assets.
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
