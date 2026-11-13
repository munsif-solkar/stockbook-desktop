# Supabase setup

Stockbook 2.0 keeps accounts and ledger data in Supabase: Postgres for storage,
Supabase Auth for sign-in, and row-level security plus `stockbook_*` functions
for tenant isolation. Each Supabase auth user gets one Stockbook workspace.

You can use a hosted Supabase project or a local one started with the Supabase
CLI. These steps cover a hosted project.

## 1. Create a project

1. Open <https://supabase.com/dashboard> and create a project.
2. Save the database password somewhere safe.
3. Wait for the project to finish provisioning.

## 2. Apply the schema

The schema, functions, policies, indexes, and grants live in one migration:

```
supabase/migrations/20261113001000_stockbook_supabase.sql
```

**SQL editor (easiest):** open the file, copy all of it into the Supabase SQL
editor, and run it.

**psql:** use the connection string from **Project Settings → Database**:

```bash
psql "$SUPABASE_DB_URL" -v ON_ERROR_STOP=1 \
  -f supabase/migrations/20261113001000_stockbook_supabase.sql
```

The migration is idempotent. Re-run it after updating Stockbook to pick up new
or changed functions.

## 3. Get the API credentials

Open **Project Settings → API** and copy:

- **Project URL** — `https://YOUR-PROJECT.supabase.co`
- **Anon key** or **publishable key** — the public key

Do **not** use the `service_role` or secret key in the desktop app. It bypasses
row-level security.

## 4. Configure the desktop app

Create `config.json` in the Stockbook data directory:

| Platform | Path |
| --- | --- |
| Linux | `~/.local/share/Stockbook/config.json` (or `$XDG_DATA_HOME/Stockbook/config.json`) |
| Windows | `%LOCALAPPDATA%\Stockbook\config.json` |

```json
{
  "supabaseUrl": "https://YOUR-PROJECT.supabase.co",
  "supabaseAnonKey": "YOUR-ANON-OR-PUBLISHABLE-KEY"
}
```

Keep the file owner-only (`chmod 600`) where the platform supports it.

Environment variables override the file, which is useful for development:

| Variable | Purpose |
| --- | --- |
| `STOCKBOOK_SUPABASE_URL` | Project URL |
| `STOCKBOOK_SUPABASE_ANON_KEY` | Public anon/publishable key |
| `STOCKBOOK_DATA_DIR` | Use a different data directory |

Precedence is environment → `config.json` → local-development defaults. Restart
the app after changing the backend.

## 5. Email confirmation

Hosted Supabase enables **Confirm email** by default:

- **Keep it on for production.** New users get a confirmation email and must
  click the link before their first sign-in. The app shows “Check your email to
  confirm the account”.
- **Turn it off for local testing.** Dashboard → **Authentication → Sign In /
  Providers → Email → Confirm email**. New sign-ups are then active
  immediately.
- If you create users in **Authentication → Users**, enable **Auto Confirm
  User** so they can sign in right away.

The workspace row is created on the first successful sign-in, using the company
name from sign-up metadata when present.

## 6. First sign-in

1. Start Stockbook.
2. Register with a company name, email, and password, or sign in with an
   existing Supabase user.
3. Add items, purchases, and sales. The data is written to Supabase immediately.

## 7. Verify

In the app, open **Backup & restore**; the **Supabase backend** line should show
your project URL.

From the Supabase SQL editor:

```sql
select * from public.stockbook_companies;
select count(*) from public.stockbook_products;
select count(*) from public.stockbook_sales;
```

## Multiple computers

Install Stockbook on each computer, use the same `config.json`, and sign in with
the same account. Both computers see the same workspace. Session files are
independent and can be signed out separately.

## Troubleshooting

| Symptom | Likely cause |
| --- | --- |
| “Incorrect email or password” | The app is pointed at a different Supabase project, the user does not exist there, or the password is wrong. Check the backend URL in **Backup & restore**. |
| “Confirm your email address…” | The account exists but has not been confirmed. Confirm it in **Authentication → Users** or resend the email. |
| “Could not reach the Stockbook backend” | Wrong URL, no network, or the project is paused. Test `https://YOUR-PROJECT.supabase.co/auth/v1/health`. |
| `Secret API key required` in a browser | Normal for the bare `/rest/v1/` root with a publishable key. Table and RPC endpoints still work. |
| `permission denied for function` | The migration or its grants did not finish. Re-run it. |
| Data missing on another computer | Refresh the page. Each sign-in has its own session; data is shared, cached views are not. |

## Resetting a workspace

Restoring a backup replaces the signed-in workspace’s items, purchases, and
sales. To start over instead, sign in and restore an empty Stockbook backup, or
delete the rows in the Supabase SQL editor while signed in as the project owner:

```sql
delete from public.stockbook_sales where company_id = 1;
delete from public.stockbook_purchases where company_id = 1;
delete from public.stockbook_products where company_id = 1;
```

Take a backup first.
