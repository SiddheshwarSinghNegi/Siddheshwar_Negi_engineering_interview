package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/array/banking-api/internal/database"
	"github.com/array/banking-api/internal/integrations/northwind"
	"github.com/array/banking-api/internal/repositories"
	"github.com/array/banking-api/internal/services"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"log/slog"
)

func TestNorthwindHandler_GetBankInfo_Success(t *testing.T) {
	bankInfo := northwind.BankInfo{
		Name:          "Test Bank",
		RoutingNumber: "021000089",
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/bank" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(bankInfo)
	}))
	defer server.Close()

	client := northwind.NewClient(server.URL, "test-key")
	db := database.SetupTestDB(t)
	defer database.CleanupTestDB(t, db)
	nwExtRepo := repositories.NewNorthwindExternalAccountRepository(db.DB)
	nwTransferRepo := repositories.NewNorthwindTransferRepository(db.DB)
	accountSvc := services.NewNorthwindAccountService(client, nwExtRepo, slog.Default())
	transferSvc := services.NewNorthwindTransferService(client, nwTransferRepo, slog.Default())
	handler := NewNorthwindHandler(client, accountSvc, transferSvc)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/northwind/bank", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("user_id", uuid.New())

	err := handler.GetBankInfo(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	var body struct {
		Data    northwind.BankInfo `json:"data"`
		Message string             `json:"message"`
	}
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&body))
	assert.Equal(t, bankInfo.Name, body.Data.Name)
	assert.Equal(t, bankInfo.RoutingNumber, body.Data.RoutingNumber)
}

func TestNorthwindHandler_GetBankInfo_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"message":"internal error"}`))
	}))
	defer server.Close()

	client := northwind.NewClient(server.URL, "test-key")
	db := database.SetupTestDB(t)
	defer database.CleanupTestDB(t, db)
	nwExtRepo := repositories.NewNorthwindExternalAccountRepository(db.DB)
	nwTransferRepo := repositories.NewNorthwindTransferRepository(db.DB)
	accountSvc := services.NewNorthwindAccountService(client, nwExtRepo, slog.Default())
	transferSvc := services.NewNorthwindTransferService(client, nwTransferRepo, slog.Default())
	handler := NewNorthwindHandler(client, accountSvc, transferSvc)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/northwind/bank", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("user_id", uuid.New())

	err := handler.GetBankInfo(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadGateway, rec.Code)
}

func TestNorthwindHandler_GetDomains_Success(t *testing.T) {
	domains := []northwind.Domain{
		{Name: "ach", Description: "ACH transfers"},
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/domains" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(domains)
	}))
	defer server.Close()

	client := northwind.NewClient(server.URL, "test-key")
	db := database.SetupTestDB(t)
	defer database.CleanupTestDB(t, db)
	nwExtRepo := repositories.NewNorthwindExternalAccountRepository(db.DB)
	nwTransferRepo := repositories.NewNorthwindTransferRepository(db.DB)
	accountSvc := services.NewNorthwindAccountService(client, nwExtRepo, slog.Default())
	transferSvc := services.NewNorthwindTransferService(client, nwTransferRepo, slog.Default())
	handler := NewNorthwindHandler(client, accountSvc, transferSvc)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/northwind/domains", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("user_id", uuid.New())

	err := handler.GetDomains(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	var body struct {
		Data    []northwind.Domain `json:"data"`
		Message string             `json:"message"`
	}
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&body))
	require.Len(t, body.Data, 1)
	assert.Equal(t, "ach", body.Data[0].Name)
}

