CREATE TABLE tickets (
    id          BIGSERIAL PRIMARY KEY,
    title       VARCHAR(200) NOT NULL,
    description TEXT,
    status      VARCHAR(20)  NOT NULL DEFAULT 'open'
                CHECK (status IN ('open', 'in_progress', 'closed')),
    priority    VARCHAR(10)  NOT NULL DEFAULT 'medium'
                CHECK (priority IN ('low', 'medium', 'high')),
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT now()
);

-- Serves the default listing and its newest-first ordering.
CREATE INDEX idx_tickets_created ON tickets (created_at DESC, id DESC);
-- Serves the status-filtered listing without a separate sort step.
CREATE INDEX idx_tickets_status_created ON tickets (status, created_at DESC, id DESC);

