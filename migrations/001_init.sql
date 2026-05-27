CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE IF NOT EXISTS subscriptions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    service_name TEXT NOT NULL,
    price INTEGER NOT NULL CHECK (price > 0),
    user_id UUID NOT NULL,
    start_month DATE NOT NULL,
    end_month DATE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (end_month IS NULL OR end_month >= start_month)
);

CREATE INDEX IF NOT EXISTS subscriptions_user_id_idx
    ON subscriptions (user_id);

CREATE INDEX IF NOT EXISTS subscriptions_service_name_idx
    ON subscriptions (service_name);

CREATE INDEX IF NOT EXISTS subscriptions_period_idx
    ON subscriptions (start_month, end_month);