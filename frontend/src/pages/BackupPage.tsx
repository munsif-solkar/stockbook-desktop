import { useEffect, useState } from "react";
import { api, type BackupResult, type RestoreResult, type UpdateInfo } from "../api";
import { navigate, useRouter } from "../router";

type Busy = "" | "backup" | "restore" | "check" | "update";

function formatBytes(bytes: number | undefined) {
  if (!bytes || bytes < 0) return "";
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
}

export function BackupPage() {
  const { location } = useRouter();
  const autoCheck = location.query.get("updates") === "1";

  const [busy, setBusy] = useState<Busy>("");
  const [version, setVersion] = useState("");
  const [dataPath, setDataPath] = useState("");
  const [backup, setBackup] = useState<BackupResult | null>(null);
  const [backupError, setBackupError] = useState("");
  const [restore, setRestore] = useState<RestoreResult | null>(null);
  const [restoreError, setRestoreError] = useState("");
  const [update, setUpdate] = useState<UpdateInfo | null>(null);
  const [updateError, setUpdateError] = useState("");
  const [restarting, setRestarting] = useState(false);

  useEffect(() => {
    api.Version().then(setVersion).catch(() => setVersion(""));
    api.DataLocation().then(setDataPath).catch(() => setDataPath(""));
  }, []);

  useEffect(() => {
    if (!autoCheck) return;
    let cancelled = false;
    setBusy("check");
    api
      .CheckForUpdates()
      .then((result) => {
        if (!cancelled) setUpdate(result);
      })
      .catch((error: unknown) => {
        if (!cancelled) {
          setUpdateError(error instanceof Error ? error.message : String(error));
        }
      })
      .finally(() => {
        if (!cancelled) setBusy("");
      });
    return () => {
      cancelled = true;
    };
    // Re-run when the sidebar shortcut activates the ?updates=1 query,
    // including when the user is already on this page.
  }, [autoCheck]);

  async function handleBackup() {
    setBackup(null);
    setBackupError("");
    setBusy("backup");
    try {
      const result = await api.BackupDatabase();
      if (!result.canceled) {
        if (result.error) setBackupError(result.error);
        else setBackup(result);
      }
    } catch (error) {
      setBackupError(error instanceof Error ? error.message : String(error));
    } finally {
      setBusy("");
    }
  }

  async function handleRestore() {
    const confirmed = window.confirm(
      "Restore a Stockbook backup?\n\n" +
        "This replaces every account, item, purchase, and sale in the local " +
        "database with the contents of the backup file. A safety snapshot of " +
        "the current data is kept and you will need to sign in again.",
    );
    if (!confirmed) return;

    setRestore(null);
    setRestoreError("");
    setBusy("restore");
    try {
      const result = await api.RestoreDatabase();
      if (result.canceled) return;
      if (result.error) {
        setRestoreError(result.error);
        return;
      }
      if (result.restored) {
        setRestore(result);
        window.alert(
          "Restore complete. Sign in with the account from the restored backup." +
            (result.safetyBackup
              ? `\n\nA copy of your previous data was saved at:\n${result.safetyBackup}`
              : ""),
        );
        window.dispatchEvent(new Event("stockbook:unauthorized"));
        navigate("/login", true);
      }
    } catch (error) {
      setRestoreError(error instanceof Error ? error.message : String(error));
    } finally {
      setBusy("");
    }
  }

  async function handleCheck() {
    setUpdate(null);
    setUpdateError("");
    setBusy("check");
    try {
      const result = await api.CheckForUpdates();
      setUpdate(result);
    } catch (error) {
      setUpdateError(error instanceof Error ? error.message : String(error));
    } finally {
      setBusy("");
    }
  }

  async function handleUpdate() {
    if (!update?.updateAvailable) return;
    const confirmed = window.confirm(
      `Install Stockbook ${update.latestVersion ?? ""}?\n\n` +
        "The update is downloaded from the official GitHub release, verified " +
        "against its published checksum, and installed over this copy. " +
        "Stockbook will restart automatically.",
    );
    if (!confirmed) return;

    setUpdateError("");
    setBusy("update");
    try {
      const result = await api.ApplyUpdate();
      if (result.error) {
        setUpdateError(result.error);
        if (result.updated) setUpdate({ ...update, updateAvailable: false });
        return;
      }
      setRestarting(true);
    } catch (error) {
      setUpdateError(error instanceof Error ? error.message : String(error));
    } finally {
      setBusy("");
    }
  }

  return (
    <>
      <div className="page-head">
        <div>
          <h1 className="page-title">Backup &amp; restore</h1>
          <p className="page-sub">
            Keep a safe copy of your ledger, or move it to another computer.
          </p>
        </div>
      </div>

      <div className="backup-grid">
        <section className="panel backup-card">
          <div className="section-head">
            <div>
              <h2 className="section-title">Backup</h2>
              <p className="section-sub">Download a copy of the whole local database.</p>
            </div>
          </div>
          <div className="panel-body">
            <p className="backup-copy">
              The backup is an ordinary SQLite <code>.db</code> file. It includes
              every account, item, purchase, and sale stored by Stockbook on this
              computer. Keep it on another drive or in cloud storage.
            </p>
            <div className="backup-path">
              <span>Live data file</span>
              <code title={dataPath}>{dataPath || "Loading…"}</code>
            </div>

            {backup ? (
              <div className="notice backup-result" role="status">
                <span>
                  Backup saved ({formatBytes(backup.size)}).
                  <span className="backup-result-path">{backup.path}</span>
                </span>
              </div>
            ) : null}
            {backupError ? (
              <div className="form-error" role="alert">
                {backupError}
              </div>
            ) : null}

            <div className="backup-actions">
              <button
                className="btn btn-primary"
                type="button"
                onClick={handleBackup}
                disabled={busy !== ""}
              >
                {busy === "backup" ? "Creating backup…" : "Backup now"}
              </button>
            </div>
          </div>
        </section>

        <section className="panel backup-card">
          <div className="section-head">
            <div>
              <h2 className="section-title">Restore</h2>
              <p className="section-sub">Replace this computer's data from a backup.</p>
            </div>
          </div>
          <div className="panel-body">
            <p className="backup-copy">
              Choose a backup created by Stockbook. The file is checked before
              anything changes, and a safety snapshot of the current database is
              kept in the <code>backups</code> folder beside your data.
            </p>
            <div className="backup-warning">
              Restoring replaces the current accounts, items, purchases, and sales,
              then signs you out. Other copies of Stockbook must be closed.
            </div>

            {restore?.safetyBackup ? (
              <div className="notice backup-result" role="status">
                <span>
                  Restored. Safety snapshot:
                  <span className="backup-result-path">{restore.safetyBackup}</span>
                </span>
              </div>
            ) : null}
            {restoreError ? (
              <div className="form-error" role="alert">
                {restoreError}
              </div>
            ) : null}

            <div className="backup-actions">
              <button
                className="btn btn-secondary"
                type="button"
                onClick={handleRestore}
                disabled={busy !== ""}
              >
                {busy === "restore" ? "Restoring…" : "Restore from file…"}
              </button>
            </div>
          </div>
        </section>
      </div>

      <section className="panel panel-spaced backup-card" id="updates">
        <div className="section-head">
          <div>
            <h2 className="section-title">App updates</h2>
            <p className="section-sub">
              Check GitHub releases for a newer Stockbook build.
            </p>
          </div>
          <span className="badge">
            {version ? `Version ${version}` : "Version"}
          </span>
        </div>
        <div className="panel-body">
          {restarting ? (
            <div className="notice" role="status">
              <span>Update installed. Stockbook is restarting…</span>
            </div>
          ) : null}

          {update?.error ? (
            <div className="form-error" role="alert">
              {update.error}
            </div>
          ) : null}
          {updateError ? (
            <div className="form-error" role="alert">
              {updateError}
            </div>
          ) : null}

          {update && !update.error ? (
            <div className="update-summary">
              <p>
                {update.updateAvailable ? (
                  <>
                    A newer version is available:{" "}
                    <strong>{update.latestVersion}</strong>
                    {update.releaseName ? ` — ${update.releaseName}` : ""}.
                  </>
                ) : (
                  <>
                    You are on the latest version
                    {update.latestVersion ? ` (${update.latestVersion})` : ""}.
                  </>
                )}
              </p>
              {update.updateAvailable && update.notes ? (
                <div className="update-notes">{update.notes}</div>
              ) : null}
              {update.releaseUrl ? (
                <p className="update-link">
                  Release page: <code>{update.releaseUrl}</code>
                </p>
              ) : null}
            </div>
          ) : (
            <p className="backup-copy">
              Stockbook compares its version with the latest public release on
              GitHub. Updates are downloaded only from the official repository and
              verified against the release checksum.
            </p>
          )}

          <div className="backup-actions">
            {update?.updateAvailable && update.assetName ? (
              <button
                className="btn btn-primary"
                type="button"
                onClick={handleUpdate}
                disabled={busy !== "" || restarting}
              >
                {busy === "update"
                  ? "Downloading and installing…"
                  : `Install ${update.latestVersion ?? "update"}`}
              </button>
            ) : null}
            <button
              className="btn btn-secondary"
              type="button"
              onClick={handleCheck}
              disabled={busy !== "" || restarting}
            >
              {busy === "check" ? "Checking…" : "Check for updates"}
            </button>
          </div>
        </div>
      </section>
    </>
  );
}
