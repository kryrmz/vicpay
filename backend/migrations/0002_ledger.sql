-- 0002_ledger: append-only, double-entry ledger.
-- Ported and consolidated from KiramoPay's migration 020 with its later
-- fragments (029 escrow, 040 savings) merged into a single system-account
-- catalog. Money is stored as BIGINT minor units (never float). The journal is
-- the source of truth; user_wallet accounts also carry a cached balance for
-- hot-path reads, reconciled against the journal via wallet_journal_drift.

CREATE TABLE ledger_accounts (
    id                   uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    code                 text UNIQUE,                         -- set for system accounts, e.g. SYSTEM:FEES:USD
    type                 text NOT NULL CHECK (type IN (
                             'user_wallet',
                             'system_fees', 'system_suspense', 'system_external',
                             'system_reserve', 'system_escrow', 'system_savings')),
    user_id              uuid REFERENCES users (id) ON DELETE RESTRICT,
    currency             text NOT NULL,
    cached_balance_minor bigint,                              -- maintained for user_wallet; NULL for system accounts
    created_at           timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT chk_user_wallet_shape CHECK (
        (type = 'user_wallet' AND user_id IS NOT NULL AND cached_balance_minor IS NOT NULL)
        OR (type <> 'user_wallet' AND code IS NOT NULL))
);

-- One wallet per (user, currency).
CREATE UNIQUE INDEX uq_user_wallet ON ledger_accounts (user_id, currency)
    WHERE type = 'user_wallet';

CREATE TABLE journal_postings (
    id              uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    idempotency_key text        NOT NULL UNIQUE,
    description     text        NOT NULL,
    metadata        jsonb       NOT NULL DEFAULT '{}'::jsonb,
    created_at      timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE journal_entries (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    posting_id   uuid        NOT NULL REFERENCES journal_postings (id) ON DELETE RESTRICT,
    account_id   uuid        NOT NULL REFERENCES ledger_accounts (id) ON DELETE RESTRICT,
    direction    text        NOT NULL CHECK (direction IN ('debit', 'credit')),
    amount_minor bigint      NOT NULL CHECK (amount_minor > 0),
    currency     text        NOT NULL,
    created_at   timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_entries_posting ON journal_entries (posting_id);
CREATE INDEX idx_entries_account ON journal_entries (account_id);

-- Balance invariant: within each posting, debits must equal credits per
-- currency. DEFERRABLE INITIALLY DEFERRED so the check runs at COMMIT, once all
-- of a posting's entries are inserted.
CREATE FUNCTION fn_journal_balanced() RETURNS trigger AS $$
DECLARE
    unbalanced int;
BEGIN
    SELECT count(*) INTO unbalanced FROM (
        SELECT currency,
               sum(CASE WHEN direction = 'debit' THEN amount_minor ELSE -amount_minor END) AS net
        FROM journal_entries
        WHERE posting_id = NEW.posting_id
        GROUP BY currency
    ) t WHERE net <> 0;
    IF unbalanced > 0 THEN
        RAISE EXCEPTION 'journal posting % is not balanced', NEW.posting_id
            USING ERRCODE = 'check_violation';
    END IF;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE CONSTRAINT TRIGGER trg_journal_balanced
    AFTER INSERT ON journal_entries
    DEFERRABLE INITIALLY DEFERRED
    FOR EACH ROW EXECUTE FUNCTION fn_journal_balanced();

-- Append-only: any UPDATE or DELETE on the journal is rejected.
CREATE FUNCTION fn_journal_immutable() RETURNS trigger AS $$
BEGIN
    RAISE EXCEPTION 'journal is append-only: % on % is forbidden', TG_OP, TG_TABLE_NAME
        USING ERRCODE = 'restrict_violation';
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_postings_immutable BEFORE UPDATE OR DELETE ON journal_postings
    FOR EACH ROW EXECUTE FUNCTION fn_journal_immutable();
CREATE TRIGGER trg_entries_immutable BEFORE UPDATE OR DELETE ON journal_entries
    FOR EACH ROW EXECUTE FUNCTION fn_journal_immutable();

-- Derived balance straight from the journal: the true source of truth.
CREATE VIEW ledger_account_balances AS
SELECT a.id AS account_id,
       a.currency,
       COALESCE(sum(CASE WHEN e.direction = 'credit' THEN e.amount_minor ELSE -e.amount_minor END), 0) AS balance_minor
FROM ledger_accounts a
LEFT JOIN journal_entries e ON e.account_id = a.id
GROUP BY a.id, a.currency;

-- Reconciliation view: any row here means a user wallet's cache disagrees with
-- the journal. Under a correct engine this view is always empty.
CREATE VIEW wallet_journal_drift AS
SELECT a.id AS account_id,
       a.cached_balance_minor,
       b.balance_minor AS journal_balance_minor
FROM ledger_accounts a
JOIN ledger_account_balances b ON b.account_id = a.id
WHERE a.type = 'user_wallet'
  AND a.cached_balance_minor IS DISTINCT FROM b.balance_minor;

-- Consolidated system-account catalog (one migration, all currencies).
INSERT INTO ledger_accounts (code, type, currency) VALUES
    ('SYSTEM:FEES:USD',     'system_fees',     'USD'),
    ('SYSTEM:FEES:CRC',     'system_fees',     'CRC'),
    ('SYSTEM:SUSPENSE:USD', 'system_suspense', 'USD'),
    ('SYSTEM:SUSPENSE:CRC', 'system_suspense', 'CRC'),
    ('SYSTEM:EXTERNAL:USD', 'system_external', 'USD'),
    ('SYSTEM:EXTERNAL:CRC', 'system_external', 'CRC'),
    ('SYSTEM:RESERVE:USD',  'system_reserve',  'USD'),
    ('SYSTEM:RESERVE:CRC',  'system_reserve',  'CRC'),
    ('SYSTEM:ESCROW:USD',   'system_escrow',   'USD'),
    ('SYSTEM:ESCROW:CRC',   'system_escrow',   'CRC'),
    ('SYSTEM:SAVINGS:USD',  'system_savings',  'USD'),
    ('SYSTEM:SAVINGS:CRC',  'system_savings',  'CRC');
