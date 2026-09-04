import { GridIcon, TagIcon, CalendarIcon, LocationIcon } from "../../../components/layout/icons";

const filters = [
  { key: "categories", label: "All Categories", icon: GridIcon },
  { key: "price", label: "Price Range", icon: TagIcon },
  { key: "date", label: "Date", icon: CalendarIcon },
  { key: "location", label: "Location", icon: LocationIcon },
];

export default function FiltersSidebar({ activeFilter, onSelectFilter, onReset }) {
  return (
    <aside className="w-64 shrink-0 flex flex-col gap-4">
      <div>
        <h2 className="font-display text-xl font-semibold text-primary">Filters</h2>
        <p className="text-sm text-ink/60">Narrow your search</p>
      </div>

      <nav className="flex flex-col gap-1">
        {filters.map(({ key, label, icon: Icon }) => (
          <button
            key={key}
            onClick={() => onSelectFilter(key)}
            className={`flex items-center gap-2 px-2 py-2 rounded-md text-sm font-semibold transition-colors ${
              activeFilter === key
                ? "bg-primary/10 text-primary"
                : "text-ink/70 hover:bg-background"
            }`}
          >
            <Icon />
            {label}
          </button>
        ))}
      </nav>

      <div className="border-t border-border pt-4">
        <button onClick={onReset} className="text-sm font-semibold text-primary">
          Reset All
        </button>
      </div>
    </aside>
  );
}