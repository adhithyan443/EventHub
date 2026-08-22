export default function Input({ label, error, leftAddon, rightElement, className = "", ...props }) {
  return (
    <div className="flex flex-col gap-2 w-full">
      {label && <label className="text-sm font-semibold text-ink">{label}</label>}
      <div className="relative flex items-stretch">
        {leftAddon && (
          <span className="flex items-center px-3 rounded-l-md border border-r-0 border-border bg-background text-sm text-ink/70">
            {leftAddon}
          </span>
        )}
        <input
          className={`w-full border px-4 py-3 text-sm text-ink placeholder:text-ink/50 focus:outline-none focus:ring-2 focus:ring-primary/40 ${leftAddon ? "rounded-r-md" : "rounded-md"
            } ${rightElement ? "pr-11" : ""} ${error ? "border-red-500" : "border-border"} ${className}`}
          {...props}
        />
        {rightElement && (
          <span className="absolute right-3 top-1/2 -translate-y-1/2">{rightElement}</span>
        )}
      </div>
      {error && <p className="text-xs text-red-500">{error}</p>}
    </div>
  );
}