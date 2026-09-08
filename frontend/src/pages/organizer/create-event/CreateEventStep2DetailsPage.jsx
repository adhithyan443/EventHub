import { useNavigate } from "react-router-dom";
import EventCreationStepper from "../../../components/organizer/EventCreationStepper";
import EventCreationFooter from "../../../components/organizer/EventCreationFooter";
import useEventCreationStore from "../../../store/eventCreationStore";
import { ORGANIZER_ROUTES } from "../../../constants/eventConstants";

export default function CreateEventStep2DetailsPage() {
  const navigate = useNavigate();

  const dateTime = useEventCreationStore((state) => state.dateTime);
  const updateDateTime = useEventCreationStore((state) => state.updateDateTime);

  const eventDetails = useEventCreationStore((state) => state.eventDetails);
  const updateEventDetails = useEventCreationStore((state) => state.updateEventDetails);
  const updateContactInformation = useEventCreationStore((state) => state.updateContactInformation);
  const updateCancellationPolicy = useEventCreationStore((state) => state.updateCancellationPolicy);
  const updateVisibility = useEventCreationStore((state) => state.updateVisibility);

  const cancellation =
    typeof eventDetails.cancellationPolicy === "object" && eventDetails.cancellationPolicy !== null
      ? eventDetails.cancellationPolicy
      : {
          allowCancellation: true,
          cancellationDeadline: "48 hours before event",
          refundPolicy: "PARTIAL",
          refundPercentage: 80,
          organizerPolicyAccepted: true,
        };

  const handleBack = () => {
    navigate(ORGANIZER_ROUTES.CREATE_STEP_1);
  };

  const handleContinue = () => {
    navigate(ORGANIZER_ROUTES.CREATE_STEP_3);
  };

  return (
    <div className="min-h-full flex flex-col justify-between">
      <div className="p-8 max-w-4xl mx-auto w-full flex flex-col gap-6">
        {/* Stepper Progress Bar */}
        <EventCreationStepper currentStep={2} />

        {/* Card Container */}
        <div className="bg-white border border-[#bcc9c6]/50 rounded-2xl shadow-sm overflow-hidden flex flex-col">
          {/* Header */}
          <div className="p-6 border-b border-[#bcc9c6]/30 bg-[#f9f9ff]">
            <h1 className="text-xl font-bold text-[#141b2b]">Date, Time & Event Details</h1>
            <p className="text-xs text-[#565e74] mt-1">
              Configure event scheduling, highlights, attendee rules, and organizer contact details.
            </p>
          </div>

          <div className="p-8 flex flex-col gap-8">
            {/* Section: Date & Time */}
            <div className="flex flex-col gap-4">
              <h3 className="font-bold text-[14px] text-[#3d4947] uppercase tracking-wider">
                Date & Schedule
              </h3>

              <div className="grid grid-cols-1 md:grid-cols-3 gap-5">
                <div className="flex flex-col gap-2">
                  <label className="text-[13px] font-semibold text-[#141b2b]">
                    Event Date <span className="text-[#ba1a1a]">*</span>
                  </label>
                  <input
                    type="date"
                    value={dateTime.eventDate}
                    onChange={(e) => updateDateTime({ eventDate: e.target.value })}
                    className="w-full bg-[#f9f9ff] border border-[#bcc9c6] rounded-lg px-4 py-2.5 text-sm text-[#141b2b] focus:outline-none focus:border-[#00685f]"
                  />
                </div>

                <div className="flex flex-col gap-2">
                  <label className="text-[13px] font-semibold text-[#141b2b]">
                    Start Time <span className="text-[#ba1a1a]">*</span>
                  </label>
                  <input
                    type="time"
                    disabled={dateTime.isAllDay}
                    value={dateTime.startTime}
                    onChange={(e) => updateDateTime({ startTime: e.target.value })}
                    className={`w-full bg-[#f9f9ff] border border-[#bcc9c6] rounded-lg px-4 py-2.5 text-sm text-[#141b2b] focus:outline-none focus:border-[#00685f] ${
                      dateTime.isAllDay ? "opacity-50 cursor-not-allowed" : ""
                    }`}
                  />
                </div>

                <div className="flex flex-col gap-2">
                  <label className="text-[13px] font-semibold text-[#141b2b]">
                    End Time <span className="text-[#ba1a1a]">*</span>
                  </label>
                  <input
                    type="time"
                    disabled={dateTime.isAllDay}
                    value={dateTime.endTime}
                    onChange={(e) => updateDateTime({ endTime: e.target.value })}
                    className={`w-full bg-[#f9f9ff] border border-[#bcc9c6] rounded-lg px-4 py-2.5 text-sm text-[#141b2b] focus:outline-none focus:border-[#00685f] ${
                      dateTime.isAllDay ? "opacity-50 cursor-not-allowed" : ""
                    }`}
                  />
                </div>
              </div>

              {/* All-Day Checkbox */}
              <label className="flex items-center gap-2.5 text-xs text-[#141b2b] font-medium cursor-pointer select-none">
                <input
                  type="checkbox"
                  checked={dateTime.isAllDay}
                  onChange={(e) => updateDateTime({ isAllDay: e.target.checked })}
                  className="rounded border-[#bcc9c6] text-[#00685f] focus:ring-[#00685f] size-4"
                />
                <span>This is an all-day event (no specific start / end hour required)</span>
              </label>
            </div>

            <hr className="border-t border-[#bcc9c6]/40" />

            {/* Section: Highlights & Rules */}
            <div className="flex flex-col gap-6">
              <h3 className="font-bold text-[14px] text-[#3d4947] uppercase tracking-wider">
                Event Highlights & Guidelines
              </h3>

              <div className="flex flex-col gap-2">
                <label className="text-[13px] font-semibold text-[#141b2b]">
                  Event Highlights
                </label>
                <textarea
                  rows={3}
                  value={eventDetails.highlights}
                  onChange={(e) => updateEventDetails({ highlights: e.target.value })}
                  placeholder="Key highlights and major features of this event..."
                  className="w-full bg-[#f9f9ff] border border-[#bcc9c6] rounded-lg p-4 text-sm text-[#141b2b] focus:outline-none focus:border-[#00685f]"
                />
                <span className="text-[11px] text-[#565e74]">Short bullet points work best to grab attention.</span>
              </div>

              <div className="flex flex-col gap-2">
                <label className="text-[13px] font-semibold text-[#141b2b]">
                  Event Rules & Instructions
                </label>
                <textarea
                  rows={3}
                  value={eventDetails.rules}
                  onChange={(e) => updateEventDetails({ rules: e.target.value })}
                  placeholder="e.g. Valid photo ID required, no outside liquids, dress code guidelines..."
                  className="w-full bg-[#f9f9ff] border border-[#bcc9c6] rounded-lg p-4 text-sm text-[#141b2b] focus:outline-none focus:border-[#00685f]"
                />
              </div>
            </div>

            <hr className="border-t border-[#bcc9c6]/40" />

            {/* Section: Contact Information */}
            <div className="flex flex-col gap-4">
              <h3 className="font-bold text-[14px] text-[#3d4947] uppercase tracking-wider">
                Organizer Contact Information
              </h3>

              <div className="grid grid-cols-1 md:grid-cols-3 gap-5">
                <div className="flex flex-col gap-2">
                  <label className="text-[13px] font-semibold text-[#141b2b]">
                    Contact Name
                  </label>
                  <input
                    type="text"
                    value={eventDetails.contactInformation.name}
                    onChange={(e) => updateContactInformation({ name: e.target.value })}
                    placeholder="e.g. Help Desk"
                    className="w-full bg-[#f9f9ff] border border-[#bcc9c6] rounded-lg px-4 py-2.5 text-sm text-[#141b2b] focus:outline-none focus:border-[#00685f]"
                  />
                </div>

                <div className="flex flex-col gap-2">
                  <label className="text-[13px] font-semibold text-[#141b2b]">
                    Contact Phone
                  </label>
                  <input
                    type="tel"
                    value={eventDetails.contactInformation.phone}
                    onChange={(e) => updateContactInformation({ phone: e.target.value })}
                    placeholder="+1 (555) 000-0000"
                    className="w-full bg-[#f9f9ff] border border-[#bcc9c6] rounded-lg px-4 py-2.5 text-sm text-[#141b2b] focus:outline-none focus:border-[#00685f]"
                  />
                </div>

                <div className="flex flex-col gap-2">
                  <label className="text-[13px] font-semibold text-[#141b2b]">
                    Contact Email
                  </label>
                  <input
                    type="email"
                    value={eventDetails.contactInformation.email}
                    onChange={(e) => updateContactInformation({ email: e.target.value })}
                    placeholder="help@eventhub.com"
                    className="w-full bg-[#f9f9ff] border border-[#bcc9c6] rounded-lg px-4 py-2.5 text-sm text-[#141b2b] focus:outline-none focus:border-[#00685f]"
                  />
                </div>
              </div>
            </div>

            <hr className="border-t border-[#bcc9c6]/40" />

            {/* Section: Cancellation Policy (Figma Fidelity) */}
            <div className="flex flex-col gap-5">
              <div className="flex items-center gap-2">
                <span className="text-lg">↩️</span>
                <h3 className="font-bold text-[16px] text-[#141b2b]">
                  Cancellation Policy
                </h3>
              </div>

              <div className="flex flex-col gap-3">
                <label className="text-[14px] font-semibold text-[#141b2b]">
                  Do you allow customers to cancel their tickets?
                </label>

                <div className="flex flex-col gap-3">
                  {/* Option 1: Yes, allow cancellation */}
                  <div
                    onClick={() => updateCancellationPolicy({ allowCancellation: true })}
                    className={`rounded-[8px] p-[13px] flex gap-[12px] items-start cursor-pointer transition-all ${
                      cancellation.allowCancellation
                        ? "bg-[#f0fdfa] border border-[#0d9488]"
                        : "bg-white border border-[#bcc9c6] hover:border-gray-400"
                    }`}
                  >
                    <div className="pt-0.5 shrink-0">
                      {cancellation.allowCancellation ? (
                        <div className="bg-[#0d9488] rounded-full size-[18px] flex items-center justify-center">
                          <svg className="size-2.5 text-white" fill="none" viewBox="0 0 16 16" stroke="currentColor" strokeWidth="2.5">
                            <path strokeLinecap="round" strokeLinejoin="round" d="M3.5 8.5l3 3 6-6" />
                          </svg>
                        </div>
                      ) : (
                        <div className="bg-white border border-[#6b7280] rounded-full size-[16px]" />
                      )}
                    </div>
                    <div className="flex flex-col">
                      <span className="font-semibold text-[14px] text-[#141b2b]">
                        Yes, allow cancellation
                      </span>
                      <span className="text-[14px] text-[#565e74]">
                        Set terms for attendee cancellations.
                      </span>
                    </div>
                  </div>

                  {/* Option 2: No, cancellation not allowed */}
                  <div
                    onClick={() => updateCancellationPolicy({ allowCancellation: false })}
                    className={`rounded-[8px] p-[13px] flex gap-[12px] items-start cursor-pointer transition-all ${
                      !cancellation.allowCancellation
                        ? "bg-[#f0fdfa] border border-[#0d9488]"
                        : "bg-white border border-[#bcc9c6] hover:border-gray-400"
                    }`}
                  >
                    <div className="pt-0.5 shrink-0">
                      {!cancellation.allowCancellation ? (
                        <div className="bg-[#0d9488] rounded-full size-[18px] flex items-center justify-center">
                          <svg className="size-2.5 text-white" fill="none" viewBox="0 0 16 16" stroke="currentColor" strokeWidth="2.5">
                            <path strokeLinecap="round" strokeLinejoin="round" d="M3.5 8.5l3 3 6-6" />
                          </svg>
                        </div>
                      ) : (
                        <div className="bg-white border border-[#6b7280] rounded-full size-[16px]" />
                      )}
                    </div>
                    <div className="flex flex-col">
                      <span className="font-semibold text-[14px] text-[#141b2b]">
                        No, cancellation not allowed
                      </span>
                      <span className="text-[14px] text-[#565e74]">
                        All sales are final.
                      </span>
                    </div>
                  </div>
                </div>

                {/* Indented Subsection for "Yes, allow cancellation" */}
                {cancellation.allowCancellation && (
                  <div className="border-l-2 border-[#bcc9c6] pl-[18px] ml-3 flex flex-col gap-4 max-w-md my-2">
                    {/* Cancellation deadline */}
                    <div className="flex flex-col gap-1">
                      <label className="text-[12px] font-medium tracking-[0.6px] text-[#565e74] uppercase">
                        Cancellation deadline
                      </label>
                      <select
                        value={cancellation.cancellationDeadline}
                        onChange={(e) =>
                          updateCancellationPolicy({ cancellationDeadline: e.target.value })
                        }
                        className="w-full bg-[#f9f9ff] border border-[#bcc9c6] rounded-[8px] px-[13px] py-[9px] text-[14px] text-[#141b2b] focus:outline-none focus:border-[#00685f]"
                      >
                        <option value="24 hours before event">24 hours before event</option>
                        <option value="48 hours before event">48 hours before event</option>
                        <option value="7 days before event">7 days before event</option>
                        <option value="14 days before event">14 days before event</option>
                      </select>
                    </div>

                    {/* Refund Policy Radios */}
                    <div className="flex flex-col gap-2">
                      <label className="text-[12px] font-medium tracking-[0.6px] text-[#565e74] uppercase">
                        Refund Policy
                      </label>
                      <div className="flex items-center gap-6">
                        {[
                          { id: "FULL", label: "Full" },
                          { id: "PARTIAL", label: "Partial" },
                          { id: "NO_REFUND", label: "No Refund" },
                        ].map((opt) => {
                          const isChecked = cancellation.refundPolicy === opt.id;
                          return (
                            <label
                              key={opt.id}
                              onClick={() => updateCancellationPolicy({ refundPolicy: opt.id })}
                              className="flex items-center gap-2 cursor-pointer select-none"
                            >
                              {isChecked ? (
                                <div className="bg-[#0d9488] rounded-full size-[18px] flex items-center justify-center">
                                  <svg className="size-2.5 text-white" fill="none" viewBox="0 0 16 16" stroke="currentColor" strokeWidth="2.5">
                                    <path strokeLinecap="round" strokeLinejoin="round" d="M3.5 8.5l3 3 6-6" />
                                  </svg>
                                </div>
                              ) : (
                                <div className="bg-white border border-[#6b7280] rounded-full size-[16px]" />
                              )}
                              <span className="text-[14px] text-[#141b2b]">{opt.label}</span>
                            </label>
                          );
                        })}
                      </div>
                    </div>

                    {/* Conditional for Partial Refund */}
                    {cancellation.refundPolicy === "PARTIAL" && (
                      <div className="bg-[#f1f3ff] border border-[#bcc9c6] rounded-[8px] p-[13px] flex items-center justify-between">
                        <span className="text-[14px] text-[#141b2b]">Refund Percentage</span>
                        <div className="flex items-center gap-2">
                          <input
                            type="number"
                            min="1"
                            max="100"
                            value={cancellation.refundPercentage}
                            onChange={(e) =>
                              updateCancellationPolicy({
                                refundPercentage: Number(e.target.value) || 0,
                              })
                            }
                            className="w-[80px] bg-[#f9f9ff] border border-[#bcc9c6] rounded-[6px] px-[9px] py-[5px] text-center text-[14px] text-[#141b2b] focus:outline-none focus:border-[#00685f]"
                          />
                          <span className="text-[14px] text-[#565e74]">%</span>
                        </div>
                      </div>
                    )}
                  </div>
                )}

                {/* Organizer Cancellation Policy Notice */}
                <div className="border-t border-[#bcc9c6]/40 pt-4 flex flex-col gap-3 mt-2">
                  <h4 className="font-semibold text-[14px] text-[#141b2b]">
                    Organizer Cancellation Policy
                  </h4>
                  <div className="bg-[rgba(255,218,214,0.2)] border border-[#ffdad6] rounded-[8px] p-[17px] flex gap-[12px] items-start">
                    <svg className="size-4 shrink-0 text-[#ba1a1a] mt-0.5" fill="currentColor" viewBox="0 0 20 20">
                      <path fillRule="evenodd" d="M8.257 3.099c.765-1.36 2.722-1.36 3.486 0l5.58 9.92c.75 1.334-.213 2.98-1.742 2.98H4.42c-1.53 0-2.493-1.646-1.743-2.98l5.58-9.92zM11 13a1 1 0 11-2 0 1 1 0 012 0zm-1-8a1 1 0 00-1 1v3a1 1 0 002 0V6a1 1 0 00-1-1z" clipRule="evenodd" />
                    </svg>
                    <p className="text-[14px] text-[#3d4947] leading-[20px]">
                      If an organizer cancels an event, affected customers should be notified and eligible refunds will be processed according to the platform policy.
                    </p>
                  </div>

                  <label className="flex items-center gap-3 cursor-pointer pt-1 select-none">
                    <input
                      type="checkbox"
                      checked={cancellation.organizerPolicyAccepted}
                      onChange={(e) =>
                        updateCancellationPolicy({ organizerPolicyAccepted: e.target.checked })
                      }
                      className="size-4 rounded border-[#bcc9c6] text-[#00685f] focus:ring-[#00685f]"
                    />
                    <span className="text-[14px] text-[#141b2b]">
                      I understand the EventHub event cancellation policy.
                    </span>
                  </label>
                </div>
              </div>
            </div>

            <hr className="border-t border-[#bcc9c6]/40" />

            {/* Section: Event Visibility (Figma Fidelity) */}
            <div className="flex flex-col gap-4">
              <div className="flex items-center gap-2">
                <span className="text-lg">👁️</span>
                <h3 className="font-bold text-[16px] text-[#141b2b]">
                  Event Visibility
                </h3>
              </div>

              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                {/* Public Option */}
                <div
                  onClick={() => updateVisibility("PUBLIC")}
                  className={`rounded-[8px] p-[13px] flex gap-[12px] items-start cursor-pointer transition-all ${
                    eventDetails.visibility === "PUBLIC"
                      ? "bg-[#f0fdfa] border border-[#0d9488]"
                      : "bg-white border border-[#bcc9c6] hover:border-gray-400"
                  }`}
                >
                  <div className="pt-0.5 shrink-0">
                    {eventDetails.visibility === "PUBLIC" ? (
                      <div className="bg-[#0d9488] rounded-full size-[18px] flex items-center justify-center">
                        <svg className="size-2.5 text-white" fill="none" viewBox="0 0 16 16" stroke="currentColor" strokeWidth="2.5">
                          <path strokeLinecap="round" strokeLinejoin="round" d="M3.5 8.5l3 3 6-6" />
                        </svg>
                      </div>
                    ) : (
                      <div className="bg-white border border-[#6b7280] rounded-full size-[16px]" />
                    )}
                  </div>
                  <div className="flex flex-col gap-1 w-full">
                    <div className="flex items-center justify-between w-full">
                      <span className="font-semibold text-[14px] text-[#141b2b]">
                        Public
                      </span>
                      <svg className="size-4 text-[#0d9488]" fill="none" stroke="currentColor" viewBox="0 0 24 24" strokeWidth="2">
                        <path strokeLinecap="round" strokeLinejoin="round" d="M21 12a9 9 0 01-9 9m9-9a9 9 0 00-9-9m9 9H3m9 9a9 9 0 01-9-9m9 9c1.657 0 3-4.03 3-9s-1.343-9-3-9m0 18c-1.657 0-3-4.03-3-9s1.343-9 3-9m-9 9a9 9 0 019-9" />
                      </svg>
                    </div>
                    <span className="text-[14px] text-[#565e74] leading-[20px]">
                      Anyone can discover and book this event. It will appear in search results.
                    </span>
                  </div>
                </div>

                {/* Private Option */}
                <div
                  onClick={() => updateVisibility("PRIVATE")}
                  className={`rounded-[8px] p-[13px] flex gap-[12px] items-start cursor-pointer transition-all ${
                    eventDetails.visibility === "PRIVATE"
                      ? "bg-[#f0fdfa] border border-[#0d9488]"
                      : "bg-white border border-[#bcc9c6] hover:border-gray-400"
                  }`}
                >
                  <div className="pt-0.5 shrink-0">
                    {eventDetails.visibility === "PRIVATE" ? (
                      <div className="bg-[#0d9488] rounded-full size-[18px] flex items-center justify-center">
                        <svg className="size-2.5 text-white" fill="none" viewBox="0 0 16 16" stroke="currentColor" strokeWidth="2.5">
                          <path strokeLinecap="round" strokeLinejoin="round" d="M3.5 8.5l3 3 6-6" />
                        </svg>
                      </div>
                    ) : (
                      <div className="bg-white border border-[#6b7280] rounded-full size-[16px]" />
                    )}
                  </div>
                  <div className="flex flex-col gap-1 w-full">
                    <div className="flex items-center justify-between w-full">
                      <span className="font-semibold text-[14px] text-[#141b2b]">
                        Private
                      </span>
                      <svg className="size-4 text-[#6d7a77]" fill="none" stroke="currentColor" viewBox="0 0 24 24" strokeWidth="2">
                        <path strokeLinecap="round" strokeLinejoin="round" d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z" />
                      </svg>
                    </div>
                    <span className="text-[14px] text-[#565e74] leading-[20px]">
                      Only customers with the direct event link can access it. Hidden from search.
                    </span>
                  </div>
                </div>
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
  );
}
