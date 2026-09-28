package seaflowsdk

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// defaultHTTPTimeout bounds one call. A run is only *created* here, so no
// baseURL in the contract waits on model generation.
const defaultHTTPTimeout = 60 * time.Second

// maxFallbackMessageBytes caps how much of a non-JSON response body is quoted
// back in an error.
const maxFallbackMessageBytes = 4 << 10

// EndUserHeader is the header the Engine reads the integrating product's end
// user from (ADR-0010). It is set from ClientOptions.EndUserID.
const EndUserHeader = "X-Infra-User-Id"

var (
	// ErrMissingBaseURL is returned by the first call on a client built without a
	// BaseURL or APIBaseURL. The SDK never guesses a destination.
	ErrMissingBaseURL = errors.New("sea-flow-sdk-go: BaseURL is required")
	// ErrMissingProductionKey is returned by a credentialed call on a client built
	// without a ProductionKey.
	ErrMissingProductionKey = errors.New("sea-flow-sdk-go: ProductionKey is required")
	// ErrMissingIdentifier is returned when a call omits a path identifier or a
	// required request field. The check runs locally so a caller mistake reads as
	// one instead of as the Engine's 400/404. The wrapper names the field, so the
	// sentinel itself carries no prefix.
	ErrMissingIdentifier = errors.New("missing required value")
)

// QueryParams carries optional query values. Zero values are omitted, so a
// caller can pass "unset" as the Go zero value.
type QueryParams map[string]any

// APIError is the Engine's failure envelope, with the HTTP status kept separate
// from the body's code because the contract mirrors them but does not have to.
type APIError struct {
	HTTPStatus int
	Code       int
	Message    string
}

func (e *APIError) Error() string {
	if e.Code == 0 || e.Code == e.HTTPStatus {
		return fmt.Sprintf("workflow engine HTTP %d: %s", e.HTTPStatus, e.Message)
	}
	return fmt.Sprintf("workflow engine HTTP %d (code %d): %s", e.HTTPStatus, e.Code, e.Message)
}

// IsNotFound reports whether a call failed because the resource does not exist
// or is not visible to the caller.
func IsNotFound(err error) bool { return hasStatus(err, http.StatusNotFound) }

// IsConflict reports whether a call failed because an active run blocks it or a
// duplicate exists.
func IsConflict(err error) bool { return hasStatus(err, http.StatusConflict) }

func hasStatus(err error, status int) bool {
	var apiError *APIError
	if !errors.As(err, &apiError) {
		return false
	}
	return apiError.HTTPStatus == status
}

