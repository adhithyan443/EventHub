import { useNavigate } from "react-router-dom";
import { useState } from "react";
import EventCreationStepper from "../../../components/organizer/EventCreationStepper";
import EventCreationFooter from "../../../components/organizer/EventCreationFooter";
import useEventCreationStore from "../../../store/eventCreationStore";
import { ORGANIZER_ROUTES } from "../../../constants/eventConstants";

const MAX_HIGHLIGHTS_LENGTH = 1000;
const MAX_RULES_LENGTH = 1000;
const MAX_CONTACT_NAME_LENGTH = 100;
const MAX_PHONE_LENGTH = 20;

const VALID_CANCELLATION_DEADLINES = [
  "24 hours before event",
  "48 hours before event",
  "7 days before event",
  "14 days before event",
];

const VALID_REFUND_POLICIES = ["FULL", "PARTIAL", "NO_REFUND"];
const VALID_VISIBILITIES = ["PUBLIC", "PRIVATE"];

const EMAIL_REGEX = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
const PHONE_REGEX = /^\+?[0-9\s().-]{7,20}$/;

export default function CreateEventStep2DetailsPage() {
  const navigate = useNavigate();

  const [validationError, setValidationError] = useState("");

  const dateTime = useEventCreationStore((state) => state.dateTime);
  const updateDateTime = useEventCreationStore(
    (state) => state.updateDateTime
  );

  const eventDetails = useEventCreationStore((state) => state.eventDetails);
  const updateEventDetails = useEventCreationStore(
    (state) => state.updateEventDetails
  );
  const updateContactInformation = useEventCreationStore(
    (state) => state.updateContactInformation
  );
  const updateCancellationPolicy = useEventCreationStore(
    (state) => state.updateCancellationPolicy
  );
  const updateVisibility = useEventCreationStore(
    (state) => state.updateVisibility
  );

  const cancellation =
    typeof eventDetails.cancellationPolicy === "object" &&
      eventDetails.cancellationPolicy !== null
      ? eventDetails.cancellationPolicy
      : {
        allowCancellation: true,
        cancellationDeadline: "",
        refundPolicy: "",
        refundPercentage: "",
        organizerPolicyAccepted: false,
      };

  const handleBack = () => {
    setValidationError("");
    navigate(ORGANIZER_ROUTES.CREATE_STEP_1);
  };

  const handleContinue = () => {
    setValidationError("");



    if (!dateTime.eventDate) {
      setValidationError("Please select the event date.");
      return;
    }

    // Make sure the selected date is actually valid.
    const selectedDate = new Date(`${dateTime.eventDate}T00:00:00`);

    if (Number.isNaN(selectedDate.getTime())) {
      setValidationError("Please select a valid event date.");
      return;
    }

    /*
     * Prevent past event dates.
     */
    const today = new Date();
    today.setHours(0, 0, 0, 0);

    if (selectedDate < today) {
      setValidationError("Event date cannot be in the past.");
      return;
    }

    /*
     * ---------------------------------------------------------
     * 2. TIME VALIDATION
     * ---------------------------------------------------------
     */

    if (!dateTime.isAllDay) {
      if (!dateTime.startTime) {
        setValidationError("Please select the event start time.");
        return;
      }

      if (!dateTime.endTime) {
        setValidationError("Please select the event end time.");
        return;
      }

      /*
       * Validate time format.
       */
      const timeRegex = /^([01]\d|2[0-3]):([0-5]\d)$/;

      if (!timeRegex.test(dateTime.startTime)) {
        setValidationError("Please select a valid event start time.");
        return;
      }

      if (!timeRegex.test(dateTime.endTime)) {
        setValidationError("Please select a valid event end time.");
        return;
      }

      /*
       * Compare start and end times.
       */
      const startDateTime = new Date(
        `${dateTime.eventDate}T${dateTime.startTime}:00`
      );

      const endDateTime = new Date(
        `${dateTime.eventDate}T${dateTime.endTime}:00`
      );

      if (
        Number.isNaN(startDateTime.getTime()) ||
        Number.isNaN(endDateTime.getTime())
      ) {
        setValidationError("Please enter valid event date and time values.");
        return;
      }

      /*
       * End time must be after start time.
       */
      if (endDateTime <= startDateTime) {
        setValidationError("End time must be later than start time.");
        return;
      }

      /*
       * If the event is today, start time must still be
       * in the future.
       */
      const now = new Date();

      if (selectedDate.getTime() === today.getTime()) {
        if (startDateTime <= now) {
          setValidationError(
            "Event start time must be later than the current time."
          );
          return;
        }
      }
    }

    /*
     * ---------------------------------------------------------
     * 3. EVENT HIGHLIGHTS VALIDATION
     * ---------------------------------------------------------
     *
     * These fields are optional in the current UI.
     * If provided, validate their length.
     */

    const highlights = eventDetails.highlights?.trim() || "";

    if (highlights.length > MAX_HIGHLIGHTS_LENGTH) {
      setValidationError(
        `Event highlights cannot exceed ${MAX_HIGHLIGHTS_LENGTH} characters.`
      );
      return;
    }

    /*
     * ---------------------------------------------------------
     * 4. EVENT RULES VALIDATION
     * ---------------------------------------------------------
     */

    const rules = eventDetails.rules?.trim() || "";

    if (rules.length > MAX_RULES_LENGTH) {
      setValidationError(
        `Event rules cannot exceed ${MAX_RULES_LENGTH} characters.`
      );
      return;
    }

    /*
     * ---------------------------------------------------------
     * 5. CONTACT INFORMATION VALIDATION
     * ---------------------------------------------------------
     *
     * Contact information is currently optional.
     *
     * However, if the organizer starts entering contact
     * information, all three fields must be completed.
     */

    const contact = eventDetails.contactInformation || {};

    const contactName = contact.name?.trim() || "";
    const contactPhone = contact.phone?.trim() || "";
    const contactEmail = contact.email?.trim() || "";

    const hasAnyContactInformation =
      contactName !== "" ||
      contactPhone !== "" ||
      contactEmail !== "";

    if (hasAnyContactInformation) {
      if (!contactName) {
        setValidationError(
          "Please enter the organizer contact name."
        );
        return;
      }

      if (contactName.length > MAX_CONTACT_NAME_LENGTH) {
        setValidationError(
          `Contact name cannot exceed ${MAX_CONTACT_NAME_LENGTH} characters.`
        );
        return;
      }

      if (!contactPhone) {
        setValidationError(
          "Please enter the organizer contact phone number."
        );
        return;
      }

      if (contactPhone.length > MAX_PHONE_LENGTH) {
        setValidationError(
          `Contact phone number cannot exceed ${MAX_PHONE_LENGTH} characters.`
        );
        return;
      }

      if (!PHONE_REGEX.test(contactPhone)) {
        setValidationError(
          "Please enter a valid contact phone number."
        );
        return;
      }

      if (!contactEmail) {
        setValidationError(
          "Please enter the organizer contact email."
        );
        return;
      }

      if (!EMAIL_REGEX.test(contactEmail)) {
        setValidationError(
          "Please enter a valid contact email address."
        );
        return;
      }
    }

    /*
     * ---------------------------------------------------------
     * 6. CANCELLATION POLICY VALIDATION
     * ---------------------------------------------------------
     */

    if (!eventDetails.cancellationPolicy) {
      setValidationError(
        "Please configure the cancellation policy."
      );
      return;
    }

    /*
     * allowCancellation must be an actual boolean.
     */
    if (typeof cancellation.allowCancellation !== "boolean") {
      setValidationError(
        "Please select whether ticket cancellation is allowed."
      );
      return;
    }

    /*
     * When cancellation is enabled, validate all
     * cancellation configuration.
     */
    if (cancellation.allowCancellation) {
      if (!cancellation.cancellationDeadline) {
        setValidationError(
          "Please select a cancellation deadline."
        );
        return;
      }

      if (
        !VALID_CANCELLATION_DEADLINES.includes(
          cancellation.cancellationDeadline
        )
      ) {
        setValidationError(
          "Please select a valid cancellation deadline."
        );
        return;
      }

      if (!cancellation.refundPolicy) {
        setValidationError(
          "Please select a refund policy."
        );
        return;
      }

      if (!VALID_REFUND_POLICIES.includes(cancellation.refundPolicy)) {
        setValidationError(
          "Please select a valid refund policy."
        );
        return;
      }

      /*
       * Partial refund requires a valid percentage.
       */
      if (cancellation.refundPolicy === "PARTIAL") {
        const refundPercentage = Number(
          cancellation.refundPercentage
        );

        if (!Number.isFinite(refundPercentage)) {
          setValidationError(
            "Please enter a valid refund percentage."
          );
          return;
        }

        if (
          refundPercentage < 1 ||
          refundPercentage > 100
        ) {
          setValidationError(
            "Refund percentage must be between 1% and 100%."
          );
          return;
        }

        /*
         * Only allow whole-number percentages.
         */
        if (!Number.isInteger(refundPercentage)) {
          setValidationError(
            "Refund percentage must be a whole number."
          );
          return;
        }
      }
    }

    /*
     * Organizer must explicitly accept the platform policy.
     */
    if (cancellation.organizerPolicyAccepted !== true) {
      setValidationError(
        "Please accept the EventHub event cancellation policy."
      );
      return;
    }

    /*
     * ---------------------------------------------------------
     * 7. VISIBILITY VALIDATION
     * ---------------------------------------------------------
     */

    if (!VALID_VISIBILITIES.includes(eventDetails.visibility)) {
      setValidationError(
        "Please select a valid event visibility option."
      );
      return;
    }

    /*
     * ---------------------------------------------------------
     * ALL VALIDATIONS PASSED
     * ---------------------------------------------------------
     */

    navigate(ORGANIZER_ROUTES.CREATE_STEP_3);
  };

  return (
    <div className="min-h-full flex flex-col justify-between">
      <div className="p-4 sm:p-6 lg:p-8 max-w-4xl mx-auto w-full flex flex-col gap-6">
        {/* Stepper Progress Bar */}
        <EventCreationStepper currentStep={2} />

        {/* Card Container */}
        <div className="bg-white border border-[#bcc9c6]/50 rounded-2xl shadow-sm overflow-hidden flex flex-col">
          {/* Header */}
          <div className="p-4 sm:p-6 border-b border-[#bcc9c6]/30 bg-[#f9f9ff]">
            <h1 className="text-xl font-bold text-[#141b2b]">
              Date, Time & Event Details
            </h1>

            <p className="text-xs text-[#565e74] mt-1">
              Configure event scheduling, highlights, attendee
              rules, and organizer contact details.
            </p>
          </div>

          {/* Validation Error */}
          {validationError && (
            <div className="mx-4 sm:mx-8 mt-4 sm:mt-8 rounded-lg border border-[#ffdad6] bg-[#fff5f3] px-4 py-3">
              <p className="text-sm font-medium text-[#ba1a1a]">
                {validationError}
              </p>
            </div>
          )}

          <div className="p-4 sm:p-6 md:p-8 flex flex-col gap-6 sm:gap-8">
            {/* =================================================
                DATE & SCHEDULE
            ================================================== */}
            <div className="flex flex-col gap-4">
              <h3 className="font-bold text-[14px] text-[#3d4947] uppercase tracking-wider">
                Date & Schedule
              </h3>

              <div className="grid grid-cols-1 md:grid-cols-3 gap-5">
                {/* Event Date */}
                <div className="flex flex-col gap-2">
                  <label className="text-[13px] font-semibold text-[#141b2b]">
                    Event Date{" "}
                    <span className="text-[#ba1a1a]">*</span>
                  </label>

                  <input
                    type="date"
                    min={new Date().toISOString().split("T")[0]}
                    value={dateTime.eventDate}
                    onChange={(e) => {
                      setValidationError("");
                      updateDateTime({
                        eventDate: e.target.value,
                      });
                    }}
                    className="w-full bg-[#f9f9ff] border border-[#bcc9c6] rounded-lg px-4 py-2.5 text-sm text-[#141b2b] focus:outline-none focus:border-[#00685f]"
                  />
                </div>

                {/* Start Time */}
                <div className="flex flex-col gap-2">
                  <label className="text-[13px] font-semibold text-[#141b2b]">
                    Start Time{" "}
                    <span className="text-[#ba1a1a]">*</span>
                  </label>

                  <input
                    type="time"
                    disabled={dateTime.isAllDay}
                    value={dateTime.startTime}
                    onChange={(e) => {
                      setValidationError("");
                      updateDateTime({
                        startTime: e.target.value,
                      });
                    }}
                    className={`w-full bg-[#f9f9ff] border border-[#bcc9c6] rounded-lg px-4 py-2.5 text-sm text-[#141b2b] focus:outline-none focus:border-[#00685f] ${dateTime.isAllDay
                      ? "opacity-50 cursor-not-allowed"
                      : ""
                      }`}
                  />
                </div>

                {/* End Time */}
                <div className="flex flex-col gap-2">
                  <label className="text-[13px] font-semibold text-[#141b2b]">
                    End Time{" "}
                    <span className="text-[#ba1a1a]">*</span>
                  </label>

                  <input
                    type="time"
                    disabled={dateTime.isAllDay}
                    value={dateTime.endTime}
                    onChange={(e) => {
                      setValidationError("");
                      updateDateTime({
                        endTime: e.target.value,
                      });
                    }}
                    className={`w-full bg-[#f9f9ff] border border-[#bcc9c6] rounded-lg px-4 py-2.5 text-sm text-[#141b2b] focus:outline-none focus:border-[#00685f] ${dateTime.isAllDay
                      ? "opacity-50 cursor-not-allowed"
                      : ""
                      }`}
                  />
                </div>
              </div>

              {/* All Day */}
              <label className="flex items-center gap-2.5 text-xs text-[#141b2b] font-medium cursor-pointer select-none">
                <input
                  type="checkbox"
                  checked={dateTime.isAllDay}
                  onChange={(e) => {
                    setValidationError("");
                    updateDateTime({
                      isAllDay: e.target.checked,
                    });
                  }}
                  className="rounded border-[#bcc9c6] text-[#00685f] focus:ring-[#00685f] size-4"
                />

                <span>
                  This is an all-day event (no specific start /
                  end hour required)
                </span>
              </label>
            </div>

            <hr className="border-t border-[#bcc9c6]/40" />

            {/* =================================================
                HIGHLIGHTS & RULES
            ================================================== */}
            <div className="flex flex-col gap-6">
              <h3 className="font-bold text-[14px] text-[#3d4947] uppercase tracking-wider">
                Event Highlights & Guidelines
              </h3>

              {/* Highlights */}
              <div className="flex flex-col gap-2">
                <label className="text-[13px] font-semibold text-[#141b2b]">
                  Event Highlights
                </label>

                <textarea
                  rows={3}
                  maxLength={MAX_HIGHLIGHTS_LENGTH}
                  value={eventDetails.highlights}
                  onChange={(e) => {
                    setValidationError("");
                    updateEventDetails({
                      highlights: e.target.value,
                    });
                  }}
                  placeholder="Key highlights and major features of this event..."
                  className="w-full bg-[#f9f9ff] border border-[#bcc9c6] rounded-lg p-4 text-sm text-[#141b2b] focus:outline-none focus:border-[#00685f]"
                />

                <div className="flex justify-between">
                  <span className="text-[11px] text-[#565e74]">
                    Short bullet points work best to grab
                    attention.
                  </span>

                  <span className="text-[11px] text-[#565e74]">
                    {(eventDetails.highlights?.length || 0)}/{MAX_HIGHLIGHTS_LENGTH}
                  </span>
                </div>
              </div>

              {/* Rules */}
              <div className="flex flex-col gap-2">
                <label className="text-[13px] font-semibold text-[#141b2b]">
                  Event Rules & Instructions
                </label>

                <textarea
                  rows={3}
                  maxLength={MAX_RULES_LENGTH}
                  value={eventDetails.rules}
                  onChange={(e) => {
                    setValidationError("");
                    updateEventDetails({
                      rules: e.target.value,
                    });
                  }}
                  placeholder="e.g. Valid photo ID required, no outside liquids, dress code guidelines..."
                  className="w-full bg-[#f9f9ff] border border-[#bcc9c6] rounded-lg p-4 text-sm text-[#141b2b] focus:outline-none focus:border-[#00685f]"
                />

                <div className="flex justify-end">
                  <span className="text-[11px] text-[#565e74]">
                    {(eventDetails.rules?.length || 0)}/{MAX_RULES_LENGTH}
                  </span>
                </div>
              </div>
            </div>

            <hr className="border-t border-[#bcc9c6]/40" />

            {/* =================================================
                CONTACT INFORMATION
            ================================================== */}
            <div className="flex flex-col gap-4">
              <h3 className="font-bold text-[14px] text-[#3d4947] uppercase tracking-wider">
                Organizer Contact Information
              </h3>

              <div className="grid grid-cols-1 md:grid-cols-3 gap-5">
                {/* Contact Name */}
                <div className="flex flex-col gap-2">
                  <label className="text-[13px] font-semibold text-[#141b2b]">
                    Contact Name
                  </label>

                  <input
                    type="text"
                    maxLength={MAX_CONTACT_NAME_LENGTH}
                    value={eventDetails.contactInformation?.name || ""}
                    onChange={(e) => {
                      setValidationError("");
                      updateContactInformation({
                        name: e.target.value,
                      });
                    }}
                    placeholder="e.g. Help Desk"
                    className="w-full bg-[#f9f9ff] border border-[#bcc9c6] rounded-lg px-4 py-2.5 text-sm text-[#141b2b] focus:outline-none focus:border-[#00685f]"
                  />
                </div>

                {/* Contact Phone */}
                <div className="flex flex-col gap-2">
                  <label className="text-[13px] font-semibold text-[#141b2b]">
                    Contact Phone
                  </label>

                  <input
                    type="tel"
                    maxLength={MAX_PHONE_LENGTH}
                    value={eventDetails.contactInformation?.phone || ""}
                    onChange={(e) => {
                      setValidationError("");
                      updateContactInformation({
                        phone: e.target.value,
                      });
                    }}
                    placeholder="+91 9876543210"
                    className="w-full bg-[#f9f9ff] border border-[#bcc9c6] rounded-lg px-4 py-2.5 text-sm text-[#141b2b] focus:outline-none focus:border-[#00685f]"
                  />
                </div>

                {/* Contact Email */}
                <div className="flex flex-col gap-2">
                  <label className="text-[13px] font-semibold text-[#141b2b]">
                    Contact Email
                  </label>

                  <input
                    type="email"
                    value={eventDetails.contactInformation?.email || ""}
                    onChange={(e) => {
                      setValidationError("");
                      updateContactInformation({
                        email: e.target.value,
                      });
                    }}
                    placeholder="help@eventhub.com"
                    className="w-full bg-[#f9f9ff] border border-[#bcc9c6] rounded-lg px-4 py-2.5 text-sm text-[#141b2b] focus:outline-none focus:border-[#00685f]"
                  />
                </div>
              </div>
            </div>

            <hr className="border-t border-[#bcc9c6]/40" />

            {/* =================================================
                CANCELLATION POLICY
            ================================================== */}
            <div className="flex flex-col gap-5">
              <div className="flex items-center gap-2">
                <span className="text-lg">↩️</span>

                <h3 className="font-bold text-[16px] text-[#141b2b]">
                  Cancellation Policy
                </h3>
              </div>

              <div className="flex flex-col gap-3">
                <label className="text-[14px] font-semibold text-[#141b2b]">
                  Do you allow customers to cancel their
                  tickets?
                </label>

                <div className="flex flex-col gap-3">
                  {/* Allow Cancellation */}
                  <div
                    role="button"
                    tabIndex={0}
                    onClick={() => {
                      setValidationError("");
                      updateCancellationPolicy({
                        allowCancellation: true,
                      });
                    }}
                    className={`rounded-[8px] p-[13px] flex gap-[12px] items-start cursor-pointer transition-all ${cancellation.allowCancellation
                      ? "bg-[#f0fdfa] border border-[#0d9488]"
                      : "bg-white border border-[#bcc9c6] hover:border-gray-400"
                      }`}
                  >
                    <div className="pt-0.5 shrink-0">
                      {cancellation.allowCancellation ? (
                        <div className="bg-[#0d9488] rounded-full size-[18px] flex items-center justify-center">
                          <svg
                            className="size-2.5 text-white"
                            fill="none"
                            viewBox="0 0 16 16"
                            stroke="currentColor"
                            strokeWidth="2.5"
                          >
                            <path
                              strokeLinecap="round"
                              strokeLinejoin="round"
                              d="M3.5 8.5l3 3 6-6"
                            />
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

                  {/* Cancellation Not Allowed */}
                  <div
                    role="button"
                    tabIndex={0}
                    onClick={() => {
                      setValidationError("");
                      updateCancellationPolicy({
                        allowCancellation: false,
                      });
                    }}
                    className={`rounded-[8px] p-[13px] flex gap-[12px] items-start cursor-pointer transition-all ${!cancellation.allowCancellation
                      ? "bg-[#f0fdfa] border border-[#0d9488]"
                      : "bg-white border border-[#bcc9c6] hover:border-gray-400"
                      }`}
                  >
                    <div className="pt-0.5 shrink-0">
                      {!cancellation.allowCancellation ? (
                        <div className="bg-[#0d9488] rounded-full size-[18px] flex items-center justify-center">
                          <svg
                            className="size-2.5 text-white"
                            fill="none"
                            viewBox="0 0 16 16"
                            stroke="currentColor"
                            strokeWidth="2.5"
                          >
                            <path
                              strokeLinecap="round"
                              strokeLinejoin="round"
                              d="M3.5 8.5l3 3 6-6"
                            />
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

                {/* Cancellation Configuration */}
                {cancellation.allowCancellation && (
                  <div className="border-l-2 border-[#bcc9c6] pl-[18px] ml-3 flex flex-col gap-4 max-w-md my-2">
                    {/* Deadline */}
                    <div className="flex flex-col gap-1">
                      <label className="text-[12px] font-medium tracking-[0.6px] text-[#565e74] uppercase">
                        Cancellation deadline
                      </label>

                      <select
                        value={
                          cancellation.cancellationDeadline || ""
                        }
                        onChange={(e) => {
                          setValidationError("");
                          updateCancellationPolicy({
                            cancellationDeadline:
                              e.target.value,
                          });
                        }}
                        className="w-full bg-[#f9f9ff] border border-[#bcc9c6] rounded-[8px] px-[13px] py-[9px] text-[14px] text-[#141b2b] focus:outline-none focus:border-[#00685f]"
                      >
                        <option value="" disabled>
                          Select cancellation deadline
                        </option>

                        <option value="24 hours before event">
                          24 hours before event
                        </option>

                        <option value="48 hours before event">
                          48 hours before event
                        </option>

                        <option value="7 days before event">
                          7 days before event
                        </option>

                        <option value="14 days before event">
                          14 days before event
                        </option>
                      </select>
                    </div>

                    {/* Refund Policy */}
                    <div className="flex flex-col gap-2">
                      <label className="text-[12px] font-medium tracking-[0.6px] text-[#565e74] uppercase">
                        Refund Policy
                      </label>

                      <div className="flex items-center gap-6">
                        {[
                          { id: "FULL", label: "Full" },
                          {
                            id: "PARTIAL",
                            label: "Partial",
                          },
                          {
                            id: "NO_REFUND",
                            label: "No Refund",
                          },
                        ].map((opt) => {
                          const isChecked =
                            cancellation.refundPolicy ===
                            opt.id;

                          return (
                            <label
                              key={opt.id}
                              onClick={() => {
                                setValidationError("");
                                updateCancellationPolicy({
                                  refundPolicy: opt.id,
                                });
                              }}
                              className="flex items-center gap-2 cursor-pointer select-none"
                            >
                              {isChecked ? (
                                <div className="bg-[#0d9488] rounded-full size-[18px] flex items-center justify-center">
                                  <svg
                                    className="size-2.5 text-white"
                                    fill="none"
                                    viewBox="0 0 16 16"
                                    stroke="currentColor"
                                    strokeWidth="2.5"
                                  >
                                    <path
                                      strokeLinecap="round"
                                      strokeLinejoin="round"
                                      d="M3.5 8.5l3 3 6-6"
                                    />
                                  </svg>
                                </div>
                              ) : (
                                <div className="bg-white border border-[#6b7280] rounded-full size-[16px]" />
                              )}

                              <span className="text-[14px] text-[#141b2b]">
                                {opt.label}
                              </span>
                            </label>
                          );
                        })}
                      </div>
                    </div>

                    {/* Partial Refund */}
                    {cancellation.refundPolicy ===
                      "PARTIAL" && (
                        <div className="bg-[#f1f3ff] border border-[#bcc9c6] rounded-[8px] p-[13px] flex items-center justify-between">
                          <span className="text-[14px] text-[#141b2b]">
                            Refund Percentage
                          </span>

                          <div className="flex items-center gap-2">
                            <input
                              type="number"
                              min="1"
                              max="100"
                              step="1"
                              value={
                                cancellation.refundPercentage ??
                                ""
                              }
                              onChange={(e) => {
                                setValidationError("");

                                updateCancellationPolicy({
                                  refundPercentage:
                                    e.target.value === ""
                                      ? ""
                                      : Number(e.target.value),
                                });
                              }}
                              className="w-[80px] bg-[#f9f9ff] border border-[#bcc9c6] rounded-[6px] px-[9px] py-[5px] text-center text-[14px] text-[#141b2b] focus:outline-none focus:border-[#00685f]"
                            />

                            <span className="text-[14px] text-[#565e74]">
                              %
                            </span>
                          </div>
                        </div>
                      )}
                  </div>
                )}

                {/* Organizer Cancellation Policy */}
                <div className="border-t border-[#bcc9c6]/40 pt-4 flex flex-col gap-3 mt-2">
                  <h4 className="font-semibold text-[14px] text-[#141b2b]">
                    Organizer Cancellation Policy
                  </h4>

                  <div className="bg-[rgba(255,218,214,0.2)] border border-[#ffdad6] rounded-[8px] p-[17px] flex gap-[12px] items-start">
                    <svg
                      className="size-4 shrink-0 text-[#ba1a1a] mt-0.5"
                      fill="currentColor"
                      viewBox="0 0 20 20"
                    >
                      <path
                        fillRule="evenodd"
                        d="M8.257 3.099c.765-1.36 2.722-1.36 3.486 0l5.58 9.92c.75 1.334-.213 2.98-1.742 2.98H4.42c-1.53 0-2.493-1.646-1.743-2.98l5.58-9.92zM11 13a1 1 0 11-2 0 1 1 0 012 0zm-1-8a1 1 0 00-1 1v3a1 1 0 002 0V6a1 1 0 00-1-1z"
                        clipRule="evenodd"
                      />
                    </svg>

                    <p className="text-[14px] text-[#3d4947] leading-[20px]">
                      If an organizer cancels an event, affected
                      customers should be notified and eligible
                      refunds will be processed according to the
                      platform policy.
                    </p>
                  </div>

                  <label className="flex items-center gap-3 cursor-pointer pt-1 select-none">
                    <input
                      type="checkbox"
                      checked={
                        cancellation.organizerPolicyAccepted ===
                        true
                      }
                      onChange={(e) => {
                        setValidationError("");

                        updateCancellationPolicy({
                          organizerPolicyAccepted:
                            e.target.checked,
                        });
                      }}
                      className="size-4 rounded border-[#bcc9c6] text-[#00685f] focus:ring-[#00685f]"
                    />

                    <span className="text-[14px] text-[#141b2b]">
                      I understand the EventHub event cancellation
                      policy.
                    </span>
                  </label>
                </div>
              </div>
            </div>

            <hr className="border-t border-[#bcc9c6]/40" />

            {/* =================================================
                EVENT VISIBILITY
            ================================================== */}
            <div className="flex flex-col gap-4">
              <div className="flex items-center gap-2">
                <span className="text-lg">👁️</span>

                <h3 className="font-bold text-[16px] text-[#141b2b]">
                  Event Visibility
                </h3>
              </div>

              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                {/* Public */}
                <div
                  role="button"
                  tabIndex={0}
                  onClick={() => {
                    setValidationError("");
                    updateVisibility("PUBLIC");
                  }}
                  className={`rounded-[8px] p-[13px] flex gap-[12px] items-start cursor-pointer transition-all ${eventDetails.visibility === "PUBLIC"
                    ? "bg-[#f0fdfa] border border-[#0d9488]"
                    : "bg-white border border-[#bcc9c6] hover:border-gray-400"
                    }`}
                >
                  <div className="pt-0.5 shrink-0">
                    {eventDetails.visibility === "PUBLIC" ? (
                      <div className="bg-[#0d9488] rounded-full size-[18px] flex items-center justify-center">
                        <svg
                          className="size-2.5 text-white"
                          fill="none"
                          viewBox="0 0 16 16"
                          stroke="currentColor"
                          strokeWidth="2.5"
                        >
                          <path
                            strokeLinecap="round"
                            strokeLinejoin="round"
                            d="M3.5 8.5l3 3 6-6"
                          />
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

                      <svg
                        className="size-4 text-[#0d9488]"
                        fill="none"
                        stroke="currentColor"
                        viewBox="0 0 24 24"
                        strokeWidth="2"
                      >
                        <path
                          strokeLinecap="round"
                          strokeLinejoin="round"
                          d="M21 12a9 9 0 01-9 9m9-9a9 9 0 00-9-9m9 9H3m9 9a9 9 0 01-9-9m9 9c1.657 0 3-4.03 3-9s-1.343-9-3-9m0 18c-1.657 0-3-4.03-3-9s1.343-9 3-9m-9 9a9 9 0 019-9"
                        />
                      </svg>
                    </div>

                    <span className="text-[14px] text-[#565e74] leading-[20px]">
                      Anyone can discover and book this event.
                      It will appear in search results.
                    </span>
                  </div>
                </div>

                {/* Private */}
                <div
                  role="button"
                  tabIndex={0}
                  onClick={() => {
                    setValidationError("");
                    updateVisibility("PRIVATE");
                  }}
                  className={`rounded-[8px] p-[13px] flex gap-[12px] items-start cursor-pointer transition-all ${eventDetails.visibility === "PRIVATE"
                    ? "bg-[#f0fdfa] border border-[#0d9488]"
                    : "bg-white border border-[#bcc9c6] hover:border-gray-400"
                    }`}
                >
                  <div className="pt-0.5 shrink-0">
                    {eventDetails.visibility === "PRIVATE" ? (
                      <div className="bg-[#0d9488] rounded-full size-[18px] flex items-center justify-center">
                        <svg
                          className="size-2.5 text-white"
                          fill="none"
                          viewBox="0 0 16 16"
                          stroke="currentColor"
                          strokeWidth="2.5"
                        >
                          <path
                            strokeLinecap="round"
                            strokeLinejoin="round"
                            d="M3.5 8.5l3 3 6-6"
                          />
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

                      <svg
                        className="size-4 text-[#6d7a77]"
                        fill="none"
                        stroke="currentColor"
                        viewBox="0 0 24 24"
                        strokeWidth="2"
                      >
                        <path
                          strokeLinecap="round"
                          strokeLinejoin="round"
                          d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z"
                        />
                      </svg>
                    </div>

                    <span className="text-[14px] text-[#565e74] leading-[20px]">
                      Only customers with the direct event link
                      can access it. Hidden from search.
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