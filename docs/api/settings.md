# Settings

Global application settings. Every endpoint requires an authenticated admin user.

## Endpoints

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| `GET` | `/settings` | 🔒👑 | Read current settings |
| `PATCH` | `/settings` | 🔒👑 | Update supplied settings |

🔒 = valid Bearer token. 👑 = `is_admin: true`.

## Settings

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `allow_signup_without_invite` | boolean | `true` | Permit registration without a valid invite token |
| `first_init_completed` | boolean | `false` | Whether the initial admin has been created |

`allow_signup_without_invite` defaults to `true` to permit first-admin bootstrap via `/auth/register/admin`. Set it to `false` after initial setup to require invitations.

`first_init_completed` is set to `true` automatically after `/auth/register/admin` succeeds. Once set, the init endpoint permanently returns `404`.

## Examples

### Read

```http
GET /settings
Authorization: Bearer <admin-token>
```

```json
{
  "allow_signup_without_invite": true,
  "first_init_completed": false
}
```

### Disable open signup

```http
PATCH /settings
Authorization: Bearer <admin-token>
Content-Type: application/json

{"allow_signup_without_invite": false}
```

```json
{
  "allow_signup_without_invite": false,
  "first_init_completed": false
}
```

When disabled, `POST /auth/register` without `invite` returns `403`:

```json
{"error":"an invite is required"}
```

An invalid supplied invite always returns `403`, even while open signup is enabled.

## Persistence

Settings use a singleton `settings` database row (`id = 1`). Startup creates it with defaults using `INSERT ... ON CONFLICT DO NOTHING`; startup also repairs a missing singleton row. Updates use an atomic PostgreSQL upsert, so later settings can be added without adding routes or tables.
