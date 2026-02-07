package northwind

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// Client is the NorthWind Bank API client
type Client struct {
	baseURL             string
	apiKey              string
	httpClient          *http.Client
	maxRetries          int
	retryInitialBackoff time.Duration
}

// ClientOption configures the NorthWind client
type ClientOption func(*Client)

// WithRetry enables retries with exponential backoff
func WithRetry(maxRetries int, initialBackoffMs int) ClientOption {
	return func(c *Client) {
		c.maxRetries = maxRetries
		c.retryInitialBackoff = time.Duration(initialBackoffMs) * time.Millisecond
	}
}

// NewClient creates a new NorthWind API client
func NewClient(baseURL, apiKey string, opts ...ClientOption) *Client {
	c := &Client{
		baseURL: baseURL,
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// APIError represents an error returned by the NorthWind API
type APIError struct {
	StatusCode int
	Body       string
	Parsed     *APIErrorResponse
}

func (e *APIError) Error() string {
	if e.Parsed != nil {
		msg := e.Parsed.Message
		if msg == "" {
			msg = e.Parsed.Error
		}
		if msg != "" {
			return fmt.Sprintf("northwind api error (HTTP %d): %s", e.StatusCode, msg)
		}
	}
	return fmt.Sprintf("northwind api error (HTTP %d): %s", e.StatusCode, e.Body)
}

// doRequest executes an HTTP request to the NorthWind API with optional retries.
// Retries on network errors and 5xx responses; does not retry on 4xx.
func (c *Client) doRequest(ctx context.Context, method, path string, body interface{}) ([]byte, int, error) {
	fullURL := c.baseURL + path

	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewBuffer(jsonBody)
	}

	var lastErr error
	var lastStatus int

	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, 0, ctx.Err()
			case <-time.After(c.retryBackoff(attempt)):
				// proceed to retry
			}
		}

		req, err := http.NewRequestWithContext(ctx, method, fullURL, reqBody)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to create request: %w", err)
		}

		req.Header.Set("Authorization", "Bearer "+c.apiKey)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")
		if traceID, ok := ctx.Value(traceIDKey).(string); ok && traceID != "" {
			req.Header.Set("X-Trace-ID", traceID)
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("failed to execute request: %w", err)
			continue
		}

		respBody, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			lastErr = fmt.Errorf("failed to read response body: %w", err)
			lastStatus = resp.StatusCode
			continue
		}

		if resp.StatusCode >= 400 {
			apiErr := &APIError{StatusCode: resp.StatusCode, Body: string(respBody)}
			var parsed APIErrorResponse
			if json.Unmarshal(respBody, &parsed) == nil {
				apiErr.Parsed = &parsed
			}
			// Do not retry 4xx
			if resp.StatusCode < 500 {
				return nil, resp.StatusCode, apiErr
			}
			lastErr = apiErr
			lastStatus = resp.StatusCode
			continue
		}

		return respBody, resp.StatusCode, nil
	}

	if apiErr, ok := lastErr.(*APIError); ok {
		return nil, lastStatus, apiErr
	}
	return nil, lastStatus, lastErr
}

func (c *Client) retryBackoff(attempt int) time.Duration {
	if c.retryInitialBackoff <= 0 {
		return 0
	}
	// Exponential: initial * 2^attempt
	d := c.retryInitialBackoff * time.Duration(1<<uint(attempt-1))
	if d > 10*time.Second {
		return 10 * time.Second
	}
	return d
}

type contextKey string

const traceIDKey contextKey = "trace_id"

// WithTraceID returns a context with the trace ID set for propagation
func WithTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, traceIDKey, traceID)
}

// --- API Methods ---

// GetBankInfo retrieves NorthWind bank information
func (c *Client) GetBankInfo(ctx context.Context) (*BankInfo, error) {
	body, _, err := c.doRequest(ctx, http.MethodGet, "/bank", nil)
	if err != nil {
		return nil, err
	}
	var result BankInfo
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to decode bank info: %w", err)
	}
	return &result, nil
}

// GetDomains retrieves NorthWind domains
func (c *Client) GetDomains(ctx context.Context) ([]Domain, error) {
	body, _, err := c.doRequest(ctx, http.MethodGet, "/domains", nil)
	if err != nil {
		return nil, err
	}
	var result []Domain
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to decode domains: %w", err)
	}
	return result, nil
}

// ListAccounts lists external accounts from NorthWind
func (c *Client) ListAccounts(ctx context.Context, limit, offset int, accountType, status string) ([]ExternalAccount, error) {
	params := url.Values{}
	if limit > 0 {
		params.Set("limit", strconv.Itoa(limit))
	}
	if offset > 0 {
		params.Set("offset", strconv.Itoa(offset))
	}
	if accountType != "" {
		params.Set("type", accountType)
	}
	if status != "" {
		params.Set("status", status)
	}

	path := "/external/accounts"
	if len(params) > 0 {
		path += "?" + params.Encode()
	}

	body, _, err := c.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	var result []ExternalAccount
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to decode accounts: %w", err)
	}
	return result, nil
}

