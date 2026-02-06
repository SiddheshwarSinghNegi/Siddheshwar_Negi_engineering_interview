-- Create northwind_external_accounts table for storing validated external accounts
CREATE TABLE IF NOT EXISTS northwind_external_accounts (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NULL REFERENCES users(id) ON DELETE SET NULL,
    account_holder_name TEXT NOT NULL,
    account_number TEXT NOT NULL,
    routing_number TEXT NOT NULL,
    institution_name TEXT NULL,
    validated BOOLEAN NOT NULL DEFAULT false,
    validation_time TIMESTAMP NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Unique constraint: one registration per account_number + routing_number per user
CREATE UNIQUE INDEX idx_nw_ext_accounts_unique
    ON northwind_external_accounts(user_id, account_number, routing_number);

CREATE INDEX idx_nw_ext_accounts_user_id ON northwind_external_accounts(user_id);
CREATE INDEX idx_nw_ext_accounts_account_number ON northwind_external_accounts(account_number);

COMMENT ON TABLE northwind_external_accounts IS 'Registered and validated external bank accounts for NorthWind transfers';
