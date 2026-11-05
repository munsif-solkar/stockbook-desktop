# Stockbook Desktop

[![Test](https://github.com/munsif-solkar/stockbook-desktop/actions/workflows/test.yml/badge.svg)](https://github.com/munsif-solkar/stockbook-desktop/actions/workflows/test.yml)
![License: MIT](https://img.shields.io/badge/license-MIT-blue)

A local-first inventory, purchasing, and sales ledger for small retail shops.
Stockbook Desktop is a native desktop application built with Go and
[Wails](https://wails.io)

Everything runs on the computer it is installed on. There is no server, no
network API, no account in the cloud, and no telemetry. The database is a plain
SQLite file you own and can copy, inspect, or back up at any time.

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

## Features

- **Accounts**: register, sign in, sign out, and keep each company's data
  private.
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
- **Backup & restore**: save the whole local database to a `.db` file, and
  restore it with a validated, automatic safety snapshot.
- **In-app updates**: check GitHub releases, then download and install a newer
  build that is verified against the release checksum.
- INR formatting by default, with date-range presets and pagination.

## Where your data lives

Data is stored in the platform data directory, **never inside the executable or
the AppImage**:

```
Linux:    $XDG_DATA_HOME/Stockbook/stockbook.db   (defaults to ~/.local/share/Stockbook)
Windows:  %LOCALAPPDATA%\Stockbook\stockbook.db
```

The session token sits next to it in `session`, mode `0600`. All files the app
creates are owner-only thanks to a restrictive process umask.

Replacing, updating, or deleting `stockbook` / `Stockbook.exe` does not touch
that database. A new build opens the same file and keeps every account, item,
purchase, and sale.

Environment overrides:

| Variable | Purpose |
| --- | --- |
| `STOCKBOOK_DATA_DIR` | Use a different directory for the database and session |
| `STOCKBOOK_DB` | Point at a specific database file |

On first run, if the target database does not exist, the app looks for a legacy
`data/stockbook.db` beside the working directory or the executable and copies it
(with its WAL) into place.

### Backups and durability

Each time the app starts with existing data, it writes a consistent snapshot to
a `backups` folder beside the database:

```
Linux:    ~/.local/share/Stockbook/backups/
Windows:  %LOCALAPPDATA%\Stockbook\backups\
```

The ten most recent snapshots are kept and older ones are pruned. Snapshots are
ordinary `.db` files. To restore one:

1. Close Stockbook.
2. Copy the snapshot over `stockbook.db`.
3. Delete any `stockbook.db-wal` and `stockbook.db-shm` files beside it.

Backups protect against a bad upgrade, a mistaken file operation, or a corrupted
database, but they live on the same disk. Copy the `backups` folder to external
storage periodically for protection against disk failure.

The Windows and Linux copies are independent: each operating system keeps its
own local database. Use a backup file to move a ledger between machines.

Only deleting the data directory, wiping the OS user profile, or losing the disk
removes the ledger.

### Backup & restore

Open **Backup & restore** in the sidebar to save a copy of the whole database.
The result is an ordinary SQLite `.db` file containing every account, item,
purchase, and sale. Keep it on another drive or in cloud storage.

Restoring validates the selected file before changing anything:

1. The backup must be a healthy SQLite database with Stockbook's tables.
2. A safety snapshot of the current data is written to the `backups` folder.
3. The database is swapped atomically and the app signs out.
4. Sign in with the account stored in the restored backup.

To undo a restore, close Stockbook, copy the `restore-safety-*.db` snapshot over
`stockbook.db`, and delete any `stockbook.db-wal` and `stockbook.db-shm` files
beside it.

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

- Passwords are hashed with scrypt and a per-user random salt.
- Session tokens are random 256-bit values stored only as SHA-256 hashes in
  SQLite; the raw token lives in the owner-only session file.
- Sessions expire after seven days and are deleted on logout.
- Login attempts are rate limited in memory.
- Every query and mutation is scoped to the signed-in company, and SQL uses
  bound parameters throughout.
- Input validation happens on both the client and the Go side; money is stored
  as integer cents to avoid floating-point drift.
- WebKit runs with the default context menu disabled and no devtools in
  production builds.
- Writes use `synchronous = FULL`, so an acknowledged purchase or sale is on
  disk even after a power cut.
- Sessions are written through a temporary file and renamed into place, so a
  crash cannot leave a truncated token.

## Build from source

Prerequisites:

- Go 1.25+
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
current Windows 10 and 11. Data is stored in
`%LOCALAPPDATA%\Stockbook\stockbook.db`, and the session file sits beside it.

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

The AppImage bundles the binary, desktop entry, and icon. It reads and writes the
same data directory described above, so it can be tested with a throwaway
directory without touching live data:

```bash
XDG_DATA_HOME=/tmp/stockbook-test \
  ./build/appimage/Stockbook-x86_64.AppImage
```

## Tests

```bash
go test ./...
go vet ./...
```

- `actions_test.go` covers registration, login, session files, item creation,
  newest-first item ordering, purchases, sales, payment splits, stock
  restoration, deletion rules, and validation in a temporary database.
- `backup_test.go` covers full-database backups, restore validation, safety
  snapshots, signing out after restore, and rejection of invalid files.
- `version_test.go` covers version comparison, update asset selection, and
  download URL validation.
- `db_test.go` covers opening databases at paths with spaces and punctuation and
  verifies startup backups, backup contents, and retention pruning.
- `parity_test.go` compares the desktop queries against a copy of real data and
  is opt-in so it never touches the live database:

```bash
mkdir -p /tmp/stockbook-copy
cp data/stockbook.db* /tmp/stockbook-copy/
STOCKBOOK_TEST_DATA=/tmp/stockbook-copy go test -run TestParityReport -v
```

## Project layout

```
main.go, app.go        Wails entry point and window setup
actions*.go            Auth and mutation actions exposed to the frontend
queries.go             Read queries and aggregations
db.go                  SQLite opening, PRAGMAs, migrations, backups
backup.go              Database backup, validation, and restore
updates.go, version.go GitHub release checks and verified self-updates
auth.go                scrypt password hashing and session handling
models.go, validation.go, format.go
frontend/              React + TypeScript + Vite SPA
build/                 Icons, platform metadata, packaging
```

## License

MIT. See [LICENSE](LICENSE).
