# Migrating from Stockbook 1.x (SQLite)

Stockbook 2.0 stores data in Supabase and does not read the old local
`stockbook.db`. Existing ledgers are migrated once through a JSON backup:

1. convert the old SQLite file to a Stockbook JSON backup;
2. restore that JSON into the new Supabase workspace.

The conversion is read-only. Keep the original database until you have verified
the import.

## Before you start

1. Close Stockbook 1.x so its write-ahead log is flushed.
2. Copy the old data somewhere safe, including any sidecar files:

   ```
   Linux:    ~/.local/share/Stockbook/stockbook.db
             ~/.local/share/Stockbook/stockbook.db-wal
             ~/.local/share/Stockbook/stockbook.db-shm

   Windows:  %LOCALAPPDATA%\Stockbook\stockbook.db
             %LOCALAPPDATA%\Stockbook\stockbook.db-wal
             %LOCALAPPDATA%\Stockbook\stockbook.db-shm
   ```

3. Set up Supabase and configure the app as described in
   [supabase-setup.md](supabase-setup.md).

## 1. Convert the database

The converter uses only the Python standard library.

```bash
# See which companies are in the file.
python3 scripts/migrate-sqlite-to-json.py --list

# Export the only company, or choose one by id.
python3 scripts/migrate-sqlite-to-json.py \
  --db "$HOME/.local/share/Stockbook/stockbook.db" \
  --company 1 \
  --out "$HOME/stockbook-backup.json"
```

On Windows, use the `%LOCALAPPDATA%` path for `--db`. If the database has just
one company, `--company` can be omitted.

The output is the same format the app writes from **Backup & restore**.

## 2. Import into Supabase

1. Start Stockbook 2.0 and sign in to the workspace that should receive the
   data.
2. Open **Backup & restore**.
3. Under **Restore**, choose the JSON file.
4. Confirm. The app validates the file, writes a safety snapshot, replaces the
   current workspace’s items, purchases, and sales, and signs you out.
5. Sign in again and check **Items**, **Inventory**, **Purchases**, and
   **Sales**.

Restoring replaces the signed-in workspace’s ledger. If you have already added
data to the Supabase workspace, export it first or use a fresh account.

## 3. Verify

Compare against the old app:

- number of items and stock on hand;
- purchase count and spend;
- sales count, revenue, and profit;
- low-stock items.

The migration preserves amounts (integer paise/cents), quantities, timestamps,
and notes. Product/purchase/sale IDs may differ in Supabase; relationships are
remapped during import.

## 4. Clear the old local data

After verification, delete the old local files if you no longer need them:

```
stockbook.db
stockbook.db-wal
stockbook.db-shm
session          (legacy 1.x session file)
backups/         (old SQLite snapshots)
```

Keep `config.json`, the current Supabase `session.json`, and the JSON backup you
imported. The JSON file is your rollback copy.

## Multiple companies

A 1.x database can hold more than one company. Export each one separately:

```bash
python3 scripts/migrate-sqlite-to-json.py --list
python3 scripts/migrate-sqlite-to-json.py --company 1 --out company-1.json
python3 scripts/migrate-sqlite-to-json.py --company 2 --out company-2.json
```

Sign in to the matching Supabase account for each workspace and restore the
corresponding file.

## Rollback

The old app still runs if you keep its database and binary. Do not write to both
the old SQLite file and the new Supabase workspace after migrating; choose one
as the source of truth.
