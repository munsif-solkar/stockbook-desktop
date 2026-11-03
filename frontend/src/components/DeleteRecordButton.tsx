import { useRef, useState, type FormEvent } from "react";
import { api } from "../api";
import { SubmitButton } from "./SubmitButton";

export function DeleteRecordButton({
  kind,
  id,
  productName,
  quantity,
  summary,
  onDeleted,
}: {
  kind: "purchase" | "sale";
  id: number;
  productName: string;
  quantity: number;
  summary: string;
  onDeleted: () => void;
}) {
  const dialogRef = useRef<HTMLDialogElement>(null);
  const [error, setError] = useState("");
  const [pending, setPending] = useState(false);
  const titleId = `delete-${kind}-${id}-title`;

  const label =
    kind === "purchase"
      ? `Delete purchase for ${productName}`
      : `Delete sale of ${productName}`;

  function open() {
    setError("");
    dialogRef.current?.showModal();
  }

  async function submit(event: FormEvent) {
    event.preventDefault();
    setPending(true);
    setError("");
    try {
      const result =
        kind === "purchase"
          ? await api.DeletePurchase(id)
          : await api.DeleteSale(id);
      if (result.error) {
        setError(result.error);
        return;
      }
      dialogRef.current?.close();
      onDeleted();
    } catch (deleteError) {
      setError(
        deleteError instanceof Error ? deleteError.message : String(deleteError),
      );
    } finally {
      setPending(false);
    }
  }

  return (
    <>
      <button
        className="btn btn-ghost btn-sm row-delete"
        type="button"
        aria-label={label}
        title={label}
        onClick={open}
      >
        <svg
          className="btn-icon"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          strokeWidth="2"
          strokeLinecap="round"
          strokeLinejoin="round"
          aria-hidden="true"
        >
          <path d="M4 7h16M9 7V5a1 1 0 0 1 1-1h4a1 1 0 0 1 1 1v2M6 7l1 13a1 1 0 0 0 1 1h8a1 1 0 0 0 1-1l1-13M10 11v6M14 11v6" />
        </svg>
      </button>

      <dialog
        ref={dialogRef}
        aria-labelledby={titleId}
        onClick={(event) => {
          if (event.target === event.currentTarget) dialogRef.current?.close();
        }}
      >
        <div className="dialog-card">
          <div className="dialog-head">
            <div>
              <h2 className="dialog-title" id={titleId}>
                {kind === "purchase" ? "Delete purchase?" : "Delete sale?"}
              </h2>
              <p className="dialog-sub">
                {kind === "purchase"
                  ? "The record is removed and stock is reduced."
                  : "The record is removed and stock is restored."}
              </p>
            </div>
            <button
              className="dialog-close"
              type="button"
              onClick={() => dialogRef.current?.close()}
              aria-label="Close"
            >
              ×
            </button>
          </div>

          <form onSubmit={submit}>
            <div className="dialog-body">
              {error ? (
                <div className="form-error" role="alert">
                  {error}
                </div>
              ) : null}

              <div className="confirm-summary">
                <div className="row-title">{productName}</div>
                <div className="row-sub">{summary}</div>
              </div>

              <p className="confirm-copy">
                {kind === "purchase"
                  ? `${quantity} unit${
                      quantity === 1 ? "" : "s"
                    } will be removed from stock. If those units have already been sold, this purchase cannot be deleted.`
                  : `${quantity} unit${
                      quantity === 1 ? "" : "s"
                    } will be added back to stock. The sale will no longer count toward revenue or profit.`}
              </p>
            </div>

            <div className="dialog-foot">
              <button
                className="btn btn-secondary"
                type="button"
                onClick={() => dialogRef.current?.close()}
              >
                Cancel
              </button>
              <SubmitButton
                className="btn btn-danger"
                pending={pending}
                pendingLabel="Deleting…"
              >
                Delete
              </SubmitButton>
            </div>
          </form>
        </div>
      </dialog>
    </>
  );
}
