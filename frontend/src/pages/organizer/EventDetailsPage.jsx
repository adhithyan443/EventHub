import { useState } from "react";
import { useParams, useNavigate, Link } from "react-router-dom";
import { ORGANIZER_ROUTES } from "../../constants/eventConstants";
import StatusBadge from "../../components/organizer/events/StatusBadge";
import ProgressBar from "../../components/organizer/events/ProgressBar";
import {
  ArrowLeftIcon,
  CalendarDetailIcon,
  ClockIcon,
  MapPinDetailIcon,
  RupeeIcon,
  TicketDetailIcon,
  EditIcon,
  ShareIcon,
  ExternalLinkIcon,
  CheckCircleIcon,
} from "../../components/organizer/events/OrganizerEventsIcons";
import { getOrganizerEventById } from "./mockOrganizerEvents";

export default function EventDetailsPage() {
  const { eventId } = useParams();
  const navigate = useNavigate();

  const event = getOrganizerEventById(eventId);
  const [copySuccess, setCopySuccess] = useState(false);

  const handleShare = () => {
    if (navigator.clipboard) {
      navigator.clipboard.writeText(window.location.href);
      setCopySuccess(true);
      setTimeout(() => setCopySuccess(false), 2000);
    }
  };

  return (
    <div className="p-4 sm:p-6 lg:p-8 max-w-7xl mx-auto flex flex-col gap-6 w-full animate-fade-in">
      {/* Top Breadcrumb & Return Action */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
        <div className="flex items-center gap-2 text-xs sm:text-sm text-[#565e74]">
          <Link
            to={ORGANIZER_ROUTES.MY_EVENTS}
            className="flex items-center gap-1.5 text-[#00685f] hover:underline font-medium"
          >
            <ArrowLeftIcon className="size-4" />
            <span>Back to My Events</span>
          </Link>
          <span>/</span>
          <span className="text-[#141b2b] font-medium truncate max-w-[200px] sm:max-w-xs">
            {event.name}
          </span>
        </div>

        {/* Action Buttons */}
        <div className="flex items-center gap-2 self-start sm:self-auto flex-wrap">
          <button
            type="button"
            onClick={handleShare}
            className="border border-[#bcc9c6] bg-white hover:bg-[#f1f3ff] text-[#141b2b] text-xs sm:text-sm font-medium px-3.5 py-2 rounded-lg flex items-center gap-1.5 transition-colors cursor-pointer"
          >
            <ShareIcon className="size-3.5 text-[#565e74]" />
            <span>{copySuccess ? "Link Copied!" : "Share"}</span>
          </button>

          <button
            type="button"
            onClick={() => navigate(ORGANIZER_ROUTES.CREATE_STEP_1)}
            className="border border-[#bcc9c6] bg-white hover:bg-[#f1f3ff] text-[#141b2b] text-xs sm:text-sm font-medium px-3.5 py-2 rounded-lg flex items-center gap-1.5 transition-colors cursor-pointer"
          >
            <EditIcon className="size-3.5 text-[#565e74]" />
            <span>Edit Event</span>
          </button>

          <button
            type="button"
            onClick={() => navigate("/events")}
            className="bg-[#00685f] hover:bg-[#005a52] text-white text-xs sm:text-sm font-medium px-4 py-2 rounded-lg flex items-center gap-1.5 transition-colors shadow-2xs cursor-pointer"
          >
            <ExternalLinkIcon className="size-3.5 text-white" />
            <span>View Public Page</span>
          </button>
        </div>
      </div>

      {/* Main Title & Status Header */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3 bg-white border border-[#bcc9c6] rounded-xl p-4 sm:p-6 shadow-2xs">
        <div className="flex flex-col gap-1.5">
          <div className="flex items-center gap-3 flex-wrap">
            <h1 className="text-xl sm:text-2xl lg:text-3xl font-bold text-[#141b2b]">
              {event.name}
            </h1>
            <StatusBadge status={event.status} />
          </div>
          <p className="text-xs sm:text-sm text-[#565e74]">
            Event ID: <span className="font-mono text-[#141b2b]">EV-{event.id.toString().padStart(5, "0")}</span> • Created by Organizer Arjun Kumar
          </p>
        </div>

        <div className="flex items-center gap-2 self-start sm:self-auto">
          <span className="bg-[#f1f3ff] text-[#565e74] text-xs font-semibold px-3 py-1.5 rounded-md uppercase tracking-wider">
            {event.visibility || "PUBLIC"}
          </span>
          <span className="bg-[#00685f]/10 text-[#00685f] text-xs font-semibold px-3 py-1.5 rounded-md uppercase tracking-wider">
            {event.category}
          </span>
        </div>
      </div>

      {/* Key Metrics Grid (4 Bento Cards) */}
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
        {/* Total Revenue */}
        <div className="bg-white border border-[#bcc9c6] rounded-xl p-4 sm:p-5 shadow-2xs flex flex-col justify-between gap-3">
          <div className="flex items-center justify-between">
            <span className="text-xs font-semibold text-[#565e74] uppercase tracking-wider">
              Total Revenue
            </span>
            <div className="size-8 rounded-lg bg-[#00685f]/10 flex items-center justify-center text-[#00685f]">
              <RupeeIcon className="size-4" />
            </div>
          </div>
          <div className="flex flex-col">
            <span className="text-2xl sm:text-3xl font-bold text-[#141b2b]">
              {event.revenue}
            </span>
            <span className="text-xs text-[#565e74] mt-0.5">
              Gross sales to date
            </span>
          </div>
        </div>

        {/* Tickets Sold */}
        <div className="bg-white border border-[#bcc9c6] rounded-xl p-4 sm:p-5 shadow-2xs flex flex-col justify-between gap-3">
          <div className="flex items-center justify-between">
            <span className="text-xs font-semibold text-[#565e74] uppercase tracking-wider">
              Tickets Sold
            </span>
            <div className="size-8 rounded-lg bg-[#dce2f7] flex items-center justify-center text-[#00685f]">
              <TicketDetailIcon className="size-4 text-[#00685f]" />
            </div>
          </div>
          <div className="flex flex-col gap-1.5">
            <div className="flex items-baseline justify-between">
              <span className="text-2xl sm:text-3xl font-bold text-[#141b2b]">
                {event.sold}
                <span className="text-sm font-normal text-[#565e74]">
                  {" "}/ {event.total.toLocaleString()}
                </span>
              </span>
              <span className="text-xs font-bold text-[#00685f]">
                {event.pct}%
              </span>
            </div>
            <ProgressBar pct={event.pct} />
          </div>
        </div>

        {/* Date & Schedule */}
        <div className="bg-white border border-[#bcc9c6] rounded-xl p-4 sm:p-5 shadow-2xs flex flex-col justify-between gap-3">
          <div className="flex items-center justify-between">
            <span className="text-xs font-semibold text-[#565e74] uppercase tracking-wider">
              Event Date
            </span>
            <div className="size-8 rounded-lg bg-[#f1f3ff] flex items-center justify-center text-[#141b2b]">
              <CalendarDetailIcon className="size-4 text-[#141b2b]" />
            </div>
          </div>
          <div className="flex flex-col">
            <span className="text-xl sm:text-2xl font-bold text-[#141b2b]">
              {event.date}
            </span>
            <span className="text-xs text-[#565e74] mt-0.5">
              {event.timeRange}
            </span>
          </div>
        </div>

        {/* Venue & Location */}
        <div className="bg-white border border-[#bcc9c6] rounded-xl p-4 sm:p-5 shadow-2xs flex flex-col justify-between gap-3">
          <div className="flex items-center justify-between">
            <span className="text-xs font-semibold text-[#565e74] uppercase tracking-wider">
              Location / City
            </span>
            <div className="size-8 rounded-lg bg-[#f1f3ff] flex items-center justify-center text-[#141b2b]">
              <MapPinDetailIcon className="size-4 text-[#141b2b]" />
            </div>
          </div>
          <div className="flex flex-col">
            <span className="text-xl sm:text-2xl font-bold text-[#141b2b] truncate">
              {event.venue}
            </span>
            <span className="text-xs text-[#565e74] mt-0.5 truncate">
              {event.venueDetails?.name || event.venue}
            </span>
          </div>
        </div>
      </div>

      {/* Main 2-Column Content Grid */}
      <div className="grid grid-cols-1 lg:grid-cols-12 gap-6 lg:gap-8 items-start">
        {/* Left Column (8 cols): Banner, Overview, Ticket Tiers, Guidelines */}
        <div className="lg:col-span-8 flex flex-col gap-6">
          {/* Banner & Description Card */}
          <div className="bg-white border border-[#bcc9c6] rounded-xl overflow-hidden shadow-2xs">
            <div className="h-56 sm:h-72 w-full bg-gray-100 relative">
              <img
                src={event.image}
                alt={event.name}
                className="size-full object-cover"
              />
              <div className="absolute top-4 left-4 flex gap-2">
                <span className="bg-[#00685f] text-white text-[11px] font-bold px-3 py-1 rounded-full uppercase tracking-wider shadow-sm">
                  {event.category}
                </span>
                <span className="bg-white/90 backdrop-blur-xs text-[#141b2b] text-[11px] font-bold px-3 py-1 rounded-full shadow-sm">
                  {event.ageRestriction}
                </span>
              </div>
            </div>

            <div className="p-5 sm:p-6 flex flex-col gap-4">
              <h2 className="text-xl font-bold text-[#141b2b]">
                About This Event
              </h2>
              <p className="text-sm text-[#565e74] leading-relaxed whitespace-pre-line">
                {event.description}
              </p>

              {/* Highlights */}
              {event.highlights && event.highlights.length > 0 && (
                <div className="mt-2 pt-4 border-t border-[#bcc9c6]/40 flex flex-col gap-2.5">
                  <h3 className="text-xs font-bold text-[#141b2b] uppercase tracking-wider">
                    Key Highlights
                  </h3>
                  <div className="grid grid-cols-1 sm:grid-cols-2 gap-2">
                    {event.highlights.map((h, i) => (
                      <div key={i} className="flex items-start gap-2 text-xs sm:text-sm text-[#565e74]">
                        <CheckCircleIcon className="size-4 text-[#00685f] shrink-0 mt-0.5" />
                        <span>{h}</span>
                      </div>
                    ))}
                  </div>
                </div>
              )}
            </div>
          </div>

          {/* Ticket Tiers Breakdown Card */}
          <div className="bg-white border border-[#bcc9c6] rounded-xl p-5 sm:p-6 shadow-2xs flex flex-col gap-4">
            <div className="flex items-center justify-between pb-3 border-b border-[#bcc9c6]/40">
              <div className="flex flex-col">
                <h2 className="text-lg font-bold text-[#141b2b]">
                  Ticket Tiers & Inventory
                </h2>
                <p className="text-xs text-[#565e74]">
                  {event.ticketTypes?.length || 0} active ticket tier configurations
                </p>
              </div>
              <span className="text-xs font-semibold px-2.5 py-1 rounded-full bg-[#00685f]/10 text-[#00685f]">
                {event.total.toLocaleString()} Total Capacity
              </span>
            </div>

            <div className="overflow-x-auto">
              <table className="w-full text-left border-collapse">
                <thead>
                  <tr className="bg-[#f1f3ff] text-[11px] font-bold uppercase tracking-wider text-[#565e74]">
                    <th className="py-2.5 px-3 rounded-l-lg">Tier Name</th>
                    <th className="py-2.5 px-3">Price</th>
                    <th className="py-2.5 px-3">Sold / Cap</th>
                    <th className="py-2.5 px-3 min-w-[120px]">Progress</th>
                    <th className="py-2.5 px-3 rounded-r-lg text-right">Status</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-[#bcc9c6]/30 text-xs sm:text-sm">
                  {(event.ticketTypes || []).map((t) => {
                    const tierPct = t.capacity > 0 ? Math.round((t.sold / t.capacity) * 100) : 0;
                    return (
                      <tr key={t.id} className="hover:bg-[#f9f9ff]">
                        <td className="py-3 px-3">
                          <div className="flex flex-col">
                            <span className="font-semibold text-[#141b2b]">{t.name}</span>
                            <span className="text-[11px] text-[#565e74] line-clamp-1">{t.description}</span>
                          </div>
                        </td>
                        <td className="py-3 px-3 font-semibold text-[#141b2b] whitespace-nowrap">
                          {t.price}
                        </td>
                        <td className="py-3 px-3 text-[#565e74] whitespace-nowrap">
                          {t.sold} / {t.capacity}
                        </td>
                        <td className="py-3 px-3 min-w-[120px]">
                          <div className="flex flex-col gap-1">
                            <div className="flex justify-between text-[11px] font-medium">
                              <span>{tierPct}%</span>
                            </div>
                            <ProgressBar pct={tierPct} />
                          </div>
                        </td>
                        <td className="py-3 px-3 text-right">
                          <span
                            className={`inline-block px-2.5 py-0.5 rounded-full text-[11px] font-medium ${
                              t.status === "Sold Out"
                                ? "bg-red-50 text-red-600 border border-red-200"
                                : t.status === "Selling Fast"
                                  ? "bg-amber-50 text-amber-700 border border-amber-200"
                                  : "bg-emerald-50 text-emerald-700 border border-emerald-200"
                            }`}
                          >
                            {t.status}
                          </span>
                        </td>
                      </tr>
                    );
                  })}
                </tbody>
              </table>
            </div>
          </div>

          {/* Attendee Guidelines & Rules Card */}
          <div className="bg-white border border-[#bcc9c6] rounded-xl p-5 sm:p-6 shadow-2xs flex flex-col gap-4">
            <h2 className="text-lg font-bold text-[#141b2b] pb-2 border-b border-[#bcc9c6]/40">
              Attendee Guidelines & Instructions
            </h2>
            <div className="grid grid-cols-1 sm:grid-cols-2 gap-4 text-xs sm:text-sm">
              <div className="flex flex-col gap-1.5">
                <span className="font-semibold text-[#141b2b] uppercase tracking-wider text-[11px] text-[#565e74]">
                  Event Rules & Regulations
                </span>
                <p className="text-[#565e74] whitespace-pre-line leading-relaxed">
                  {event.rules || "Standard venue rules apply."}
                </p>
              </div>

              <div className="flex flex-col gap-1.5">
                <span className="font-semibold text-[#141b2b] uppercase tracking-wider text-[11px] text-[#565e74]">
                  Arrival & Check-in Notes
                </span>
                <p className="text-[#565e74] whitespace-pre-line leading-relaxed">
                  {event.attendeeInformation || "Check-in instructions will be delivered with ticket confirmation."}
                </p>
              </div>
            </div>
          </div>
        </div>

        {/* Right Column (4 cols): Schedule & Venue, Contact, Cancellation, Management Tools */}
        <div className="lg:col-span-4 flex flex-col gap-6">
          {/* Schedule & Venue Card */}
          <div className="bg-white border border-[#bcc9c6] rounded-xl p-5 sm:p-6 shadow-2xs flex flex-col gap-4">
            <h3 className="text-base font-bold text-[#141b2b] pb-2 border-b border-[#bcc9c6]/40">
              Schedule & Location
            </h3>

            {/* Date & Time */}
            <div className="flex flex-col gap-2">
              <div className="flex items-start gap-3">
                <div className="size-8 rounded-lg bg-[#f1f3ff] flex items-center justify-center text-[#141b2b] shrink-0 mt-0.5">
                  <CalendarDetailIcon className="size-4" />
                </div>
                <div className="flex flex-col">
                  <span className="text-xs font-semibold text-[#565e74] uppercase tracking-wider">
                    Date & Timing
                  </span>
                  <span className="text-sm font-bold text-[#141b2b] mt-0.5">
                    {event.date}
                  </span>
                  <span className="text-xs text-[#565e74] flex items-center gap-1 mt-0.5">
                    <ClockIcon className="size-3 text-[#565e74]" />
                    <span>{event.timeRange} ({event.duration})</span>
                  </span>
                </div>
              </div>
            </div>

            {/* Venue Details */}
            <div className="pt-3 border-t border-[#bcc9c6]/30 flex flex-col gap-2">
              <div className="flex items-start gap-3">
                <div className="size-8 rounded-lg bg-[#f1f3ff] flex items-center justify-center text-[#141b2b] shrink-0 mt-0.5">
                  <MapPinDetailIcon className="size-4" />
                </div>
                <div className="flex flex-col">
                  <span className="text-xs font-semibold text-[#565e74] uppercase tracking-wider">
                    Venue ({event.locationType || "PHYSICAL"})
                  </span>
                  <span className="text-sm font-bold text-[#141b2b] mt-0.5">
                    {event.venueDetails?.name || event.venue}
                  </span>
                  {event.venueDetails?.address && (
                    <span className="text-xs text-[#565e74] mt-0.5">
                      {event.venueDetails.address}
                    </span>
                  )}
                  {event.venueDetails?.city && (
                    <span className="text-xs text-[#565e74]">
                      {event.venueDetails.city}, {event.venueDetails.state} {event.venueDetails.postalCode}
                    </span>
                  )}
                </div>
              </div>
            </div>
          </div>

          {/* Organizer Contact Card */}
          <div className="bg-white border border-[#bcc9c6] rounded-xl p-5 sm:p-6 shadow-2xs flex flex-col gap-3">
            <h3 className="text-base font-bold text-[#141b2b] pb-2 border-b border-[#bcc9c6]/40">
              Organizer Contact
            </h3>
            <div className="flex flex-col gap-2 text-xs sm:text-sm">
              <div className="flex justify-between py-1 border-b border-[#bcc9c6]/20">
                <span className="text-[#565e74]">Lead Organizer:</span>
                <span className="font-semibold text-[#141b2b]">{event.contactInformation?.name || "Arjun Kumar"}</span>
              </div>
              <div className="flex justify-between py-1 border-b border-[#bcc9c6]/20">
                <span className="text-[#565e74]">Phone:</span>
                <span className="font-semibold text-[#141b2b]">{event.contactInformation?.phone || "+91 98765 43210"}</span>
              </div>
              <div className="flex justify-between py-1">
                <span className="text-[#565e74]">Email:</span>
                <span className="font-semibold text-[#00685f] truncate max-w-[170px]">{event.contactInformation?.email || "organizer@eventhub.com"}</span>
              </div>
            </div>
          </div>

          {/* Cancellation Policy Card */}
          <div className="bg-white border border-[#bcc9c6] rounded-xl p-5 sm:p-6 shadow-2xs flex flex-col gap-3">
            <h3 className="text-base font-bold text-[#141b2b] pb-2 border-b border-[#bcc9c6]/40">
              Cancellation & Refund
            </h3>
            <div className="flex flex-col gap-2 text-xs sm:text-sm">
              <div className="flex justify-between py-1 border-b border-[#bcc9c6]/20">
                <span className="text-[#565e74]">Refund Policy:</span>
                <span className="font-semibold text-[#141b2b]">
                  {event.cancellationPolicy?.refundPolicy === "FULL"
                    ? "Full Refund"
                    : event.cancellationPolicy?.refundPolicy === "PARTIAL"
                      ? "Partial Refund"
                      : "Non-refundable"}
                </span>
              </div>
              <div className="flex justify-between py-1 border-b border-[#bcc9c6]/20">
                <span className="text-[#565e74]">Deadline:</span>
                <span className="font-semibold text-[#141b2b]">{event.cancellationPolicy?.deadline || "N/A"}</span>
              </div>
              {event.cancellationPolicy?.lastCancellationDate && (
                <div className="flex justify-between py-1">
                  <span className="text-[#565e74]">Last Date:</span>
                  <span className="font-semibold text-[#141b2b]">{event.cancellationPolicy.lastCancellationDate}</span>
                </div>
              )}
            </div>
          </div>

          {/* Quick Management Tools */}
          <div className="bg-white border border-[#bcc9c6] rounded-xl p-5 sm:p-6 shadow-2xs flex flex-col gap-3">
            <h3 className="text-base font-bold text-[#141b2b] pb-2 border-b border-[#bcc9c6]/40">
              Event Management
            </h3>
            <div className="flex flex-col gap-2">
              <button
                type="button"
                onClick={() => alert("Downloading attendee report for " + event.name)}
                className="w-full border border-[#bcc9c6] hover:bg-[#f1f3ff] text-[#141b2b] text-xs sm:text-sm font-semibold py-2.5 px-3 rounded-lg text-center transition-colors cursor-pointer"
              >
                📥 Download Guestlist (CSV)
              </button>
              <button
                type="button"
                onClick={() => navigate(ORGANIZER_ROUTES.ATTENDEES)}
                className="w-full border border-[#bcc9c6] hover:bg-[#f1f3ff] text-[#141b2b] text-xs sm:text-sm font-semibold py-2.5 px-3 rounded-lg text-center transition-colors cursor-pointer"
              >
                👥 View Registered Attendees
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
