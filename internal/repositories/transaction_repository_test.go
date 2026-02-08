package repositories

import (
	"testing"
	"time"

	"github.com/array/banking-api/internal/database"
	"github.com/array/banking-api/internal/models"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/suite"
)

func TestTransactionRepositorySuite(t *testing.T) {
	suite.Run(t, new(TransactionRepositorySuite))
}

type TransactionRepositorySuite struct {
	suite.Suite
	db       *database.DB
	repo     TransactionRepositoryInterface
	testUser *models.User
	testAcct *models.Account
}

func (s *TransactionRepositorySuite) SetupTest() {
	s.db = database.SetupTestDB(s.T())
	s.repo = NewTransactionRepository(s.db.DB)

	s.testUser = database.CreateTestUser(s.T(), s.db, "txuser@example.com")
	s.testAcct = &models.Account{
		UserID:        s.testUser.ID,
		AccountNumber: "1012345678",
		AccountType:   models.AccountTypeChecking,
		Balance:       decimal.NewFromFloat(1000),
		Status:        models.AccountStatusActive,
		Currency:      "USD",
	}
	s.NoError(s.db.Create(s.testAcct).Error)
}

func (s *TransactionRepositorySuite) TearDownTest() {
	database.CleanupTestDB(s.T(), s.db)
}

func (s *TransactionRepositorySuite) TestCreate() {
	tx := &models.Transaction{
		AccountID:       s.testAcct.ID,
		TransactionType: models.TransactionTypeCredit,
		Amount:          decimal.NewFromFloat(50),
		BalanceBefore:   decimal.NewFromFloat(1000),
		BalanceAfter:    decimal.NewFromFloat(1050),
		Description:     "deposit",
		Reference:       models.GenerateTransactionReference(),
		Status:          models.TransactionStatusCompleted,
	}
	err := s.repo.Create(tx)
	s.NoError(err)
	s.NotEqual(uuid.Nil, tx.ID)
	s.NotZero(tx.CreatedAt)
}

func (s *TransactionRepositorySuite) TestGetByID_Success() {
	tx := &models.Transaction{
		AccountID:       s.testAcct.ID,
		TransactionType: models.TransactionTypeDebit,
		Amount:          decimal.NewFromFloat(25),
		BalanceBefore:   decimal.NewFromFloat(1000),
		BalanceAfter:    decimal.NewFromFloat(975),
		Description:     "withdrawal",
		Reference:       models.GenerateTransactionReference(),
		Status:          models.TransactionStatusCompleted,
	}
	s.NoError(s.repo.Create(tx))

	found, err := s.repo.GetByID(tx.ID)
	s.NoError(err)
	s.Equal(tx.ID, found.ID)
	s.Equal(tx.AccountID, found.AccountID)
	s.Equal(tx.Amount.String(), found.Amount.String())
}

func (s *TransactionRepositorySuite) TestGetByID_NotFound() {
	_, err := s.repo.GetByID(uuid.New())
	s.Error(err)
	s.Equal(ErrTransactionNotFound, err)
}

func (s *TransactionRepositorySuite) TestGetByAccountID_Pagination() {
	for i := 0; i < 5; i++ {
		amount := float64(10 * (i + 1))
		tx := &models.Transaction{
			AccountID:       s.testAcct.ID,
			TransactionType: models.TransactionTypeCredit,
			Amount:          decimal.NewFromFloat(amount),
			BalanceBefore:   decimal.Zero,
			BalanceAfter:    decimal.NewFromFloat(amount), // credit: 0 + amount
			Description:     "tx",
			Reference:       models.GenerateTransactionReference(),
			Status:          models.TransactionStatusCompleted,
		}
		s.NoError(s.repo.Create(tx))
	}

	list, total, err := s.repo.GetByAccountID(s.testAcct.ID, 0, 2)
	s.NoError(err)
	s.Equal(int64(5), total)
	s.Len(list, 2)

	list2, total2, err := s.repo.GetByAccountID(s.testAcct.ID, 2, 2)
	s.NoError(err)
	s.Equal(int64(5), total2)
	s.Len(list2, 2)
}

func (s *TransactionRepositorySuite) TestGetByReference() {
	ref := models.GenerateTransactionReference()
	tx := &models.Transaction{
		AccountID:       s.testAcct.ID,
		TransactionType: models.TransactionTypeCredit,
		Amount:          decimal.NewFromFloat(100),
		BalanceBefore:   decimal.Zero,
		BalanceAfter:    decimal.NewFromFloat(100),
		Description:     "ref test",
		Reference:       ref,
		Status:          models.TransactionStatusCompleted,
	}
	s.NoError(s.repo.Create(tx))

	found, err := s.repo.GetByReference(ref)
	s.NoError(err)
	s.Equal(ref, found.Reference)
	s.Equal(tx.ID, found.ID)
}

