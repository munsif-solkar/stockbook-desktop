# Self-hosting Stockbook

The official Stockbook downloads include two built-in choices at sign-in:
**On this computer** uses a local SQLite ledger, and **Stockbook Cloud** connects
to the hosted Supabase project compiled into the app. Normal users do not need
this guide, and local mode needs no project at all.

This guide is for people building Stockbook from source who want the cloud
choice to use their own Supabase project — for a private deployment, a different
region, or a fork.

## 1. Create the Supabase project

1. Create a project at <https://supabase.com/dashboard>.
2. Apply the migrations in `supabase/migrations/` in filename order. You can
   paste each file into **SQL Editor**, or use `psql`:

   ```bash
   psql "$SUPABASE_DB_URL" -f supabase/migrations/20261113001000_stockbook_supabase.sql
   psql "$SUPABASE_DB_URL" -f supabase/migrations/20261114000100_export_inventory_only.sql
   ```

3. In **Authentication → Providers → Email**, decide whether new users must
   confirm their email. If confirmation is enabled, users must confirm before
   their first sign-in.

## 2. Configure the build

Edit [`config.json`](../config.json) at the repository root:

```json
{
  "supabaseUrl": "https://YOUR-PROJECT.supabase.co",
  "supabaseAnonKey": "YOUR-ANON-OR-PUBLISHABLE-KEY"
}
```

Copy both values from **Project Settings → API**:

- **Project URL** → `supabaseUrl`
- **anon** or **publishable** key → `supabaseAnonKey`

The file is compiled into the binary with `go:embed`. Changing it requires a
rebuild. End users never see or edit it.

> **Never use a `service_role` key here.** It bypasses row-level security and
> would be shipped to every user. Only the public anon/publishable key belongs in
> the desktop app.

### Environment overrides

For development and tests, environment variables override the compiled config:

| Variable | Purpose |
| --- | --- |
| `STOCKBOOK_SUPABASE_URL` | Supabase API URL |
| `STOCKBOOK_SUPABASE_ANON_KEY` | Public anon/publishable key |
| `STOCKBOOK_DATA_DIR` | Session/data directory |
| `STOCKBOOK_DB` | Local SQLite database file (local mode) |

A local Supabase CLI project works too: `supabase start` prints the API URL and
anon key to export.

## 3. Build and verify

```bash
go test ./...
wails build -tags webkit2_41
./build/bin/stockbook
```

Choose **Stockbook Cloud**, register a test account, add an item, record a
purchase and a sale, then export the data. The exported JSON must contain
`products`, `purchases`, and `sales`, and must not contain a `company`, `email`,
or password field. Importing that file into a fresh **On this computer**
workspace proves the shared format works in both directions.

Local mode needs no Supabase project: it opens `stockbook.db` in the Stockbook
data folder and supports the same features.

To run the opt-in live test against a throwaway project:

```bash
STOCKBOOK_TEST_SUPABASE=1 \
STOCKBOOK_TEST_SUPABASE_URL=https://YOUR-TEST-PROJECT.supabase.co \
STOCKBOOK_TEST_SUPABASE_ANON_KEY=YOUR-TEST-ANON-KEY \
STOCKBOOK_TEST_EMAIL=you+stockbook-test@gmail.com \
  go test -run Live -v
```

Use a separate project for this. The test creates an auth user and writes rows.

## Troubleshooting

**“Stockbook could not open its workspace” on launch** — the selected backend
could not start. In cloud mode, the compiled URL/key may be wrong or the
Supabase project may be paused; check the dashboard and rebuild if needed. In
local mode, the data folder may not be writable. Choose the other option on the
same screen to continue.

**“Email not confirmed”** — the project requires confirmation. Confirm the
address from the email, or disable confirmation in the Supabase Auth settings.

**Sign-up fails with a database error** — the migrations were not applied, or
were applied out of order. Apply every file in `supabase/migrations/` in
filename order.

**An import says “not a Stockbook data file”** — the file is not a Stockbook
JSON export. Do not rename an arbitrary JSON file; export the workspace from
Stockbook, or extract a 1.x ledger with
[the migration guide](migrating-from-1.x.md).
