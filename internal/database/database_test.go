package database

import (
	"testing"
	"time"

	"github.com/array/banking-api/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestDB_AutoMigrate(t *testing.T) {
	db := SetupTestDB(t)
	defer CleanupTestDB(t, db)
	err := db.AutoMigrate()
	require.NoError(t, err)
}

func TestDB_CreateIndexes(t *testing.T) {
	db := SetupTestDB(t)
	defer CleanupTestDB(t, db)
	// CreateIndexes may log errors for sqlite-unsupported syntax but returns nil
	err := db.CreateIndexes()
	assert.NoError(t, err)
}

func TestDB_HealthCheck(t *testing.T) {
	db := SetupTestDB(t)
	defer CleanupTestDB(t, db)
	err := db.HealthCheck()
	require.NoError(t, err)
}

func TestDB_Transaction(t *testing.T) {
	db := SetupTestDB(t)
	defer CleanupTestDB(t, db)
	err := db.Transaction(func(tx *gorm.DB) error {
		u := &models.User{
			Email:        "tx@example.com",
			PasswordHash: "hash",
			FirstName:    "Tx",
			LastName:     "User",
			Role:         models.RoleCustomer,
		}
		return tx.Create(u).Error
	})
	require.NoError(t, err)
	var count int64
	require.NoError(t, db.Model(&models.User{}).Where("email = ?", "tx@example.com").Count(&count).Error)
	assert.Equal(t, int64(1), count)
}

func TestDB_CleanupExpiredTokens(t *testing.T) {
	db := SetupTestDB(t)
	defer CleanupTestDB(t, db)

	user := CreateTestUser(t, db, "cleanup@example.com")
	past := time.Now().Add(-1 * time.Hour)

	rt := &models.RefreshToken{
		UserID:    user.ID,
		TokenHash: "expired_hash",
		ExpiresAt: past,
	}
	require.NoError(t, db.Create(rt).Error)

	bt := &models.BlacklistedToken{
		JTI:       uuid.New().String(),
		ExpiresAt: past,
	}
	require.NoError(t, db.Create(bt).Error)

	err := db.CleanupExpiredTokens()
	require.NoError(t, err)

	var refreshCount, blacklistCount int64
	require.NoError(t, db.Model(&models.RefreshToken{}).Where("token_hash = ?", "expired_hash").Count(&refreshCount).Error)
	require.NoError(t, db.Model(&models.BlacklistedToken{}).Count(&blacklistCount).Error)
	assert.Equal(t, int64(0), refreshCount)
	assert.Equal(t, int64(0), blacklistCount)
}
