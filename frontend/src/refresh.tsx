import {
  createContext,
  useCallback,
  useContext,
  useMemo,
  useState,
  type ReactNode,
} from "react";

type RefreshValue = {
  refreshKey: number;
  refresh: () => void;
};

const RefreshContext = createContext<RefreshValue | null>(null);

export function RefreshProvider({ children }: { children: ReactNode }) {
  const [refreshKey, setRefreshKey] = useState(0);
  const refresh = useCallback(() => setRefreshKey((value) => value + 1), []);
  const value = useMemo(() => ({ refreshKey, refresh }), [refreshKey, refresh]);
  return (
    <RefreshContext.Provider value={value}>{children}</RefreshContext.Provider>
  );
}

export function useRefresh() {
  const value = useContext(RefreshContext);
  if (!value) throw new Error("Refresh context is not available.");
  return value;
}
