import { useEffect, useState, type FormEvent } from "react";
import type { RangeFilters } from "../api";
import { isPresetActive, rangePresetHref } from "../filters";
import { Link, navigate } from "../router";

export function RangeFilter({
  pathname,
  filters,
  categories,
}: {
  pathname: string;
  filters: RangeFilters;
  categories: string[];
}) {
  const options = [...categories];
  if (filters.category && !options.includes(filters.category)) {
    options.push(filters.category);
  }

  const [from, setFrom] = useState(filters.from ?? "");
  const [to, setTo] = useState(filters.to ?? "");
  const [category, setCategory] = useState(filters.category ?? "");

  useEffect(() => {
    setFrom(filters.from ?? "");
    setTo(filters.to ?? "");
    setCategory(filters.category ?? "");
  }, [filters.from, filters.to, filters.category]);

  const presets: Array<{ days: number | "all"; label: string }> = [
    { days: 7, label: "7d" },
    { days: 30, label: "30d" },
    { days: 90, label: "90d" },
    { days: 365, label: "1y" },
    { days: "all", label: "All" },
  ];

  function apply(event: FormEvent) {
    event.preventDefault();
    let nextFrom = from;
    let nextTo = to;
    if (nextFrom && nextTo && nextFrom > nextTo) {
      [nextFrom, nextTo] = [nextTo, nextFrom];
    }
    const params = new URLSearchParams();
    if (nextFrom) params.set("from", nextFrom);
    if (nextTo) params.set("to", nextTo);
    if (category) params.set("category", category);
    const query = params.toString();
    navigate(query ? `${pathname}?${query}` : pathname);
  }

  const reset = "";

  return (
    <form className="toolbar" onSubmit={apply} aria-label="Filters">
      <div className="seg" aria-label="Quick date ranges">
        {presets.map((preset) => {
          const active =
            preset.days === "all"
              ? filters.allTime
              : isPresetActive(filters, preset.days);
          return (
            <Link
              key={preset.label}
              className={active ? "seg-item active" : "seg-item"}
              to={rangePresetHref(pathname, preset.days, filters.category)}
            >
              {preset.label}
            </Link>
          );
        })}
      </div>

      <div className="toolbar-fields">
        <label className="sr-only" htmlFor={`${pathname}-from`}>
          From date
        </label>
        <input
          className="input input-sm"
          id={`${pathname}-from`}
          name="from"
          type="date"
          value={from}
          onChange={(event) => setFrom(event.target.value)}
          aria-label="From date"
        />
        <span className="toolbar-sep" aria-hidden="true">
          →
        </span>
        <label className="sr-only" htmlFor={`${pathname}-to`}>
          To date
        </label>
        <input
          className="input input-sm"
          id={`${pathname}-to`}
          name="to"
          type="date"
          value={to}
          onChange={(event) => setTo(event.target.value)}
          aria-label="To date"
        />
        <label className="sr-only" htmlFor={`${pathname}-category`}>
          Category
        </label>
        <select
          className="select select-sm"
          id={`${pathname}-category`}
          name="category"
          value={category}
          onChange={(event) => setCategory(event.target.value)}
          aria-label="Category"
        >
          <option value="">All categories</option>
          {options.map((item) => (
            <option key={item} value={item}>
              {item}
            </option>
          ))}
        </select>
        <button className="btn btn-primary btn-sm" type="submit">
          Apply
        </button>
        <Link
          className="btn btn-ghost btn-sm"
          to={reset ? `${pathname}?${reset}` : pathname}
        >
          Reset
        </Link>
      </div>
    </form>
  );
}
