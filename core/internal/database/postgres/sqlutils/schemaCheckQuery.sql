		SELECT
			COUNT(*) = 3 AS has_expected_columns,
			BOOL_OR(column_name = 'id' AND data_type = 'integer' AND is_nullable = 'NO' AND column_default LIKE 'nextval(%') AS has_id,
			BOOL_OR(column_name = 'name' AND data_type = 'text' AND is_nullable = 'NO') AS has_name,
			BOOL_OR(column_name = 'applied_at' AND data_type = 'timestamp with time zone' AND is_nullable = 'NO' AND column_default IS NOT NULL AND column_default LIKE '%now()%') AS has_applied_at
		FROM information_schema.columns
		WHERE table_schema = current_schema()
		  AND table_name = 'schema_migrations'