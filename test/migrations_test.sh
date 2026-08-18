#!/bin/sh

set -eu

psql() {
	docker compose exec -T postgres psql \
		-v ON_ERROR_STOP=1 \
		-U order_user \
		-d order_db \
		"$@"
}

make migrate-up

psql <<'SQL'
DO $$
DECLARE
    actual_columns TEXT;
BEGIN
    SELECT string_agg(
        column_name || ':' || data_type || ':' || is_nullable,
        ',' ORDER BY ordinal_position
    )
    INTO actual_columns
    FROM information_schema.columns
    WHERE table_schema = 'public' AND table_name = 'orders';

    IF actual_columns <> 'id:bigint:NO,guid:uuid:NO,user_guid:uuid:YES,total_price:bigint:NO,currency:character varying:NO,status:character varying:NO,created_at:timestamp with time zone:NO,updated_at:timestamp with time zone:NO' THEN
        RAISE EXCEPTION 'unexpected orders columns: %', actual_columns;
    END IF;

    SELECT string_agg(
        column_name || ':' || data_type || ':' || is_nullable,
        ',' ORDER BY ordinal_position
    )
    INTO actual_columns
    FROM information_schema.columns
    WHERE table_schema = 'public' AND table_name = 'order_items';

    IF actual_columns <> 'id:bigint:NO,guid:uuid:NO,order_guid:uuid:NO,product_guid:uuid:NO,quantity:integer:NO,unit_price:bigint:NO,created_at:timestamp with time zone:NO,updated_at:timestamp with time zone:NO' THEN
        RAISE EXCEPTION 'unexpected order_items columns: %', actual_columns;
    END IF;

    IF EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_schema = 'public'
          AND table_name IN ('orders', 'order_items')
          AND column_name = 'id'
          AND column_default NOT LIKE 'nextval(%'
    ) THEN
        RAISE EXCEPTION 'BIGSERIAL id default is missing';
    END IF;

    IF EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_schema = 'public'
          AND table_name IN ('orders', 'order_items')
          AND column_name IN ('created_at', 'updated_at')
          AND column_default <> 'now()'
    ) THEN
        RAISE EXCEPTION 'timestamp default is not NOW()';
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = 'public'
          AND table_name = 'orders'
          AND column_name = 'currency'
          AND character_maximum_length = 3
    ) OR NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = 'public'
          AND table_name = 'orders'
          AND column_name = 'status'
          AND character_maximum_length = 32
    ) THEN
        RAISE EXCEPTION 'orders VARCHAR limits are incorrect';
    END IF;

    IF (
        SELECT COUNT(*)
        FROM pg_constraint AS constraint_definition
        JOIN pg_class AS table_definition
          ON table_definition.oid = constraint_definition.conrelid
        JOIN pg_namespace AS schema_definition
          ON schema_definition.oid = table_definition.relnamespace
        WHERE schema_definition.nspname = 'public'
          AND table_definition.relname IN ('orders', 'order_items')
          AND (
              (
                  constraint_definition.contype = 'p'
                  AND pg_get_constraintdef(constraint_definition.oid) = 'PRIMARY KEY (guid)'
              ) OR (
                  constraint_definition.contype = 'u'
                  AND pg_get_constraintdef(constraint_definition.oid) = 'UNIQUE (id)'
              )
          )
    ) <> 4 THEN
        RAISE EXCEPTION 'guid primary keys or id unique constraints are missing';
    END IF;

    IF NOT EXISTS (
        SELECT 1
        FROM pg_indexes
        WHERE schemaname = 'public'
          AND tablename = 'order_items'
          AND indexname = 'idx_order_items_order_guid'
    ) THEN
        RAISE EXCEPTION 'idx_order_items_order_guid is missing';
    END IF;

    IF NOT EXISTS (
        SELECT 1
        FROM information_schema.referential_constraints AS rc
        JOIN information_schema.table_constraints AS tc
          ON tc.constraint_schema = rc.constraint_schema
         AND tc.constraint_name = rc.constraint_name
        JOIN information_schema.constraint_column_usage AS ccu
          ON ccu.constraint_schema = rc.unique_constraint_schema
         AND ccu.constraint_name = rc.unique_constraint_name
        WHERE rc.constraint_schema = 'public'
          AND tc.table_name = 'order_items'
          AND ccu.table_name = 'orders'
          AND rc.delete_rule = 'CASCADE'
    ) THEN
        RAISE EXCEPTION 'order_items -> orders ON DELETE CASCADE FK is missing';
    END IF;
END
$$;

INSERT INTO orders (guid, total_price, currency, status)
VALUES ('00000000-0000-0000-0000-000000003002', 12000, 'RUB', 'pending');

INSERT INTO order_items (guid, order_guid, product_guid, quantity, unit_price)
VALUES (
    '00000000-0000-0000-0001-000000003002',
    '00000000-0000-0000-0000-000000003002',
    '00000000-0000-0000-0002-000000003002',
    2,
    6000
);

DELETE FROM orders
WHERE guid = '00000000-0000-0000-0000-000000003002';

DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM order_items
        WHERE guid = '00000000-0000-0000-0001-000000003002'
    ) THEN
        RAISE EXCEPTION 'ON DELETE CASCADE did not remove order item';
    END IF;
END
$$;
SQL

make migrate-down

if [ "$(psql -Atc "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = 'public' AND table_name IN ('orders', 'order_items')")" -ne 0 ]; then
	echo "orders or order_items still exists after migrate-down" >&2
	exit 1
fi

make migrate-up
make migrate-up
