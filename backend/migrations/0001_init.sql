-- 0001_init: base extensions and the users table.
-- PII (phone, email) is encrypted at the application layer (AES-256-GCM) and
-- stored as ciphertext; a keyed HMAC "blind index" column enables equality
-- lookups without decrypting. This keeps PII confidentiality independent of any
-- Postgres session GUC, so it is safe under a transactional connection pooler
-- (PgBouncer) -- unlike KiramoPay's current_setting()-based approach.

CREATE EXTENSION IF NOT EXISTS pgcrypto WITH SCHEMA public;

CREATE TABLE users (
    id             uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    phone_hmac     bytea       NOT NULL UNIQUE,
    phone_cipher   bytea       NOT NULL,
    email_hmac     bytea       UNIQUE,
    email_cipher   bytea,
    password_hash  text        NOT NULL,
    phone_verified boolean     NOT NULL DEFAULT false,
    kyc_level      smallint    NOT NULL DEFAULT 0 CHECK (kyc_level BETWEEN 0 AND 2),
    status         text        NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'suspended', 'closed')),
    created_at     timestamptz NOT NULL DEFAULT now(),
    updated_at     timestamptz NOT NULL DEFAULT now()
);
