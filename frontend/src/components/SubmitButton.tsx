import type { ReactNode } from "react";

export function SubmitButton({
  children,
  pendingLabel = "Saving…",
  className = "btn btn-primary",
  disabled = false,
  pending = false,
  onClick,
}: {
  children: ReactNode;
  pendingLabel?: string;
  className?: string;
  disabled?: boolean;
  pending?: boolean;
  onClick?: () => void;
}) {
  return (
    <button
      className={className}
      type="submit"
      disabled={pending || disabled}
      onClick={onClick}
    >
      {pending ? pendingLabel : children}
    </button>
  );
}
