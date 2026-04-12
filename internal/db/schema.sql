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
    name       TEXT    NOT NULL,
    created_at INTEGER NOT NULL DEFAULT 0
);
