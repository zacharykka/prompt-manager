CREATE TABLE IF NOT EXISTS users (
    id TEXT PRIMARY KEY,
    email TEXT NOT NULL UNIQUE,
    hashed_password TEXT NOT NULL,
    role TEXT NOT NULL DEFAULT 'admin',
    status TEXT NOT NULL DEFAULT 'active',
    last_login_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS prompts (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    description TEXT,
    tags TEXT,
    active_version_id TEXT,
    created_by TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS prompt_versions (
    id TEXT PRIMARY KEY,
    prompt_id TEXT NOT NULL,
    version_number INTEGER NOT NULL,
    body TEXT NOT NULL,
    variables_schema TEXT,
    status TEXT NOT NULL DEFAULT 'draft',
    metadata TEXT,
    created_by TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (prompt_id) REFERENCES prompts(id) ON DELETE CASCADE
);

CREATE UNIQUE INDEX IF NOT EXISTS prompt_versions_unique_version ON prompt_versions(prompt_id, version_number);
CREATE INDEX IF NOT EXISTS prompt_versions_prompt_idx ON prompt_versions(prompt_id, created_at);

CREATE TABLE IF NOT EXISTS prompt_execution_logs (
    id TEXT PRIMARY KEY,
    prompt_id TEXT NOT NULL,
    prompt_version_id TEXT NOT NULL,
    user_id TEXT,
    status TEXT NOT NULL,
    duration_ms INTEGER,
    request_payload TEXT,
    response_metadata TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (prompt_id) REFERENCES prompts(id) ON DELETE CASCADE,
    FOREIGN KEY (prompt_version_id) REFERENCES prompt_versions(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS prompt_execution_logs_lookup_idx ON prompt_execution_logs(prompt_id, created_at DESC);
