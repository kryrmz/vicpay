-- Least-privilege application role. Run ONCE, as the database owner, AFTER the
-- migrations have created the schema. This is a privilege wall that complements
-- the append-only triggers: even if a trigger were dropped, the application role
-- still cannot UPDATE or DELETE the journal.
--
-- Usage:
--   psql "$DATABASE_DIRECT_URL" -v app_password="'CHANGE_ME'" -f deploy/roles.sql
-- (the password must be quoted inside the value, hence the inner single quotes)

\set ON_ERROR_STOP on

DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'vicpay_app') THEN
    EXECUTE format('CREATE ROLE vicpay_app LOGIN PASSWORD %L', :app_password);
  ELSE
    EXECUTE format('ALTER ROLE vicpay_app LOGIN PASSWORD %L', :app_password);
  END IF;
END $$;

-- Start from nothing, then grant exactly what the app needs.
REVOKE ALL ON ALL TABLES IN SCHEMA public FROM vicpay_app;
GRANT USAGE ON SCHEMA public TO vicpay_app;

-- Read everything (including the balance/drift views) and insert new rows
-- (users, wallets, journal postings/entries, tokens, otp challenges).
GRANT SELECT, INSERT ON ALL TABLES IN SCHEMA public TO vicpay_app;

-- Update only the tables whose rows legitimately change over their lifetime.
GRANT UPDATE ON
    users,
    ledger_accounts,
    refresh_tokens,
    password_reset_tokens,
    otp_challenges
TO vicpay_app;

-- The journal is append-only for the app: no UPDATE, no DELETE, ever.
REVOKE UPDATE, DELETE ON journal_postings, journal_entries FROM vicpay_app;

-- Migration bookkeeping is owner-managed; the app may only read it.
REVOKE ALL ON schema_migrations FROM vicpay_app;
GRANT SELECT ON schema_migrations TO vicpay_app;

-- Re-run this script after adding new migrations so fresh tables inherit the
-- same grants (or manage this with ALTER DEFAULT PRIVILEGES for the owner role).
