import { Link, useRouter } from "../router";

const items = [
  {
    href: "/overview",
    label: "Overview",
    icon: (
      <>
        <rect x="3" y="3" width="7" height="7" rx="1.5" />
        <rect x="14" y="3" width="7" height="7" rx="1.5" />
        <rect x="3" y="14" width="7" height="7" rx="1.5" />
        <rect x="14" y="14" width="7" height="7" rx="1.5" />
      </>
    ),
  },
  {
    href: "/items",
    label: "Items",
    icon: (
      <>
        <path d="M20.6 13.4 13.4 20.6a2 2 0 0 1-2.8 0l-7.2-7.2A2 2 0 0 1 3 12V5a2 2 0 0 1 2-2h7a2 2 0 0 1 1.4.6l7.2 7.2a2 2 0 0 1 0 2.6Z" />
        <circle cx="7.5" cy="7.5" r="1.2" />
      </>
    ),
  },
  {
    href: "/inventory",
    label: "Inventory",
    icon: (
      <>
        <path d="M21 8.2 12 3 3 8.2v7.6L12 21l9-5.2V8.2Z" />
        <path d="m3.3 8.3 8.7 5 8.7-5" />
        <path d="M12 13.3V21" />
      </>
    ),
  },
  {
    href: "/purchase",
    label: "Purchases",
    icon: (
      <>
        <path d="M3 4h2l2.2 10.3a2 2 0 0 0 2 1.7h7.8a2 2 0 0 0 2-1.6L20.5 7H6" />
        <circle cx="9.5" cy="19" r="1.2" />
        <circle cx="17.5" cy="19" r="1.2" />
      </>
    ),
  },
  {
    href: "/sales",
    label: "Sales",
    icon: (
      <>
        <path d="M4 19V10" />
        <path d="M10 19V4" />
        <path d="M16 19v-6" />
        <path d="M22 19H2" />
      </>
    ),
  },
  {
    href: "/data",
    label: "Export/import data",
    icon: (
      <>
        <ellipse cx="12" cy="5" rx="8" ry="3" />
        <path d="M4 5v6c0 1.7 3.6 3 8 3s8-1.3 8-3V5" />
        <path d="M4 11v6c0 1.7 3.6 3 8 3s8-1.3 8-3v-6" />
      </>
    ),
  },
];

export function SidebarNav() {
  const { location } = useRouter();

  return (
    <nav className="nav" aria-label="Main navigation">
      {items.map((item) => {
        const active =
          location.path === item.href || location.path.startsWith(`${item.href}/`);
        return (
          <Link
            className={active ? "nav-link active" : "nav-link"}
            to={item.href}
            key={item.href}
            aria-current={active ? "page" : undefined}
          >
            <svg
              className="nav-icon"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              strokeWidth="1.7"
              strokeLinecap="round"
              strokeLinejoin="round"
              aria-hidden="true"
            >
              {item.icon}
            </svg>
            {item.label}
          </Link>
        );
      })}
    </nav>
  );
}
