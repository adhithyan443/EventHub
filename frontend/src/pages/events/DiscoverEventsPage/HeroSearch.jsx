import { SearchIcon, LocationIcon, CalendarIcon } from "../../../components/layout/icons";

export default function HeroSearch() {
  return (
    <section className="bg-gradient-to-b from-primary/5 to-background py-16 px-6">
      <div className="max-w-3xl mx-auto flex flex-col items-center text-center gap-3">
        <h1 className="font-display text-4xl md:text-5xl font-bold text-ink">
          Find your next experience.
        </h1>
        <p className="text-ink/60">
          Discover concerts, sports, conferences, workshops, and more.
        </p>

        <div className="mt-6 w-full bg-white rounded-lg border border-border shadow-sm flex flex-col md:flex-row items-stretch divide-y md:divide-y-0 md:divide-x divide-border">
          <div className="flex items-center gap-2 px-4 py-3 flex-1">
            <span className="text-ink/40"><SearchIcon /></span>
            <input
              type="text"
              placeholder="Search events, artists, venues..."
              className="bg-transparent text-sm text-ink placeholder:text-ink/50 focus:outline-none w-full"
            />
          </div>
          <div className="flex items-center gap-2 px-4 py-3 flex-1">
            <span className="text-ink/40"><LocationIcon /></span>
            <input
              type="text"
              placeholder="Anywhere"
              className="bg-transparent text-sm text-ink placeholder:text-ink/50 focus:outline-none w-full"
            />
          </div>
          <div className="flex items-center gap-2 px-4 py-3 flex-1">
            <span className="text-ink/40"><CalendarIcon /></span>
            <input
              type="text"
              placeholder="Any Date"
              className="bg-transparent text-sm text-ink placeholder:text-ink/50 focus:outline-none w-full"
            />
          </div>
          <button className="bg-primary text-white font-semibold px-8 py-3 md:rounded-r-lg hover:bg-primary/90 transition-colors">
            Search
          </button>
        </div>
      </div>
    </section>
  );
}