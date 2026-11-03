import { Link } from "../router";

export function NotFoundPage() {
  return (
    <main className="auth-page">
      <div className="auth-brand">
        <span className="brand-mark" aria-hidden="true">
          S
        </span>
        Stockbook
      </div>
      <div className="auth-card">
        <h1 className="auth-title">Page not found</h1>
        <p className="auth-sub">
          That page does not exist or the item has been removed.
        </p>
        <Link className="btn btn-primary btn-block" to="/overview">
          Back to overview
        </Link>
      </div>
    </main>
  );
}
