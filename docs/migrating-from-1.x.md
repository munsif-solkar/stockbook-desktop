# Migrating from Stockbook 1.x (SQLite)

Stockbook 2.1 keeps the local-first path that 1.x used. The sign-in screen offers
**On this computer** and **Stockbook Cloud**:

- **On this computer** opens the existing 1.x SQLite ledger directly. Accounts,
  sessions, items, purchases, sales, and backups keep working in the same folder.
- **Stockbook Cloud** is a hosted workspace for access from more than one
  computer. Moving an old workspace there is a one-time export and import.

## Keep using the old database locally

1. Install and open Stockbook 2.1.
2. Choose **On this computer**.
3. Sign in with the email and password you used in 1.x.

That is the whole migration. Stockbook opens `stockbook.db` in the usual data
folder, restores the existing 1.x `session` file when it is still valid, and
continues writing startup snapshots to `backups/`. Nothing is deleted or
converted.

Old data locations:

```
Linux:    ~/.local/share/Stockbook/stockbook.db
          ~/.local/share/Stockbook/session
          ~/.local/share/Stockbook/backups/

Windows:  %LOCALAPPDATA%\Stockbook\stockbook.db
          %LOCALAPPDATA%\Stockbook\session
          %LOCALAPPDATA%\Stockbook\backups/
```

`STOCKBOOK_DATA_DIR` overrides the folder if you used a custom location, and
`STOCKBOOK_DB` points at a specific database file.

A 1.x database can hold more than one company. Each account still signs in to
its own workspace on the same computer; switch accounts by signing out and back
in.

## Move a local workspace to Stockbook Cloud

Old passwords cannot move to the cloud because 1.x stored local scrypt hashes
that Supabase Auth cannot import. Your inventory can move at any time.

1. Open Stockbook in local mode and sign in to the workspace you want to move.
2. On **Export/import data**, choose **Export to JSON…** and save the file
   somewhere safe.
3. On the sign-in screen, switch to **Stockbook Cloud**.
4. Create a cloud account (or sign in to an existing one) and set its password.
5. On **Export/import data**, choose **Import from JSON file…** and select the
   file from step 2.
6. Verify the items, stock, purchases, revenue, and profit.

The JSON contains items, purchases, and sales only. It never contains the old
account email, password hash, or session, which is why it can be imported into
any account on either backend.

If the old database held several companies, repeat the export for each local
account and import each file into the matching cloud account. Each import
replaces only the workspace you are signed in to.

## What happens to the data

- **Local mode** keeps `stockbook.db`, `session`, and `backups/` exactly where
  1.x kept them.
- **Cloud mode** stores business data in Supabase Postgres and keeps only
  `session.json` on disk.
- The two modes keep separate sessions. Switching back to **On this computer**
  resumes the local account that was signed in before.
- Exports and import safety copies are ordinary JSON files. Keep them until you
  are satisfied with the new workspace.

## Rollback

Local mode is non-destructive, so the old database remains available as long as
you do not delete it. If you move a workspace to the cloud and want to return,
export from the cloud and import that file back into local mode; IDs are remapped
during import, but items, quantities, dates, and amounts are preserved.

Do not write to both a local ledger and a cloud workspace after moving and expect
them to merge. Pick one as the source of truth and use JSON export/import to move
between them.

## Standalone JSON export

The app handles normal 1.x data. If you need to inspect a database without
opening Stockbook, or extract JSON on a machine where the old database cannot be
opened, the optional Python utility still works. It uses only the standard
library.

```bash
# See which companies are in the file.
python3 supabase/migrate-sqlite-to-json.py --list \
  --db "$HOME/.local/share/Stockbook/stockbook.db"

# Export the only company, or choose one by id.
python3 supabase/migrate-sqlite-to-json.py \
  --db "$HOME/.local/share/Stockbook/stockbook.db" \
  --company 1 \
  --out "$HOME/stockbook-backup.json"
```

The output is the same format **Export/import data** accepts. On Windows, use the
`%LOCALAPPDATA%` path for `--db`.
