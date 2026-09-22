import { useState } from "react";
import { useNavigate } from "react-router-dom";

import EventCreationStepper from "../../../components/organizer/EventCreationStepper";
import EventCreationFooter from "../../../components/organizer/EventCreationFooter";
import PayoutBankAccountCard from "../../../components/organizer/PayoutBankAccountCard";

import useEventCreationStore from "../../../store/eventCreationStore";

import {
  TICKET_MODES,
  ORGANIZER_ROUTES,
  isOnlineEvent,
  isPhysicalEvent,
  isHybridEvent,
} from "../../../constants/eventConstants";

import { createOrganizerEvent } from "../../../api/organizerApi";

import imgDefaultBanner from "../../../assets/organizer/0b2568d2a1321299cd93ab73936efb2e8bc467fe.png";

const VALID_VISIBILITIES = ["PUBLIC", "PRIVATE"];

const VALID_REFUND_POLICIES = [
  "FULL",
  "PARTIAL",
  "NO_REFUND",
];

const VALID_CANCELLATION_DEADLINES = [
  "24 hours before event",
  "48 hours before event",
  "7 days before event",
  "14 days before event",
];

const isValidEmail = (email) =>
  /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email);

const isValidPhone = (phone) =>
  /^\+?[0-9\s().-]{7,20}$/.test(phone);

const getTotalSeatingCapacity = (categories) =>
  categories.reduce(
    (total, category) =>
      total +
      (category.rows || []).reduce(
        (rowTotal, row) =>
          rowTotal + (row.seats?.length || 0),
        0
      ),
    0
  );

const getTotalSeatingRows = (categories) =>
  categories.reduce(
    (total, category) =>
      total + (category.rows?.length || 0),
    0
  );

const getSeatingRowNames = (categories) =>
  categories
    .flatMap((category) => category.rows || [])
    .map(
      (row) =>
        row.rowLetter?.trim() ||
        row.name?.trim()
    )
    .filter(Boolean);

