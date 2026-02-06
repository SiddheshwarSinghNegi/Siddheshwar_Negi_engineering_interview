package handlers

import (
	"errors"
	"net/http"

	appErrors "github.com/array/banking-api/internal/errors"
	"github.com/array/banking-api/internal/integrations/northwind"
	"github.com/array/banking-api/internal/services"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

// NorthwindHandler handles NorthWind integration endpoints
type NorthwindHandler struct {
	client     *northwind.Client
	accountSvc *services.NorthwindAccountService
	transferSvc *services.NorthwindTransferService
}

// NewNorthwindHandler creates a new NorthWind handler
func NewNorthwindHandler(
	client *northwind.Client,
	accountSvc *services.NorthwindAccountService,
	transferSvc *services.NorthwindTransferService,
) *NorthwindHandler {
	return &NorthwindHandler{
		client:     client,
		accountSvc: accountSvc,
		transferSvc: transferSvc,
	}
}

// --- Bank Info & Domains ---

// GetBankInfo retrieves NorthWind bank information
func (h *NorthwindHandler) GetBankInfo(c echo.Context) error {
	info, err := h.client.GetBankInfo(c.Request().Context())
	if err != nil {
		return SendError(c, appErrors.NorthwindAPIError, appErrors.WithDetails(err.Error()))
	}
	return c.JSON(http.StatusOK, SuccessResponse{
		Data:    info,
		Message: "Bank info retrieved",
	})
}

// GetDomains retrieves NorthWind domains
func (h *NorthwindHandler) GetDomains(c echo.Context) error {
	domains, err := h.client.GetDomains(c.Request().Context())
	if err != nil {
		return SendError(c, appErrors.NorthwindAPIError, appErrors.WithDetails(err.Error()))
	}
	return c.JSON(http.StatusOK, SuccessResponse{
		Data:    domains,
		Message: "Domains retrieved",
	})
}

// --- External Accounts ---

// ValidateAndRegister validates and registers an external account
func (h *NorthwindHandler) ValidateAndRegister(c echo.Context) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return SendError(c, appErrors.AuthMissingToken)
	}

	var req services.ValidateAndRegisterRequest
	if err := c.Bind(&req); err != nil {
		return SendError(c, appErrors.ValidationGeneral, appErrors.WithDetails("Invalid request body"))
	}
	if err := c.Validate(req); err != nil {
		return err
	}

	resp, err := h.accountSvc.ValidateAndRegister(c.Request().Context(), userID, req)
	if err != nil {
		if errors.Is(err, services.ErrExternalAccountValidationFailed) {
			return c.JSON(http.StatusUnprocessableEntity, SuccessResponse{
				Data:    resp,
				Message: "Account validation failed",
			})
		}
		return SendSystemError(c, err)
	}

	return c.JSON(http.StatusCreated, SuccessResponse{
		Data:    resp,
		Message: "External account validated and registered",
	})
}

// ListRegisteredAccounts lists the user's registered external accounts
func (h *NorthwindHandler) ListRegisteredAccounts(c echo.Context) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return SendError(c, appErrors.AuthMissingToken)
	}

	offset := getIntParam(c, "offset", 0)
	limit := getIntParam(c, "limit", 20)
	if limit > 100 {
		limit = 100
	}

	accounts, total, err := h.accountSvc.ListRegisteredAccounts(c.Request().Context(), userID, offset, limit)
	if err != nil {
		return SendSystemError(c, err)
	}

	return c.JSON(http.StatusOK, SuccessResponse{
		Data:    accounts,
		Message: "Registered external accounts retrieved",
		Meta: map[string]interface{}{
			"total":  total,
			"offset": offset,
			"limit":  limit,
		},
	})
}

// ListAccessibleAccounts lists accessible accounts from NorthWind API
func (h *NorthwindHandler) ListAccessibleAccounts(c echo.Context) error {
	accounts, err := h.accountSvc.ListAccessibleAccounts(c.Request().Context())
	if err != nil {
		return SendError(c, appErrors.NorthwindAPIError, appErrors.WithDetails(err.Error()))
	}
	return c.JSON(http.StatusOK, SuccessResponse{
		Data:    accounts,
		Message: "Accessible NorthWind accounts retrieved",
	})
}

// --- Transfers ---

// CreateTransfer initiates a new external transfer
func (h *NorthwindHandler) CreateTransfer(c echo.Context) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return SendError(c, appErrors.AuthMissingToken)
	}

	var req services.CreateTransferRequest
	if err := c.Bind(&req); err != nil {
		return SendError(c, appErrors.ValidationGeneral, appErrors.WithDetails("Invalid request body"))
	}
	if err := c.Validate(req); err != nil {
		return err
	}

	resp, err := h.transferSvc.CreateTransfer(c.Request().Context(), userID, req)
	if err != nil {
		if errors.Is(err, services.ErrNWTransferValidationFailed) {
			return SendError(c, appErrors.NorthwindTransferValidationFail, appErrors.WithDetails(err.Error()))
		}
		if errors.Is(err, services.ErrNWTransferInsufficientBal) {
			return SendError(c, appErrors.NorthwindTransferInsufficientBal, appErrors.WithDetails(err.Error()))
		}
		if errors.Is(err, services.ErrNWTransferInitiateFailed) {
			return SendError(c, appErrors.NorthwindTransferInitiateFail, appErrors.WithDetails(err.Error()))
		}
		return SendSystemError(c, err)
	}

	return c.JSON(http.StatusCreated, SuccessResponse{
		Data:    resp,
		Message: "Transfer initiated successfully",
	})
}

