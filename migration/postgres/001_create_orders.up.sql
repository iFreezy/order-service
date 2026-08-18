CREATE TABLE orders (
    id BIGSERIAL NOT NULL UNIQUE,
    guid UUID NOT NULL PRIMARY KEY,
    user_guid UUID,
    total_price BIGINT NOT NULL,
    currency VARCHAR(3) NOT NULL,
    status VARCHAR(32) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