const validateEventForPublish = ({
  basicInformation,
  locationType,
  onlineUrl,
  venue,
  dateTime,
  eventDetails,
  ticketMode,
  ticketTypes,
  seatingConfig,
  ticketSalesSettings,
}) => {
  const isOnline = isOnlineEvent(locationType);
  const isPhysical = isPhysicalEvent(locationType);
  const isHybrid = isHybridEvent(locationType);

  // --------------------------------------------------
  // Basic Information
  // --------------------------------------------------

  if (!basicInformation.title?.trim()) {
    return "Event title is required.";
  }

  if (!basicInformation.categoryId?.trim()) {
    return "Please select an event category.";
  }

  if (!basicInformation.description?.trim()) {
    return "Event description is required.";
  }

  if (!basicInformation.language?.trim()) {
    return "Event language is required.";
  }

  if (!basicInformation.bannerFile) {
    return "Please select an event banner before publishing.";
  }

  // --------------------------------------------------
  // Date & Time
  // --------------------------------------------------

  if (!dateTime.eventDate) {
    return "Event date is required.";
  }

  const eventDate = new Date(
    `${dateTime.eventDate}T00:00:00`
  );

  if (Number.isNaN(eventDate.getTime())) {
    return "Please provide a valid event date.";
  }

  const today = new Date();

  const todayDate = new Date(
    today.getFullYear(),
    today.getMonth(),
    today.getDate()
  );

  if (eventDate < todayDate) {
    return "Event date cannot be in the past.";
  }

  if (!dateTime.isAllDay) {
    if (!dateTime.startTime) {
      return "Event start time is required.";
    }

    if (!dateTime.endTime) {
      return "Event end time is required.";
    }

    if (
      dateTime.endTime <= dateTime.startTime
    ) {
      return "Event end time must be after the start time.";
    }

    if (
      dateTime.eventDate ===
      [
        today.getFullYear(),
        String(today.getMonth() + 1).padStart(2, "0"),
        String(today.getDate()).padStart(2, "0"),
      ].join("-")
    ) {
      const [hours, minutes] =
        dateTime.startTime.split(":").map(Number);

      const startDateTime = new Date();

      startDateTime.setHours(
        hours,
        minutes,
        0,
        0
      );

      if (startDateTime <= today) {
        return "Event start time must be later than the current time.";
      }
    }
  }

  // --------------------------------------------------
  // Location
  // --------------------------------------------------

  if (isOnline || isHybrid) {
    if (!onlineUrl?.trim()) {
      return "Online events require a virtual stream or meeting URL.";
    }

    try {
      new URL(onlineUrl.trim());
    } catch {
      return "Please provide a valid online event URL.";
    }
  }

  if (isPhysical || isHybrid) {
    if (!venue?.name?.trim()) {
      return "Please select a physical venue.";
    }

    if (!venue.address?.trim()) {
      return "Venue address is required.";
    }

    if (!venue.city?.trim()) {
      return "Venue city is required.";
    }

    if (!venue.country?.trim()) {
      return "Venue country is required.";
    }

    if (
      !Number.isFinite(Number(venue.latitude)) ||
      !Number.isFinite(Number(venue.longitude))
    ) {
      return "Valid venue coordinates are required.";
    }
  }

  // --------------------------------------------------
  // Ticket Sales Settings
  // --------------------------------------------------

  if (
    !ticketSalesSettings?.salesStartDate
  ) {
    return "Ticket sales start date is required.";
  }

  if (
    !ticketSalesSettings?.salesEndDate
  ) {
    return "Ticket sales end date is required.";
  }

  if (
    ticketSalesSettings.salesStartDate >
    ticketSalesSettings.salesEndDate
  ) {
    return "Ticket sales start date must be before the end date.";
  }

  if (
    ticketSalesSettings.salesEndDate >
    dateTime.eventDate
  ) {
    return "Ticket sales end date cannot be after the event date.";
  }

  const maxTicketsPerBooking = Number(
    ticketSalesSettings.maxTicketsPerBooking
  );

  if (
    !Number.isInteger(maxTicketsPerBooking) ||
    maxTicketsPerBooking <= 0
  ) {
    return "Maximum tickets per booking must be a positive whole number.";
  }

  // --------------------------------------------------
  // General Admission
  // --------------------------------------------------

  if (
    ticketMode === TICKET_MODES.GENERAL
  ) {
    if (
      !Array.isArray(ticketTypes) ||
      ticketTypes.length === 0
    ) {
      return "Add at least one ticket type before publishing.";
    }

    let totalCapacity = 0;

    for (const ticket of ticketTypes) {
      if (!ticket.name?.trim()) {
        return "Every ticket type must have a name.";
      }

      const price = Number(ticket.price);

      if (
        !Number.isFinite(price) ||
        price < 0
      ) {
        return `Enter a valid price for the "${ticket.name}" ticket type.`;
      }

      const capacity = Number(ticket.capacity);

      if (
        !Number.isInteger(capacity) ||
        capacity <= 0
      ) {
        return `Enter a valid capacity for the "${ticket.name}" ticket type.`;
      }

      totalCapacity += capacity;
    }

    if (
      maxTicketsPerBooking > totalCapacity
    ) {
      return "Maximum tickets per booking cannot exceed total ticket capacity.";
    }
  }

  // --------------------------------------------------
  // Reserved Seating
  // --------------------------------------------------

  if (
    !isOnline &&
    ticketMode === TICKET_MODES.SEATED
  ) {
    const categories =
      seatingConfig?.categories || [];

    if (categories.length === 0) {
      return "Add at least one seating category before publishing.";
    }

    const totalCapacity =
      getTotalSeatingCapacity(categories);

    if (totalCapacity <= 0) {
      return "Generate at least one seat before publishing.";
    }

    if (
      maxTicketsPerBooking > totalCapacity
    ) {
      return "Maximum tickets per booking cannot exceed total seating capacity.";
    }

    const categoryIds = new Set(
      categories.map((category) => category.id)
    );

    const seatIds = new Set();

    for (const category of categories) {
      if (!category.name?.trim()) {
        return "Every seating category must have a name.";
      }

      const price = Number(
        String(category.price ?? "").replace(
          /[₹,\s]/g,
          ""
        )
      );

      if (
        !Number.isFinite(price) ||
        price < 0
      ) {
        return `Enter a valid price for the "${category.name}" seating category.`;
      }

      if (
        !category.rows ||
        category.rows.length === 0
      ) {
        return `The "${category.name}" category must contain at least one row.`;
      }

      const rowNames = new Set();

      for (const row of category.rows) {
        const rowName =
          row.rowLetter?.trim() ||
          row.name?.trim();

        if (!rowName) {
          return `Every row in the "${category.name}" category must have a name.`;
        }

        const normalizedRowName =
          rowName.toLowerCase();

        if (
          rowNames.has(normalizedRowName)
        ) {
          return `The "${category.name}" category contains duplicate row "${rowName}".`;
        }

        rowNames.add(normalizedRowName);

        if (
          !Array.isArray(row.seats) ||
          row.seats.length === 0
        ) {
          return `Row "${rowName}" in the "${category.name}" category must contain at least one seat.`;
        }

        for (const seat of row.seats) {
          if (
            !seat.id ||
            !seat.categoryId
          ) {
            return `The "${rowName}" row in the "${category.name}" category contains an invalid seat configuration.`;
          }

          if (
            !categoryIds.has(
              seat.categoryId
            )
          ) {
            return `The "${rowName}" row contains a seat assigned to an invalid category.`;
          }

          if (seatIds.has(seat.id)) {
            return `Duplicate seat ID "${seat.id}" was found in the seating layout.`;
          }

          seatIds.add(seat.id);
        }
      }
    }
  }

  // --------------------------------------------------
  // Cancellation Policy
  // --------------------------------------------------

  const cancellationPolicy =
    eventDetails?.cancellationPolicy;

  if (
    !cancellationPolicy ||
    typeof cancellationPolicy !== "object"
  ) {
    return "Cancellation policy is required.";
  }

  if (
    typeof cancellationPolicy.allowCancellation !==
    "boolean"
  ) {
    return "Cancellation policy configuration is invalid.";
  }

  if (
    cancellationPolicy.allowCancellation
  ) {
    if (
      !VALID_CANCELLATION_DEADLINES.includes(
        cancellationPolicy.cancellationDeadline
      )
    ) {
      return "Please select a valid cancellation deadline.";
    }

    if (
      !VALID_REFUND_POLICIES.includes(
        cancellationPolicy.refundPolicy
      )
    ) {
      return "Please select a valid refund policy.";
    }

    if (
      cancellationPolicy.refundPolicy ===
      "PARTIAL"
    ) {
      const refundPercentage = Number(
        cancellationPolicy.refundPercentage
      );

      if (
        !Number.isInteger(
          refundPercentage
        ) ||
        refundPercentage <= 0 ||
        refundPercentage > 100
      ) {
        return "Partial refund percentage must be between 1 and 100.";
      }
    }
  }

  if (
    cancellationPolicy.organizerPolicyAccepted !==
    true
  ) {
    return "You must accept the organizer cancellation policy before publishing.";
  }

  // --------------------------------------------------
  // Visibility
  // --------------------------------------------------

  if (
    !VALID_VISIBILITIES.includes(
      eventDetails?.visibility
    )
  ) {
    return "Please select a valid event visibility.";
  }

  // --------------------------------------------------
  // Organizer Contact
  // --------------------------------------------------

  const contact =
    eventDetails?.contactInformation || {};

  const hasContactInformation =
    Boolean(
      contact.name?.trim() ||
      contact.phone?.trim() ||
      contact.email?.trim()
    );

  if (hasContactInformation) {
    if (!contact.name?.trim()) {
      return "Organizer contact name is required.";
    }

    if (!contact.phone?.trim()) {
      return "Organizer contact phone is required.";
    }

    if (!isValidPhone(contact.phone.trim())) {
      return "Please provide a valid organizer contact phone number.";
    }

    if (!contact.email?.trim()) {
      return "Organizer contact email is required.";
    }

    if (!isValidEmail(contact.email.trim())) {
      return "Please provide a valid organizer contact email.";
    }
  }

  return "";
};

