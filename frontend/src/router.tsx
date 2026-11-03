import {
  createContext,
  useContext,
  useEffect,
  useMemo,
  useState,
  type AnchorHTMLAttributes,
  type ReactNode,
} from "react";

export type Location = { path: string; query: URLSearchParams };

export function parseLocation(): Location {
  const raw = window.location.hash.replace(/^#/, "") || "/overview";
  const [pathPart, queryPart] = raw.split("?");
  const path = pathPart && pathPart.startsWith("/") ? pathPart : "/overview";
  return { path, query: new URLSearchParams(queryPart ?? "") };
}

export function navigate(to: string, replace = false) {
  const clean = to.startsWith("/") ? to : `/${to}`;
  const target = `#${clean}`;
  if (replace) {
    window.history.replaceState(null, "", target);
    window.dispatchEvent(new Event("stockbook:navigate"));
    return;
  }
  if (window.location.hash === target) {
    window.dispatchEvent(new Event("stockbook:navigate"));
    return;
  }
  window.location.hash = target;
}

type RouterValue = {
  location: Location;
  navigate: typeof navigate;
};

const RouterContext = createContext<RouterValue | null>(null);

export function RouterProvider({ children }: { children: ReactNode }) {
  const [location, setLocation] = useState<Location>(parseLocation);

  useEffect(() => {
    const update = () => setLocation(parseLocation());
    window.addEventListener("hashchange", update);
    window.addEventListener("stockbook:navigate", update);
    return () => {
      window.removeEventListener("hashchange", update);
      window.removeEventListener("stockbook:navigate", update);
    };
  }, []);

  const value = useMemo(() => ({ location, navigate }), [location]);
  return <RouterContext.Provider value={value}>{children}</RouterContext.Provider>;
}

export function useRouter() {
  const value = useContext(RouterContext);
  if (!value) throw new Error("Router is not available.");
  return value;
}

type LinkProps = AnchorHTMLAttributes<HTMLAnchorElement> & {
  to: string;
  replace?: boolean;
};

export function Link({ to, replace, onClick, children, ...rest }: LinkProps) {
  return (
    <a
      {...rest}
      href={`#${to.startsWith("/") ? to : `/${to}`}`}
      onClick={(event) => {
        onClick?.(event);
        if (
          event.defaultPrevented ||
          event.button !== 0 ||
          event.metaKey ||
          event.ctrlKey ||
          event.shiftKey ||
          event.altKey
        ) {
          return;
        }
        event.preventDefault();
        navigate(to, replace);
      }}
    >
      {children}
    </a>
  );
}
