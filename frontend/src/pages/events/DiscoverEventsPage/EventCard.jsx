import { CalendarIcon, LocationIcon } from "../../../components/layout/icons";

export default function EventCard({ event }) {
  return (
    <article className="bg-[#f9f9ff] rounded-xl overflow-hidden border border-border flex flex-col">
      <div className="relative h-44 w-full">
        <img src={event.image} alt={event.title} className="h-full w-full object-cover" />
        <span className="absolute left-2 top-2 flex items-center gap-1.5 rounded-full bg-white/90 backdrop-blur-sm border border-border px-2.5 py-1 text-xs font-medium tracking-wide text-ink">
          <span className="h-2 w-2 rounded-full bg-primary" />
          {event.category}
        </span>
        {event.badge && (
          <span className="absolute right-2 top-2 rounded-full bg-red-100 px-2.5 py-1 text-xs font-bold tracking-wide text-red-700">
            {event.badge}
          </span>
        )}
      </div>

      <div className="flex flex-1 flex-col gap-4 p-6">
        <h3 className="font-display text-xl font-semibold text-ink leading-snug">
          {event.title}
        </h3>

        <div className="flex flex-col gap-1 text-sm text-ink/70">
          <span className="flex items-center gap-2"><CalendarIcon /> {event.date}</span>
          <span className="flex items-center gap-2"><LocationIcon /> {event.location}</span>
        </div>

        <div className="mt-auto flex items-center justify-between border-t border-border pt-3">
          <span className="text-sm font-semibold text-ink">From ${event.price}</span>
          <button className="rounded-md border border-border px-4 py-1.5 text-xs font-medium text-ink/70 hover:bg-background">
            View Details
          </button>
        </div>
      </div>
    </article>
  );
}