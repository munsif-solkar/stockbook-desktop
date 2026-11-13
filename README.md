# Stockbook Desktop

[![Test](https://github.com/munsif-solkar/stockbook-desktop/actions/workflows/test.yml/badge.svg)](https://github.com/munsif-solkar/stockbook-desktop/actions/workflows/test.yml)
![License: MIT](https://img.shields.io/badge/license-MIT-blue)

A real-time inventory, purchasing, and sales ledger for small retail shops.
Stockbook Desktop is a native desktop application built with Go and
[Wails](https://wails.io), backed by [Supabase](https://supabase.com)
(Postgres, Auth, and row-level security).

Accounts, items, purchases, and sales live in Supabase, so the same workspace
can be opened from more than one computer. The desktop app keeps only a
Supabase session file on disk; there is no local database.

## Quick start

Stockbook 2.0 needs a Supabase project. A download by itself cannot store data.

1. Create a project at <https://supabase.com/dashboard>.
2. Apply the schema from
   `supabase/migrations/20261113001000_stockbook_supabase.sql` in the Supabase
   SQL editor, or with `psql`.
3. Copy your **Project URL** and **anon/publishable key** from **Project
   Settings → API** into `config.json` in the Stockbook data directory:

   ```json
   {
     "supabaseUrl": "https://YOUR-PROJECT.supabase.co",
     "supabaseAnonKey": "YOUR-ANON-OR-PUBLISHABLE-KEY"
   }
   ```

   Start from [`config.example.json`](config.example.json). Put the file at
   `~/.local/share/Stockbook/config.json` on Linux or
   `%LOCALAPPDATA%\Stockbook\config.json` on Windows.
4. Start Stockbook, register or sign in, and begin adding items.

The full walkthrough, including email confirmation and troubleshooting, is in
[docs/supabase-setup.md](docs/supabase-setup.md).

## Download

Prebuilt binaries are attached to the
[latest release](https://github.com/munsif-solkar/stockbook-desktop/releases/latest):

| Platform | File | Notes |
| --- | --- | --- |
| Linux x86-64 | `Stockbook-x86_64.AppImage` | `chmod +x` then run. No installation needed. |
| Windows 10/11 x86-64 | `Stockbook-windows-amd64.zip` | Unzip and run `Stockbook.exe`. |
| Windows 10/11 x86-64 | `stockbook.exe` | The same executable without the zip. |
| Checksums | `SHA256SUMS.txt` | Verify downloads with `sha256sum -c SHA256SUMS.txt`. |

Windows builds need the Microsoft Edge WebView2 Runtime, which ships with
current Windows 10 and 11. If SmartScreen warns about an unsigned app, choose
**More info → Run anyway**.

## Upgrading from Stockbook 1.x

Stockbook 2.0 is a breaking release: the local SQLite database is no longer
read or written. Before using 2.0:

1. Set up Supabase and `config.json` as described in the quick start above.
2. Convert the old ledger with the bundled script:

   ```bash
   python3 scripts/migrate-sqlite-to-json.py --list
   python3 scripts/migrate-sqlite-to-json.py \
     --db "$HOME/.local/share/Stockbook/stockbook.db" \
     --company 1 --out "$HOME/stockbook-backup.json"
   ```

3. In Stockbook 2.0, sign in and restore that JSON file from **Backup &
   restore**.
4. Verify items, stock, purchases, revenue, and profit, then remove the old
   `stockbook.db` and its `-wal`/`-shm` sidecars when you no longer need them.

Keep the JSON export as a rollback copy. The complete procedure, including
Windows paths and multi-company databases, is in
[docs/migrating-from-1.x.md](docs/migrating-from-1.x.md).

## Features

- **Accounts**: register and sign in with Supabase Auth, and keep each
  company's data private with row-level security.
- **Overview**: sales, purchase spend, gross profit, units sold, cash/UPI split,
  sales trend, top items, low stock, and recent sales.
- **Items**: catalog CRUD with SKU, category, default prices, and low-stock
  limits.
- **Inventory**: stock on hand, stock value, retail value, low-stock filter, and
  a searchable "Update stock" dialog that records a purchase.
- **Purchases**: filtered history, totals, average cost, and safe deletion.
- **Sales**: filtered history with cost basis and profit, category performance,
  cashflow split, and safe deletion that returns units to stock.
- **New sell**: quick-sale dialog with item search, stock limits, editable price,
  and cash/UPI payment modes.
- **Backup & restore**: export the whole workspace to a JSON file, and restore it
  with a validated, automatic safety snapshot.
- **In-app updates**: check GitHub releases, then download and install a newer
  build that is verified against the release checksum.
- INR formatting by default, with date-range presets and pagination.

## Supabase backend

Business data is stored in these Postgres tables:

| Table | Purpose |
| --- | --- |
| `stockbook_companies` | One workspace per Supabase auth user |
| `stockbook_products` | Item catalog and stock on hand |
| `stockbook_purchases` | Stock-in records |
| `stockbook_sales` | Sales with cost basis and profit |

All reads and writes go through `stockbook_*` SQL functions. The functions scope
every row to the signed-in user's company, and row-level security is enabled on
the tables as a second layer. Money is stored as integer paise/cents.

Apply the schema once per Supabase project:

```bash
psql "$SUPABASE_DB_URL" -f supabase/migrations/20261113001000_stockbook_supabase.sql
```

### Configuration

Point the app at a Supabase project in either of two ways.

**Config file (recommended for the packaged app).** Create `config.json` in the
Stockbook data directory:

```json
{
  "supabaseUrl": "https://YOUR-PROJECT.supabase.co",
  "supabaseAnonKey": "YOUR-ANON-OR-PUBLISHABLE-KEY"
}
```

Paths:

```
Linux:    $XDG_DATA_HOME/Stockbook/config.json   (defaults to ~/.local/share/Stockbook)
Windows:  %LOCALAPPDATA%\Stockbook\config.json
```

**Environment variables.**

| Variable | Default | Purpose |
| --- | --- | --- |
| `STOCKBOOK_SUPABASE_URL` | `http://127.0.0.1:54321` | Supabase API URL |
| `STOCKBOOK_SUPABASE_ANON_KEY` | local Supabase anon key | Public anon key |
| `STOCKBOOK_DATA_DIR` | platform data directory | Config and session files |

Precedence is environment, then `config.json`, then the local-development
defaults. Restart the app after changing the backend.

The anon/publishable key is public by design; row-level security and the RPC
functions enforce access. Never ship the service-role key in the desktop app.

### What stays on the computer

Only the optional `config.json` and the Supabase session file are stored
locally:

```
Linux:    $XDG_DATA_HOME/Stockbook/{config.json,session.json}   (defaults to ~/.local/share/Stockbook)
Windows:  %LOCALAPPDATA%\Stockbook\{config.json,session.json}
```

The session file contains the access and refresh tokens, is written with mode
`0600`, and is removed on sign-out. `config.json` holds only the project URL and
public anon key. Replacing or updating the executable does not touch either file.
A new build resumes the same Supabase workspace.

If Supabase is unreachable, the app opens on the sign-in screen and shows the
connection error instead of starting with stale local data.

### Backups and durability

Open **Backup & restore** in the sidebar to export the whole workspace. The
result is a JSON file containing every item, purchase, and sale, plus the
workspace name. Keep it on another drive or in cloud storage.

Restoring validates the selected file before changing anything:

1. The file must be a Stockbook JSON backup with products, purchases, and sales.
2. A safety snapshot of the current workspace is written to the `backups` folder
   in the Stockbook data directory.
3. The workspace rows are replaced atomically in one database transaction.
4. The app signs out. Sign in again to open the restored workspace.

To undo a restore, use the `restore-safety-*.json` file from that folder with the
same restore flow.

### App updates

Click **Check for updates** in the sidebar (or in **Backup & restore**) to
compare the running version with the latest public GitHub release.

When a newer release exists, **Install update** downloads the build for this
platform, verifies it against the release's `SHA256SUMS.txt`, replaces the
executable, and restarts. On Windows the previous executable is renamed to
`.old` and removed on the next launch. Ledger data is never touched by an app
update.

Automatic updates require the release to contain the platform asset
(`Stockbook-x86_64.AppImage` or `stockbook.exe`) and `SHA256SUMS.txt`. If those
are missing, the app points to the release page for a manual download. The app
only downloads over HTTPS from the official `munsif-solkar/stockbook-desktop`
repository and its GitHub asset hosts.

## Security notes

- Accounts and passwords are handled by Supabase Auth; Stockbook never stores a
  password or password hash.
- Session tokens are short-lived Supabase access tokens plus a rotating refresh
  token. Only the refresh token and access token are kept, in an owner-only
  `session.json`, and they are removed on sign-out.
- Every read and write goes through a `stockbook_*` `security definer` function
  that derives the company from `auth.uid()`. Client-side table writes cannot
  change stock or money.
- Row-level security is enabled on all workspace tables as a second layer.
- Login attempts are rate limited in memory, on top of Supabase Auth limits.
- Input validation happens on both the client and the Go side; money is stored
  as integer paise/cents to avoid floating-point drift.
- The anon key shipped in the app is public by design. The service-role key is
  never embedded or sent to the desktop client.
- WebKit runs with the default context menu disabled and no devtools in
  production builds.
- The session file is written through a temporary file and renamed into place,
  so a crash cannot leave a truncated token.

## Build from source

Prerequisites:

- Go 1.25+
- Node.js 20+
- Wails v2 (`go install github.com/wailsapp/wails/v2/cmd/wails@latest`)
- A Supabase project (local or hosted) with the Stockbook migration applied
- Linux only: `webkit2gtk-4.1` development files and the usual GTK build
  dependencies. On distributions that ship only the 4.1 package, build with the
  `webkit2_41` tag.

```bash
git clone https://github.com/munsif-solkar/stockbook-desktop.git
cd stockbook-desktop

# Apply the schema to your Supabase database once.
psql "$SUPABASE_DB_URL" -f supabase/migrations/20261113001000_stockbook_supabase.sql

# Point the app at your project (defaults to a local Supabase on :54321).
export STOCKBOOK_SUPABASE_URL="https://YOUR-PROJECT.supabase.co"
export STOCKBOOK_SUPABASE_ANON_KEY="YOUR-ANON-KEY"

go test ./...
wails build -tags webkit2_41
./build/bin/stockbook
```

With a local Supabase CLI project, `supabase start` prints the API URL and anon
key to use. Sign-up confirmation must be disabled, or new users have to confirm
their email before the first sign-in.

For frontend-only checks:

```bash
cd frontend
npm install
npm run build        # tsc --noEmit && vite build
```

### Windows build

Wails can cross-compile the 64-bit Windows app from Linux:

```bash
wails build -platform windows/amd64
```

The result is `build/bin/stockbook.exe`, a GUI executable with the frontend and
icon embedded. It needs the Microsoft Edge WebView2 Runtime, which ships with
current Windows 10 and 11. The workspace uses the same Supabase project, and the
session file is stored in `%LOCALAPPDATA%\Stockbook\session.json`.

### AppImage

Build the Linux binary, copy it into the bundled AppDir, then package it:

```bash
wails build -tags webkit2_41
cp build/bin/stockbook build/appimage/Stockbook.AppDir/usr/bin/stockbook
cd build/appimage
ARCH=x86_64 appimagetool Stockbook.AppDir Stockbook-x86_64.AppImage
```

Run it directly:

```bash
./build/appimage/Stockbook-x86_64.AppImage
```

The AppImage bundles the binary, desktop entry, and icon. It keeps only the
Supabase session file in the data directory described above, so it can be tested
with a throwaway directory without touching a real session:

```bash
XDG_DATA_HOME=/tmp/stockbook-test \
  ./build/appimage/Stockbook-x86_64.AppImage
```

## Tests

```bash
go test ./...
go vet ./...
```

- `version_test.go` covers version comparison, update asset selection, and
  download URL validation.
- `integration_live_test.go` exercises the full flow against a real Supabase
  project: registration, items, purchases, sales, overview, stock rules, session
  restore, JSON backup, and restore. It is opt-in so it never touches production
  data:

```bash
STOCKBOOK_TEST_SUPABASE=1 \
STOCKBOOK_TEST_EMAIL=you+stockbook-test@gmail.com \
  go test -run TestLiveSupabaseFlow -v
```

The live test registers that email address and writes rows in the configured
project. Use an address you control and delete the user afterwards to clean up.
Hosted Supabase rejects placeholder domains such as `example.com` during
sign-up, so use a real inbox.

## Project layout

```
main.go, app.go        Wails entry point and window setup
actions*.go            Auth and mutation actions exposed to the frontend
queries.go             RPC reads and trend shaping
db.go                  Supabase client, auth calls, RPC helper, config
backup.go              JSON workspace export, validation, and restore
updates.go, version.go GitHub release checks and verified self-updates
auth.go                Session file persistence and sign-out
models.go, validation.go, format.go
supabase/migrations/   Postgres schema, RPC functions, RLS, grants
frontend/              React + TypeScript + Vite SPA
build/                 Icons, platform metadata, packaging
```

## Documentation

- [docs/supabase-setup.md](docs/supabase-setup.md) — create the project, apply
  the schema, configure the app, and troubleshoot sign-in.
- [docs/migrating-from-1.x.md](docs/migrating-from-1.x.md) — convert a 1.x
  SQLite ledger and restore it into Supabase.
- [config.example.json](config.example.json) — template for the backend config
  file.
- [CHANGELOG.md](CHANGELOG.md) — release history.

## License

MIT. See [LICENSE](LICENSE).
