/**
 * Small reusable UI primitives. Component boundaries follow behavior,
 * not pixels: Button / Badge / EmptyState / Menu / Modal / ResizablePanel.
 */

import {
  useCallback,
  useEffect,
  useRef,
  useState,
  type ReactNode,
} from "react";
import { cn } from "../utils/cn";
import { clamp } from "../utils/helpers";
import { Icon, type IconName } from "./icons";

/* ------------------------------------------------------------------ */
/* Button                                                              */
/* ------------------------------------------------------------------ */

type ButtonVariant = "solid" | "soft" | "outline" | "ghost" | "green";

export function Button({
  variant = "outline",
  size = "md",
  icon,
  children,
  onClick,
  disabled,
  className,
  title,
}: {
  variant?: ButtonVariant;
  size?: "sm" | "md" | "lg";
  icon?: IconName;
  children?: ReactNode;
  onClick?: () => void;
  disabled?: boolean;
  className?: string;
  title?: string;
}) {
  const base =
    "inline-flex items-center justify-center gap-1.5 rounded-lg font-medium transition-colors duration-100 select-none whitespace-nowrap";
  const sizes = {
    sm: "h-7 px-2.5 text-[12px]",
    md: "h-8.5 px-3.5 text-[13px]",
    lg: "h-10 px-4 text-[13.5px]",
  }[size];
  const variants: Record<ButtonVariant, string> = {
    solid: "bg-iris-600 text-white hover:bg-iris-700 disabled:bg-iris-300",
    soft: "bg-iris-100 text-iris-700 hover:bg-iris-200/70 disabled:opacity-50",
    green: "bg-moss-100 text-moss-700 hover:bg-moss-200/70 disabled:opacity-50",
    outline:
      "border border-line bg-paper text-ink-700 hover:border-iris-300 hover:text-iris-700 disabled:opacity-40",
    ghost: "text-ink-500 hover:bg-cream-100 hover:text-ink-900 disabled:opacity-40",
  };
  return (
    <button
      type="button"
      title={title}
      className={cn(base, sizes, variants[variant], disabled && "cursor-not-allowed", className)}
      onClick={onClick}
      disabled={disabled}
    >
      {icon && <Icon name={icon} size={size === "sm" ? 13 : 15} />}
      {children}
    </button>
  );
}

/* ------------------------------------------------------------------ */
/* Icon button (square, for toolbars)                                  */
/* ------------------------------------------------------------------ */

export function IconBtn({
  name,
  onClick,
  title,
  active,
  disabled,
  size = 16,
  className,
}: {
  name: IconName;
  onClick?: (e: React.MouseEvent) => void;
  title?: string;
  active?: boolean;
  disabled?: boolean;
  size?: number;
  className?: string;
}) {
  return (
    <button
      type="button"
      title={title}
      aria-label={title}
      onClick={onClick}
      disabled={disabled}
      className={cn(
        "inline-flex h-7 w-7 items-center justify-center rounded-md transition-colors",
        active ? "bg-iris-100 text-iris-700" : "text-ink-500 hover:bg-cream-100 hover:text-ink-900",
        disabled && "pointer-events-none opacity-30",
        className,
      )}
    >
      <Icon name={name} size={size} />
    </button>
  );
}

/* ------------------------------------------------------------------ */
/* Badge                                                               */
/* ------------------------------------------------------------------ */

export function Badge({
  tone = "neutral",
  children,
  className,
}: {
  tone?: "moss" | "iris" | "blue" | "hay" | "neutral";
  children: ReactNode;
  className?: string;
}) {
  const tones = {
    moss: "bg-moss-100 text-moss-700",
    iris: "bg-iris-100 text-iris-700",
    blue: "bg-sky-100 text-sky-700",
    hay: "bg-hay-100 text-hay-700",
    neutral: "bg-cream-100 text-ink-500",
  };
  return (
    <span className={cn("inline-flex items-center rounded-full px-2 py-0.5 text-[11px] font-medium", tones[tone], className)}>
      {children}
    </span>
  );
}

/* ------------------------------------------------------------------ */
/* Empty state                                                         */
/* ------------------------------------------------------------------ */

export function EmptyState({
  icon,
  title,
  hint,
  children,
}: {
  icon: IconName;
  title: string;
  hint?: string;
  children?: ReactNode;
}) {
  return (
    <div className="fade-in flex flex-col items-center justify-center gap-2 px-6 py-10 text-center">
      <div className="flex h-10 w-10 items-center justify-center rounded-full bg-cream-100 text-ink-400">
        <Icon name={icon} size={18} />
      </div>
      <p className="text-[13px] font-medium text-ink-700">{title}</p>
      {hint && <p className="max-w-[240px] text-[12px] leading-relaxed text-mute">{hint}</p>}
      {children}
    </div>
  );
}

/* ------------------------------------------------------------------ */
/* Menu (dropdown / contextual three-dot menu)                         */
/* ------------------------------------------------------------------ */

