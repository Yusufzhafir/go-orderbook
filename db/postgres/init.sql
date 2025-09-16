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
    ledger_id     BIGINT       REFERENCES ticker(id),
    ledger_tb_id  BIGINT       NOT NULL,
    tb_account_id NUMERIC(38,0) NOT NULL,
    is_escrow     BOOLEAN      DEFAULT FALSE,
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, ledger_id)
);

CREATE TABLE orders (
    id          BIGINT      PRIMARY KEY,         
    user_id     BIGINT      NOT NULL,            
    ticker_id   BIGINT      NOT NULL,            
    side        SMALLINT    NOT NULL,            
    ticker_ledger_id BIGINT NOT NULL,            
    type        SMALLINT    NOT NULL,            
    quantity    BIGINT      NOT NULL,            
    price       BIGINT      NOT NULL,            
    is_active   BOOLEAN     NOT NULL DEFAULT TRUE, 
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    closed_at   TIMESTAMPTZ             DEFAULT NULL
);

CREATE TABLE trades (
    id                BIGSERIAL PRIMARY KEY,
    ticker_id         BIGINT    NOT NULL,      
    order_taker_id    BIGINT    NOT NULL,      
    order_maker_id    BIGINT    NOT NULL,      
    ledger_transfer_id NUMERIC(38,0) NOT NULL, 
    user_ledger_id    BIGINT    NOT NULL,      
    ticker_ledger_id  BIGINT    NOT NULL,      
    type    SMALLINT  NOT NULL,               
    quantity BIGINT   NOT NULL,
    price    BIGINT   NOT NULL,
    traded_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