export default function CreateEventReviewPublishPage() {
  const navigate = useNavigate();

  const basicInformation = useEventCreationStore(
    (state) => state.basicInformation
  );

  const locationType = useEventCreationStore(
    (state) =>
      state.locationType || state.eventType
  );

  const onlineUrl = useEventCreationStore(
    (state) => state.onlineUrl
  );

  const venue = useEventCreationStore(
    (state) => state.venue
  );

  const dateTime = useEventCreationStore(
    (state) => state.dateTime
  );

  const eventDetails = useEventCreationStore(
    (state) => state.eventDetails
  );

  const ticketMode = useEventCreationStore(
    (state) => state.ticketMode
  );

  const ticketTypes = useEventCreationStore(
    (state) => state.ticketTypes
  );

  const seatingConfig = useEventCreationStore(
    (state) => state.seatingConfiguration
  );

  const ticketSalesSettings =
    useEventCreationStore(
      (state) => state.ticketSalesSettings
    );

  const resetForm = useEventCreationStore(
    (state) => state.resetForm
  );

  const [publishing, setPublishing] =
    useState(false);

  const [publishError, setPublishError] =
    useState("");

  const [showSuccessModal, setShowSuccessModal] =
    useState(false);

  const isOnline =
    isOnlineEvent(locationType);

  const isPhysical =
    isPhysicalEvent(locationType);

  const isHybrid =
    isHybridEvent(locationType);

  const showSeatedLayout =
    !isOnline &&
    ticketMode === TICKET_MODES.SEATED;

  const seatingCategories =
    seatingConfig?.categories || [];

  const totalSeatingRows =
    getTotalSeatingRows(
      seatingCategories
    );

  const seatingRowNames =
    getSeatingRowNames(
      seatingCategories
    );

  const totalSeatingCapacity =
    getTotalSeatingCapacity(
      seatingCategories
    );

  const seatingCategoryCount =
    seatingCategories.length;

  const totalTicketCapacity =
    ticketTypes.reduce(
      (total, ticket) =>
        total + (Number(ticket.capacity) || 0),
      0
    );

  const handleBack = () => {
    if (showSeatedLayout) {
      navigate(
        ORGANIZER_ROUTES.CREATE_SEAT_CONFIG
      );
    } else {
      navigate(
        ORGANIZER_ROUTES.CREATE_TICKET_TYPES
      );
    }
  };

  const handlePublish = async () => {
    if (publishing) {
      return;
    }

    setPublishError("");

    const validationError =
      validateEventForPublish({
        basicInformation,
        locationType,
        onlineUrl,
        venue,
        dateTime,
        eventDetails,
        ticketMode,
        ticketTypes,
        seatingConfig,
        ticketSalesSettings,
      });

    if (validationError) {
      setPublishError(validationError);
      return;
    }

    try {
      setPublishing(true);

      const cancellationAllowed =
        Boolean(
          eventDetails.cancellationPolicy
            ?.allowCancellation
        );

      let cancellationDeadlineHours = 0;

      if (cancellationAllowed) {
        const deadline =
          eventDetails.cancellationPolicy
            ?.cancellationDeadline || "";

        const parsedHours =
          parseInt(deadline, 10);

        if (
          Number.isInteger(parsedHours) &&
          parsedHours > 0
        ) {
          cancellationDeadlineHours =
            parsedHours;
        } else if (
          deadline.includes("7 days")
        ) {
          cancellationDeadlineHours = 168;
        } else if (
          deadline.includes("14 days")
        ) {
          cancellationDeadlineHours = 336;
        } else {
          cancellationDeadlineHours = 48;
        }
      }

      const parsedAge = parseInt(
        basicInformation.ageRestriction,
        10
      );

      const ageRestriction =
        !Number.isNaN(parsedAge) &&
          parsedAge >= 0
          ? parsedAge
          : 0;

      const eventPayload = {
        category_id:
          basicInformation.categoryId,

        event_type: locationType,

        online_url: isPhysical
          ? ""
          : onlineUrl.trim(),

        title:
          basicInformation.title.trim(),

        description:
          basicInformation.description.trim(),

        banner_url: "",

        language:
          basicInformation.language.trim(),

        age_restriction:
          ageRestriction,

        event_date:
          dateTime.eventDate,

        start_time:
          dateTime.startTime,

        end_time:
          dateTime.endTime,

        seat_layout_type:
          showSeatedLayout
            ? "SEATED"
            : "GENERAL",

        booking_limit_per_user:
          Number(
            ticketSalesSettings.maxTicketsPerBooking
          ),

        cancellation_allowed:
          cancellationAllowed,

        cancellation_deadline_hours:
          cancellationDeadlineHours,

        venue: isOnline
          ? {
            google_place_id: "",
            name: "",
            address: "",
            city: "",
            state: "",
            country: "",
            postal_code: "",
            latitude: 0,
            longitude: 0,
          }
          : {
            google_place_id:
              venue.google_place_id?.trim() ||
              "",

            name:
              venue.name?.trim() || "",

            address:
              venue.address?.trim() || "",

            city:
              venue.city?.trim() || "",

            state:
              venue.state?.trim() || "",

            country:
              venue.country?.trim() || "",

            postal_code:
              venue.postal_code?.trim() ||
              "",

            latitude:
              Number(venue.latitude),

            longitude:
              Number(venue.longitude),
          },
      };

      const formData = new FormData();

      formData.append(
        "event",
        JSON.stringify(eventPayload)
      );

      if (basicInformation.bannerFile) {
        formData.append(
          "banner",
          basicInformation.bannerFile
        );
      }

      await createOrganizerEvent(
        formData
      );

      setShowSuccessModal(true);
    } catch (err) {
      console.error(
        "Failed to create event:",
        err
      );

      const errMsg =
        err.response?.data?.message ||
        err.message ||
        "Failed to publish event. Please try again.";

      setPublishError(errMsg);
    } finally {
      setPublishing(false);
    }
  };

  const handleReturnToDashboard = () => {
    setShowSuccessModal(false);
    resetForm();

    navigate(
      ORGANIZER_ROUTES.DASHBOARD
    );
  };

  return (
    <div className="min-h-full flex flex-col justify-between">
      <div className="p-4 sm:p-6 lg:p-8 max-w-5xl mx-auto w-full flex flex-col gap-6">
        <EventCreationStepper currentStep={5} />

        <div className="flex flex-col gap-1">
          <h1 className="text-xl sm:text-2xl font-bold text-[#141b2b]">
            Review & Publish
          </h1>

          <p className="text-xs text-[#565e74]">
            Review all event information,
            ticket configurations, and location
            settings before publishing live.
          </p>
        </div>

        {publishError && (
          <div className="rounded-lg border border-[#ba1a1a]/30 bg-[#ba1a1a]/5 px-4 py-3 text-sm text-[#ba1a1a]">
            {publishError}
          </div>
        )}

        <div className="grid grid-cols-1 lg:grid-cols-12 gap-6 lg:gap-8">
          <div className="lg:col-span-8 flex flex-col gap-6">
            <div className="bg-white border border-[#bcc9c6]/50 rounded-2xl overflow-hidden shadow-xs">
              <div className="h-56 bg-gray-100 relative">
                <img
                  src={
                    basicInformation.bannerPreviewUrl ||
                    basicInformation.banner ||
                    imgDefaultBanner
                  }
                  alt={
                    basicInformation.title ||
                    "Event banner"
                  }
                  className="size-full object-cover"
                />

                <div className="absolute top-4 left-4 flex gap-2">
                  <span className="bg-[#00685f] text-white text-[11px] font-bold px-3 py-1 rounded-full uppercase tracking-wider">
                    {basicInformation.category ||
                      "Uncategorized"}
                  </span>

                  <span className="bg-white/90 backdrop-blur text-[#141b2b] text-[11px] font-bold px-3 py-1 rounded-full">
                    {basicInformation.ageRestriction
                      ? `${basicInformation.ageRestriction}+`
                      : "All Ages"}
                  </span>
                </div>
              </div>

              <div className="p-6 flex flex-col gap-4">
                <h2 className="text-2xl font-bold text-[#141b2b]">
                  {basicInformation.title ||
                    "Untitled Event"}
                </h2>

                <p className="text-sm text-[#565e74] leading-relaxed whitespace-pre-line">
                  {basicInformation.description ||
                    "No description provided."}
                </p>
              </div>
            </div>

            <div className="bg-white border border-[#bcc9c6]/50 rounded-2xl p-6 shadow-xs flex flex-col gap-5">
              <h3 className="font-bold text-base text-[#141b2b] border-b border-gray-100 pb-3">
                Schedule & Location
              </h3>

              <div className="grid grid-cols-1 md:grid-cols-2 gap-6 text-xs">
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

                <div className="flex flex-col gap-1.5">
                  <span className="text-[#565e74] font-semibold uppercase tracking-wider text-[11px]">
                    Location ({locationType})
                  </span>

                  {isPhysical && (
                    <div className="flex flex-col">
                      <span className="text-sm font-bold text-[#141b2b]">
                        📍{" "}
                        {venue.name ||
                          "Physical Venue"}
                      </span>

                      {venue.address && (
                        <span className="text-[#565e74] mt-0.5">
                          {venue.address}
                        </span>
                      )}

                      {venue.city && (
                        <span className="text-[#565e74]">
                          {venue.city}
                          {venue.state
                            ? `, ${venue.state}`
                            : ""}
                          {venue.postal_code
                            ? ` (${venue.postal_code})`
                            : ""}
                        </span>
                      )}
                    </div>
                  )}

                  {isOnline && (
                    <div className="flex flex-col">
                      <span className="text-sm font-bold text-[#141b2b]">
                        🌐 Virtual Stream /
                        Meeting
                      </span>

                      <span className="text-[#00685f] truncate mt-0.5 font-mono text-[11px]">
                        {onlineUrl ||
                          "No stream link provided"}
                      </span>
                    </div>
                  )}

                  {isHybrid && (
                    <div className="flex flex-col gap-2">
                      <div className="flex flex-col">
                        <span className="text-sm font-bold text-[#141b2b]">
                          📍{" "}
                          {venue.name ||
                            "Physical Venue"}
                        </span>

                        {venue.address && (
                          <span className="text-[#565e74] mt-0.5">
                            {venue.address}
                          </span>
                        )}
                      </div>

                      <div className="flex flex-col">
                        <span className="text-xs font-semibold text-[#141b2b]">
                          🌐 Virtual Stream:
                        </span>

                        <span className="text-[#00685f] truncate text-[11px] font-mono">
                          {onlineUrl ||
                            "No stream link provided"}
                        </span>
                      </div>
                    </div>
                  )}
                </div>
              </div>
            </div>

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
                  {ticketTypes.map((ticket) => (
                    <div
                      key={ticket.id}
                      className="py-3 flex items-center justify-between text-xs"
                    >
                      <div className="flex flex-col">
                        <strong className="text-sm text-[#141b2b]">
                          {ticket.name}
                        </strong>

                        {ticket.description && (
                          <span className="text-[#565e74]">
                            {ticket.description}
                          </span>
                        )}
                      </div>

                      <div className="flex flex-col text-right">
                        <strong className="text-sm text-[#00685f]">
                          ₹{ticket.price}
                        </strong>

                        <span className="text-[#565e74]">
                          {ticket.capacity} capacity
                        </span>
                      </div>
                    </div>
                  ))}

                  <div className="pt-3 flex justify-between text-xs">
                    <span className="text-[#565e74]">
                      Total Capacity
                    </span>

                    <strong className="text-[#141b2b]">
                      {totalTicketCapacity}
                    </strong>
                  </div>
                </div>
              ) : (
                <div className="flex flex-col gap-4">
                  <div className="flex items-center justify-between bg-[#f9f9ff] border border-[#bcc9c6]/50 rounded-xl p-3 text-xs">
                    <div className="flex flex-col">
                      <strong className="text-sm text-[#141b2b]">
                        Reserved Seating (
                        {totalSeatingRows} Rows)
                      </strong>

                      <span className="text-[#565e74]">
                        Rows:{" "}
                        {seatingRowNames.length > 0
                          ? seatingRowNames.join(", ")
                          : "No rows configured"}
                      </span>
                    </div>

                    <div className="flex flex-col text-right">
                      <strong className="text-sm text-[#00685f]">
                        {totalSeatingCapacity}{" "}
                        Total Seats
                      </strong>

                      <span className="text-[#565e74]">
                        {seatingCategoryCount}{" "}
                        {seatingCategoryCount === 1
                          ? "Category"
                          : "Categories"}
                      </span>
                    </div>
                  </div>

                  <div className="flex flex-col divide-y divide-gray-100">
                    {seatingCategories.map(
                      (category) => (
                        <div
                          key={category.id}
                          className="py-2.5 flex items-center justify-between text-xs"
                        >
                          <div className="flex items-center gap-2">
                            <span
                              className="size-2.5 rounded-full"
                              style={{
                                backgroundColor:
                                  category.color,
                              }}
                            />

                            <strong className="text-sm text-[#141b2b]">
                              {category.name}
                            </strong>
                          </div>

                          <strong className="text-sm text-[#00685f]">
                            {category.price}
                          </strong>
                        </div>
                      )
                    )}
                  </div>
                </div>
              )}
            </div>

            <div className="bg-white border border-[#bcc9c6]/50 rounded-2xl p-6 shadow-xs flex flex-col gap-5">
              <h3 className="font-bold text-base text-[#141b2b] border-b border-gray-100 pb-3">
                Policies & Visibility
              </h3>

              <div className="grid grid-cols-1 md:grid-cols-2 gap-6 text-xs">
                <div className="flex flex-col gap-1.5">
                  <span className="text-[#565e74] font-semibold uppercase tracking-wider text-[11px]">
                    Cancellation & Refund
                  </span>

                  {eventDetails.cancellationPolicy
                    ?.allowCancellation ? (
                    <div className="flex flex-col">
                      <span className="text-sm font-bold text-[#141b2b]">
                        Yes, Cancellations
                        Allowed
                      </span>

                      <span className="text-[#565e74] mt-0.5">
                        Deadline:{" "}
                        {
                          eventDetails
                            .cancellationPolicy
                            .cancellationDeadline
                        }
                      </span>

                      <span className="text-[#00685f] font-medium">
                        Refund:{" "}
                        {eventDetails
                          .cancellationPolicy
                          .refundPolicy ===
                          "PARTIAL"
                          ? `${eventDetails.cancellationPolicy.refundPercentage}% Partial Refund`
                          : eventDetails
                            .cancellationPolicy
                            .refundPolicy ===
                            "FULL"
                            ? "100% Full Refund"
                            : "No Refund"}
                      </span>
                    </div>
                  ) : (
                    <div className="flex flex-col">
                      <span className="text-sm font-bold text-[#141b2b]">
                        Cancellations Not
                        Allowed
                      </span>

                      <span className="text-[#565e74] mt-0.5">
                        All ticket sales are
                        final.
                      </span>
                    </div>
                  )}
                </div>

                <div className="flex flex-col gap-1.5">
                  <span className="text-[#565e74] font-semibold uppercase tracking-wider text-[11px]">
                    Event Discovery
                  </span>

                  <div className="flex flex-col">
                    <span className="text-sm font-bold text-[#141b2b] flex items-center gap-1.5">
                      {eventDetails.visibility ===
                        "PUBLIC"
                        ? "🌐 Public Event"
                        : "🔒 Private Event"}
                    </span>

                    <span className="text-[#565e74] mt-0.5">
                      {eventDetails.visibility ===
                        "PUBLIC"
                        ? "Discoverable in search feeds and platform recommendations."
                        : "Accessible exclusively via direct invitation / link."}
                    </span>
                  </div>
                </div>
              </div>
            </div>

            <PayoutBankAccountCard />
          </div>

          <div className="lg:col-span-4 flex flex-col gap-6">
            <div className="bg-white border border-[#bcc9c6]/50 rounded-2xl p-6 shadow-xs flex flex-col gap-5">
              <h3 className="font-bold text-base text-[#141b2b]">
                Pre-Publish Checklist
              </h3>

              <div className="flex flex-col gap-3.5 text-xs">
                <div className="flex items-center gap-3">
                  <span className="size-5 rounded-full bg-emerald-100 text-emerald-700 flex items-center justify-center font-bold">
                    ✓
                  </span>

                  <span className="text-[#141b2b] font-medium">
                    Basic details provided
                  </span>
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

                  <span className="text-[#141b2b] font-medium">
                    Event schedule verified
                  </span>
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

                  <span className="text-[#141b2b] font-medium">
                    Linked bank account active
                  </span>
                </div>
              </div>

              <div className="p-4 rounded-xl bg-[#00685f]/5 border border-[#00685f]/20 text-xs text-[#00685f] leading-relaxed">
                Your event will be immediately
                visible according to your
                visibility settings once published.
              </div>
            </div>

            <div className="bg-white border border-[#bcc9c6]/50 rounded-2xl p-6 shadow-xs flex flex-col gap-3 text-xs">
              <h4 className="font-bold text-[#141b2b]">
                Organizer Contact
              </h4>

              <span className="text-[#565e74]">
                Contact:{" "}
                <strong>
                  {eventDetails.contactInformation
                    ?.name || "Not provided"}
                </strong>
              </span>

              <span className="text-[#565e74]">
                Phone:{" "}
                <strong>
                  {eventDetails.contactInformation
                    ?.phone || "Not provided"}
                </strong>
              </span>

              <span className="text-[#565e74]">
                Email:{" "}
                <strong>
                  {eventDetails.contactInformation
                    ?.email || "Not provided"}
                </strong>
              </span>
            </div>
          </div>
        </div>
      </div>

      <EventCreationFooter
        isReviewStep={true}
        onBack={handleBack}
        onContinue={handlePublish}
        continueLabel={
          publishing
            ? "Publishing Event..."
            : "🚀 Publish Event"
        }
      />

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
                <strong>
                  {basicInformation.title}
                </strong>{" "}
                has been listed and is ready
                for attendee bookings and
                ticket purchases.
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
                Date:{" "}
                <strong className="text-[#141b2b]">
                  {dateTime.eventDate}
                </strong>
              </span>

              <span className="text-[#565e74]">
                Location:{" "}
                <strong className="text-[#141b2b]">
                  {isOnline
                    ? "Virtual Stream Event"
                    : isHybrid
                      ? `${venue.name || "Physical Venue"} + Virtual Stream`
                      : venue.name ||
                      "Physical Venue"}
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