// envelope is the platform response wrapper every path answers with:
// {"code":0,"message":"ok","data":...} on success and {"code":<status>,...} on
// failure.
type envelope struct {
	Code    *int            `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

// Transport performs the HTTP work: it derives the API URL, sets the Production
// Key and end-user header, and unwraps the envelope.
type Transport struct {
	baseURL       string
	apiBaseURL    string
	productionKey string
	endUserID     string
	headers       map[string]string
	httpClient    *http.Client
}

// NewTransport builds a transport. BaseURL names the gateway or Engine root;
// APIBaseURL can explicitly replace the derived <BaseURL>/api/v1 address.
func NewTransport(options ClientOptions) *Transport {
	httpClient := options.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: defaultHTTPTimeout}
	}
	baseURL := normalizeBaseURL(options.BaseURL)
	return &Transport{
		baseURL:       baseURL,
		apiBaseURL:    resolveAPIBaseURL(baseURL, options.APIBaseURL),
		productionKey: strings.TrimSpace(options.ProductionKey),
		endUserID:     strings.TrimSpace(options.EndUserID),
		headers:       cloneStringMap(options.Headers),
		httpClient:    httpClient,
	}
}

// WithEndUser copies the transport with a different default end user, leaving
// the original untouched.
func (t *Transport) WithEndUser(endUserID string) *Transport {
	clone := *t
	clone.endUserID = strings.TrimSpace(endUserID)
	clone.headers = cloneStringMap(t.headers)
	return &clone
}

func (t *Transport) GetJSON(ctx context.Context, path string, query QueryParams, dst any) error {
	return t.request(ctx, http.MethodGet, path, query, nil, dst)
}

func (t *Transport) PostJSON(ctx context.Context, path string, body any, dst any) error {
	return t.request(ctx, http.MethodPost, path, nil, body, dst)
}

func (t *Transport) PutJSON(ctx context.Context, path string, body any, dst any) error {
	return t.request(ctx, http.MethodPut, path, nil, body, dst)
}

func (t *Transport) PatchJSON(ctx context.Context, path string, body any, dst any) error {
	return t.request(ctx, http.MethodPatch, path, nil, body, dst)
}

func (t *Transport) DeleteJSON(ctx context.Context, path string, query QueryParams, dst any) error {
	return t.request(ctx, http.MethodDelete, path, query, nil, dst)
}

func (t *Transport) request(ctx context.Context, method, path string, query QueryParams, body any, dst any) error {
	requestURL, err := t.buildURL(path, query)
	if err != nil {
		return err
	}
	if t.productionKey == "" {
		return ErrMissingProductionKey
	}

	var reader io.Reader
	hasBody := body != nil
	if hasBody {
		raw, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("sea-flow-sdk-go: encode %s %s: %w", method, path, err)
		}
		reader = bytes.NewReader(raw)
	}

	request, err := http.NewRequestWithContext(ctx, method, requestURL, reader)
	if err != nil {
		return fmt.Errorf("sea-flow-sdk-go: build %s %s: %w", method, path, err)
	}
	request.Header.Set("Accept", "application/json")
	if hasBody {
		request.Header.Set("Content-Type", "application/json")
	}
	t.applyHeaders(request)

	response, err := t.httpClient.Do(request)
	if err != nil {
		return err
	}
	defer func() { _ = response.Body.Close() }()

	raw, err := io.ReadAll(response.Body)
	if err != nil {
		return fmt.Errorf("sea-flow-sdk-go: read %s %s: %w", method, path, err)
	}
	return decodeEnvelope(response.StatusCode, raw, dst)
}

// applyHeaders puts the caller's headers on first, then the credential and the
// end-user identifier. EndUserID wins over a header of the same name so that a
// client built from ClientOptions.Headers cannot silently shadow it.
func (t *Transport) applyHeaders(request *http.Request) {
	for key, value := range t.headers {
		if strings.TrimSpace(key) != "" {
			request.Header.Set(key, value)
		}
	}
	request.Header.Del("Authorization")
	request.Header.Set("Authorization", "Bearer "+t.productionKey)
	if t.endUserID != "" {
		request.Header.Set(EndUserHeader, t.endUserID)
	} else {
		request.Header.Del(EndUserHeader)
	}
}

func (t *Transport) buildURL(path string, query QueryParams) (string, error) {
	if t.apiBaseURL == "" {
		return "", ErrMissingBaseURL
	}
	base, err := url.Parse(t.apiBaseURL)
	if err != nil {
		return "", fmt.Errorf("sea-flow-sdk-go: invalid APIBaseURL %q: %w", t.apiBaseURL, err)
	}
	if base.Scheme == "" || base.Host == "" {
		return "", fmt.Errorf("sea-flow-sdk-go: invalid APIBaseURL %q: a scheme and host are required", t.apiBaseURL)
	}
	base.Path = strings.TrimRight(base.Path, "/") + path
	if encoded := queryValues(query).Encode(); encoded != "" {
		base.RawQuery = encoded
	}
	return base.String(), nil
}

func queryValues(query QueryParams) url.Values {
	values := url.Values{}
	for key, value := range query {
		switch typed := value.(type) {
		case nil:
		case string:
			if typed != "" {
				values.Set(key, typed)
			}
		case int:
			if typed != 0 {
				values.Set(key, strconv.Itoa(typed))
			}
		case int64:
			if typed != 0 {
				values.Set(key, strconv.FormatInt(typed, 10))
			}
		case bool:
			values.Set(key, strconv.FormatBool(typed))
		default:
			values.Set(key, fmt.Sprint(typed))
		}
	}
	return values
}

// decodeEnvelope turns one response into either a decoded payload or an
// *APIError. A non-2xx status and a non-zero body code are both failures: the
// contract mirrors them, and reading only one of the two would let a proxy or a
// future Engine slip an error past the caller.
func decodeEnvelope(status int, raw []byte, dst any) error {
	var decoded envelope
	if err := json.Unmarshal(raw, &decoded); err != nil {
		if status >= 200 && status <= 299 {
			return fmt.Errorf("sea-flow-sdk-go: decode response: %w", err)
		}
		return &APIError{HTTPStatus: status, Code: status, Message: fallbackMessage(status, raw)}
	}

	code := status
	if decoded.Code != nil {
		code = *decoded.Code
	}
	if status < 200 || status > 299 || code != 0 {
		message := strings.TrimSpace(decoded.Message)
		if message == "" {
			message = fallbackMessage(status, raw)
		}
		return &APIError{HTTPStatus: status, Code: code, Message: message}
	}

	if dst == nil || len(decoded.Data) == 0 || string(decoded.Data) == "null" {
		return nil
	}
	if err := json.Unmarshal(decoded.Data, dst); err != nil {
		return fmt.Errorf("sea-flow-sdk-go: decode response data: %w", err)
	}
	return nil
}

func fallbackMessage(status int, raw []byte) string {
	text := strings.TrimSpace(string(raw))
	if text == "" {
		if message := strings.TrimSpace(http.StatusText(status)); message != "" {
			return message
		}
		return "request failed"
	}
	if len(text) > maxFallbackMessageBytes {
		text = text[:maxFallbackMessageBytes]
	}
	return text
}

func normalizeBaseURL(baseURL string) string {
	return strings.TrimRight(strings.TrimSpace(baseURL), "/")
}

func resolveAPIBaseURL(baseURL, apiBaseURL string) string {
	if explicit := normalizeBaseURL(apiBaseURL); explicit != "" {
		return explicit
	}
	if baseURL == "" {
		return ""
	}
	parsed, err := url.Parse(baseURL)
	if err != nil {
		return baseURL
	}
	path := strings.TrimRight(parsed.Path, "/")
	if !strings.HasSuffix(path, "/api/v1") {
		path += "/api/v1"
	}
	parsed.Path = path
	return normalizeBaseURL(parsed.String())
}
