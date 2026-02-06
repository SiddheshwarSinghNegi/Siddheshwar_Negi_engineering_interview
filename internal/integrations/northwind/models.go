package northwind

import "time"

// --- Request Models ---

// AccountValidationRequest represents a request to validate an external account
type AccountValidationRequest struct {
	AccountNumber string `json:"account_number"`
	RoutingNumber string `json:"routing_number"`
	AccountType   string `json:"account_type,omitempty"`
}

// TransferRequest represents a request to initiate or validate a transfer
type TransferRequest struct {
	Amount             float64         `json:"amount"`
	Currency           string          `json:"currency"`
	Description        string          `json:"description,omitempty"`
	Direction          string          `json:"direction"`
	TransferType       string          `json:"transfer_type"`
	ReferenceNumber    string          `json:"reference_number"`
	ScheduledDate      string          `json:"scheduled_date,omitempty"`
	SourceAccount      AccountDetails  `json:"source_account"`
	DestinationAccount AccountDetails  `json:"destination_account"`
}

// AccountDetails represents bank account details in a transfer
type AccountDetails struct {
	AccountHolderName string `json:"account_holder_name"`
	AccountNumber     string `json:"account_number"`
	RoutingNumber     string `json:"routing_number"`
	InstitutionName   string `json:"institution_name,omitempty"`
}

// BatchTransferRequest represents a batch of transfers
type BatchTransferRequest struct {
	Transfers []TransferRequest `json:"transfers"`
}

// CancelRequest represents a transfer cancel request
type CancelRequest struct {
	Reason string `json:"reason"`
}

// ReverseRequest represents a transfer reversal request
type ReverseRequest struct {
	Reason      string `json:"reason"`
	Description string `json:"description,omitempty"`
}

// TransferListFilters represents filters for listing transfers
type TransferListFilters struct {
	Status       string `json:"status,omitempty"`
	Direction    string `json:"direction,omitempty"`
	TransferType string `json:"transfer_type,omitempty"`
	Limit        int    `json:"limit,omitempty"`
	Offset       int    `json:"offset,omitempty"`
}

// --- Response Models ---

// BankInfo represents NorthWind bank information
type BankInfo struct {
	Name          string `json:"name"`
	RoutingNumber string `json:"routing_number"`
	SwiftCode     string `json:"swift_code,omitempty"`
	Address       string `json:"address,omitempty"`
	City          string `json:"city,omitempty"`
	State         string `json:"state,omitempty"`
	ZipCode       string `json:"zip_code,omitempty"`
	Country       string `json:"country,omitempty"`
}

// Domain represents a NorthWind domain
type Domain struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// ExternalAccount represents an external account from NorthWind
type ExternalAccount struct {
	AccountNumber     string `json:"account_number"`
	RoutingNumber     string `json:"routing_number"`
	AccountHolderName string `json:"account_holder_name"`
	AccountType       string `json:"account_type,omitempty"`
	InstitutionName   string `json:"institution_name,omitempty"`
	Status            string `json:"status,omitempty"`
}

// AccountValidationResponse represents the response from account validation
type AccountValidationResponse struct {
	Valid             bool   `json:"valid"`
	AccountNumber     string `json:"account_number,omitempty"`
	RoutingNumber     string `json:"routing_number,omitempty"`
	AccountHolderName string `json:"account_holder_name,omitempty"`
	InstitutionName   string `json:"institution_name,omitempty"`
	AccountType       string `json:"account_type,omitempty"`
	Message           string `json:"message,omitempty"`
}

// AccountBalance represents an account balance from NorthWind
type AccountBalance struct {
	AccountNumber    string  `json:"account_number"`
	AvailableBalance float64 `json:"available_balance"`
	CurrentBalance   float64 `json:"current_balance"`
	Currency         string  `json:"currency"`
}

// TransferResponse represents a transfer response from NorthWind
type TransferResponse struct {
	TransferID             string   `json:"transfer_id"`
	Status                 string   `json:"status"`
	Amount                 float64  `json:"amount"`
	Currency               string   `json:"currency"`
	Direction              string   `json:"direction"`
	TransferType           string   `json:"transfer_type"`
	ReferenceNumber        string   `json:"reference_number"`
	Description            string   `json:"description,omitempty"`
	ScheduledDate          string   `json:"scheduled_date,omitempty"`
	SourceAccount          AccountDetails `json:"source_account"`
	DestinationAccount     AccountDetails `json:"destination_account"`
	InitiatedDate          string   `json:"initiated_date,omitempty"`
	ProcessingDate         string   `json:"processing_date,omitempty"`
	ExpectedCompletionDate string   `json:"expected_completion_date,omitempty"`
	CompletedDate          string   `json:"completed_date,omitempty"`
	Fee                    *float64 `json:"fee,omitempty"`
	ExchangeRate           *float64 `json:"exchange_rate,omitempty"`
	ErrorCode              string   `json:"error_code,omitempty"`
	ErrorMessage           string   `json:"error_message,omitempty"`
	CreatedAt              string   `json:"created_at,omitempty"`
	UpdatedAt              string   `json:"updated_at,omitempty"`
}

// TransferValidationResponse represents transfer validation result
type TransferValidationResponse struct {
	Valid  bool                     `json:"valid"`
	Issues []TransferValidationIssue `json:"issues,omitempty"`
}

// TransferValidationIssue represents a single validation issue
type TransferValidationIssue struct {
	Field    string `json:"field,omitempty"`
	Message  string `json:"message"`
	Severity string `json:"severity"` // "error" or "warning"
}

// BatchTransferResponse represents a batch transfer response
type BatchTransferResponse struct {
	Transfers []TransferResponse `json:"transfers"`
	TotalCount int               `json:"total_count"`
	SuccessCount int             `json:"success_count"`
	FailedCount  int             `json:"failed_count"`
}

// TransferStatusResponse represents a transfer status response from NorthWind
type TransferStatusResponse = TransferResponse

// HealthResponse represents the NorthWind health check response
type HealthResponse struct {
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp,omitempty"`
}

// APIErrorResponse represents an error response from NorthWind
type APIErrorResponse struct {
	Error   string `json:"error,omitempty"`
	Message string `json:"message,omitempty"`
	Code    string `json:"code,omitempty"`
	Details string `json:"details,omitempty"`
}

// PaginatedResponse wraps a paginated list response
type PaginatedResponse struct {
	Data       interface{} `json:"data"`
	TotalCount int         `json:"total_count,omitempty"`
	Limit      int         `json:"limit,omitempty"`
	Offset     int         `json:"offset,omitempty"`
}

// AccountListResponse for listing accounts
type AccountListResponse struct {
	Accounts   []ExternalAccount `json:"accounts"`
	TotalCount int               `json:"total_count,omitempty"`
}

// TransferListResponse for listing transfers
type TransferListResponse struct {
	Transfers  []TransferResponse `json:"transfers"`
	TotalCount int                `json:"total_count,omitempty"`
}
