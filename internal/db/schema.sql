CREATE TABLE IF NOT EXISTS coins (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    api_id      TEXT    NOT NULL UNIQUE,
    name        TEXT    NOT NULL,
    ticker      TEXT    NOT NULL,
    rate        REAL    NOT NULL DEFAULT 0,
    price_change REAL   NOT NULL DEFAULT 0,
    market_rank INTEGER NOT NULL DEFAULT 0,
    updated_at  INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS portfolios (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    name       TEXT    NOT NULL UNIQUE,
    created_at INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS holdings (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    portfolio_id INTEGER NOT NULL REFERENCES portfolios(id) ON DELETE CASCADE,
    coin_id      INTEGER NOT NULL REFERENCES coins(id) ON DELETE CASCADE,
    amount       REAL    NOT NULL,
    UNIQUE(portfolio_id, coin_id)
);

CREATE TABLE IF NOT EXISTS currencies (
    code TEXT PRIMARY KEY,
    name TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS settings (
    key TEXT NOT NULL UNIQUE,
    value TEXT NOT NULL
);

INSERT OR IGNORE INTO settings (key, value) VALUES ('selected_currency', 'usd');