// ValidateAccount validates an external account with NorthWind
func (c *Client) ValidateAccount(ctx context.Context, req AccountValidationRequest) (*AccountValidationResponse, error) {
	body, _, err := c.doRequest(ctx, http.MethodPost, "/external/accounts/validate", req)
	if err != nil {
		return nil, err
	}
	var result AccountValidationResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to decode validation response: %w", err)
	}
	return &result, nil
}

// GetAccountBalance retrieves balance for an external account
func (c *Client) GetAccountBalance(ctx context.Context, accountNumber string) (*AccountBalance, error) {
	path := fmt.Sprintf("/external/accounts/%s/balance", url.PathEscape(accountNumber))
	body, _, err := c.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	var result AccountBalance
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to decode account balance: %w", err)
	}
	return &result, nil
}

// ListTransfers lists external transfers from NorthWind
func (c *Client) ListTransfers(ctx context.Context, filters TransferListFilters) ([]TransferResponse, error) {
	params := url.Values{}
	if filters.Status != "" {
		params.Set("status", filters.Status)
	}
	if filters.Direction != "" {
		params.Set("direction", filters.Direction)
	}
	if filters.TransferType != "" {
		params.Set("transfer_type", filters.TransferType)
	}
	if filters.Limit > 0 {
		params.Set("limit", strconv.Itoa(filters.Limit))
	}
	if filters.Offset > 0 {
		params.Set("offset", strconv.Itoa(filters.Offset))
	}

	path := "/external/transfers"
	if len(params) > 0 {
		path += "?" + params.Encode()
	}

	body, _, err := c.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	var result []TransferResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to decode transfers: %w", err)
	}
	return result, nil
}

// ValidateTransfer validates a transfer request with NorthWind
func (c *Client) ValidateTransfer(ctx context.Context, req TransferRequest) (*TransferValidationResponse, error) {
	body, _, err := c.doRequest(ctx, http.MethodPost, "/external/transfers/validate", req)
	if err != nil {
		return nil, err
	}
	var result TransferValidationResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to decode transfer validation: %w", err)
	}
	return &result, nil
}

// InitiateTransfer initiates a transfer via NorthWind
func (c *Client) InitiateTransfer(ctx context.Context, req TransferRequest) (*TransferResponse, error) {
	body, _, err := c.doRequest(ctx, http.MethodPost, "/external/transfers/initiate", req)
	if err != nil {
		return nil, err
	}
	var result TransferResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to decode transfer response: %w", err)
	}
	return &result, nil
}

// BatchTransfers submits a batch of transfers
func (c *Client) BatchTransfers(ctx context.Context, req BatchTransferRequest) (*BatchTransferResponse, error) {
	body, _, err := c.doRequest(ctx, http.MethodPost, "/external/transfers/batch", req)
	if err != nil {
		return nil, err
	}
	var result BatchTransferResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to decode batch transfer response: %w", err)
	}
	return &result, nil
}

// GetTransferStatus retrieves the status of a transfer
func (c *Client) GetTransferStatus(ctx context.Context, transferID string) (*TransferStatusResponse, error) {
	path := fmt.Sprintf("/external/transfers/%s", url.PathEscape(transferID))
	body, _, err := c.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	var result TransferStatusResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to decode transfer status: %w", err)
	}
	return &result, nil
}

// CancelTransfer cancels a pending transfer
func (c *Client) CancelTransfer(ctx context.Context, transferID, reason string) (*TransferResponse, error) {
	path := fmt.Sprintf("/external/transfers/%s/cancel", url.PathEscape(transferID))
	body, _, err := c.doRequest(ctx, http.MethodPost, path, CancelRequest{Reason: reason})
	if err != nil {
		return nil, err
	}
	var result TransferResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to decode cancel response: %w", err)
	}
	return &result, nil
}

// ReverseTransfer reverses a completed transfer
func (c *Client) ReverseTransfer(ctx context.Context, transferID, reason, description string) (*TransferResponse, error) {
	path := fmt.Sprintf("/external/transfers/%s/reverse", url.PathEscape(transferID))
	body, _, err := c.doRequest(ctx, http.MethodPost, path, ReverseRequest{
		Reason:      reason,
		Description: description,
	})
	if err != nil {
		return nil, err
	}
	var result TransferResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to decode reverse response: %w", err)
	}
	return &result, nil
}

// Reset resets NorthWind state (development only)
func (c *Client) Reset(ctx context.Context) error {
	_, _, err := c.doRequest(ctx, http.MethodPost, "/external/reset", nil)
	return err
}

// Health checks NorthWind API health
func (c *Client) Health(ctx context.Context) (*HealthResponse, error) {
	body, _, err := c.doRequest(ctx, http.MethodGet, "/health", nil)
	if err != nil {
		return nil, err
	}
	var result HealthResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to decode health response: %w", err)
	}
	return &result, nil
}
