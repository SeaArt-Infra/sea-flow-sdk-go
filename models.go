package seaflowsdk

import "context"

// ModelsResource answers what the canvas may bind to an execution node.
type ModelsResource struct {
	transport *Transport
}

// List lists the models the caller may bind, under their model entitlement. The
// answer comes from the live gateway catalog, and the gateway stays the final
// authority at run time.
func (r *ModelsResource) List(ctx context.Context) ([]Model, error) {
	var result []Model
	if err := r.transport.GetJSON(ctx, "/seaflow/models", nil, &result); err != nil {
		return nil, err
	}
	return result, nil
}
