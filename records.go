package seaflowsdk

import "context"

// RecordsResource is the replayable run history across every workspace.
type RecordsResource struct {
	transport *Transport
}

// List lists the caller's run records, newest first.
func (r *RecordsResource) List(ctx context.Context, options ListOptions) ([]RunRecord, error) {
	var result []RunRecord
	if err := r.transport.GetJSON(ctx, "/seaflow/records", options.query(), &result); err != nil {
		return nil, err
	}
	return result, nil
}
