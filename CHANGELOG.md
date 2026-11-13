# Changelog

All notable changes to Stockbook Desktop are documented here. This project uses
[Semantic Versioning](https://semver.org/).

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
