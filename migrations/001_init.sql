CREATE TABLE IF NOT EXISTS users (
    id          INTEGER PRIMARY KEY,
    username    TEXT    NOT NULL DEFAULT '',
    first_name  TEXT    NOT NULL DEFAULT '',
    last_name   TEXT    NOT NULL DEFAULT '',
    admin_id    INTEGER NOT NULL DEFAULT 0,
    is_blocked  INTEGER NOT NULL DEFAULT 0,
    created_at  DATETIME NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS connections (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id         INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    label           TEXT    NOT NULL,
    link            TEXT    NOT NULL,
    xui_client_id   TEXT    NOT NULL DEFAULT '',
    inbound_id      INTEGER NOT NULL DEFAULT 0,
    is_active       INTEGER NOT NULL DEFAULT 1,
    created_at      DATETIME NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_connections_user_id ON connections(user_id);

CREATE TABLE IF NOT EXISTS payments (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id      INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    period_year  INTEGER NOT NULL,
    period_month INTEGER NOT NULL,
    paid_at      DATETIME,
    confirmed_by INTEGER,
    created_at   DATETIME NOT NULL DEFAULT (datetime('now')),
    UNIQUE (user_id, period_year, period_month)
);

CREATE INDEX IF NOT EXISTS idx_payments_period ON payments(period_year, period_month);
