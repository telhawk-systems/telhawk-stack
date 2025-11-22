interface FilterChipProps {
  label: string;
  value?: string;
  onRemove?: () => void;
  onClick?: () => void;
}

const CloseIcon = () => (
  <svg className="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
  </svg>
);

export function FilterChip({ label, value, onRemove, onClick }: FilterChipProps) {
  return (
    <span
      className={`inline-flex items-center gap-1 h-7 px-3 text-sm bg-gray-100 text-gray-700 rounded ${
        onClick ? 'cursor-pointer hover:bg-gray-200' : ''
      }`}
      onClick={onClick}
    >
      {value ? (
        <>
          <span className="text-gray-500">{label}:</span>
          <span className="font-medium">{value}</span>
        </>
      ) : (
        <span>{label}</span>
      )}
      {onRemove && (
        <button
          onClick={(e) => {
            e.stopPropagation();
            onRemove();
          }}
          className="ml-1 p-0.5 hover:bg-gray-300 rounded transition-colors"
          aria-label={`Remove ${label} filter`}
        >
          <CloseIcon />
        </button>
      )}
    </span>
  );
}
