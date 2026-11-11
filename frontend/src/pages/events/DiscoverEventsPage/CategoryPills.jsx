export default function CategoryPills({
  categories = [],
  selectedId = "",
  selected = "",
  onSelect,
  loading = false,
  error = "",
  onRetry,
}) {
  const activeSelection = selectedId || selected || "";
  const isAllEventsSelected = !activeSelection || activeSelection === "All Events";

  return (
    <div className="max-w-7xl mx-auto px-4 sm:px-6 py-3 sm:py-6 border-b border-border overflow-x-auto scrollbar-none">
      <div className="flex items-center gap-2 sm:gap-4 w-max">
        <button
          type="button"
          onClick={() => onSelect && onSelect("", "All Events")}
          className={`h-8 sm:h-10 px-4 sm:px-6 rounded-full text-xs sm:text-sm font-semibold whitespace-nowrap transition-colors ${
            isAllEventsSelected
              ? "bg-primary text-white"
              : "bg-border text-ink/70 hover:bg-border/70"
          }`}
        >
          All Events
        </button>

        {loading && (
          <div className="flex items-center gap-2 sm:gap-4">
            {[1, 2, 3, 4].map((n) => (
              <div
                key={`cat-skeleton-${n}`}
                className="h-8 sm:h-10 w-24 sm:w-28 rounded-full bg-border/60 animate-pulse"
              />
            ))}
          </div>
        )}

        {!loading && error && categories.length === 0 && (
          <div className="flex items-center gap-2 text-xs text-red-600 px-2">
            <span>Unable to load categories.</span>
            {onRetry && (
              <button
                type="button"
                onClick={onRetry}
                className="underline font-semibold hover:text-red-700"
              >
                Retry
              </button>
            )}
          </div>
        )}

        {!loading &&
          categories.map((cat) => {
            const catId = cat.ID || cat.id;
            const catName = cat.Name || cat.name;
            const isSelected = activeSelection === catId || activeSelection === catName;

            return (
              <button
                key={catId}
                type="button"
                onClick={() => onSelect && onSelect(catId, catName)}
                className={`h-8 sm:h-10 px-4 sm:px-6 rounded-full text-xs sm:text-sm font-semibold whitespace-nowrap transition-colors ${
                  isSelected
                    ? "bg-primary text-white"
                    : "bg-border text-ink/70 hover:bg-border/70"
                }`}
              >
                {catName}
              </button>
            );
          })}
      </div>
    </div>
  );
}