-- Create regulator_notifications table for tracking webhook delivery to regulator
CREATE TABLE IF NOT EXISTS regulator_notifications (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    transfer_id UUID NOT NULL REFERENCES northwind_transfers(id) ON DELETE CASCADE,
    terminal_status TEXT NOT NULL CHECK (terminal_status IN ('COMPLETED', 'FAILED')),
    delivered BOOLEAN NOT NULL DEFAULT false,
    attempt_count INT NOT NULL DEFAULT 0,
    first_attempt_at TIMESTAMP NULL,
    last_attempt_at TIMESTAMP NULL,
    next_attempt_at TIMESTAMP NULL,
    last_http_status INT NULL,
    last_error TEXT NULL,
    payload JSONB NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Only one notification per transfer + terminal status
CREATE UNIQUE INDEX idx_reg_notif_transfer_status ON regulator_notifications(transfer_id, terminal_status);
CREATE INDEX idx_reg_notif_pending ON regulator_notifications(delivered, next_attempt_at) WHERE delivered = false;

-- Trigger to update updated_at
CREATE TRIGGER update_regulator_notifications_updated_at BEFORE UPDATE ON regulator_notifications
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

COMMENT ON TABLE regulator_notifications IS 'Regulator webhook notifications for terminal transfer states with retry tracking';
