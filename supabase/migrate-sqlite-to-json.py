#!/usr/bin/env python3
"""Convert a Stockbook 1.x SQLite database into a Stockbook JSON data file.

Stockbook 2.x can use Supabase or a local SQLite ledger. Stockbook 2.1 opens an
old local database directly in local mode; this script is optional. It is useful
for extracting a JSON copy when the desktop app cannot open the database, or when
you want to move a workspace to Stockbook Cloud from a machine without the app.
It reads the old `stockbook.db` and writes the JSON format that the app's
"Export/import data" page can import. It does not modify the database.

The output contains inventory data only: items, purchases, and sales. Account
emails and passwords are never written to the JSON file.

Usage:
    python3 supabase/migrate-sqlite-to-json.py --list
    python3 supabase/migrate-sqlite-to-json.py --out ~/stockbook-backup.json
    python3 supabase/migrate-sqlite-to-json.py \
        --db "/path/to/stockbook.db" --company 1 --out backup.json

Only the Python standard library is required.
"""

from __future__ import annotations

import argparse
import datetime
import json
import os
import sqlite3
import sys


def default_db_path() -> str:
    if os.name == "nt":
        base = os.environ.get("LOCALAPPDATA") or os.path.expanduser("~")
        return os.path.join(base, "Stockbook", "stockbook.db")
    xdg = os.environ.get("XDG_DATA_HOME")
    if xdg:
        return os.path.join(xdg, "Stockbook", "stockbook.db")
    return os.path.join(os.path.expanduser("~"), ".local", "share", "Stockbook", "stockbook.db")


def connect(path: str) -> sqlite3.Connection:
    if not os.path.exists(path):
        raise SystemExit(f"Database not found: {path}")
    connection = sqlite3.connect(f"file:{path}?mode=ro", uri=True)
    connection.row_factory = sqlite3.Row
    return connection


def require_tables(connection: sqlite3.Connection) -> None:
    names = {
        row[0]
        for row in connection.execute(
            "select name from sqlite_master where type = 'table'"
        )
    }
    missing = {"companies", "products", "purchases", "sales"} - names
    if missing:
        raise SystemExit(
            "This does not look like a Stockbook 1.x database. Missing tables: "
            + ", ".join(sorted(missing))
        )


def companies(connection: sqlite3.Connection) -> list[sqlite3.Row]:
    return list(
        connection.execute("select id, name, email from companies order by id")
    )


def build_backup(
    connection: sqlite3.Connection, company: sqlite3.Row
) -> dict:
    company_id = company["id"]

    products = [
        {
            "id": row["id"],
            "name": row["name"],
            "sku": row["sku"],
            "category": row["category"],
            "purchasePrice": row["purchase_price"],
            "sellPrice": row["sell_price"],
            "quantity": row["quantity"],
            "lowStockThreshold": row["low_stock_threshold"],
            "createdAt": row["created_at"],
            "updatedAt": row["updated_at"],
        }
        for row in connection.execute(
            """
            select id, name, sku, category, purchase_price, sell_price,
                   quantity, low_stock_threshold, created_at, updated_at
            from products
            where company_id = ?
            order by id
            """,
            (company_id,),
        )
    ]

    purchases = [
        {
            "id": row["id"],
            "productId": row["product_id"],
            "quantity": row["quantity"],
            "unitCost": row["unit_cost"],
            "totalCost": row["total_cost"],
            "purchasedAt": row["purchased_at"],
            "note": row["note"],
            "createdAt": row["created_at"],
        }
        for row in connection.execute(
            """
            select id, product_id, quantity, unit_cost, total_cost,
                   purchased_at, note, created_at
            from purchases
            where company_id = ?
            order by id
            """,
            (company_id,),
        )
    ]

    sales = [
        {
            "id": row["id"],
            "productId": row["product_id"],
            "quantity": row["quantity"],
            "unitPrice": row["unit_price"],
            "totalPrice": row["total_price"],
            "unitCost": row["unit_cost"],
            "totalCost": row["total_cost"],
            "profit": row["profit"],
            "paymentMode": row["payment_mode"],
            "soldAt": row["sold_at"],
            "note": row["note"],
            "createdAt": row["created_at"],
        }
        for row in connection.execute(
            """
            select id, product_id, quantity, unit_price, total_price,
                   unit_cost, total_cost, profit, payment_mode, sold_at,
                   note, created_at
            from sales
            where company_id = ?
            order by id
            """,
            (company_id,),
        )
    ]

    return {
        "format": "stockbook-backup",
        "version": 1,
        "exportedAt": datetime.datetime.now(datetime.timezone.utc)
        .isoformat()
        .replace("+00:00", "Z"),
        "products": products,
        "purchases": purchases,
        "sales": sales,
    }


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument(
        "--db",
        default=default_db_path(),
        help="path to the old stockbook.db (default: the platform data directory)",
    )
    parser.add_argument(
        "--company",
        type=int,
        help="company id to export when the database has more than one account",
    )
    parser.add_argument(
        "--out",
        help="output JSON path (default: stockbook-backup-YYYY-MM-DD.json)",
    )
    parser.add_argument(
        "--list",
        action="store_true",
        help="list companies in the database and exit",
    )
    args = parser.parse_args()

    connection = connect(args.db)
    try:
        require_tables(connection)
        accounts = companies(connection)
        if not accounts:
            raise SystemExit("The database has no companies to export.")

        if args.list:
            for account in accounts:
                print(f"{account['id']}\t{account['name']}\t{account['email']}")
            return 0

        selected = None
        if args.company is not None:
            selected = next((a for a in accounts if a["id"] == args.company), None)
            if selected is None:
                raise SystemExit(f"Company {args.company} was not found.")
        elif len(accounts) == 1:
            selected = accounts[0]
        else:
            print("This database contains more than one company. Choose one:", file=sys.stderr)
            for account in accounts:
                print(
                    f"  --company {account['id']}   {account['name']} <{account['email']}>",
                    file=sys.stderr,
                )
            return 2

        backup = build_backup(connection, selected)
    finally:
        connection.close()

    out = args.out or (
        "stockbook-backup-" + datetime.date.today().isoformat() + ".json"
    )
    out = os.path.abspath(os.path.expanduser(out))
    os.makedirs(os.path.dirname(out), exist_ok=True)
    with open(out, "w", encoding="utf-8") as handle:
        json.dump(backup, handle, indent=2)
        handle.write("\n")
    try:
        os.chmod(out, 0o600)
    except OSError:
        pass

    print(f"Wrote {out}")
    print(
        "  company:   {name}\n"
        "  products:  {products}\n"
        "  purchases: {purchases}\n"
        "  sales:     {sales}".format(
            name=selected["name"],
            products=len(backup["products"]),
            purchases=len(backup["purchases"]),
            sales=len(backup["sales"]),
        )
    )
    print(
        "\nNext: open Stockbook 2.x, sign in, then Export/import data -> Import "
        "from JSON file and choose this file."
    )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
