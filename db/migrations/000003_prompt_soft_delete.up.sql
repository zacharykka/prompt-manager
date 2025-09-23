ALTER TABLE prompts ADD COLUMN status TEXT NOT NULL DEFAULT 'active';
ALTER TABLE prompts ADD COLUMN deleted_at TIMESTAMP;

CREATE TABLE IF NOT EXISTS prompt_audit_logs (
    id TEXT PRIMARY KEY,
    prompt_id TEXT NOT NULL,
    action TEXT NOT NULL,
    payload TEXT,
    created_by TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (prompt_id) REFERENCES prompts(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS prompt_audit_logs_prompt_idx ON prompt_audit_logs(prompt_id, created_at DESC);