export function Menu({
  button,
  children,
  align = "right",
  width = 200,
}: {
  button: (open: boolean) => ReactNode;
  children: (close: () => void) => ReactNode;
  align?: "left" | "right";
  width?: number;
}) {
  const [open, setOpen] = useState(false);
  const ref = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!open) return;
    const onDown = (e: MouseEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node)) setOpen(false);
    };
    document.addEventListener("mousedown", onDown);
    return () => document.removeEventListener("mousedown", onDown);
  }, [open]);

  return (
    <div className="relative" ref={ref}>
      <div onClick={() => setOpen((o) => !o)}>{button(open)}</div>
      {open && (
        <div
          className={cn(
            "fade-in absolute z-50 mt-1 overflow-hidden rounded-lg border border-line bg-paper py-1 shadow-lg shadow-ink-900/8",
            align === "right" ? "right-0" : "left-0",
          )}
          style={{ width }}
        >
          {children(() => setOpen(false))}
        </div>
      )}
    </div>
  );
}

export function MenuItem({
  icon,
  label,
  onClick,
  hint,
  tone = "default",
}: {
  icon?: IconName;
  label: string;
  onClick?: () => void;
  hint?: string;
  tone?: "default" | "accent";
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      className={cn(
        "flex w-full items-center gap-2.5 px-3 py-1.5 text-left text-[12.5px] transition-colors",
        tone === "accent" ? "text-iris-700 hover:bg-iris-50" : "text-ink-700 hover:bg-cream-100",
      )}
    >
      {icon && <Icon name={icon} size={14} className="shrink-0 opacity-70" />}
      <span className="flex-1">{label}</span>
      {hint && <span className="text-[11px] text-mute">{hint}</span>}
    </button>
  );
}

/* ------------------------------------------------------------------ */
/* Modal                                                               */
/* ------------------------------------------------------------------ */

export function Modal({
  title,
  subtitle,
  onClose,
  children,
  width = 520,
}: {
  title: string;
  subtitle?: string;
  onClose: () => void;
  children: ReactNode;
  width?: number;
}) {
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => e.key === "Escape" && onClose();
    document.addEventListener("keydown", onKey);
    return () => document.removeEventListener("keydown", onKey);
  }, [onClose]);

  return (
    <div
      className="fixed inset-0 z-[70] flex items-center justify-center bg-ink-900/25 p-6"
      onMouseDown={(e) => e.target === e.currentTarget && onClose()}
    >
      <div
        className="fade-in max-h-[80vh] overflow-y-auto rounded-xl border border-line bg-paper shadow-2xl shadow-ink-900/20"
        style={{ width, maxWidth: "100%" }}
      >
        <div className="flex items-start justify-between border-b border-line-soft px-5 py-4">
          <div>
            <h3 className="font-serif text-[16px] font-semibold text-ink-900">{title}</h3>
            {subtitle && <p className="mt-0.5 text-[12px] text-mute">{subtitle}</p>}
          </div>
          <IconBtn name="x" title="Close" onClick={onClose} />
        </div>
        <div className="px-5 py-4">{children}</div>
      </div>
    </div>
  );
}

/* ------------------------------------------------------------------ */
/* Spinner                                                             */
/* ------------------------------------------------------------------ */

export function Spinner({ size = 14, className }: { size?: number; className?: string }) {
  return (
    <svg
      width={size}
      height={size}
      viewBox="0 0 24 24"
      fill="none"
      className={cn("animate-spin", className)}
      aria-hidden="true"
    >
      <circle cx="12" cy="12" r="9" stroke="currentColor" strokeOpacity="0.25" strokeWidth="2.5" />
      <path d="M21 12a9 9 0 0 0-9-9" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round" />
    </svg>
  );
}

/* ------------------------------------------------------------------ */
/* ResizablePanel — right-side panel with drag-to-resize               */
/* ------------------------------------------------------------------ */

export function ResizablePanel({
  width,
  min = 280,
  max = 560,
  onWidth,
  children,
  className,
}: {
  width: number;
  min?: number;
  max?: number;
  onWidth: (w: number) => void;
  children: ReactNode;
  className?: string;
}) {
  const onPointerDown = useCallback(
    (e: React.PointerEvent) => {
      e.preventDefault();
      const startX = e.clientX;
      const startW = width;
      const move = (ev: PointerEvent) => {
        onWidth(clamp(startW - (ev.clientX - startX), min, max));
      };
      const up = () => {
        window.removeEventListener("pointermove", move);
        window.removeEventListener("pointerup", up);
      };
      window.addEventListener("pointermove", move);
      window.addEventListener("pointerup", up);
    },
    [width, min, max, onWidth],
  );

  return (
    <aside
      className={cn("relative flex h-full min-h-0 flex-col border-l border-line bg-paper", className)}
      style={{ width }}
    >
      <div
        role="separator"
        title="Drag to resize · double-click to reset"
        onPointerDown={onPointerDown}
        onDoubleClick={() => onWidth(360)}
        className="absolute inset-y-0 -left-0.5 z-10 w-1.5 cursor-col-resize hover:bg-iris-300/50"
      />
      {children}
    </aside>
  );
}
