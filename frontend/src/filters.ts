import type { RangeFilters } from "./api";
import { addDaysISO, todayISO } from "./format";

export function isISODate(value: string) {
  if (!/^\d{4}-\d{2}-\d{2}$/.test(value)) return false;
  const date = new Date(`${value}T00:00:00.000Z`);
  return !Number.isNaN(date.getTime()) && date.toISOString().slice(0, 10) === value;
}

export function readPage(value: string | null) {
  const page = Number(value);
  return Number.isInteger(page) && page > 0 ? page : 1;
}

export function readParam(value: string | null, max = 100) {
  const raw = value?.trim();
  return raw ? raw.slice(0, max) : null;
}

function cleanCategory(value: string | null) {
  const category = value?.trim().slice(0, 50);
  return category ? category : null;
}

export function readRangeFilters(
  query: URLSearchParams,
  defaultDays = 30,
): RangeFilters {
  const category = cleanCategory(query.get("category"));
  const allTime = query.get("all") === "1";

  if (allTime) {
    return { from: null, to: null, allTime: true, category };
  }

  const rawFrom = query.get("from")?.trim() ?? "";
  const rawTo = query.get("to")?.trim() ?? "";
  let from: string | null = isISODate(rawFrom) ? rawFrom : null;
  let to: string | null = isISODate(rawTo) ? rawTo : null;

  if (!from && !to) {
    const today = todayISO();
    return {
      from: addDaysISO(today, -(defaultDays - 1)),
      to: today,
      allTime: false,
      category,
    };
  }

  if (from && to && from > to) {
    [from, to] = [to, from];
  }

  return { from, to, allTime: false, category };
}

export function filterQuery(filters: RangeFilters) {
  const params = new URLSearchParams();
  if (filters.allTime) {
    params.set("all", "1");
  } else {
    if (filters.from) params.set("from", filters.from);
    if (filters.to) params.set("to", filters.to);
  }
  if (filters.category) params.set("category", filters.category);
  return params;
}

export function rangePresetHref(
  pathname: string,
  days: number | "all",
  category: string | null,
) {
  const params = new URLSearchParams();
  if (days === "all") {
    params.set("all", "1");
  } else {
    const today = todayISO();
    params.set("from", addDaysISO(today, -(days - 1)));
    params.set("to", today);
  }
  if (category) params.set("category", category);
  const query = params.toString();
  return query ? `${pathname}?${query}` : pathname;
}

export function isPresetActive(filters: RangeFilters, days: number) {
  if (!filters.from || !filters.to) return false;
  const today = todayISO();
  return filters.to === today && filters.from === addDaysISO(today, -(days - 1));
}

export function withPage(
  pathname: string,
  params: URLSearchParams,
  page: number,
) {
  const next = new URLSearchParams(params);
  if (page > 1) next.set("page", String(page));
  else next.delete("page");
  const query = next.toString();
  return query ? `${pathname}?${query}` : pathname;
}

export function queryHref(pathname: string, params: URLSearchParams) {
  const query = params.toString();
  return query ? `${pathname}?${query}` : pathname;
}

export function formQuery(form: HTMLFormElement) {
  const params = new URLSearchParams();
  new FormData(form).forEach((value, key) => {
    if (typeof value === "string" && value.trim() !== "") {
      params.set(key, value.trim());
    }
  });
  return params;
}

export function toRangeInput(filters: RangeFilters, page: number) {
  return {
    from: filters.from ?? "",
    to: filters.to ?? "",
    allTime: filters.allTime,
    category: filters.category ?? "",
    page,
  };
}
