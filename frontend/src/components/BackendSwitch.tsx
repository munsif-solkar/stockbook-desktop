import { useEffect, useState } from "react";
import { api, type BackendInfo } from "../api";

// BackendSwitch lets people choose where Stockbook keeps their ledger. Local
// is the default and keeps everything on this computer; Stockbook Cloud syncs
// the workspace through an account. Switching reloads the window so the app
// starts cleanly on the newly selected backend.
export function BackendSwitch() {
  const [info, setInfo] = useState<BackendInfo | null>(null);
  const [error, setError] = useState("");
  const [switching, setSwitching] = useState("");

  useEffect(() => {
    let cancelled = false;
    api
      .BackendInfo()
      .then((result) => {
        if (!cancelled) setInfo(result);
      })
      .catch((cause: unknown) => {
        if (!cancelled) {
          setError(cause instanceof Error ? cause.message : String(cause));
        }
      });
    return () => {
      cancelled = true;
    };
  }, []);

  async function choose(mode: "local" | "cloud") {
    if (!info || switching) return;
    // Clicking the selected option normally does nothing, but after a boot
    // error it retries the same backend, which is how a temporary failure or a
    // corrupt backend.json gets recovered.
    if (info.mode === mode && !info.error) return;
    setError("");
    setSwitching(mode);
    try {
      const result = await api.SetBackend(mode);
      if (result.error) {
        setError(result.error);
        setSwitching("");
        return;
      }
      window.location.reload();
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : String(cause));
      setSwitching("");
    }
  }

  if (!info) {
    return (
      <div className="backend-switch backend-switch-loading">
        {error ? `Could not load storage options: ${error}` : "Loading storage options…"}
      </div>
    );
  }

  return (
    <div className="backend-block">
      <span className="backend-label">Where should Stockbook keep your data?</span>
      <div className="backend-options" role="group" aria-label="Storage location">
        <button
          type="button"
          className={`backend-option${info.isLocal ? " selected" : ""}`}
          onClick={() => choose("local")}
          disabled={Boolean(switching)}
          aria-pressed={info.isLocal}
        >
          <span className="backend-option-mark" aria-hidden="true">
            ⌂
          </span>
          <span className="backend-option-text">
            <span className="backend-option-title">On this computer</span>
            <span className="backend-option-note">
              Local workspace. No account sync, no internet needed.
            </span>
          </span>
          {switching === "local" ? <span className="backend-option-busy">Switching…</span> : null}
        </button>
        <button
          type="button"
          className={`backend-option${info.isCloud ? " selected" : ""}`}
          onClick={() => choose("cloud")}
          disabled={Boolean(switching)}
          aria-pressed={info.isCloud}
        >
          <span className="backend-option-mark" aria-hidden="true">
            ☁
          </span>
          <span className="backend-option-text">
            <span className="backend-option-title">Stockbook Cloud</span>
            <span className="backend-option-note">
              Sign in from any computer. Your workspace follows you.
            </span>
          </span>
          {switching === "cloud" ? <span className="backend-option-busy">Switching…</span> : null}
        </button>
      </div>
      <p className="backend-hint">
        Each choice keeps its own workspace. To move inventory between them, use
        Export/import data.
      </p>
      {error || info.error ? (
        <div className="form-error" role="alert">
          {error || info.error}
        </div>
      ) : null}
    </div>
  );
}
