const categories = ["All Events", "Music", "Sports", "Comedy", "Technology", "Conferences", "Workshops", "Theatre"];

export default function CategoryPills({ selected, onSelect }) {
  return (
    <div className="max-w-7xl mx-auto px-6 py-6 border-b border-border overflow-x-auto">
      <div className="flex gap-4 w-max">
        {categories.map((cat) => (
          <button
            key={cat}
            onClick={() => onSelect(cat)}
            className={`h-10 px-6 rounded-full text-sm font-semibold whitespace-nowrap transition-colors ${
              selected === cat
                ? "bg-primary text-white"
                : "bg-border text-ink/70 hover:bg-border/70"
            }`}
          >
            {cat}
          </button>
        ))}
      </div>
    </div>
  );
}