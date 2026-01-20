# Database Migrations

This directory contains database migration files that are automatically run when the application starts.

## How Migrations Work

- Migrations are executed in **alphabetical order** by filename
- Each migration runs **only once** (tracked in `schema_migrations` table)
- Migrations run automatically on app startup before the server starts

## Adding a New Migration

1. Create a new `.sql` file with the naming pattern: `XXX_description.sql`

   - `XXX` = sequential number (001, 002, 003, etc.)
   - `description` = brief description of what the migration does

2. Write your SQL in the file

3. Deploy your code - the migration will run automatically on startup

## Example

```sql
-- migrations/003_add_user_profiles.sql
CREATE TABLE IF NOT EXISTS user_profiles (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL,
    bio TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_user_profiles_user_id ON user_profiles(user_id);
```

## Migration Best Practices

- ✅ Use `IF NOT EXISTS` / `IF EXISTS` for safety
- ✅ Test migrations locally before deploying
- ✅ Keep migrations small and focused
- ✅ Never modify an already-deployed migration file
- ✅ Use descriptive names
- ❌ Don't delete old migration files (they're part of your schema history)

## Checking Migration Status

The `schema_migrations` table tracks which migrations have been applied:

```sql
SELECT * FROM schema_migrations ORDER BY applied_at;
```