func (s *TransactionRepositorySuite) TestGetByReference_NotFound() {
	_, err := s.repo.GetByReference("nonexistent-ref")
	s.Error(err)
	s.Equal(ErrTransactionNotFound, err)
}

func (s *TransactionRepositorySuite) TestUpdateWithOptimisticLock() {
	tx := &models.Transaction{
		AccountID:       s.testAcct.ID,
		TransactionType: models.TransactionTypeDebit,
		Amount:          decimal.NewFromFloat(50),
		BalanceBefore:   decimal.NewFromFloat(1000),
		BalanceAfter:    decimal.NewFromFloat(950),
		Description:     "pending",
		Reference:       models.GenerateTransactionReference(),
		Status:          models.TransactionStatusPending,
		Version:         1,
	}
	s.NoError(s.repo.Create(tx))

	tx.Status = models.TransactionStatusCompleted
	now := time.Now()
	tx.ProcessedAt = &now
	// Keep AccountID set so BeforeUpdate validation sees it when repo calls Updates(tx)
	err := s.repo.UpdateWithOptimisticLock(tx, 1)
	s.NoError(err)

	updated, err := s.repo.GetByID(tx.ID)
	s.NoError(err)
	s.Equal(models.TransactionStatusCompleted, updated.Status)
}

func (s *TransactionRepositorySuite) TestUpdateWithOptimisticLock_Conflict() {
	tx := &models.Transaction{
		AccountID:       s.testAcct.ID,
		TransactionType: models.TransactionTypeCredit,
		Amount:          decimal.NewFromFloat(50),
		BalanceBefore:   decimal.Zero,
		BalanceAfter:    decimal.NewFromFloat(50),
		Description:     "v1",
		Reference:       models.GenerateTransactionReference(),
		Status:          models.TransactionStatusCompleted,
		Version:         1,
	}
	s.NoError(s.repo.Create(tx))

	tx.Version = 2
	err := s.repo.UpdateWithOptimisticLock(tx, 99) // wrong expected version; AccountID already set from create
	s.Error(err)
	s.Equal(models.ErrOptimisticLockConflict, err)
}

func (s *TransactionRepositorySuite) TestGetPendingTransactions() {
	tx1 := &models.Transaction{
		AccountID:       s.testAcct.ID,
		TransactionType: models.TransactionTypeDebit,
		Amount:          decimal.NewFromFloat(10),
		BalanceBefore:   decimal.NewFromFloat(1000),
		BalanceAfter:    decimal.NewFromFloat(990),
		Description:     "pending",
		Reference:       models.GenerateTransactionReference(),
		Status:          models.TransactionStatusPending,
	}
	s.NoError(s.repo.Create(tx1))

	tx2 := &models.Transaction{
		AccountID:       s.testAcct.ID,
		TransactionType: models.TransactionTypeCredit,
		Amount:          decimal.NewFromFloat(20),
		BalanceBefore:   decimal.Zero,
		BalanceAfter:    decimal.NewFromFloat(20),
		Description:     "completed",
		Reference:       models.GenerateTransactionReference(),
		Status:          models.TransactionStatusCompleted,
	}
	s.NoError(s.repo.Create(tx2))

	pending, err := s.repo.GetPendingTransactions(0, 10)
	s.NoError(err)
	s.GreaterOrEqual(len(pending), 1)
	for _, p := range pending {
		s.Equal(models.TransactionStatusPending, p.Status)
	}
}

func (s *TransactionRepositorySuite) TestGetRecentByAccountID() {
	for i := 0; i < 3; i++ {
		amount := float64(i + 1)
		tx := &models.Transaction{
			AccountID:       s.testAcct.ID,
			TransactionType: models.TransactionTypeCredit,
			Amount:          decimal.NewFromFloat(amount),
			BalanceBefore:   decimal.Zero,
			BalanceAfter:    decimal.NewFromFloat(amount), // credit: 0 + amount
			Description:     "recent",
			Reference:       models.GenerateTransactionReference(),
			Status:          models.TransactionStatusCompleted,
		}
		s.NoError(s.repo.Create(tx))
	}

	recent, err := s.repo.GetRecentByAccountID(s.testAcct.ID, 2)
	s.NoError(err)
	s.LessOrEqual(len(recent), 2)
}
