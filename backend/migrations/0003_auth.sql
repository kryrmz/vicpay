-- 0003_auth: session and verification tables.
-- refresh_tokens form a rotation family (family_id) with a parent chain so the
-- server can detect reuse of a rotated token and revoke the whole family.
-- password_reset_tokens and otp_challenges are single-use and time-boxed; codes
-- and tokens are stored only as hashes, never in plaintext.

CREATE TABLE refresh_tokens (
    jti           text PRIMARY KEY,             -- the token id (also embedded in the JWT)
    user_id       uuid        NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    family_id     text        NOT NULL,
    parent_jti    text,
    token_hash    text        NOT NULL,          -- sha256 of the issued token
    issued_at     timestamptz NOT NULL DEFAULT now(),
    family_origin timestamptz NOT NULL DEFAULT now(),
    expires_at    timestamptz NOT NULL,
    used_at       timestamptz,                   -- set when rotated; a second use is reuse
    revoked_at    timestamptz
);

CREATE INDEX idx_refresh_user ON refresh_tokens (user_id);
CREATE INDEX idx_refresh_family ON refresh_tokens (family_id);

CREATE TABLE password_reset_tokens (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    uuid        NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    token_hash text        NOT NULL,
    expires_at timestamptz NOT NULL,
    consumed_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_reset_user ON password_reset_tokens (user_id);

CREATE TABLE otp_challenges (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    recipient    text        NOT NULL,            -- E.164 phone
    purpose      text        NOT NULL,
    code_hash    text        NOT NULL,
    attempts     int         NOT NULL DEFAULT 0,
    max_attempts int         NOT NULL,
    expires_at   timestamptz NOT NULL,
    consumed_at  timestamptz,
    created_at   timestamptz NOT NULL DEFAULT now()
);

-- Lookups fetch the most recent active challenge for a (recipient, purpose).
CREATE INDEX idx_otp_active ON otp_challenges (recipient, purpose, created_at DESC)
    WHERE consumed_at IS NULL;
