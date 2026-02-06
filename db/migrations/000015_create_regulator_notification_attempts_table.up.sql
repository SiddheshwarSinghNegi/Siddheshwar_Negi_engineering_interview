-- Create regulator_notification_attempts table for hard audit proof of every delivery attempt
CREATE TABLE IF NOT EXISTS regulator_notification_attempts (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    notification_id UUID NOT NULL REFERENCES regulator_notifications(id) ON DELETE CASCADE,
    attempted_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    http_status INT NULL,
    error TEXT NULL,
    response_body TEXT NULL
);

CREATE INDEX idx_reg_notif_attempts_notification_id ON regulator_notification_attempts(notification_id);
CREATE INDEX idx_reg_notif_attempts_attempted_at ON regulator_notification_attempts(attempted_at);

COMMENT ON TABLE regulator_notification_attempts IS 'Audit trail of individual regulator webhook delivery attempts';