// GetTransfer retrieves a specific transfer
func (h *NorthwindHandler) GetTransfer(c echo.Context) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return SendError(c, appErrors.AuthMissingToken)
	}

	transferID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return SendError(c, appErrors.ValidationGeneral, appErrors.WithDetails("Invalid transfer ID"))
	}

	transfer, err := h.transferSvc.GetTransfer(c.Request().Context(), userID, transferID)
	if err != nil {
		if errors.Is(err, services.ErrNWTransferNotFound) {
			return SendError(c, appErrors.NorthwindTransferNotFound)
		}
		return SendSystemError(c, err)
	}

	return c.JSON(http.StatusOK, SuccessResponse{
		Data: transfer,
	})
}

// ListTransfers lists the user's NorthWind transfers
func (h *NorthwindHandler) ListTransfers(c echo.Context) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return SendError(c, appErrors.AuthMissingToken)
	}

	offset := getIntParam(c, "offset", 0)
	limit := getIntParam(c, "limit", 20)
	if limit > 100 {
		limit = 100
	}
	status := c.QueryParam("status")
	direction := c.QueryParam("direction")
	transferType := c.QueryParam("transfer_type")

	transfers, total, err := h.transferSvc.ListTransfers(c.Request().Context(), userID, status, direction, transferType, offset, limit)
	if err != nil {
		return SendSystemError(c, err)
	}

	return c.JSON(http.StatusOK, SuccessResponse{
		Data:    transfers,
		Message: "Transfers retrieved",
		Meta: map[string]interface{}{
			"total":  total,
			"offset": offset,
			"limit":  limit,
		},
	})
}

// CancelTransfer cancels a pending transfer
func (h *NorthwindHandler) CancelTransfer(c echo.Context) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return SendError(c, appErrors.AuthMissingToken)
	}

	transferID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return SendError(c, appErrors.ValidationGeneral, appErrors.WithDetails("Invalid transfer ID"))
	}

	var req struct {
		Reason string `json:"reason" validate:"required"`
	}
	if err := c.Bind(&req); err != nil {
		return SendError(c, appErrors.ValidationGeneral, appErrors.WithDetails("Invalid request body"))
	}

	transfer, err := h.transferSvc.CancelTransfer(c.Request().Context(), userID, transferID, req.Reason)
	if err != nil {
		if errors.Is(err, services.ErrNWTransferNotFound) {
			return SendError(c, appErrors.NorthwindTransferNotFound)
		}
		return SendError(c, appErrors.NorthwindTransferCancelFail, appErrors.WithDetails(err.Error()))
	}

	return c.JSON(http.StatusOK, SuccessResponse{
		Data:    transfer,
		Message: "Transfer cancelled",
	})
}

// ReverseTransfer reverses a completed transfer
func (h *NorthwindHandler) ReverseTransfer(c echo.Context) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return SendError(c, appErrors.AuthMissingToken)
	}

	transferID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return SendError(c, appErrors.ValidationGeneral, appErrors.WithDetails("Invalid transfer ID"))
	}

	var req struct {
		Reason      string `json:"reason" validate:"required"`
		Description string `json:"description,omitempty"`
	}
	if err := c.Bind(&req); err != nil {
		return SendError(c, appErrors.ValidationGeneral, appErrors.WithDetails("Invalid request body"))
	}

	transfer, err := h.transferSvc.ReverseTransfer(c.Request().Context(), userID, transferID, req.Reason, req.Description)
	if err != nil {
		if errors.Is(err, services.ErrNWTransferNotFound) {
			return SendError(c, appErrors.NorthwindTransferNotFound)
		}
		return SendError(c, appErrors.NorthwindTransferReverseFail, appErrors.WithDetails(err.Error()))
	}

	return c.JSON(http.StatusOK, SuccessResponse{
		Data:    transfer,
		Message: "Transfer reversed",
	})
}

// --- NorthWind Health ---

// NorthwindHealth checks NorthWind API health
func (h *NorthwindHandler) NorthwindHealth(c echo.Context) error {
	health, err := h.client.Health(c.Request().Context())
	if err != nil {
		return SendError(c, appErrors.NorthwindAPIUnavailable, appErrors.WithDetails(err.Error()))
	}
	return c.JSON(http.StatusOK, SuccessResponse{
		Data:    health,
		Message: "NorthWind API is healthy",
	})
}

// NorthwindReset resets NorthWind state (development only)
func (h *NorthwindHandler) NorthwindReset(c echo.Context) error {
	if err := h.client.Reset(c.Request().Context()); err != nil {
		return SendError(c, appErrors.NorthwindAPIError, appErrors.WithDetails(err.Error()))
	}
	return c.JSON(http.StatusOK, SuccessResponse{
		Message: "NorthWind state reset",
	})
}
