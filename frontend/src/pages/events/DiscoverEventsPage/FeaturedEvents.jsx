import EventCard from "./EventCard";

export default function FeaturedEvents({ events }) {
  return (
    <section className="flex flex-col gap-6">
      <div className="flex items-end justify-between">
        <h2 className="font-display text-3xl font-bold text-ink">Featured Events</h2>
        <a href="#" className="text-sm font-semibold text-primary">View all featured</a>
      </div>
      <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
        {events.map((event) => (
          <EventCard key={event.id} event={event} />
        ))}
      </div>
    </section>
  );
}