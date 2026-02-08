package validation

import (
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewValidator(t *testing.T) {
	v := NewValidator()
	require.NotNil(t, v)
	require.NotNil(t, v.GetValidate())
}

func TestGetValidator(t *testing.T) {
	instance = nil // reset singleton for test
	v1 := GetValidator()
	v2 := GetValidator()
	require.NotNil(t, v1)
	assert.Same(t, v1, v2)
}

func TestEchoValidator(t *testing.T) {
	ev := EchoValidator()
	require.NotNil(t, ev)
	_, ok := ev.(echo.Validator)
	assert.True(t, ok)
}

func TestEchoValidator_Validate(t *testing.T) {
	ev := EchoValidator()
	type validStruct struct {
		AccountNumber string `json:"account_number" validate:"account_number"`
	}
	err := ev.Validate(&validStruct{AccountNumber: ""})
	assert.Error(t, err)
	err = ev.Validate(&validStruct{AccountNumber: "1234567890"})
	assert.NoError(t, err)
}

func TestValidateAccountNumber(t *testing.T) {
	v := NewValidator().GetValidate()
	type s struct {
		Num string `json:"num" validate:"account_number"`
	}
	tests := []struct {
		name string
		num  string
		want bool
	}{
		{"empty", "", false},
		{"too short", "123", false},
		{"9 digits", "123456789", false},
		{"10 digits", "1234567890", true},
		{"12 digits", "123456789012", true},
		{"13 digits", "1234567890123", false},
		{"letters", "1234567890a", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.Struct(&s{Num: tt.num})
			if tt.want {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}

func TestValidateTransactionAmount(t *testing.T) {
	v := NewValidator().GetValidate()
	type s struct {
		Amount float64 `json:"amount" validate:"transaction_amount"`
	}
	tests := []struct {
		name   string
		amount float64
		want   bool
	}{
		{"zero", 0, false},
		{"negative", -1, false},
		{"valid int", 100, true},
		{"one decimal", 10.5, true},
		{"two decimals", 10.99, true},
		{"three decimals", 10.999, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.Struct(&s{Amount: tt.amount})
			if tt.want {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}

func TestValidatePositiveAmount(t *testing.T) {
	v := NewValidator().GetValidate()
	type sInt struct {
		N int `json:"n" validate:"positive_amount"`
	}
	type sFloat struct {
		N float64 `json:"n" validate:"positive_amount"`
	}
	require.Error(t, v.Struct(&sInt{N: 0}))
	require.Error(t, v.Struct(&sInt{N: -1}))
	require.NoError(t, v.Struct(&sInt{N: 1}))
	require.Error(t, v.Struct(&sFloat{N: 0}))
	require.NoError(t, v.Struct(&sFloat{N: 0.01}))
}

func TestValidateCustomerID(t *testing.T) {
	v := NewValidator().GetValidate()
	type s struct {
		ID string `json:"id" validate:"customer_id"`
	}
	require.Error(t, v.Struct(&s{ID: ""}))
	require.Error(t, v.Struct(&s{ID: "not-uuid"}))
	require.NoError(t, v.Struct(&s{ID: "550e8400-e29b-41d4-a716-446655440000"}))
}

func TestValidateAccountType(t *testing.T) {
	v := NewValidator().GetValidate()
	type s struct {
		Type string `json:"type" validate:"account_type"`
	}
	for _, valid := range []string{"checking", "savings", "credit", "CHECKING"} {
		require.NoError(t, v.Struct(&s{Type: valid}))
	}
	require.Error(t, v.Struct(&s{Type: "invalid"}))
	require.Error(t, v.Struct(&s{Type: ""}))
}

func TestValidateTransactionType(t *testing.T) {
	v := NewValidator().GetValidate()
	type s struct {
		Type string `json:"type" validate:"transaction_type"`
	}
	for _, valid := range []string{"deposit", "withdrawal", "transfer", "DEPOSIT"} {
		require.NoError(t, v.Struct(&s{Type: valid}))
	}
	require.Error(t, v.Struct(&s{Type: "invalid"}))
}

func TestGetValidate(t *testing.T) {
	val := NewValidator()
	inner := val.GetValidate()
	require.NotNil(t, inner)
}
