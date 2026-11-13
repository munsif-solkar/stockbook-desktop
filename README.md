# Stockbook Desktop

[![Test](https://github.com/munsif-solkar/stockbook-desktop/actions/workflows/test.yml/badge.svg)](https://github.com/munsif-solkar/stockbook-desktop/workflows/Test/badge.svg)
![License: MIT](https://img.shields.io/badge/license-MIT-blue)

A real-time inventory, purchasing, and sales ledger for small retail shops.
Stockbook Desktop is a native desktop application built with Go and
[Wails](https://wails.io), with a local-first design.

**Your data lives on your computer by default.** At the sign-in screen you choose
where Stockbook keeps this workspace:

- **On this computer** — a local SQLite ledger in the same folder Stockbook 1.x
  used. No account server, no internet connection, no setup.
- **Stockbook Cloud** — a hosted [Supabase](https://supabase.com) workspace you
  can open from any computer with your account.

Both choices are built in. You never enter a Supabase URL or key.

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

## Quick start

1. Download and open Stockbook.
2. Choose **On this computer** or **Stockbook Cloud**.
3. Create a workspace (company name, email, password) or sign in.
4. Start adding items, recording purchases, and recording sales.

Each workspace has its own items, purchases, and sales. In cloud mode the same
account can sign in from another computer; in local mode the workspace stays on
the computer that holds the database file.

## Upgrading from Stockbook 1.x

Stockbook 1.x kept everything in a local SQLite database, and Stockbook 2.1
opens that database directly. There is no conversion step and no data is
deleted:

1. Install Stockbook 2.1 and open it.
2. Choose **On this computer** at the sign-in screen.
3. Sign in with the same email and password you used in 1.x.

The app reads `stockbook.db` and the 1.x `session` file from the same data
folder, so existing accounts, sessions, items, purchases, sales, and backups keep
working. Startup snapshots continue to be written to the `backups` folder.

To move a 1.x workspace to Stockbook Cloud instead:

1. Open Stockbook 2.1 in local mode and sign in to the old workspace.
2. On **Export/import data**, choose **Export to JSON…**.
3. Switch to **Stockbook Cloud** on the sign-in screen and create or sign in to a
   cloud account.
4. On **Export/import data**, choose **Import from JSON file…** and select the
   exported file.

The JSON contains inventory data only: items, purchases, and sales. It never
contains the old email, password hash, or session, so old passwords cannot be
migrated; use the password of the cloud account you sign in with. See
[docs/migrating-from-1.x.md](docs/migrating-from-1.x.md) for the full walkthrough,
Windows paths, and multi-company databases.

## Features

- **Two storage choices**: a local SQLite ledger or Stockbook Cloud, selected on
  the sign-in screen. Each remembers its own session.
- **Accounts**: local accounts with scrypt password hashing, or Supabase Auth for
  cloud workspaces. Data is scoped to one company per workspace.
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
- **Export/import data**: save the workspace as a JSON file containing only
  inventory data, and import one back with validation and an automatic safety
  copy. Files move freely between local and cloud workspaces, and between
  accounts.
- **In-app updates**: check GitHub releases, then download and install a newer
  build that is verified against the release checksum.
- INR formatting by default, with date-range presets and pagination.

## Your data

The sign-in choice is stored in `backend.json` and contains only the mode name:

```json
{ "mode": "local" }
```

It never contains keys, tokens, or account details. When the file is absent,
Stockbook starts in local mode, except for a Stockbook 2.0 installation that
already has a `session.json`, which opens in cloud mode so its workspace is not
lost.

### On this computer

```
Linux:    $XDG_DATA_HOME/Stockbook/    (defaults to ~/.local/share/Stockbook)
Windows:  %LOCALAPPDATA%\Stockbook\
```

| File | Purpose |
| --- | --- |
| `stockbook.db` | Items, purchases, sales, and local accounts (SQLite) |
| `stockbook.db-wal`, `stockbook.db-shm` | SQLite write-ahead log |
| `session` | The signed-in local session token |
| `backups/` | Startup snapshots and import safety copies |

`STOCKBOOK_DB` can point the local database at a different file, and
`STOCKBOOK_DATA_DIR` changes the data folder.

### Stockbook Cloud

Business data lives in Supabase Postgres. The desktop app stores only the cloud
session file:

```
Linux:    $XDG_DATA_HOME/Stockbook/session.json
Windows:  %LOCALAPPDATA%\Stockbook\session.json
```

The session file contains the access and refresh tokens, is written with mode
`0600`, and is removed on sign-out. Replacing or updating the executable does
not touch it.

If the cloud backend is unreachable, the sign-in screen shows the connection
error and offers the local workspace as an alternative.

### Export/import data

Open **Export/import data** in the sidebar.

**Export** writes a JSON copy of the signed-in workspace. Keep it on another
drive or in cloud storage.

**Import** validates the selected file before changing anything:

1. The file must be a Stockbook JSON file with products, purchases, and sales.
2. Every row is checked: identifiers, names, categories, SKUs, prices,
   quantities, dates, payment modes, notes, and references between records.
   Duplicate IDs or SKUs, orphaned purchases or sales, negative values, and
   invalid dates are rejected. Any account fields in a file are ignored and
   never written back out.
3. A safety copy of the current workspace is written to the `backups` folder in
   the Stockbook data directory.
4. The workspace rows are replaced atomically in one transaction.
5. The account and sign-in stay exactly as they are.

Exports contain items, purchases, and sales only. No email address, password,
account record, or session token is written into an export, so a file exported
from a local workspace can be imported into any cloud account — even a different
one — and the other way around. To undo an import, import the
`import-safety-*.json` file from the `backups` folder using the same flow.

## App updates

Click **Check for updates** in the sidebar (or on **Export/import data**) to
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

## Self-hosting and building from source

The official builds compile a Stockbook Cloud project URL and public
publishable key from [`config.json`](config.json) at the repository root. End
users never see or edit this file, and local mode works without it.

If you are building Stockbook for your own Supabase project:

1. Create a project at <https://supabase.com/dashboard>.
2. Apply the migrations in `supabase/migrations/` in filename order:

   ```bash
   psql "$SUPABASE_DB_URL" -f supabase/migrations/20261113001000_stockbook_supabase.sql
   psql "$SUPABASE_DB_URL" -f supabase/migrations/20261114000100_export_inventory_only.sql
   ```

3. Put your **Project URL** and **anon/publishable key** in `config.json` at the
   repository root:

   ```json
   {
     "supabaseUrl": "https://YOUR-PROJECT.supabase.co",
     "supabaseAnonKey": "YOUR-ANON-OR-PUBLISHABLE-KEY"
   }
   ```

4. Build the app. The file is embedded into the binary with `go:embed`.

**Never put a `service_role` key in `config.json`.** It would be shipped to every
user. Only the public anon/publishable key belongs there; row-level security and
the RPC functions enforce access.

### Environment overrides

Environment variables are for development and tests. They take precedence over
the compiled `config.json`:

| Variable | Purpose |
| --- | --- |
| `STOCKBOOK_SUPABASE_URL` | Override the Supabase API URL |
| `STOCKBOOK_SUPABASE_ANON_KEY` | Override the public anon key |
| `STOCKBOOK_DATA_DIR` | Use a different session/data directory |
| `STOCKBOOK_DB` | Use a different local SQLite database file |

### Prerequisites

- Go 1.26+
- Node.js 20+
- Wails v2 (`go install github.com/wailsapp/wails/v2/cmd/wails@latest`)
- Linux only: `webkit2gtk-4.1` development files and the usual GTK build
  dependencies. On distributions that ship only the 4.1 package, build with the
  `webkit2_41` tag.

```bash
git clone https://github.com/munsif-solkar/stockbook-desktop.git
cd stockbook-desktop

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
current Windows 10 and 11. Cloud sessions are stored in
`%LOCALAPPDATA%\Stockbook\session.json`; local ledgers use
`%LOCALAPPDATA%\Stockbook\stockbook.db`.

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

The AppImage bundles the binary, desktop entry, and icon. Local ledgers and
cloud sessions use the data directory described above.

## Supabase backend

Business data is stored in these Postgres tables for cloud workspaces:

| Table | Purpose |
| --- | --- |
| `stockbook_companies` | One workspace per Supabase auth user |
| `stockbook_products` | Item catalog and stock on hand |
| `stockbook_purchases` | Stock-in records |
| `stockbook_sales` | Sales with cost basis and profit |

All reads and writes go through `stockbook_*` SQL functions. The functions scope
every row to the signed-in user's company, and row-level security is enabled on
the tables as a second layer. Money is stored as integer paise/cents.

Local workspaces use the same ledger schema in SQLite, so the two backends share
the same export format and business rules.

## Security notes

- **Local mode** stores accounts in SQLite with scrypt password hashes and
  hashed session tokens. The database and session file are owner-only.
- **Cloud mode** delegates accounts and passwords to Supabase Auth; Stockbook
  never stores a password or password hash. Session tokens are short-lived
  access tokens plus a rotating refresh token in an owner-only `session.json`,
  removed on sign-out.
- Cloud reads and writes go through `stockbook_*` `security definer` functions
  that derive the company from `auth.uid()`. Client-side table writes cannot
  change stock or money.
- Row-level security is enabled on all cloud workspace tables as a second layer.
- Login attempts are rate limited in memory in both modes.
- Input validation happens on both the client and the Go side; money is stored
  as integer paise/cents to avoid floating-point drift.
- The anon/publishable key shipped in the app is public by design. The
  service-role key is never embedded or sent to the desktop client.
- Exported JSON files are fully validated on import and contain no account
  fields, email addresses, password hashes, or session tokens.
- WebKit runs with the default context menu disabled and no devtools in
  production builds.
- Session files are written through a temporary file and renamed into place, so
  a crash cannot leave a truncated token.
- Local imports run inside a single transaction; a failure leaves the previous
  ledger untouched.

## Tests

```bash
go test ./...
go vet ./...
```

- `local_test.go` covers local registration, login, sessions, item/purchase/sale
  flows, export, import into a different account, rollback on failure, and
  session expiry.
- `datafile_test.go` covers JSON validation: broken files, wrong types, invalid
  ranges, duplicate IDs and SKUs, orphaned references, account-field stripping,
  normalization, and stable round trips.
- `backup_test.go` checks that cloud exports drop account fields.
- `version_test.go` covers version comparison, update asset selection, and
  download URL validation.
- `integration_live_test.go` exercises the cloud flow against a real Supabase
  project: registration, items, purchases, sales, overview, stock rules, session
  restore, JSON export, and import. It is opt-in and requires an explicit test
  backend:

```bash
STOCKBOOK_TEST_SUPABASE=1 \
STOCKBOOK_TEST_SUPABASE_URL=https://YOUR-TEST-PROJECT.supabase.co \
STOCKBOOK_TEST_SUPABASE_ANON_KEY=YOUR-TEST-ANON-KEY \
STOCKBOOK_TEST_EMAIL=you+stockbook-test@gmail.com \
  go test -run Live -v
```

The live test registers that email address and writes rows in the configured
project. Use a throwaway project and an address you control. Hosted Supabase
rejects placeholder domains such as `example.com` during sign-up, so use a real
inbox.

## Project layout

```
main.go, app.go        Wails entry point and window setup
backend.go             Local/cloud selection, dispatch, and session restore
local_db.go            Local SQLite schema, data paths, startup backups
local_auth.go          Local accounts, scrypt hashing, sessions
local_queries.go       Local reads and dashboard shaping
local_actions.go       Local item, purchase, and sale mutations
local_backup.go        Local export and transactional import
datafile.go            Shared JSON validation, normalization, and export shape
backup.go              Export/import dialogs, safety copies, backend routing
actions*.go            Cloud auth and mutation actions exposed to the frontend
queries.go             Cloud RPC reads and trend shaping
db.go                  Supabase client, auth calls, RPC helper, embedded config
updates.go, version.go GitHub release checks and verified self-updates
auth.go                Cloud session file persistence and sign-out
models.go, validation.go, format.go
config.json            Build-time cloud URL and public publishable key
supabase/migrations/   Postgres schema, RPC functions, RLS, grants
supabase/migrate-sqlite-to-json.py
                       Optional standalone 1.x export utility
frontend/              React + TypeScript + Vite SPA
build/                 Icons, platform metadata, packaging
```

## Documentation

- [docs/migrating-from-1.x.md](docs/migrating-from-1.x.md) — how 1.x data opens
  in local mode and how to move it to the cloud.
- [docs/self-hosting.md](docs/self-hosting.md) — point a source build at your own
  Supabase project.
- [CHANGELOG.md](CHANGELOG.md) — release history.

## License

MIT. See [LICENSE](LICENSE).
