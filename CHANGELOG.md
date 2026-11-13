# Changelog

All notable changes to Stockbook Desktop are documented here. This project uses
[Semantic Versioning](https://semver.org/).

## [2.1.0] - 2026-11-14

### Added

- A storage choice on the sign-in screen: **On this computer** (local SQLite) or
  **Stockbook Cloud** (hosted Supabase). Neither asks for a Supabase URL or key.
- Local mode restores the Stockbook 1.x ledger: the same `stockbook.db`,
  `session`, and `backups` folder, local accounts with scrypt password hashes,
  and local session restore.
- `backend.json` records only the selected mode. Existing Stockbook 2.0
  installations with a `session.json` open in cloud mode; every other install
  starts local-first.
- Local and cloud sessions are remembered separately, so switching back resumes
  the workspace that was already signed in.
- A build-time `config.json` at the repository root, embedded into the binary
  with `go:embed`, so the cloud choice connects to the hosted Supabase project
  with no setup.

### Changed

- **Backup & restore** is now **Export/import data**, and it works with either
  backend.
- Exports contain inventory data only: items, purchases, and sales. The
  workspace/account block, email address, and any password data are excluded.
- Imports are fully validated before anything is touched: broken JSON, wrong
  types, invalid ranges, duplicate IDs or SKUs, orphaned references, invalid
  dates, and unknown payment modes are rejected. Local imports run in one
  transaction and roll back on failure; cloud imports use the validated payload.
- The same JSON file moves between local and cloud, and between accounts. A file
  exported locally imports into any cloud account, even a different one.
- Importing no longer signs the user out. The frontend reloads the workspace and
  keeps the session.
- End users no longer create or edit a data-directory `config.json`. Environment
  overrides remain for development and tests.
- The 1.x-to-JSON conversion prompt is gone because local mode opens the old
  database directly. The standalone converter remains at
  `supabase/migrate-sqlite-to-json.py` as an optional utility.
- Rewrote the migration, self-hosting, and README documentation for the
  local/cloud model.

### Fixed

- The live Supabase integration test now requires explicit test project
  credentials, so it can no longer create users or write rows in the hosted
  production project by accident.


## [2.0.0] - 2026-11-13

### Breaking

- Moved accounts and business data from the local SQLite database to Supabase
  (Postgres, Auth, and row-level security). The same workspace can now be
  opened from more than one computer.
- Stockbook 2.0 does not read the 1.x `stockbook.db`. Existing ledgers must be
  converted and restored once; see
  [docs/migrating-from-1.x.md](docs/migrating-from-1.x.md) and
  `scripts/migrate-sqlite-to-json.py`.
- A Supabase project and a `config.json` (or environment variables) are now
  required. See [docs/supabase-setup.md](docs/supabase-setup.md).

### Changed

- Desktop access goes through `stockbook_*` `security definer` functions scoped
  to `auth.uid()`, with RLS enabled on all workspace tables.
- Backup & restore now exports and imports a JSON workspace file instead of a
  SQLite snapshot. Restore still validates the file and writes a safety
  snapshot first.
- Only the Supabase session file is kept on disk; there is no local database.
- Added an optional `config.json` in the Stockbook data directory so a packaged
  build can point at a hosted Supabase project without environment variables.
- The workspace name from sign-up metadata is now used when the account is
  confirmed and signs in for the first time.
- Added an opt-in live Supabase integration test.
- Added clear login feedback for unconfirmed email addresses, and renamed the
  backend labels in the UI to Supabase.

## [1.1.0] - 2026-11-04

### Added

- Backup & restore page in the sidebar. Backups are complete SQLite `.db`
  snapshots; restore validates the file, keeps a safety snapshot, swaps the
  database atomically, and signs out.
- In-app update checks against the latest GitHub release, with verified
  downloads, checksum validation, self-replacement, and automatic restart.
- "Check for updates" shortcut in the sidebar and the app version on the
  Backup & restore page.
- Newest-first ordering on the Items page so a newly created item appears at
  the top, ready for adding stock.

### Reliability

- Interrupted restores recover the previous database on the next launch.
- Restore safety snapshots are kept separately from rolling startup backups.
- Windows update replacement removes the previous executable on the next
  launch.

## [1.0.0] - 2026-11-03

First desktop release.

### Added

- Native Go + Wails application with the full Stockbook feature set: accounts,
  overview analytics, items, inventory, purchases, sales, quick sell, filtering,
  and pagination.
- Accounts with scrypt password hashing, hashed session tokens, seven-day
  expiry, and in-memory login rate limiting.
- Per-company data scoping on every query and mutation.
- INR money formatting with values stored as integer cents.
- Linux AppImage and Windows x86-64 builds.
- Automatic startup backups beside the database, with the ten most recent
  snapshots retained.
- Platform data directories: XDG on Linux, `%LOCALAPPDATA%` on Windows.
- Import of an existing web-app `data/stockbook.db` on first run.

### Fixed

- Windows startup failed with `configure database: SQL logic error: out of
  memory (1)` because the database path was converted to a `file:` URI with a
  drive letter. The native path is now passed directly to the SQLite driver.
- The Windows build stores data in `%LOCALAPPDATA%\Stockbook` instead of a
  Unix-style home directory path.

### Reliability

- `PRAGMA synchronous = FULL` so acknowledged writes survive a power cut.
- Session files are written atomically and renamed into place.
- Regression tests for special-character database paths, backup contents, and
  backup retention.
