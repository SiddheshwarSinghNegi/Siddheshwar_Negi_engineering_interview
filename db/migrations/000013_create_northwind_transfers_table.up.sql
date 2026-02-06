-- Create northwind_transfers table for tracking external transfers via NorthWind
CREATE TABLE IF NOT EXISTS northwind_transfers (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NULL REFERENCES users(id) ON DELETE SET NULL,
    northwind_transfer_id UUID NOT NULL,
    direction TEXT NOT NULL CHECK (direction IN ('INBOUND', 'OUTBOUND')),
    transfer_type TEXT NOT NULL,
    amount NUMERIC(15,2) NOT NULL,
    currency TEXT NOT NULL DEFAULT 'USD',
    description TEXT NULL,
    reference_number TEXT NOT NULL,
    scheduled_date TIMESTAMP NULL,
    source_account_number TEXT NOT NULL,
    source_routing_number TEXT NULL,
    source_account_holder_name TEXT NULL,
    destination_account_number TEXT NOT NULL,
    destination_routing_number TEXT NULL,
    destination_account_holder_name TEXT NULL,
    status TEXT NOT NULL DEFAULT 'PENDING',
    error_code TEXT NULL,
    error_message TEXT NULL,
    initiated_date TIMESTAMP NULL,
    processing_date TIMESTAMP NULL,
    expected_completion_date TIMESTAMP NULL,
    completed_date TIMESTAMP NULL,
    fee NUMERIC(15,4) NULL,
    exchange_rate NUMERIC(15,6) NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX idx_nw_transfers_nw_id ON northwind_transfers(northwind_transfer_id);
CREATE INDEX idx_nw_transfers_status ON northwind_transfers(status);
CREATE INDEX idx_nw_transfers_user_id ON northwind_transfers(user_id);
CREATE INDEX idx_nw_transfers_created_at ON northwind_transfers(created_at);

-- Trigger to update updated_at
CREATE TRIGGER update_northwind_transfers_updated_at BEFORE UPDATE ON northwind_transfers
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

COMMENT ON TABLE northwind_transfers IS 'External transfers initiated and tracked via NorthWind Bank API';
