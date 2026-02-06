package northwind

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewClient(t *testing.T) {
	c := NewClient("https://example.com", "test-key")
	if c.baseURL != "https://example.com" {
		t.Errorf("expected baseURL https://example.com, got %s", c.baseURL)
	}
	if c.apiKey != "test-key" {
		t.Errorf("expected apiKey test-key, got %s", c.apiKey)
	}
}

func TestClient_GetBankInfo_Success(t *testing.T) {
	expected := BankInfo{
		Name:          "NorthWind Bank",
		RoutingNumber: "021000089",
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/bank" {
			t.Errorf("expected /bank, got %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("expected Bearer test-key, got %s", r.Header.Get("Authorization"))
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(expected)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key")
	result, err := client.GetBankInfo(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Name != expected.Name {
		t.Errorf("expected name %s, got %s", expected.Name, result.Name)
	}
	if result.RoutingNumber != expected.RoutingNumber {
		t.Errorf("expected routing %s, got %s", expected.RoutingNumber, result.RoutingNumber)
	}
}

func TestClient_GetBankInfo_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(APIErrorResponse{
			Message: "internal server error",
		})
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key")
	_, err := client.GetBankInfo(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if apiErr.StatusCode != 500 {
		t.Errorf("expected status 500, got %d", apiErr.StatusCode)
	}
	if apiErr.Parsed == nil || apiErr.Parsed.Message != "internal server error" {
		t.Error("expected parsed error message")
	}
}

func TestClient_ValidateAccount_Valid(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/external/accounts/validate" {
			t.Errorf("expected /external/accounts/validate, got %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		var req AccountValidationRequest
		json.NewDecoder(r.Body).Decode(&req)
		if req.AccountNumber != "123456" {
			t.Errorf("expected account 123456, got %s", req.AccountNumber)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(AccountValidationResponse{
			Valid:             true,
			AccountNumber:     "123456",
			RoutingNumber:     "021000089",
			InstitutionName:   "Test Bank",
		})
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key")
	result, err := client.ValidateAccount(context.Background(), AccountValidationRequest{
		AccountNumber: "123456",
		RoutingNumber: "021000089",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Valid {
		t.Error("expected valid=true")
	}
	if result.InstitutionName != "Test Bank" {
		t.Errorf("expected institution Test Bank, got %s", result.InstitutionName)
	}
}

func TestClient_GetAccountBalance(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/external/accounts/123456/balance" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(AccountBalance{
			AccountNumber:    "123456",
			AvailableBalance: 5000.50,
			CurrentBalance:   5500.00,
			Currency:         "USD",
		})
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key")
	balance, err := client.GetAccountBalance(context.Background(), "123456")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if balance.AvailableBalance != 5000.50 {
		t.Errorf("expected 5000.50, got %f", balance.AvailableBalance)
	}
}

func TestClient_InitiateTransfer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/external/transfers/initiate" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(TransferResponse{
			TransferID:      "abc-123-def",
			Status:          "PENDING",
			Amount:          1000,
			Currency:        "USD",
			ReferenceNumber: "REF001",
		})
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key")
	result, err := client.InitiateTransfer(context.Background(), TransferRequest{
		Amount:          1000,
		Currency:        "USD",
		Direction:       "OUTBOUND",
		TransferType:    "ACH",
		ReferenceNumber: "REF001",
		SourceAccount: AccountDetails{
			AccountHolderName: "John Doe",
			AccountNumber:     "111",
			RoutingNumber:     "021000089",
		},
		DestinationAccount: AccountDetails{
			AccountHolderName: "Jane Doe",
			AccountNumber:     "222",
			RoutingNumber:     "021000089",
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.TransferID != "abc-123-def" {
		t.Errorf("expected transfer ID abc-123-def, got %s", result.TransferID)
	}
	if result.Status != "PENDING" {
		t.Errorf("expected status PENDING, got %s", result.Status)
	}
}

func TestClient_GetTransferStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/external/transfers/transfer-123" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(TransferResponse{
			TransferID: "transfer-123",
			Status:     "COMPLETED",
		})
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key")
	result, err := client.GetTransferStatus(context.Background(), "transfer-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != "COMPLETED" {
		t.Errorf("expected COMPLETED, got %s", result.Status)
	}
}

func TestClient_CancelTransfer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/external/transfers/transfer-123/cancel" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		var req CancelRequest
		json.NewDecoder(r.Body).Decode(&req)
		if req.Reason != "test reason" {
			t.Errorf("expected reason 'test reason', got %s", req.Reason)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(TransferResponse{
			TransferID: "transfer-123",
			Status:     "CANCELLED",
		})
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key")
	result, err := client.CancelTransfer(context.Background(), "transfer-123", "test reason")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != "CANCELLED" {
		t.Errorf("expected CANCELLED, got %s", result.Status)
	}
}

func TestClient_TraceIDPropagation(t *testing.T) {
	var receivedTraceID string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedTraceID = r.Header.Get("X-Trace-ID")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(HealthResponse{Status: "ok"})
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key")
	ctx := WithTraceID(context.Background(), "trace-abc-123")
	_, err := client.Health(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if receivedTraceID != "trace-abc-123" {
		t.Errorf("expected trace ID trace-abc-123, got %s", receivedTraceID)
	}
}

func TestClient_NonJSONErrorBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		w.Write([]byte("gateway timeout"))
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key")
	_, err := client.Health(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if apiErr.StatusCode != 502 {
		t.Errorf("expected 502, got %d", apiErr.StatusCode)
	}
	if apiErr.Body != "gateway timeout" {
		t.Errorf("expected raw body, got %s", apiErr.Body)
	}
}

func TestAPIError_ErrorMessage(t *testing.T) {
	tests := []struct {
		name     string
		err      *APIError
		expected string
	}{
		{
			name: "with parsed message",
			err: &APIError{
				StatusCode: 400,
				Body:       `{"message":"bad request"}`,
				Parsed:     &APIErrorResponse{Message: "bad request"},
			},
			expected: "northwind api error (HTTP 400): bad request",
		},
		{
			name: "with parsed error field",
			err: &APIError{
				StatusCode: 500,
				Body:       `{"error":"server error"}`,
				Parsed:     &APIErrorResponse{Error: "server error"},
			},
			expected: "northwind api error (HTTP 500): server error",
		},
		{
			name: "without parsed response",
			err: &APIError{
				StatusCode: 503,
				Body:       "service unavailable",
			},
			expected: "northwind api error (HTTP 503): service unavailable",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, got)
			}
		})
	}
}
