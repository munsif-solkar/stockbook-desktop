import { AuthForm } from "../components/AuthForm";
import { Link } from "../router";
import type { Company } from "../api";

export function LoginPage({
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
        <h1 className="auth-title">Welcome back</h1>
        <p className="auth-sub">Sign in to manage stock, purchases, and sales.</p>
        <AuthForm mode="login" onAuthenticated={onAuthenticated} />
      </div>
      <p className="auth-note">
        Your workspace is stored in Supabase and secured with your account.
      </p>
    </main>
  );
}
