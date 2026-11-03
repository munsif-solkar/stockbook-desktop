# Changelog

All notable changes to Stockbook Desktop are documented here. This project uses
[Semantic Versioning](https://semver.org/).

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
