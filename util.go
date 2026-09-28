package seaflowsdk

import (
	"fmt"
	"net/url"
	"strings"
)

// urlEscape keeps a caller-supplied identifier inside one path segment.
func urlEscape(value string) string {
	return url.PathEscape(value)
}

// requireIdentifier rejects an empty identifier or required field before a
// request is sent, so a missing argument reads as a caller mistake instead of
// the Engine's 400/404. The result wraps ErrMissingIdentifier, so a caller can
// test for it with errors.Is whichever field was missing.
func requireIdentifier(name, value string) error {
	if strings.TrimSpace(value) == "" {
		return fmt.Errorf("sea-flow-sdk-go: %s is required: %w", name, ErrMissingIdentifier)
	}
	return nil
}

// cloneStringMap copies caller-supplied headers so a clone of the client never
// shares a map with the original.
func cloneStringMap(source map[string]string) map[string]string {
	if len(source) == 0 {
		return nil
	}
	clone := make(map[string]string, len(source))
	for key, value := range source {
		clone[key] = value
	}
	return clone
}
