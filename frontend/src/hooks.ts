import { useEffect, useState } from "react";

export type LoaderState<T> = {
  data: T | null;
  error: string;
  loading: boolean;
};

export function useLoader<T>(
  loader: () => Promise<T>,
  deps: ReadonlyArray<unknown>,
): LoaderState<T> {
  const [state, setState] = useState<LoaderState<T>>({
    data: null,
    error: "",
    loading: true,
  });

  useEffect(() => {
    let cancelled = false;
    setState((current) => ({ ...current, loading: true, error: "" }));
    loader()
      .then((data) => {
        if (!cancelled) setState({ data, error: "", loading: false });
      })
      .catch((error: unknown) => {
        if (!cancelled) {
          setState((current) => ({
            ...current,
            loading: false,
            error: error instanceof Error ? error.message : String(error),
          }));
        }
      });
    return () => {
      cancelled = true;
    };
    // The loader is recreated every render; callers supply explicit deps.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, deps);

  return state;
}
