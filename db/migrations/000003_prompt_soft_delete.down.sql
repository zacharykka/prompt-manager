DROP INDEX IF EXISTS prompt_audit_logs_prompt_idx;
DROP TABLE IF EXISTS prompt_audit_logs;
ALTER TABLE prompts DROP COLUMN deleted_at;
ALTER TABLE prompts DROP COLUMN status;
