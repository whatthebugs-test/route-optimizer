-- Applied to the dispatch database alongside dispatch-service's Flyway migrations.
CREATE TABLE IF NOT EXISTS terminals (
    code  VARCHAR(3) PRIMARY KEY,
    name  TEXT NOT NULL,
    lat   DOUBLE PRECISION NOT NULL,
    lng   DOUBLE PRECISION NOT NULL
);

CREATE TABLE IF NOT EXISTS route_plans (
    id          TEXT PRIMARY KEY,
    terminal    VARCHAR(3) NOT NULL REFERENCES terminals (code),
    day         DATE NOT NULL,
    routes      JSONB NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS ix_route_plans_terminal_day ON route_plans (terminal, day);
