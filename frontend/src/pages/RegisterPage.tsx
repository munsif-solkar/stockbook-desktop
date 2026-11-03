import { AuthForm } from "../components/AuthForm";
import { Link } from "../router";
import type { Company } from "../api";

export function RegisterPage({
  onAuthenticated,
}: {
  onAuthenticated: (company: Company) => void;
}) {
  return (
    <main className="auth-page">
      <Link className="auth-brand" to="/">
        <span className="brand-mark" aria-hidden="true">
          S
        </span>
        Stockbook
      </Link>
      <div className="auth-card">
        <h1 className="auth-title">Create your workspace</h1>
        <p className="auth-sub">
          A clean ledger for your stock, purchases, and sales. No setup needed.
        </p>
        <AuthForm mode="register" onAuthenticated={onAuthenticated} />
      </div>
      <p className="auth-note">
        Every account gets its own private inventory, separated from other businesses.
      </p>
    </main>
  );
}
