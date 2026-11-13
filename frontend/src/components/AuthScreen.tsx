import { useEffect, useState } from "react";
import { api, type BackendInfo, type Company } from "../api";
import { Link } from "../router";
import { AuthForm } from "./AuthForm";
import { BackendSwitch } from "./BackendSwitch";

// AuthScreen is shared by the sign-in and registration routes. It shows the
// backend choice first, then copy and a form that match that backend.
export function AuthScreen({
  formMode,
  onAuthenticated,
}: {
  formMode: "login" | "register";
  onAuthenticated: (company: Company) => void;
}) {
  const [backend, setBackend] = useState<BackendInfo | null>(null);

  useEffect(() => {
    let cancelled = false;
    api
      .BackendInfo()
      .then((result) => {
        if (!cancelled) setBackend(result);
      })
      .catch(() => {
        if (!cancelled) setBackend(null);
      });
    return () => {
      cancelled = true;
    };
  }, []);

  const isLocal = backend?.isLocal ?? true;
  const isRegister = formMode === "register";

  const title = isRegister
    ? isLocal
      ? "Create your local workspace"
      : "Create your cloud workspace"
    : "Welcome back";

  const subtitle = isRegister
    ? isLocal
      ? "Your items, purchases, and sales stay on this computer."
      : "A clean ledger for your stock, purchases, and sales."
    : isLocal
      ? "Sign in to the workspace saved on this computer."
      : "Sign in to your Stockbook Cloud workspace.";

  return (
    <main className="auth-page">
      <Link className="auth-brand" to="/">
        <span className="brand-mark" aria-hidden="true">
          S
        </span>
        Stockbook
      </Link>
      <div className="auth-card">
        <BackendSwitch />
        <h1 className="auth-title">{title}</h1>
        <p className="auth-sub">{subtitle}</p>
        <AuthForm mode={formMode} onAuthenticated={onAuthenticated} />
      </div>
      <p className="auth-note">
        {isLocal
          ? "Local first: your inventory is stored on this computer. Export a JSON file any time to move it."
          : "Your workspace is stored in Stockbook Cloud and secured with your account."}
      </p>
    </main>
  );
}
