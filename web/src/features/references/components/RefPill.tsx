import { X } from "lucide-react";

export type RefPillProps = {
  label: string;
  className?: string;
  title?: string;
  onClick?: () => void;
  onHover?: (hovering: boolean) => void;
  onDelete?: () => void;
};

export function RefPill({
  label,
  className = "",
  title,
  onClick,
  onHover,
  onDelete,
}: RefPillProps) {
  const labelPadding = `py-0.5 pl-2.5 ${onDelete ? "pr-1" : "pr-2.5"}`;

  return (
    <span
      title={title}
      onMouseEnter={() => onHover?.(true)}
      onMouseLeave={() => onHover?.(false)}
      onFocus={() => onHover?.(true)}
      onBlur={() => onHover?.(false)}
      className={`inline-flex shrink-0 items-center whitespace-nowrap rounded-full border text-xs ${className}`}
    >
      {onClick ? (
        <button type="button" onClick={onClick} className={labelPadding}>
          {label}
        </button>
      ) : (
        <span className={labelPadding}>{label}</span>
      )}
      {onDelete && (
        <button
          type="button"
          aria-label={`Remove ${label}`}
          onClick={onDelete}
          className="rounded-full py-0.5 pl-0.5 pr-2 opacity-60 hover:opacity-100"
        >
          <X className="h-3 w-3" />
        </button>
      )}
    </span>
  );
}
