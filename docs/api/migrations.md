# Migrations

Folder-per-migration with `metadata.json`. Embedded in the server binary. Each migration declares whether it's safe to auto-apply.

## Directory layout

```
internal/platform/postgres/migrations/
├── 000001_create_users/
│   ├── metadata.json
│   ├── up.sql
│   └── down.sql
├── 000002_create_invites/
│   ├── metadata.json
│   ├── up.sql
│   └── down.sql
└── 000003_create_settings/
    ├── metadata.json
    ├── up.sql
    └── down.sql
```

All migrations live in a single flat directory. The `safe` field in `metadata.json` controls whether it auto-runs.

## metadata.json

```json
{
  "name": "create_users",
  "author": "artumont",
  "description": "Create the users table with UUID v7 primary key",
  "safe": true
}
```

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | Migration identifier (matches folder suffix) |
| `author` | string | Git `user.name` at creation time (auto-populated) |
| `description` | string | What the migration does |
| `safe` | bool | `true` = auto-runs on startup. `false` = needs `MIGRATE_UNSAFE=true` |

## Safety model

| `safe` | Behavior |
|--------|----------|
| `true` | Runs on every server startup automatically |
| `false` | Skipped on startup. Requires `MIGRATE_UNSAFE=true` to apply |

Unsafe migrations are logged when skipped:
```
  skipped drop_legacy (unsafe; set MIGRATE_UNSAFE=true to apply)
```

Rule: **if you'd run it in production at 3 AM with no one watching, `"safe": true`. Otherwise, `"safe": false`**.

## Commands

```bash
# Filesystem-only (no DB connection needed)
make migration-list                                    # List all migrations
migrate create <name> <desc>                          # Create safe migration
migrate create <name> <desc> --unsafe                  # Create unsafe migration

# Database commands (require DB_DSN in .env or environment)
make migrate                                           # Run pending migrations
make migrate-down                                      # Rollback last applied
make migrate-status                                    # Show applied migrations
MIGRATE_UNSAFE=true make migrate                      # Also apply unsafe migrations
```

**Setup:** Copy `.env.example` to `.env` and fill in your values. DB commands read `DB_DSN` from env or `.env`.

## Creating a migration

```bash
make migrate-create name=add_media desc="Create the media table for file uploads"
```

Creates:

```
000004_add_media/
├── metadata.json   # name, author (from git), description, safe: true
├── up.sql          # Your schema change
└── down.sql        # How to undo it
```

Fill in both SQL files. The `.down.sql` is your rollback safety net.

### Unsafe migration

```bash
make migrate-create-unsafe name=drop_legacy desc="Drop legacy invites table"
```

Same structure but `safe: false`. Won't run on startup unless `MIGRATE_UNSAFE=true`.

## Viewing migrations

```bash
make migration-list
```

Output:

```
#      NAME             DESCRIPTION                                          AUTHOR    SAFE  STATUS
000001 create_users     Create the users table with UUID v7 primary key      artumont  yes   applied
000002 create_invites   Create the invites table for registration tokens     artumont  yes   applied
000003 create_settings  Create the settings singleton and seed default row   artumont  yes   applied
000004 drop_legacy      Drop legacy invites table                            artumont  no    pending
```

## Tracking

Applied migrations recorded in `schema_migrations`:

```sql
SELECT name, author, applied_at FROM schema_migrations ORDER BY id;
```

| Column | Type | Description |
|--------|------|-------------|
| `id` | bigserial | Row ID |
| `name` | varchar | Migration name from `metadata.json` |
| `author` | varchar | Who applied it (from `metadata.json`) |
| `applied_at` | timestamptz | When it was applied |

## Production workflow

1. Create: `make migrate-create name=add_feature desc="Add feature table"`
2. Write SQL in `up.sql` and `down.sql`
3. Test: `make migrate`
4. Verify: `make migration-list`
5. Open PR, get review
6. Merge — safe migrations auto-apply on deploy boot
7. For unsafe: deploy then `MIGRATE_UNSAFE=true make migrate` explicitly

## FAQ

**Q: Can I rename a migration?**
Don't. Name is the primary key in `schema_migrations`. Create a new one.

**Q: Server starts, migration fails?**
Server exits. Fix the SQL, restart. Already-applied are skipped.

**Q: Backfill data?**
`"safe": true` if idempotent (`INSERT ... ON CONFLICT DO NOTHING`). `"safe": false` if one-time.

**Q: Author shows "unknown"?**
Run `git config user.name <your-name>`.
