-- +goose Up
CREATE TABLE profiles (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    auth_method TEXT NOT NULL CHECK (auth_method IN ('gh-cli', 'keyring')),
    keyring_ref TEXT,
    github_username TEXT,
    is_active INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL,
    last_validated_at TEXT
);

CREATE UNIQUE INDEX idx_profiles_active ON profiles(is_active) WHERE is_active = 1;

INSERT INTO profiles (id, name, auth_method, keyring_ref, github_username, is_active, created_at)
SELECT
    lower(hex(randomblob(16))),
    'gh-cli',
    'gh-cli',
    NULL,
    NULL,
    1,
    strftime('%Y-%m-%dT%H:%M:%fZ', 'now')
WHERE NOT EXISTS (SELECT 1 FROM profiles);

-- +goose Down
DROP INDEX IF EXISTS idx_profiles_active;
DROP TABLE IF EXISTS profiles;
