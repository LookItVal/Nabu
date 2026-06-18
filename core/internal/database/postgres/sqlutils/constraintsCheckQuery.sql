SELECT
    EXISTS (
        SELECT 1
        FROM information_schema.table_constraints tc
        JOIN information_schema.key_column_usage kcu
          ON tc.constraint_name = kcu.constraint_name
         AND tc.table_schema = kcu.table_schema
        WHERE tc.table_schema = current_schema()
          AND tc.table_name = 'schema_migrations'
          AND tc.constraint_type = 'PRIMARY KEY'
          AND kcu.column_name = 'id'
    ) AS has_pk,
    EXISTS (
        SELECT 1
        FROM information_schema.table_constraints tc
        JOIN information_schema.key_column_usage kcu
          ON tc.constraint_name = kcu.constraint_name
         AND tc.table_schema = kcu.table_schema
        WHERE tc.table_schema = current_schema()
          AND tc.table_name = 'schema_migrations'
          AND tc.constraint_type = 'UNIQUE'
          AND kcu.column_name = 'name'
    ) AS has_unique_name;
