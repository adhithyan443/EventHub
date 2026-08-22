export default function Input({ label, error, className = "", ...props }) {
  return (
    <div className="flex flex-col gap-2 w-full">
      {label && (
        <label className="text-sm font-semibold text-ink">{label}</label>
      )}
      <input
        className={`w-full rounded-md border px-4 py-3 text-sm text-ink placeholder:text-ink/50 focus:outline-none focus:ring-2 focus:ring-primary/40 ${
          error ? "border-red-500" : "border-border"
        } ${className}`}
        {...props}
      />
      {error && <p className="text-xs text-red-500">{error}</p>}
    </div>
  );
}