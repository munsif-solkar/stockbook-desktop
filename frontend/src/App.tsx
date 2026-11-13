import { useEffect, useState, type ReactNode } from "react";
import { api, type Company } from "./api";
import { NewSellButton } from "./components/NewSellButton";
import { SidebarNav } from "./components/SidebarNav";
import { CURRENCY } from "./format";
import { RefreshProvider, useRefresh } from "./refresh";
import { Link, navigate, useRouter } from "./router";
import { BackupPage } from "./pages/BackupPage";
import { InventoryPage } from "./pages/InventoryPage";
import { ItemsPage } from "./pages/ItemsPage";
import { LoginPage } from "./pages/LoginPage";
import { NotFoundPage } from "./pages/NotFoundPage";
import { OverviewPage } from "./pages/OverviewPage";
import { PurchasePage } from "./pages/PurchasePage";
import { RegisterPage } from "./pages/RegisterPage";
import { SalesPage } from "./pages/SalesPage";

function AppShell({
  company,
  onLogout,
  children,
}: {
  company: Company;
  onLogout: () => void;
  children: ReactNode;
}) {
  const { refresh } = useRefresh();
  const today = new Intl.DateTimeFormat("en-US", {
    weekday: "long",
    month: "long",
    day: "numeric",
  }).format(new Date());

  return (
    <div className="shell">
      <aside className="sidebar">
        <Link className="brand" to="/overview">
          <span className="brand-mark" aria-hidden="true">
            S
          </span>
          Stockbook
        </Link>

        <div className="company-block">
          <span className="company-label">Workspace</span>
          <span className="company-name" title={company.name}>
            {company.name}
          </span>
        </div>

        <SidebarNav />

        <div className="sidebar-footer">
          <span className="sidebar-email" title={company.email}>
            {company.email}
          </span>
          <div className="sidebar-footer-actions">
            <Link className="sidebar-footer-link" to="/backup?updates=1">
              Check for updates
            </Link>
            <button className="btn btn-ghost btn-sm" type="button" onClick={onLogout}>
              Sign out
            </button>
          </div>
        </div>
      </aside>

      <div className="content">
        <header className="topbar">
          <span className="topbar-context">{today}</span>
          <NewSellButton currency={CURRENCY} onChanged={refresh} />
        </header>
        <main className="main">{children}</main>
      </div>
    </div>
  );
}

export function App() {
  const { location } = useRouter();
  const [company, setCompany] = useState<Company | null>(null);
  const [checking, setChecking] = useState(true);
  const [bootError, setBootError] = useState("");

  useEffect(() => {
    let cancelled = false;
    api
      .Session()
      .then((result) => {
        if (cancelled) return;
        if (result.error) setBootError(result.error);
        setCompany(result.company);
      })
      .catch((error: unknown) => {
        if (cancelled) return;
        setBootError(
          error instanceof Error ? error.message : String(error),
        );
      })
      .finally(() => {
        if (!cancelled) setChecking(false);
      });
    return () => {
      cancelled = true;
    };
  }, []);

  useEffect(() => {
    const unauthorized = () => setCompany(null);
    window.addEventListener("stockbook:unauthorized", unauthorized);
    return () => window.removeEventListener("stockbook:unauthorized", unauthorized);
  }, []);

  const path = location.path;

  useEffect(() => {
    if (checking) return;
    const isAuthRoute = path === "/login" || path === "/register";
    if (!company && !isAuthRoute) {
      navigate("/login", true);
      return;
    }
    if (company && (isAuthRoute || path === "/")) {
      navigate("/overview", true);
    }
  }, [checking, company, path]);

  function handleAuthenticated(result: Company) {
    setCompany(result);
    navigate("/overview", true);
  }

  async function handleLogout() {
    try {
      await api.Logout();
    } catch {
      // The local session is cleared even if the record was already gone.
    }
    setCompany(null);
    navigate("/login", true);
  }

  if (checking) {
    return (
      <main className="boot-screen">
        <span className="brand-mark" aria-hidden="true">
          S
        </span>
        <p>Opening Stockbook…</p>
      </main>
    );
  }

  if (bootError) {
    return (
      <main className="auth-page">
        <div className="auth-brand">
          <span className="brand-mark" aria-hidden="true">
            S
          </span>
          Stockbook
        </div>
        <div className="auth-card">
          <h1 className="auth-title">Cannot reach the backend</h1>
          <p className="auth-sub">{bootError}</p>
        </div>
      </main>
    );
  }

  if (path === "/login") {
    return <LoginPage onAuthenticated={handleAuthenticated} />;
  }

  if (path === "/register") {
    return <RegisterPage onAuthenticated={handleAuthenticated} />;
  }

  if (!company) {
    return <LoginPage onAuthenticated={handleAuthenticated} />;
  }

  let page: ReactNode;
  switch (path) {
    case "/":
    case "/overview":
      page = <OverviewPage />;
      break;
    case "/items":
      page = <ItemsPage />;
      break;
    case "/inventory":
      page = <InventoryPage />;
      break;
    case "/purchase":
      page = <PurchasePage />;
      break;
    case "/sales":
      page = <SalesPage />;
      break;
    case "/backup":
      page = <BackupPage />;
      break;
    case "/login":
    case "/register":
      page = <OverviewPage />;
      break;
    default:
      return <NotFoundPage />;
  }

  return (
    <RefreshProvider>
      <AppShell company={company} onLogout={handleLogout}>
        {page}
      </AppShell>
    </RefreshProvider>
  );
}
