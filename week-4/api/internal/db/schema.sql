CREATE TABLE IF NOT EXISTS users (
    id            BIGSERIAL PRIMARY KEY,
    email         TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    first_name    TEXT NOT NULL DEFAULT '',
    last_name     TEXT NOT NULL DEFAULT '',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- CREATE TABLE IF NOT EXISTS is a no-op on a table that already exists, so
-- columns added after the table's first deploy need an explicit ALTER here
-- too, or older rows/deployments would break on the new queries.
ALTER TABLE users ADD COLUMN IF NOT EXISTS first_name TEXT NOT NULL DEFAULT '';
ALTER TABLE users ADD COLUMN IF NOT EXISTS last_name TEXT NOT NULL DEFAULT '';

CREATE TABLE IF NOT EXISTS instances (
    name       TEXT PRIMARY KEY,
    owner_id   BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS instances_owner_id_idx ON instances(owner_id);

-- instance_name is deliberately a plain column, not a foreign key into
-- instances(name). instances rows are deleted when an instance is deleted
-- (see DeleteInstanceRecord) — an FK with cascade semantics would destroy the
-- "instance.deleted" audit row at the exact moment it becomes the one record
-- that matters most. Audit history must outlive the resource it describes.
CREATE TABLE IF NOT EXISTS audit_logs (
    id            BIGSERIAL PRIMARY KEY,
    user_id       BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    action        TEXT NOT NULL,
    instance_name TEXT,
    metadata      JSONB NOT NULL DEFAULT '{}',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS audit_logs_user_id_idx ON audit_logs(user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS audit_logs_instance_name_idx ON audit_logs(instance_name, created_at DESC) WHERE instance_name IS NOT NULL;
