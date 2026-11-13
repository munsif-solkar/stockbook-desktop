import { useEffect, useState } from "react";
import {
  api,
  type BackendInfo,
  type ExportResult,
  type ImportResult,
  type UpdateInfo,
} from "../api";
import { useRefresh } from "../refresh";
import { useRouter } from "../router";

type Busy = "" | "export" | "import" | "check" | "update";

function formatBytes(bytes: number | undefined) {
  if (!bytes || bytes < 0) return "";
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
}

export function DataPage() {
  const { location } = useRouter();
  const { refresh } = useRefresh();
  const autoCheck = location.query.get("updates") === "1";

  const [busy, setBusy] = useState<Busy>("");
  const [version, setVersion] = useState("");
  const [dataPath, setDataPath] = useState("");
  const [backend, setBackend] = useState<BackendInfo | null>(null);
  const [exportResult, setExportResult] = useState<ExportResult | null>(null);
  const [exportError, setExportError] = useState("");
  const [importResult, setImportResult] = useState<ImportResult | null>(null);
  const [importError, setImportError] = useState("");
  const [update, setUpdate] = useState<UpdateInfo | null>(null);
  const [updateError, setUpdateError] = useState("");
  const [restarting, setRestarting] = useState(false);

  useEffect(() => {
    api.Version().then(setVersion).catch(() => setVersion(""));
    api.DataLocation().then(setDataPath).catch(() => setDataPath(""));
    api.BackendInfo().then(setBackend).catch(() => setBackend(null));
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

  async function handleExport() {
    setExportResult(null);
    setExportError("");
    setBusy("export");
    try {
      const result = await api.ExportData();
      if (!result.canceled) {
        if (result.error) setExportError(result.error);
        else setExportResult(result);
      }
    } catch (error) {
      setExportError(error instanceof Error ? error.message : String(error));
    } finally {
      setBusy("");
    }
  }

  async function handleImportFile() {
    const confirmed = window.confirm(
      "Import a Stockbook data file?\n\n" +
        "This replaces every item, purchase, and sale in your workspace with " +
        "the contents of that file. Your account stays signed in, and a safety " +
        "copy of your current data is saved first.",
    );
    if (!confirmed) return;

    setImportResult(null);
    setImportError("");
    setBusy("import");
    try {
      const result = await api.ImportData();
      if (result.canceled) return;
      if (result.error) {
        setImportError(result.error);
        return;
      }
      if (result.imported) {
        setImportResult(result);
        refresh();
        window.alert(
          "Import complete." +
            (result.safetyBackup
              ? `\n\nA copy of your previous data was saved at:\n${result.safetyBackup}`
              : ""),
        );
      }
    } catch (error) {
      setImportError(error instanceof Error ? error.message : String(error));
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
          <h1 className="page-title">Export/import data</h1>
          <p className="page-sub">
            Move your inventory to another computer or keep a safe copy.
          </p>
        </div>
        <span className="badge">
          {backend ? (backend.isLocal ? "Local workspace" : "Stockbook Cloud") : "Workspace"}
        </span>
      </div>

      <div className="backup-grid">
        <section className="panel backup-card">
          <div className="section-head">
            <div>
              <h2 className="section-title">Export data</h2>
              <p className="section-sub">Save a JSON copy of this workspace.</p>
            </div>
          </div>
          <div className="panel-body">
            <p className="backup-copy">
              The file contains your items, purchases, and sales — never your
              account details or password. It can be imported into a local
              workspace or a Stockbook Cloud account.
            </p>
            <div className="backup-path">
              <span>Stockbook data folder</span>
              <code title={dataPath}>{dataPath || "Loading…"}</code>
            </div>

            {exportResult ? (
              <div className="notice backup-result" role="status">
                <span>
                  Data exported ({formatBytes(exportResult.size)}).
                  <span className="backup-result-path">{exportResult.path}</span>
                </span>
              </div>
            ) : null}
            {exportError ? (
              <div className="form-error" role="alert">
                {exportError}
              </div>
            ) : null}

            <div className="backup-actions">
              <button
                className="btn btn-primary"
                type="button"
                onClick={handleExport}
                disabled={busy !== ""}
              >
                {busy === "export" ? "Exporting…" : "Export to JSON…"}
              </button>
            </div>
          </div>
        </section>

        <section className="panel backup-card">
          <div className="section-head">
            <div>
              <h2 className="section-title">Import data</h2>
              <p className="section-sub">Load a JSON file into this workspace.</p>
            </div>
          </div>
          <div className="panel-body">
            <p className="backup-copy">
              Choose a Stockbook JSON file. Every item, purchase, and sale is
              validated before anything changes, and a safety copy of the
              current workspace is kept in the <code>backups</code> folder in
              your Stockbook data directory.
            </p>
            <div className="backup-warning">
              Importing replaces the current items, purchases, and sales. Your
              account and sign-in stay exactly as they are.
            </div>

            {importResult?.safetyBackup ? (
              <div className="notice backup-result" role="status">
                <span>
                  Imported. Safety copy:
                  <span className="backup-result-path">{importResult.safetyBackup}</span>
                </span>
              </div>
            ) : null}
            {importError ? (
              <div className="form-error" role="alert">
                {importError}
              </div>
            ) : null}

            <div className="backup-actions">
              <button
                className="btn btn-secondary"
                type="button"
                onClick={handleImportFile}
                disabled={busy !== ""}
              >
                {busy === "import" ? "Importing…" : "Import from JSON file…"}
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
