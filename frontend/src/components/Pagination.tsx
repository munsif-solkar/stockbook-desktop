import { withPage } from "../filters";
import { Link } from "../router";

export function Pagination({
  pathname,
  params,
  page,
  totalPages,
  total,
  label = "records",
}: {
  pathname: string;
  params: URLSearchParams;
  page: number;
  totalPages: number;
  total: number;
  label?: string;
}) {
  if (total === 0) return null;

  return (
    <div className="pagination">
      <span>
        {total} {label} · page {page} of {totalPages}
      </span>
      <div className="pagination-actions">
        {page > 1 ? (
          <Link
            className="btn btn-secondary btn-sm"
            to={withPage(pathname, params, page - 1)}
          >
            Previous
          </Link>
        ) : (
          <span
            className="btn btn-secondary btn-sm"
            aria-disabled="true"
            style={{ opacity: 0.45 }}
          >
            Previous
          </span>
        )}
        {page < totalPages ? (
          <Link
            className="btn btn-secondary btn-sm"
            to={withPage(pathname, params, page + 1)}
          >
            Next
          </Link>
        ) : (
          <span
            className="btn btn-secondary btn-sm"
            aria-disabled="true"
            style={{ opacity: 0.45 }}
          >
            Next
          </span>
        )}
      </div>
    </div>
  );
}
