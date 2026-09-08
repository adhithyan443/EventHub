import { useEffect } from "react";
import { useNavigate } from "react-router-dom";
import EventCreationStepper from "../../../components/organizer/EventCreationStepper";
import EventCreationFooter from "../../../components/organizer/EventCreationFooter";
import useEventCreationStore from "../../../store/eventCreationStore";
import {
  TICKET_MODES,
  ORGANIZER_ROUTES,
  isOnlineEvent,
} from "../../../constants/eventConstants";

export default function CreateEventStep3TicketSelectionPage() {
  const navigate = useNavigate();

  const locationType = useEventCreationStore(
    (state) => state.locationType || state.eventType
  );
  const ticketMode = useEventCreationStore((state) => state.ticketMode);
  const setTicketMode = useEventCreationStore((state) => state.setTicketMode);

  const isOnline = isOnlineEvent(locationType);

  // Online events cannot use reserved seating
  useEffect(() => {
    if (isOnline && ticketMode !== TICKET_MODES.GENERAL) {
      setTicketMode(TICKET_MODES.GENERAL);
    }
  }, [isOnline, ticketMode, setTicketMode]);

  const handleBack = () => {
    navigate(ORGANIZER_ROUTES.CREATE_STEP_2);
  };

  const handleContinue = () => {
    if (!isOnline && ticketMode === TICKET_MODES.SEATED) {
      navigate(ORGANIZER_ROUTES.CREATE_SEAT_CONFIG);
    } else {
      navigate(ORGANIZER_ROUTES.CREATE_TICKET_TYPES);
    }
  };

  return (
    <div className="min-h-full flex flex-col justify-between">
      <div className="p-8 max-w-4xl mx-auto w-full flex flex-col gap-6">
        {/* Stepper Progress Bar */}
        <EventCreationStepper currentStep={3} />

        {/* Card Container */}
        <div className="bg-white border border-[#bcc9c6]/50 rounded-2xl shadow-sm overflow-hidden flex flex-col">
          {/* Header */}
          <div className="p-6 border-b border-[#bcc9c6]/30 bg-[#f9f9ff]">
            <h1 className="text-xl font-bold text-[#141b2b]">
              How should attendees select tickets?
            </h1>
            <p className="text-xs text-[#565e74] mt-1">
              Select the ticketing mode for this event. Your choice determines the next configuration step.
            </p>
          </div>

          <div className="p-8 flex flex-col gap-6">
            {isOnline && (
              <div className="flex items-center gap-3 p-4 rounded-xl bg-[#00685f]/5 border border-[#00685f]/20 text-xs text-[#00685f]">
                <span className="text-base">ℹ️</span>
                <span>
                  <strong>Virtual Event:</strong> Online events operate with General Admission ticketing (Free stream passes, Paid access tiers, VIP links). Physical seat mapping is disabled.
                </span>
              </div>
            )}

            <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
              {/* Option 1: General Admission */}
              <div
                onClick={() => setTicketMode(TICKET_MODES.GENERAL)}
                className={`rounded-xl p-6 border-2 transition-all cursor-pointer flex flex-col justify-between gap-4 ${ticketMode === TICKET_MODES.GENERAL
                  ? "bg-[#00685f]/5 border-[#00685f] shadow-md ring-1 ring-[#00685f]/20"
                  : "bg-white border-[#bcc9c6] hover:border-gray-400 hover:shadow-xs"
                  }`}
              >
                <div className="flex items-start justify-between">
                  <div className="flex items-center gap-3">
                    <div
                      className={`size-10 rounded-lg flex items-center justify-center ${ticketMode === TICKET_MODES.GENERAL
                        ? "bg-[#00685f] text-white"
                        : "bg-gray-100 text-[#565e74]"
                        }`}
                    >
                      <svg className="size-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M15 5v2m0 4v2m0 4v2M5 5a2 2 0 00-2 2v3a2 2 0 110 4v3a2 2 0 002 2h14a2 2 0 002-2v-3a2 2 0 110-4V7a2 2 0 00-2-2H5z" />
                      </svg>
                    </div>
                    <div className="flex flex-col">
                      <h3 className="text-[15px] font-bold text-[#141b2b]">
                        General Admission
                      </h3>
                      <span className="text-[11px] text-[#00685f] font-semibold">
                        Standard Ticket Types
                      </span>
                    </div>
                  </div>

                  {/* Radio Indicator */}
                  <div
                    className={`size-5 rounded-full border-2 flex items-center justify-center ${ticketMode === TICKET_MODES.GENERAL
                      ? "border-[#00685f]"
                      : "border-[#bcc9c6]"
                      }`}
                  >
                    {ticketMode === TICKET_MODES.GENERAL && (
                      <div className="size-2.5 rounded-full bg-[#00685f]" />
                    )}
                  </div>
                </div>

                <p className="text-xs text-[#565e74] leading-relaxed">
                  Customers purchase tickets without selecting a specific seat. Best for festivals, club gigs, outdoor fairs, standing room only, and open admission events.
                </p>

                <div className="pt-2 border-t border-gray-100 flex items-center gap-2 text-[11px] text-[#00685f] font-medium">
                  <span>→ Continues to Ticket Types & Pricing</span>
                </div>
              </div>

              {/* Option 2: Reserved Seating */}
              <div
                onClick={!isOnline ? () => setTicketMode(TICKET_MODES.SEATED) : undefined}
                className={`rounded-xl p-6 border-2 transition-all flex flex-col justify-between gap-4 ${isOnline
                  ? "bg-gray-50/80 border-gray-200 opacity-60 cursor-not-allowed"
                  : ticketMode === TICKET_MODES.SEATED
                    ? "bg-[#00685f]/5 border-[#00685f] shadow-md ring-1 ring-[#00685f]/20 cursor-pointer"
                    : "bg-white border-[#bcc9c6] hover:border-gray-400 hover:shadow-xs cursor-pointer"
                  }`}
              >
                <div className="flex items-start justify-between">
                  <div className="flex items-center gap-3">
                    <div
                      className={`size-10 rounded-lg flex items-center justify-center ${!isOnline && ticketMode === TICKET_MODES.SEATED
                        ? "bg-[#00685f] text-white"
                        : "bg-gray-100 text-[#565e74]"
                        }`}
                    >
                      <svg className="size-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M19 21V5a2 2 0 00-2-2H7a2 2 0 00-2 2v16m14 0h2m-2 0h-5m-9 0H3m2 0h5M9 7h1m-1 4h1m4-4h1m-1 4h1m-5 10v-5a1 1 0 011-1h2a1 1 0 011 1v5m-4 0h4" />
                      </svg>
                    </div>
                    <div className="flex flex-col">
                      <div className="flex items-center gap-2">
                        <h3 className="text-[15px] font-bold text-[#141b2b]">
                          Reserved Seating
                        </h3>
                        {isOnline && (
                          <span className="text-[10px] font-semibold px-2 py-0.5 rounded bg-gray-200 text-gray-700">
                            Physical Only
                          </span>
                        )}
                      </div>
                      <span className="text-[11px] text-[#4648d4] font-semibold">
                        Seat Map & Layout
                      </span>
                    </div>
                  </div>

                  {/* Radio Indicator */}
                  <div
                    className={`size-5 rounded-full border-2 flex items-center justify-center ${isOnline
                      ? "border-gray-300 bg-gray-100"
                      : ticketMode === TICKET_MODES.SEATED
                        ? "border-[#00685f]"
                        : "border-[#bcc9c6]"
                      }`}
                  >
                    {!isOnline && ticketMode === TICKET_MODES.SEATED && (
                      <div className="size-2.5 rounded-full bg-[#00685f]" />
                    )}
                  </div>
                </div>

                <p className="text-xs text-[#565e74] leading-relaxed">
                  {isOnline
                    ? "Reserved seating requires a physical venue. Virtual and stream-based events admit attendees via General Admission ticket types."
                    : "Customers select a specific seat before completing their booking. Ideal for auditoriums, theatres, arenas, stadiums, and seated banquet halls with tiered sections."}
                </p>

                <div className="pt-2 border-t border-gray-100 flex items-center gap-2 text-[11px] text-[#4648d4] font-medium">
                  {isOnline ? (
                    <span className="text-gray-400">Unavailable for online events</span>
                  ) : (
                    <span>→ Continues to Interactive Seat Configuration</span>
                  )}
                </div>
              </div>
            </div>
          </div>
        </div>

        {/* Persistent Footer */}
        <EventCreationFooter
          onBack={handleBack}
          onContinue={handleContinue}
        />
      </div>
    </div>
  );
}
