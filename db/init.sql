CREATE TABLE users (
    id            SERIAL        PRIMARY KEY,
    username      VARCHAR(50)   UNIQUE NOT NULL,
    password_hash TEXT          NOT NULL,
    created_at    TIMESTAMPTZ   NOT NULL DEFAULT NOW()
);

CREATE TABLE ticker (
    id                SERIAL     PRIMARY KEY,
    ticker            VARCHAR(10) UNIQUE NOT NULL,
    tb_ledger_id      BIGINT      UNIQUE NOT NULL,
    escrow_account_id NUMERIC(38,0) UNIQUE NOT NULL,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE users_ledger (
    id            BIGSERIAL    PRIMARY KEY,
    user_id       INT          REFERENCES users(id),
    ledger_id     BIGINT          REFERENCES ticker(id),
    tb_account_id NUMERIC(38,0) NOT NULL,
    is_escrow     BOOLEAN      DEFAULT FALSE,
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, ledger_id)
);
