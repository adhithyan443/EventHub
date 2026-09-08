import { useState } from "react";
import { useNavigate } from "react-router-dom";
import EventCreationStepper from "../../../components/organizer/EventCreationStepper";
import EventCreationFooter from "../../../components/organizer/EventCreationFooter";
import PayoutBankAccountCard from "../../../components/organizer/PayoutBankAccountCard";
import useEventCreationStore from "../../../store/eventCreationStore";
import {
  TICKET_MODES,
  LOCATION_TYPES,
  ORGANIZER_ROUTES,
  isOnlineEvent,
  isPhysicalEvent,
  isHybridEvent,
} from "../../../constants/eventConstants";
import imgDefaultBanner from "../../../assets/organizer/0b2568d2a1321299cd93ab73936efb2e8bc467fe.png";

export default function CreateEventReviewPublishPage() {
  const navigate = useNavigate();

  const basicInformation = useEventCreationStore((state) => state.basicInformation);
  const locationType = useEventCreationStore(
    (state) => state.locationType || state.eventType
  );
  const onlineUrl = useEventCreationStore((state) => state.onlineUrl);
  const venue = useEventCreationStore((state) => state.venue);
  const dateTime = useEventCreationStore((state) => state.dateTime);
  const eventDetails = useEventCreationStore((state) => state.eventDetails);
  const ticketMode = useEventCreationStore((state) => state.ticketMode);
  const ticketTypes = useEventCreationStore((state) => state.ticketTypes);
  const seatingConfig = useEventCreationStore((state) => state.seatingConfiguration);

  const [showSuccessModal, setShowSuccessModal] = useState(false);

  const isOnline = isOnlineEvent(locationType);
  const isPhysical = isPhysicalEvent(locationType);
  const isHybrid = isHybridEvent(locationType);
  const showSeatedLayout = !isOnline && ticketMode === TICKET_MODES.SEATED;

  // Conditional Back navigation
  const handleBack = () => {
    if (showSeatedLayout) {
      navigate(ORGANIZER_ROUTES.CREATE_SEAT_CONFIG);
    } else {
      navigate(ORGANIZER_ROUTES.CREATE_TICKET_TYPES);
    }
  };

  // UI-only publish event flow
  const handlePublish = () => {
    setShowSuccessModal(true);
  };

  const handleReturnToDashboard = () => {
    setShowSuccessModal(false);
    navigate(ORGANIZER_ROUTES.DASHBOARD);
  };

  return (
    <div className="min-h-full flex flex-col justify-between">
      <div className="p-8 max-w-5xl mx-auto w-full flex flex-col gap-6">
        {/* Stepper Progress Bar */}
        <EventCreationStepper currentStep={5} />

        {/* Header Title */}
        <div className="flex flex-col gap-1">
          <h1 className="text-2xl font-bold text-[#141b2b]">Review & Publish</h1>
          <p className="text-xs text-[#565e74]">
            Review all event information, ticket configurations, and location settings before publishing live.
          </p>
        </div>

        {/* 2-Column Review Bento Layout */}
        <div className="grid grid-cols-1 lg:grid-cols-12 gap-8">
          {/* Main Event Preview Column (8 Cols) */}
          <div className="lg:col-span-8 flex flex-col gap-6">
            {/* Event Hero Card */}
            <div className="bg-white border border-[#bcc9c6]/50 rounded-2xl overflow-hidden shadow-xs">
              <div className="h-56 bg-gray-100 relative">
                <img
                  src={basicInformation.banner || imgDefaultBanner}
                  alt={basicInformation.title}
                  className="size-full object-cover"
                />
                <div className="absolute top-4 left-4 flex gap-2">
                  <span className="bg-[#00685f] text-white text-[11px] font-bold px-3 py-1 rounded-full uppercase tracking-wider">
                    {basicInformation.category}
                  </span>
                  <span className="bg-white/90 backdrop-blur text-[#141b2b] text-[11px] font-bold px-3 py-1 rounded-full">
                    {basicInformation.ageRestriction}
                  </span>
                </div>
              </div>

              <div className="p-6 flex flex-col gap-4">
                <h2 className="text-2xl font-bold text-[#141b2b]">
                  {basicInformation.title || "Untitled Event"}
                </h2>
                <p className="text-sm text-[#565e74] leading-relaxed whitespace-pre-line">
                  {basicInformation.description}
                </p>
              </div>
            </div>

            {/* Location & Schedule Card */}
            <div className="bg-white border border-[#bcc9c6]/50 rounded-2xl p-6 shadow-xs flex flex-col gap-5">
              <h3 className="font-bold text-base text-[#141b2b] border-b border-gray-100 pb-3">
                Schedule & Location
              </h3>

              <div className="grid grid-cols-1 md:grid-cols-2 gap-6 text-xs">
                {/* Date & Time */}
                <div className="flex flex-col gap-1.5">
                  <span className="text-[#565e74] font-semibold uppercase tracking-wider text-[11px]">
                    Date & Timing
                  </span>
                  <div className="flex items-center gap-2 text-sm font-bold text-[#141b2b]">
                    📅 {dateTime.eventDate}
                  </div>
                  <span className="text-[#565e74]">
                    {dateTime.isAllDay
                      ? "All Day Event"
                      : `${dateTime.startTime} — ${dateTime.endTime}`}
                  </span>
                </div>

                {/* Location */}
                <div className="flex flex-col gap-1.5">
                  <span className="text-[#565e74] font-semibold uppercase tracking-wider text-[11px]">
                    Location ({locationType})
                  </span>
                  {isPhysical && (
                    <div className="flex flex-col">
                      <span className="text-sm font-bold text-[#141b2b]">
                        📍 {venue.name || "Physical Venue"}
                      </span>
                      {venue.address && (
                        <span className="text-[#565e74] mt-0.5">{venue.address}</span>
                      )}
                      {venue.city && (
                        <span className="text-[#565e74]">
                          {venue.city}, {venue.state} ({venue.postal_code})
                        </span>
                      )}
                    </div>
                  )}
                  {isOnline && (
                    <div className="flex flex-col">
                      <span className="text-sm font-bold text-[#141b2b]">
                        🌐 Virtual Stream / Meeting
                      </span>
                      <span className="text-[#00685f] truncate mt-0.5 font-mono text-[11px]">
                        {onlineUrl || "Stream link provided"}
                      </span>
                    </div>
                  )}
                  {isHybrid && (
                    <div className="flex flex-col gap-2">
                      <div className="flex flex-col">
                        <span className="text-sm font-bold text-[#141b2b]">
                          📍 {venue.name || "Physical Venue"}
                        </span>
                        {venue.address && (
                          <span className="text-[#565e74] mt-0.5">{venue.address}</span>
                        )}
                      </div>
                      <div className="flex flex-col">
                        <span className="text-xs font-semibold text-[#141b2b]">
                          🌐 Virtual Stream:
                        </span>
                        <span className="text-[#00685f] truncate text-[11px] font-mono">
                          {onlineUrl}
                        </span>
                      </div>
                    </div>
                  )}
                  {!isPhysical && !isOnline && !isHybrid && (
                    <span className="text-gray-400 italic">No venue selected</span>
                  )}
                </div>
              </div>
            </div>

            {/* Ticketing Summary Card */}
            <div className="bg-white border border-[#bcc9c6]/50 rounded-2xl p-6 shadow-xs flex flex-col gap-5">
              <div className="flex items-center justify-between border-b border-gray-100 pb-3">
                <h3 className="font-bold text-base text-[#141b2b]">
                  Ticketing & Pricing Structure
                </h3>
                <span className="text-xs font-semibold px-2.5 py-1 rounded-full bg-[#00685f]/10 text-[#00685f]">
                  {showSeatedLayout
                    ? "Reserved Seating"
                    : isOnline
                    ? "Online Admission"
                    : "General Admission"}
                </span>
              </div>

              {!showSeatedLayout ? (
                <div className="flex flex-col divide-y divide-gray-100">
                  {ticketTypes.map((t) => (
                    <div key={t.id} className="py-3 flex items-center justify-between text-xs">
                      <div className="flex flex-col">
                        <strong className="text-sm text-[#141b2b]">{t.name}</strong>
                        <span className="text-[#565e74]">{t.description}</span>
                      </div>
                      <div className="flex flex-col text-right">
                        <strong className="text-sm text-[#00685f]">${t.price}.00</strong>
                        <span className="text-[#565e74]">{t.capacity} capacity</span>
                      </div>
                    </div>
                  ))}
                </div>
              ) : (
                <div className="flex flex-col gap-4">
                  <div className="flex items-center justify-between bg-[#f9f9ff] border border-[#bcc9c6]/50 rounded-xl p-3 text-xs">
                    <div className="flex flex-col">
                      <strong className="text-sm text-[#141b2b]">
                        Amphitheater Seating ({seatingConfig.rows?.length || 5} Rows)
                      </strong>
                      <span className="text-[#565e74]">
                        Rows: {seatingConfig.rows?.map((r) => r.rowLetter).join(", ") || "A, B, C, D, E"} &bull; {seatingConfig.seatsPerRow || 8} seats/row
                      </span>
                    </div>
                    <div className="flex flex-col text-right">
                      <strong className="text-sm text-[#00685f]">
                        {seatingConfig.rows?.reduce((acc, r) => acc + r.seats.length, 0) || 40} Total Seats
                      </strong>
                      <span className="text-[#565e74]">Reserved Layout</span>
                    </div>
                  </div>

                  <div className="flex flex-col divide-y divide-gray-100">
                    {(seatingConfig.categories || []).map((cat) => (
                      <div key={cat.id} className="py-2.5 flex items-center justify-between text-xs">
                        <div className="flex items-center gap-2">
                          <span
                            className="size-2.5 rounded-full"
                            style={{ backgroundColor: cat.color }}
                          />
                          <strong className="text-sm text-[#141b2b]">{cat.name} Tier</strong>
                        </div>
                        <strong className="text-sm text-[#00685f]">{cat.price}</strong>
                      </div>
                    ))}
                  </div>
                </div>
              )}
            </div>

            {/* Cancellation & Visibility Card */}
            <div className="bg-white border border-[#bcc9c6]/50 rounded-2xl p-6 shadow-xs flex flex-col gap-5">
              <h3 className="font-bold text-base text-[#141b2b] border-b border-gray-100 pb-3">
                Policies & Visibility
              </h3>

              <div className="grid grid-cols-1 md:grid-cols-2 gap-6 text-xs">
                {/* Cancellation */}
                <div className="flex flex-col gap-1.5">
                  <span className="text-[#565e74] font-semibold uppercase tracking-wider text-[11px]">
                    Cancellation & Refund
                  </span>
                  {typeof eventDetails.cancellationPolicy === "object" &&
                  eventDetails.cancellationPolicy?.allowCancellation ? (
                    <div className="flex flex-col">
                      <span className="text-sm font-bold text-[#141b2b]">
                        Yes, Cancellations Allowed
                      </span>
                      <span className="text-[#565e74] mt-0.5">
                        Deadline: {eventDetails.cancellationPolicy.cancellationDeadline}
                      </span>
                      <span className="text-[#00685f] font-medium">
                        Refund: {eventDetails.cancellationPolicy.refundPolicy === "PARTIAL"
                          ? `${eventDetails.cancellationPolicy.refundPercentage}% Partial Refund`
                          : eventDetails.cancellationPolicy.refundPolicy === "FULL"
                          ? "100% Full Refund"
                          : "No Refund"}
                      </span>
                    </div>
                  ) : (
                    <div className="flex flex-col">
                      <span className="text-sm font-bold text-[#141b2b]">
                        Cancellations Not Allowed
                      </span>
                      <span className="text-[#565e74] mt-0.5">All ticket sales are final.</span>
                    </div>
                  )}
                </div>

                {/* Visibility */}
                <div className="flex flex-col gap-1.5">
                  <span className="text-[#565e74] font-semibold uppercase tracking-wider text-[11px]">
                    Event Discovery
                  </span>
                  <div className="flex flex-col">
                    <span className="text-sm font-bold text-[#141b2b] flex items-center gap-1.5">
                      {eventDetails.visibility === "PUBLIC" ? "🌐 Public Event" : "🔒 Private Event"}
                    </span>
                    <span className="text-[#565e74] mt-0.5">
                      {eventDetails.visibility === "PUBLIC"
                        ? "Discoverable in search feeds and platform recommendations."
                        : "Accessible exclusively via direct invitation / link."}
                    </span>
                  </div>
                </div>
              </div>
            </div>

            {/* Payout Bank Account Card */}
            <PayoutBankAccountCard />
          </div>

          {/* Checklist & Readiness Sidebar (4 Cols) */}
          <div className="lg:col-span-4 flex flex-col gap-6">
            <div className="bg-white border border-[#bcc9c6]/50 rounded-2xl p-6 shadow-xs flex flex-col gap-5">
              <h3 className="font-bold text-base text-[#141b2b]">Pre-Publish Checklist</h3>

              <div className="flex flex-col gap-3.5 text-xs">
                <div className="flex items-center gap-3">
                  <span className="size-5 rounded-full bg-emerald-100 text-emerald-700 flex items-center justify-center font-bold">
                    ✓
                  </span>
                  <span className="text-[#141b2b] font-medium">Basic details provided</span>
                </div>

                <div className="flex items-center gap-3">
                  <span className="size-5 rounded-full bg-emerald-100 text-emerald-700 flex items-center justify-center font-bold">
                    ✓
                  </span>
                  <span className="text-[#141b2b] font-medium">
                    {isOnline
                      ? "Virtual stream link attached"
                      : isHybrid
                      ? "Physical venue & virtual stream configured"
                      : "Google Places venue configured"}
                  </span>
                </div>

                <div className="flex items-center gap-3">
                  <span className="size-5 rounded-full bg-emerald-100 text-emerald-700 flex items-center justify-center font-bold">
                    ✓
                  </span>
                  <span className="text-[#141b2b] font-medium">Event schedule verified</span>
                </div>

                <div className="flex items-center gap-3">
                  <span className="size-5 rounded-full bg-emerald-100 text-emerald-700 flex items-center justify-center font-bold">
                    ✓
                  </span>
                  <span className="text-[#141b2b] font-medium">
                    {showSeatedLayout
                      ? "Seating layout established"
                      : isOnline
                      ? "Virtual ticket tiers ready"
                      : "General admission tiers ready"}
                  </span>
                </div>

                <div className="flex items-center gap-3">
                  <span className="size-5 rounded-full bg-emerald-100 text-emerald-700 flex items-center justify-center font-bold">
                    ✓
                  </span>
                  <span className="text-[#141b2b] font-medium">Linked bank account active</span>
                </div>
              </div>

              <div className="p-4 rounded-xl bg-[#00685f]/5 border border-[#00685f]/20 text-xs text-[#00685f] leading-relaxed">
                Your event will be immediately visible according to your visibility settings once published.
              </div>
            </div>

            {/* Organizer Contact Summary */}
            <div className="bg-white border border-[#bcc9c6]/50 rounded-2xl p-6 shadow-xs flex flex-col gap-3 text-xs">
              <h4 className="font-bold text-[#141b2b]">Organizer Contact</h4>
              <span className="text-[#565e74]">
                Contact: <strong>{eventDetails.contactInformation.name}</strong>
              </span>
              <span className="text-[#565e74]">
                Phone: <strong>{eventDetails.contactInformation.phone}</strong>
              </span>
              <span className="text-[#565e74]">
                Email: <strong>{eventDetails.contactInformation.email}</strong>
              </span>
            </div>
          </div>
        </div>
      </div>

      {/* Persistent Footer with Publish CTA */}
      <EventCreationFooter
        isReviewStep={true}
        onBack={handleBack}
        onContinue={handlePublish}
        continueLabel="🚀 Publish Event"
      />

      {/* Success / Publish Modal */}
      {showSuccessModal && (
        <div className="fixed inset-0 z-50 bg-black/50 backdrop-blur-xs flex items-center justify-center p-4">
          <div className="bg-white rounded-2xl max-w-md w-full p-8 shadow-2xl border border-[#bcc9c6] flex flex-col items-center text-center gap-5 animate-scale-up">
            <div className="size-16 rounded-full bg-emerald-100 text-emerald-700 flex items-center justify-center text-3xl">
              🎉
            </div>

            <div className="flex flex-col gap-2">
              <h3 className="text-xl font-bold text-[#141b2b]">
                Event Published Successfully!
              </h3>
              <p className="text-xs text-[#565e74] leading-relaxed">
                <strong>{basicInformation.title}</strong> has been listed and is ready for attendee bookings and ticket purchases.
              </p>
            </div>

            <div className="w-full bg-[#f9f9ff] border border-[#bcc9c6]/50 rounded-xl p-4 text-xs flex flex-col gap-1 text-left">
              <span className="text-[#565e74]">
                Mode:{" "}
                <strong className="text-[#141b2b]">
                  {showSeatedLayout
                    ? "Reserved Seating"
                    : isOnline
                    ? "Online Admission"
                    : "General Admission"}
                </strong>
              </span>
              <span className="text-[#565e74]">
                Date: <strong className="text-[#141b2b]">{dateTime.eventDate}</strong>
              </span>
              <span className="text-[#565e74]">
                Location:{" "}
                <strong className="text-[#141b2b]">
                  {isOnline
                    ? "Virtual Stream Event"
                    : isHybrid
                    ? `${venue.name || "Physical Venue"} + Virtual Stream`
                    : venue.name || "Physical Venue"}
                </strong>
              </span>
            </div>

            <button
              type="button"
              onClick={handleReturnToDashboard}
              className="w-full py-3 bg-[#00685f] hover:bg-[#005550] text-white text-sm font-semibold rounded-xl shadow-sm transition-colors cursor-pointer"
            >
              Go to Organizer Dashboard
            </button>
          </div>
        </div>
      )}
    </div>
  );
}
