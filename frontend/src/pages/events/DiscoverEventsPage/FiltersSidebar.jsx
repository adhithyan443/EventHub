import { GridIcon, TagIcon, CalendarIcon, LocationIcon } from "../../../components/layout/icons";

const filters = [
  { key: "categories", label: "All Categories", icon: GridIcon },
  { key: "price", label: "Price Range", icon: TagIcon },
  { key: "date", label: "Date", icon: CalendarIcon },
  { key: "location", label: "Location", icon: LocationIcon },
];

const getTodayString = () => {
  const d = new Date();
  return d.toISOString().split("T")[0];
};

const getTomorrowString = () => {
  const d = new Date();
  d.setDate(d.getDate() + 1);
  return d.toISOString().split("T")[0];
};

const getWeekendStrings = () => {
  const now = new Date();
  const day = now.getDay(); // 0 is Sunday, 6 is Saturday
  const diffToSaturday = day === 6 ? 0 : day === 0 ? -1 : 6 - day;
  const sat = new Date(now);
  sat.setDate(now.getDate() + diffToSaturday);
  const sun = new Date(sat);
  sun.setDate(sat.getDate() + (diffToSaturday === -1 ? 0 : 1));
  return {
    start: sat.toISOString().split("T")[0],
    end: sun.toISOString().split("T")[0],
  };
};

