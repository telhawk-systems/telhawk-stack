export interface FilterChipProps {
  label: string;
  icon?: string;
  onRemove: () => void;
  variant?: 'default' | 'event-class';
}

/**
 * FilterChip - A removable filter pill/chip component
 *
 * Design philosophy: Clear, visible filters that can be removed with one click.
 * No hidden filters - everything should be visible at a glance.
 */
export function FilterChip({ label, icon, onRemove, variant = 'default' }: FilterChipProps) {
  const baseClasses = "inline-flex items-center gap-1.5 px-3 py-1.5 rounded-full text-sm font-medium transition-colors";

  const variantClasses = variant === 'event-class'
    ? "bg-blue-100 text-blue-800 hover:bg-blue-200"
    : "bg-gray-100 text-gray-800 hover:bg-gray-200";

  return (
    <div className={`${baseClasses} ${variantClasses}`}>
      {icon && <span className="text-base leading-none">{icon}</span>}
      <span>{label}</span>
      <button
        onClick={onRemove}
        className="ml-1 hover:bg-black hover:bg-opacity-10 rounded-full p-0.5 transition-colors"
        aria-label={`Remove ${label} filter`}
      >
        <svg
          className="w-3.5 h-3.5"
          fill="none"
          stroke="currentColor"
          viewBox="0 0 24 24"
        >
          <path
            strokeLinecap="round"
            strokeLinejoin="round"
            strokeWidth={2}
            d="M6 18L18 6M6 6l12 12"
          />
        </svg>
      </button>
    </div>
  );
}