export default function FiltersSidebar({
  activeFilter = "categories",
  onSelectFilter,
  onReset,
  categories = [],
  selectedCategoryId = "",
  onSelectCategory,
  minPrice = "",
  maxPrice = "",
  onMinPriceChange,
  onMaxPriceChange,
  onClearPrice,
  startDate = "",
  endDate = "",
  onStartDateChange,
  onEndDateChange,
  onClearDate,
  city = "",
  onCityChange,
  onClearCity,
}) {
  const isCategoryActive = Boolean(selectedCategoryId);
  const isPriceActive = Boolean(minPrice !== "" || maxPrice !== "");
  const isDateActive = Boolean(startDate || endDate);
  const isLocationActive = Boolean(city && city.trim());

  const activeFiltersCount =
    (isCategoryActive ? 1 : 0) +
    (isPriceActive ? 1 : 0) +
    (isDateActive ? 1 : 0) +
    (isLocationActive ? 1 : 0);

  const getFilterIsActive = (key) => {
    switch (key) {
      case "categories":
        return isCategoryActive;
      case "price":
        return isPriceActive;
      case "date":
        return isDateActive;
      case "location":
        return isLocationActive;
      default:
        return false;
    }
  };

  return (
    <aside className="w-full lg:w-64 shrink-0 flex flex-col gap-4">
      <div className="flex items-center justify-between lg:block">
        <div>
          <h2 className="font-display text-xl font-semibold text-primary">Filters</h2>
          <p className="text-sm text-ink/60">Narrow your search</p>
        </div>
        <button
          type="button"
          onClick={onReset}
          className="lg:hidden text-sm font-semibold text-primary hover:text-primary/80 transition-colors"
        >
          Reset All {activeFiltersCount > 0 ? `(${activeFiltersCount})` : ""}
        </button>
      </div>

      <nav className="flex flex-row flex-wrap sm:flex-nowrap lg:flex-col gap-1 overflow-x-auto scrollbar-none">
        {filters.map(({ key, label, icon: Icon }) => {
          const isFilterActive = getFilterIsActive(key);
          const isSelected = activeFilter === key;

          return (
            <button
              key={key}
              type="button"
              onClick={() => onSelectFilter && onSelectFilter(key)}
              className={`relative flex items-center justify-between gap-2 px-3 py-2 rounded-md text-sm font-semibold whitespace-nowrap transition-colors ${
                isSelected
                  ? "bg-primary/10 text-primary"
                  : "text-ink/70 hover:bg-background"
              }`}
            >
              <span className="flex items-center gap-2">
                <Icon />
                {label}
              </span>
              {isFilterActive && (
                <span
                  className="h-2 w-2 rounded-full bg-primary shrink-0"
                  title="Filter active"
                />
              )}
            </button>
          );
        })}
      </nav>

      {/* Filter Control Panels */}
      <div className="rounded-lg border border-border bg-[#f9f9ff] p-4 flex flex-col gap-3">
        {/* Categories Panel */}
        {activeFilter === "categories" && (
          <div className="flex flex-col gap-2">
            <span className="text-xs font-semibold uppercase tracking-wider text-ink/60">
              Select Category
            </span>
            <div className="flex flex-col gap-1 max-h-56 overflow-y-auto pr-1">
              <button
                type="button"
                onClick={() => onSelectCategory && onSelectCategory("", "All Events")}
                className={`text-left px-3 py-1.5 rounded text-sm transition-colors ${
                  !selectedCategoryId
                    ? "bg-primary text-white font-semibold"
                    : "text-ink/80 hover:bg-border/50"
                }`}
              >
                All Categories
              </button>
              {categories.map((cat) => {
                const catId = cat.ID || cat.id;
                const catName = cat.Name || cat.name;
                const isSelected = selectedCategoryId === catId;
                return (
                  <button
                    key={catId}
                    type="button"
                    onClick={() => onSelectCategory && onSelectCategory(catId, catName)}
                    className={`text-left px-3 py-1.5 rounded text-sm transition-colors ${
                      isSelected
                        ? "bg-primary text-white font-semibold"
                        : "text-ink/80 hover:bg-border/50"
                    }`}
                  >
                    {catName}
                  </button>
                );
              })}
            </div>
            {isCategoryActive && (
              <button
                type="button"
                onClick={() => onSelectCategory && onSelectCategory("", "All Events")}
                className="text-xs text-primary hover:underline text-left mt-1"
              >
                Clear category filter
              </button>
            )}
          </div>
        )}

        {/* Price Range Panel */}
        {activeFilter === "price" && (
          <div className="flex flex-col gap-3">
            <span className="text-xs font-semibold uppercase tracking-wider text-ink/60">
              Ticket Price (₹)
            </span>
            <div className="grid grid-cols-2 gap-2">
              <div className="flex flex-col gap-1">
                <label className="text-[11px] text-ink/60">Min Price</label>
                <input
                  type="number"
                  min="0"
                  step="50"
                  placeholder="₹ 0"
                  value={minPrice}
                  onChange={(e) => onMinPriceChange && onMinPriceChange(e.target.value)}
                  className="w-full bg-white border border-border rounded px-2.5 py-1.5 text-sm text-ink focus:outline-none focus:border-primary"
                />
              </div>
              <div className="flex flex-col gap-1">
                <label className="text-[11px] text-ink/60">Max Price</label>
                <input
                  type="number"
                  min="0"
                  step="50"
                  placeholder="₹ Any"
                  value={maxPrice}
                  onChange={(e) => onMaxPriceChange && onMaxPriceChange(e.target.value)}
                  className="w-full bg-white border border-border rounded px-2.5 py-1.5 text-sm text-ink focus:outline-none focus:border-primary"
                />
              </div>
            </div>

            {/* Quick Price Presets */}
            <div className="flex flex-wrap gap-1.5 pt-1">
              <button
                type="button"
                onClick={() => {
                  onMinPriceChange && onMinPriceChange("0");
                  onMaxPriceChange && onMaxPriceChange("0");
                }}
                className={`text-xs px-2.5 py-1 rounded-full border transition-colors ${
                  minPrice === "0" && maxPrice === "0"
                    ? "bg-primary text-white border-primary"
                    : "border-border text-ink/70 hover:bg-white"
                }`}
              >
                Free
              </button>
              <button
                type="button"
                onClick={() => {
                  onMinPriceChange && onMinPriceChange("");
                  onMaxPriceChange && onMaxPriceChange("500");
                }}
                className={`text-xs px-2.5 py-1 rounded-full border transition-colors ${
                  minPrice === "" && maxPrice === "500"
                    ? "bg-primary text-white border-primary"
                    : "border-border text-ink/70 hover:bg-white"
                }`}
              >
                Under ₹500
              </button>
              <button
                type="button"
                onClick={() => {
                  onMinPriceChange && onMinPriceChange("500");
                  onMaxPriceChange && onMaxPriceChange("2000");
                }}
                className={`text-xs px-2.5 py-1 rounded-full border transition-colors ${
                  minPrice === "500" && maxPrice === "2000"
                    ? "bg-primary text-white border-primary"
                    : "border-border text-ink/70 hover:bg-white"
                }`}
              >
                ₹500 - ₹2,000
              </button>
              <button
                type="button"
                onClick={() => {
                  onMinPriceChange && onMinPriceChange("2000");
                  onMaxPriceChange && onMaxPriceChange("");
                }}
                className={`text-xs px-2.5 py-1 rounded-full border transition-colors ${
                  minPrice === "2000" && maxPrice === ""
                    ? "bg-primary text-white border-primary"
                    : "border-border text-ink/70 hover:bg-white"
                }`}
              >
                Above ₹2,000
              </button>
            </div>

            {isPriceActive && (
              <button
                type="button"
                onClick={onClearPrice}
                className="text-xs text-primary hover:underline text-left mt-1"
              >
                Clear price filter
              </button>
            )}
          </div>
        )}

        {/* Date Panel */}
        {activeFilter === "date" && (
          <div className="flex flex-col gap-3">
            <span className="text-xs font-semibold uppercase tracking-wider text-ink/60">
              Event Schedule
            </span>
            <div className="flex flex-col gap-2">
              <div className="flex flex-col gap-1">
                <label className="text-[11px] text-ink/60">From Date</label>
                <input
                  type="date"
                  value={startDate}
                  onChange={(e) => onStartDateChange && onStartDateChange(e.target.value)}
                  className="w-full bg-white border border-border rounded px-2.5 py-1.5 text-sm text-ink focus:outline-none focus:border-primary"
                />
              </div>
              <div className="flex flex-col gap-1">
                <label className="text-[11px] text-ink/60">To Date</label>
                <input
                  type="date"
                  value={endDate}
                  onChange={(e) => onEndDateChange && onEndDateChange(e.target.value)}
                  className="w-full bg-white border border-border rounded px-2.5 py-1.5 text-sm text-ink focus:outline-none focus:border-primary"
                />
              </div>
            </div>

            {/* Quick Date Presets */}
            <div className="flex flex-wrap gap-1.5 pt-1">
              <button
                type="button"
                onClick={() => {
                  const today = getTodayString();
                  onStartDateChange && onStartDateChange(today);
                  onEndDateChange && onEndDateChange(today);
                }}
                className="text-xs px-2.5 py-1 rounded-full border border-border text-ink/70 hover:bg-white transition-colors"
              >
                Today
              </button>
              <button
                type="button"
                onClick={() => {
                  const tmrw = getTomorrowString();
                  onStartDateChange && onStartDateChange(tmrw);
                  onEndDateChange && onEndDateChange(tmrw);
                }}
                className="text-xs px-2.5 py-1 rounded-full border border-border text-ink/70 hover:bg-white transition-colors"
              >
                Tomorrow
              </button>
              <button
                type="button"
                onClick={() => {
                  const { start, end } = getWeekendStrings();
                  onStartDateChange && onStartDateChange(start);
                  onEndDateChange && onEndDateChange(end);
                }}
                className="text-xs px-2.5 py-1 rounded-full border border-border text-ink/70 hover:bg-white transition-colors"
              >
                This Weekend
              </button>
            </div>

            {isDateActive && (
              <button
                type="button"
                onClick={onClearDate}
                className="text-xs text-primary hover:underline text-left mt-1"
              >
                Clear date filter
              </button>
            )}
          </div>
        )}

        {/* Location Panel */}
        {activeFilter === "location" && (
          <div className="flex flex-col gap-3">
            <span className="text-xs font-semibold uppercase tracking-wider text-ink/60">
              City or Venue Location
            </span>
            <input
              type="text"
              placeholder="e.g. Kochi, Bangalore..."
              value={city}
              onChange={(e) => onCityChange && onCityChange(e.target.value)}
              className="w-full bg-white border border-border rounded px-3 py-2 text-sm text-ink placeholder:text-ink/40 focus:outline-none focus:border-primary"
            />
            {isLocationActive && (
              <button
                type="button"
                onClick={onClearCity}
                className="text-xs text-primary hover:underline text-left"
              >
                Clear location filter
              </button>
            )}
          </div>
        )}
      </div>

      <div className="hidden lg:block border-t border-border pt-4">
        <button
          type="button"
          onClick={onReset}
          className="text-sm font-semibold text-primary hover:text-primary/80 transition-colors"
        >
          Reset All {activeFiltersCount > 0 ? `(${activeFiltersCount})` : ""}
        </button>
      </div>
    </aside>
  );
}