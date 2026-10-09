import { useEffect, useState } from "react";

import { useParams, useNavigate, useLocation, Link } from "react-router-dom";

import { ORGANIZER_ROUTES, getEditEventRoute } from "../../constants/eventConstants";

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
  TrashIcon,
  GlobeIcon,
} from "../../components/organizer/events/OrganizerEventsIcons";

import {
  getOrganizerEventById,
  deleteOrganizerEvent,
  publishOrganizerEvent,
} from "../../api/organizerApi";
import DeleteEventConfirmationModal from "../../components/organizer/events/DeleteEventConfirmationModal";
import PublishEventConfirmationModal from "../../components/organizer/events/PublishEventConfirmationModal";

export default function EventDetailsPage() {
  const { eventId } = useParams();

  const navigate = useNavigate();
  const location = useLocation();

  const [event, setEvent] = useState(null);

  const [loading, setLoading] = useState(Boolean(eventId));
  const [error, setError] = useState(
    eventId ? "" : "Event ID is missing."
  );

  const [copySuccess, setCopySuccess] = useState(false);

  const [showDeleteModal, setShowDeleteModal] = useState(false);
  const [isDeleting, setIsDeleting] = useState(false);
  const [deleteError, setDeleteError] = useState("");

  const [showPublishModal, setShowPublishModal] = useState(false);
  const [isPublishing, setIsPublishing] = useState(false);
  const [publishError, setPublishError] = useState("");

  const [toast, setToast] = useState(null);

  useEffect(() => {
    if (location.state?.successMessage) {
      setToast({
        type: "success",
        message: location.state.successMessage,
      });
      window.history.replaceState({}, document.title);
    }
  }, [location.state]);

  useEffect(() => {
    if (!toast) return;
    const timer = setTimeout(() => {
      setToast(null);
    }, 4000);
    return () => clearTimeout(timer);
  }, [toast]);

  const handleDeleteConfirm = async () => {
    if (!eventId) return;

    try {
      setIsDeleting(true);
      setDeleteError("");

      await deleteOrganizerEvent(eventId);

      setShowDeleteModal(false);
      navigate(ORGANIZER_ROUTES.MY_EVENTS, {
        state: {
          successMessage: `Event "${event?.Event?.Title || "Draft event"}" was deleted successfully.`,
        },
      });
    } catch (err) {
      console.error("Failed to delete organizer event:", err);
      const errMsg =
        err?.response?.data?.message ||
        err?.response?.data?.error ||
        err?.message ||
        "Failed to delete event. Please try again.";
      setDeleteError(errMsg);
    } finally {
      setIsDeleting(false);
    }
  };

  const fetchEventDetails = async (showPageSpinner = false) => {
    if (!eventId) return;

    try {
      if (showPageSpinner) {
        setLoading(true);
      }
      setError("");

      const response = await getOrganizerEventById(eventId);

      if (!response?.success || !response?.data) {
        throw new Error("Unable to load event details.");
      }

      setEvent(response.data);
      return response.data;
    } catch (err) {
      console.error("Failed to fetch organizer event details:", err);

      if (showPageSpinner) {
        setError(
          err?.response?.data?.message ||
            err?.response?.data?.error ||
            err?.message ||
            "Unable to load event details."
        );
      }
      throw err;
    } finally {
      if (showPageSpinner) {
        setLoading(false);
      }
    }
  };

  useEffect(() => {
    if (eventId) {
      fetchEventDetails(true);
    }
  }, [eventId]);

  const handlePublishConfirm = async () => {
    if (!eventId || isPublishing) return;

    try {
      setIsPublishing(true);
      setPublishError("");

      const response = await publishOrganizerEvent(eventId);

      // Optimistically update status to PUBLISHED immediately so the button hides
      setEvent((prev) =>
        prev
          ? {
              ...prev,
              Event: {
                ...prev.Event,
                Status: "PUBLISHED",
              },
            }
          : prev
      );

      setShowPublishModal(false);

      // Refetch event details to synchronize backend state
      try {
        await fetchEventDetails(false);
      } catch (refetchErr) {
        console.warn("Could not refetch event details after publish:", refetchErr);
      }

      setToast({
        type: "success",
        message:
          response?.message ||
          `Event "${event?.Event?.Title || "Event"}" was published successfully!`,
      });
    } catch (err) {
      console.error("Failed to publish organizer event:", err);
      const errMsg =
        err?.response?.data?.message ||
        err?.response?.data?.error ||
        err?.message ||
        "Failed to publish event. Please try again.";
      setPublishError(errMsg);
      setToast({
        type: "error",
        message: errMsg,
      });
    } finally {
      setIsPublishing(false);
    }
  };

  const handleShare = () => {
    if (navigator.clipboard) {
      navigator.clipboard.writeText(window.location.href);

      setCopySuccess(true);

      setTimeout(() => setCopySuccess(false), 2000);
    }
  };

  /*
   * API response:
   *
   * data.Event
   * data.Schedule
   * data.Venue
   * data.Setting
   * data.Cancellation
   * data.Contact
   * data.TicketTypes
   * data.SeatLayout
   * data.Statistics
   */

  const eventData = event?.Event;

  const schedule = event?.Schedule;

  const venue = event?.Venue;

  const setting = event?.Setting;

  const cancellation = event?.Cancellation;

  const contact = event?.Contact;

  const ticketTypes = event?.TicketTypes || [];

  const statistics = event?.Statistics;

  const seatLayout = event?.SeatLayout;

  const isSeated = String(setting?.SeatLayoutType || "").toUpperCase() === "SEATED";

  const formatDate = (dateValue) => {
    if (!dateValue) {
      return "N/A";
    }

    const date = new Date(dateValue);

    if (Number.isNaN(date.getTime())) {
      return dateValue;
    }

    return date.toLocaleDateString("en-IN", {
      day: "2-digit",
      month: "short",
      year: "numeric",
    });
  };

  const formatTime = (timeValue) => {
    if (!timeValue) {
      return "N/A";
    }

    const [hours, minutes] = timeValue.split(":");

    if (hours === undefined || minutes === undefined) {
      return timeValue;
    }

    const date = new Date();

    date.setHours(Number(hours), Number(minutes), 0, 0);

    return date.toLocaleTimeString("en-IN", {
      hour: "numeric",
      minute: "2-digit",
      hour12: true,
    });
  };

  const formatTimeRange = () => {
    if (schedule?.IsAllDay) {
      return "All Day";
    }

    if (!schedule?.StartTime && !schedule?.EndTime) {
      return "N/A";
    }

    return `${formatTime(schedule?.StartTime)} - ${formatTime(
      schedule?.EndTime
    )}`;
  };

  const calculateDuration = () => {
    if (!schedule?.StartTime || !schedule?.EndTime) {
      return "";
    }

    const [startHours, startMinutes] = schedule.StartTime.split(":").map(Number);

    const [endHours, endMinutes] = schedule.EndTime.split(":").map(Number);

    if (
      Number.isNaN(startHours) ||
      Number.isNaN(startMinutes) ||
      Number.isNaN(endHours) ||
      Number.isNaN(endMinutes)
    ) {
      return "";
    }

    let startTotalMinutes = startHours * 60 + startMinutes;

    let endTotalMinutes = endHours * 60 + endMinutes;

    if (endTotalMinutes < startTotalMinutes) {
      endTotalMinutes += 24 * 60;
    }

    const durationMinutes = endTotalMinutes - startTotalMinutes;

    const hours = Math.floor(durationMinutes / 60);

    const minutes = durationMinutes % 60;

    if (hours === 0) {
      return `${minutes} min`;
    }

    if (minutes === 0) {
      return `${hours} hr`;
    }

    return `${hours} hr ${minutes} min`;
  };

  const parseHighlights = (value) => {
    if (!value) {
      return [];
    }

    return value
      .split("\n")
      .map((item) => item.trim())
      .filter(Boolean);
  };

  const getTicketStatus = (ticket) => {
    if (ticket.Status === "SOLD_OUT") {
      return "Sold Out";
    }

    if (ticket.AvailableQuantity <= 0) {
      return "Sold Out";
    }

    const soldPercentage =
      ticket.TotalQuantity > 0
        ? (ticket.SoldQuantity / ticket.TotalQuantity) * 100
        : 0;

    if (soldPercentage >= 80) {
      return "Selling Fast";
    }

    return "Available";
  };

  const getCancellationDeadline = () => {
    if (!cancellation) {
      return "N/A";
    }

    if (!cancellation.CancellationAllowed) {
      return "Not Allowed";
    }

    if (!cancellation.CancellationDeadlineHours) {
      return "N/A";
    }

    return `${cancellation.CancellationDeadlineHours} hours before event`;
  };

  const getRefundPolicyLabel = () => {
    if (!cancellation) {
      return "Non-refundable";
    }

    if (!cancellation.CancellationAllowed) {
      return "Non-refundable";
    }

    if (cancellation.RefundPolicy === "FULL") {
      return "Full Refund";
    }

    if (cancellation.RefundPolicy === "PARTIAL") {
      return `Partial Refund${cancellation.RefundPercentage !== undefined
        ? ` (${cancellation.RefundPercentage}%)`
        : ""
        }`;
    }

    return "Non-refundable";
  };

  const ticketCapacity =
    statistics?.TicketCapacity ??
    ticketTypes.reduce(
      (total, ticket) => total + Number(ticket.TotalQuantity || 0),
      0
    );

  const ticketsSold =
    statistics?.TicketsSold ??
    ticketTypes.reduce(
      (total, ticket) => total + Number(ticket.SoldQuantity || 0),
      0
    );

  const percentageSold =
    statistics?.PercentageSold ??
    (ticketCapacity > 0 ? Math.round((ticketsSold / ticketCapacity) * 100) : 0);

  const revenue = statistics?.Revenue ?? 0;

  const highlights = parseHighlights(eventData?.Highlights);

  if (loading) {
    return (
      <div className="p-4 sm:p-6 lg:p-8 max-w-7xl mx-auto flex flex-col gap-6 w-full animate-fade-in">
        <div className="bg-white border border-[#bcc9c6] rounded-xl p-6 shadow-2xs">
          <div className="flex items-center justify-center min-h-48">
            <div className="flex flex-col items-center gap-3">
              <div className="size-8 border-2 border-[#00685f] border-t-transparent rounded-full animate-spin" />

              <span className="text-sm text-[#565e74]">
                Loading event details...
              </span>
            </div>
          </div>
        </div>
      </div>
    );
  }

  if (error || !eventData) {
    return (
      <div className="p-4 sm:p-6 lg:p-8 max-w-7xl mx-auto flex flex-col gap-6 w-full animate-fade-in">
        <div className="flex items-center">
          <Link
            to={ORGANIZER_ROUTES.MY_EVENTS}
            className="flex items-center gap-1.5 text-[#00685f] hover:underline font-medium text-xs sm:text-sm"
          >
            <ArrowLeftIcon className="size-4" />

            <span>Back to My Events</span>
          </Link>
        </div>

        <div className="bg-white border border-[#bcc9c6] rounded-xl p-6 shadow-2xs">
          <div className="flex flex-col items-center justify-center min-h-48 gap-3 text-center">
            <span className="text-sm font-semibold text-[#141b2b]">
              Unable to load event
            </span>

            <span className="text-xs text-[#565e74]">
              {error || "Event details could not be found."}
            </span>

            <button
              type="button"
              onClick={() => window.location.reload()}
              className="bg-[#00685f] hover:bg-[#005a52] text-white text-xs sm:text-sm font-medium px-4 py-2 rounded-lg transition-colors cursor-pointer"
            >
              Try Again
            </button>
          </div>
        </div>
      </div>
    );
  }

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
            {eventData.Title}
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

          {String(eventData?.Status || "").toUpperCase() === "DRAFT" ? (
            <button
              type="button"
              onClick={() => navigate(getEditEventRoute(eventId))}
              className="border border-[#bcc9c6] bg-white hover:bg-[#f1f3ff] text-[#141b2b] text-xs sm:text-sm font-medium px-3.5 py-2 rounded-lg flex items-center gap-1.5 transition-colors cursor-pointer"
            >
              <EditIcon className="size-3.5 text-[#565e74]" />

              <span>Edit Event</span>
            </button>
          ) : (
            <button
              type="button"
              disabled
              title="Only draft events can be edited."
              className="border border-[#bcc9c6]/50 bg-gray-50 text-[#565e74]/50 text-xs sm:text-sm font-medium px-3.5 py-2 rounded-lg flex items-center gap-1.5 cursor-not-allowed"
            >
              <EditIcon className="size-3.5 text-[#565e74]/50" />

              <span>Edit Event</span>
            </button>
          )}

          {String(eventData?.Status || "").toUpperCase() === "DRAFT" && (
            <button
              type="button"
              onClick={() => {
                setDeleteError("");
                setShowDeleteModal(true);
              }}
              className="border border-red-200 bg-white hover:bg-red-50 text-red-600 text-xs sm:text-sm font-medium px-3.5 py-2 rounded-lg flex items-center gap-1.5 transition-colors cursor-pointer"
            >
              <TrashIcon className="size-3.5 text-red-600" />

              <span>Delete Event</span>
            </button>
          )}

          {String(eventData?.Status || "").toUpperCase() === "DRAFT" && (
            <button
              type="button"
              onClick={() => {
                setPublishError("");
                setShowPublishModal(true);
              }}
              disabled={isPublishing}
              className="bg-[#00685f] hover:bg-[#005a52] text-white text-xs sm:text-sm font-medium px-4 py-2 rounded-lg flex items-center gap-1.5 transition-colors shadow-2xs cursor-pointer disabled:opacity-50 disabled:cursor-not-allowed"
            >
              <GlobeIcon className="size-3.5 text-white" />

              <span>Make Event Public</span>
            </button>
          )}

          {/* <button
            type="button"
            onClick={() => navigate("/events")}
            className="bg-[#00685f] hover:bg-[#005a52] text-white text-xs sm:text-sm font-medium px-4 py-2 rounded-lg flex items-center gap-1.5 transition-colors shadow-2xs cursor-pointer"
          >
            <ExternalLinkIcon className="size-3.5 text-white" />

            <span>View Public Page</span>
          </button> */}
        </div>
      </div>

      {/* Main Title & Status Header */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3 bg-white border border-[#bcc9c6] rounded-xl p-4 sm:p-6 shadow-2xs">
        <div className="flex flex-col gap-1.5">
          <div className="flex items-center gap-3 flex-wrap">
            <h1 className="text-xl sm:text-2xl lg:text-3xl font-bold text-[#141b2b]">
              {eventData.Title}
            </h1>

            <StatusBadge status={eventData.Status} />
          </div>

          <p className="text-xs sm:text-sm text-[#565e74]">
            Event ID:{" "}
            <span className="font-mono text-[#141b2b]">
              {eventData.ID}
            </span>{" "}
            • Created on {formatDate(eventData.CreatedAt)}
          </p>
        </div>

        <div className="flex items-center gap-2 self-start sm:self-auto">
          <span className="bg-[#f1f3ff] text-[#565e74] text-xs font-semibold px-3 py-1.5 rounded-md uppercase tracking-wider">
            {eventData.Visibility || "PUBLIC"}
          </span>

          <span className="bg-[#00685f]/10 text-[#00685f] text-xs font-semibold px-3 py-1.5 rounded-md uppercase tracking-wider">
            {eventData.CategoryName || "Uncategorized"}
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
              ₹{Number(revenue).toLocaleString("en-IN")}
            </span>

            <span className="text-xs text-[#565e74] mt-0.5">
              {statistics?.IsMock
                ? "Mock value — payments not implemented"
                : "Gross sales to date"}
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
                {ticketsSold}

                <span className="text-sm font-normal text-[#565e74]">
                  {" "}
                  / {ticketCapacity.toLocaleString()}
                </span>
              </span>

              <span className="text-xs font-bold text-[#00685f]">
                {percentageSold}%
              </span>
            </div>

            <ProgressBar pct={percentageSold} />
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
              {formatDate(schedule?.EventDate)}
            </span>

            <span className="text-xs text-[#565e74] mt-0.5">
              {formatTimeRange()}
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
              {venue?.Name || "Online Event"}
            </span>

            <span className="text-xs text-[#565e74] mt-0.5 truncate">
              {venue?.City ||
                (eventData.EventType === "ONLINE"
                  ? "Online"
                  : "Location unavailable")}
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
              {eventData.BannerImageURL ? (
                <img
                  src={eventData.BannerImageURL}
                  alt={eventData.Title}
                  className="size-full object-cover"
                />
              ) : (
                <div className="size-full flex items-center justify-center text-sm text-[#565e74]">
                  No event banner
                </div>
              )}

              <div className="absolute top-4 left-4 flex gap-2">
                <span className="bg-[#00685f] text-white text-[11px] font-bold px-3 py-1 rounded-full uppercase tracking-wider shadow-sm">
                  {eventData.CategoryName || "Uncategorized"}
                </span>

                <span className="bg-white/90 backdrop-blur-xs text-[#141b2b] text-[11px] font-bold px-3 py-1 rounded-full shadow-sm">
                  {eventData.AgeRestriction
                    ? `${eventData.AgeRestriction}+`
                    : "All Ages"}
                </span>
              </div>
            </div>

            <div className="p-5 sm:p-6 flex flex-col gap-4">
              <h2 className="text-xl font-bold text-[#141b2b]">
                About This Event
              </h2>

              <p className="text-sm text-[#565e74] leading-relaxed whitespace-pre-line">
                {eventData.Description || "No description available."}
              </p>

              {/* Highlights */}
              {highlights.length > 0 && (
                <div className="mt-2 pt-4 border-t border-[#bcc9c6]/40 flex flex-col gap-2.5">
                  <h3 className="text-xs font-bold text-[#141b2b] uppercase tracking-wider">
                    Key Highlights
                  </h3>

                  <div className="grid grid-cols-1 sm:grid-cols-2 gap-2">
                    {highlights.map((highlight, index) => (
                      <div
                        key={index}
                        className="flex items-start gap-2 text-xs sm:text-sm text-[#565e74]"
                      >
                        <CheckCircleIcon className="size-4 text-[#00685f] shrink-0 mt-0.5" />

                        <span>{highlight}</span>
                      </div>
                    ))}
                  </div>
                </div>
              )}
            </div>
          </div>

          {/* Ticket Tiers Breakdown Card */}
          <div className="bg-white border border-[#bcc9c6] rounded-xl p-5 sm:p-6 shadow-2xs flex flex-col gap-4">
            <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3 pb-3 border-b border-[#bcc9c6]/40">
              <div className="flex flex-col gap-1">
                <div className="flex items-center gap-2.5 flex-wrap">
                  <h2 className="text-lg font-bold text-[#141b2b]">
                    Ticket Tiers & Inventory
                  </h2>

                  <span
                    className={`text-xs font-semibold px-2.5 py-0.5 rounded-full ${
                      isSeated
                        ? "bg-purple-50 text-purple-700 border border-purple-200"
                        : "bg-blue-50 text-blue-700 border border-blue-200"
                    }`}
                  >
                    {isSeated ? "Reserved Seating" : "General Admission"}
                  </span>
                </div>

                <p className="text-xs text-[#565e74]">
                  {ticketTypes.length}{" "}
                  {isSeated
                    ? `active seating ${ticketTypes.length === 1 ? "category" : "categories"}`
                    : `active ticket tier ${ticketTypes.length === 1 ? "configuration" : "configurations"}`}
                </p>
              </div>

              <span className="text-xs font-semibold px-2.5 py-1 rounded-full bg-[#00685f]/10 text-[#00685f]">
                {ticketCapacity.toLocaleString()} Total Capacity
              </span>
            </div>

            {/* Ticket Type / Seating Mode Banner */}
            <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-2 p-3 rounded-lg bg-[#f1f3ff] border border-[#bcc9c6]/40 text-xs">
              <div className="flex items-center gap-2">
                <span className="text-[#565e74] font-medium">Ticketing Type:</span>
                <span className="font-bold text-[#141b2b]">
                  {isSeated ? "Reserved Seating" : "General Admission"}
                </span>
              </div>

              <span className="text-[#565e74]">
                {isSeated
                  ? "Attendees select designated reserved seats from the configured layout below."
                  : "Open seating or standing admission with standard tier-based access."}
              </span>
            </div>

            <div className="overflow-x-auto">
              <table className="w-full text-left border-collapse">
                <thead>
                  <tr className="bg-[#f1f3ff] text-[11px] font-bold uppercase tracking-wider text-[#565e74]">
                    <th className="py-2.5 px-3 rounded-l-lg">
                      {isSeated ? "Category / Tier" : "Tier Name"}
                    </th>

                    <th className="py-2.5 px-3">Price</th>

                    <th className="py-2.5 px-3">Sold / Cap</th>

                    <th className="py-2.5 px-3 min-w-[120px]">
                      Progress
                    </th>

                    <th className="py-2.5 px-3 rounded-r-lg text-right">
                      Status
                    </th>
                  </tr>
                </thead>

                <tbody className="divide-y divide-[#bcc9c6]/30 text-xs sm:text-sm">
                  {ticketTypes.length === 0 ? (
                    <tr>
                      <td colSpan={5} className="py-6 text-center text-xs text-[#565e74]">
                        No ticket tiers configured.
                      </td>
                    </tr>
                  ) : (
                    ticketTypes.map((ticket) => {
                      const capacity = Number(ticket.TotalQuantity || 0);

                      const sold = Number(ticket.SoldQuantity || 0);

                      const tierPct =
                        capacity > 0
                          ? Math.round((sold / capacity) * 100)
                          : 0;

                      const status = getTicketStatus(ticket);

                      return (
                        <tr key={ticket.ID} className="hover:bg-[#f9f9ff]">
                          <td className="py-3 px-3">
                            <div className="flex flex-col">
                              <span className="font-semibold text-[#141b2b]">
                                {ticket.Name}
                              </span>

                              {ticket.Description && ticket.Description !== ticket.Name && (
                                <span className="text-[11px] text-[#565e74] line-clamp-1">
                                  {ticket.Description}
                                </span>
                              )}
                            </div>
                          </td>

                          <td className="py-3 px-3 font-semibold text-[#141b2b] whitespace-nowrap">
                            ₹{Number(ticket.Price || 0).toLocaleString("en-IN")}
                          </td>

                          <td className="py-3 px-3 text-[#565e74] whitespace-nowrap">
                            {sold} / {capacity}
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
                              className={`inline-block px-2.5 py-0.5 rounded-full text-[11px] font-medium ${status === "Sold Out"
                                ? "bg-red-50 text-red-600 border border-red-200"
                                : status === "Selling Fast"
                                  ? "bg-amber-50 text-amber-700 border border-amber-200"
                                  : "bg-emerald-50 text-emerald-700 border border-emerald-200"
                                }`}
                            >
                              {status}
                            </span>
                          </td>
                        </tr>
                      );
                    })
                  )}
                </tbody>
              </table>
            </div>
          </div>

          {/* Configured Seat Layout Card (Only for SEATED events) */}
          {isSeated && seatLayout?.Sections && seatLayout.Sections.length > 0 && (
            <div className="bg-white border border-[#bcc9c6] rounded-xl p-5 sm:p-6 shadow-2xs flex flex-col gap-6">
              <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3 pb-3 border-b border-[#bcc9c6]/40">
                <div className="flex flex-col gap-0.5">
                  <div className="flex items-center gap-2">
                    <h2 className="text-lg font-bold text-[#141b2b]">
                      Configured Seat Layout
                    </h2>
                    <span className="text-[11px] font-semibold px-2 py-0.5 rounded-full bg-purple-50 text-purple-700 border border-purple-200">
                      Reserved Seating
                    </span>
                  </div>
                  <p className="text-xs text-[#565e74]">
                    {seatLayout.LayoutName || "Main Seating Layout"} •{" "}
                    {seatLayout.Sections.length}{" "}
                    {seatLayout.Sections.length === 1 ? "Section" : "Sections"}
                  </p>
                </div>

                {/* Seat Legend */}
                <div className="flex items-center gap-4 text-xs text-[#565e74]">
                  <div className="flex items-center gap-1.5">
                    <span className="size-3.5 rounded-md bg-white border border-[#00685f] inline-block shadow-2xs" />
                    <span>Available Seat</span>
                  </div>
                  <div className="flex items-center gap-1.5">
                    <span className="size-3.5 rounded-md bg-gray-100 border border-gray-300 inline-block line-through text-[8px] flex items-center justify-center text-gray-500 font-bold" />
                    <span>Disabled Seat</span>
                  </div>
                </div>
              </div>

              {/* Stage / Front Indicator */}
              <div className="flex flex-col items-center">
                <div className="w-full max-w-lg py-2 px-6 rounded-t-xl bg-[#f1f3ff] border border-[#bcc9c6]/50 text-center shadow-2xs">
                  <span className="text-xs font-bold tracking-widest text-[#565e74] uppercase">
                    STAGE / FRONT OF VENUE
                  </span>
                </div>
                <div className="w-full max-w-lg h-1 bg-gradient-to-r from-transparent via-[#00685f]/40 to-transparent" />
              </div>

              {/* Sections Display */}
              <div className="flex flex-col gap-6">
                {seatLayout.Sections.map((section, sIdx) => {
                  const sectionSeatCount =
                    section.Rows?.reduce(
                      (acc, r) => acc + (r.Seats?.length || 0),
                      0
                    ) || 0;

                  return (
                    <div
                      key={section.ID || sIdx}
                      className="bg-[#f9f9ff] border border-[#bcc9c6]/40 rounded-xl p-4 sm:p-5 flex flex-col gap-4 shadow-2xs"
                    >
                      {/* Section Header */}
                      <div className="flex flex-wrap items-center justify-between gap-2 pb-2.5 border-b border-[#bcc9c6]/30">
                        <div className="flex items-center gap-2">
                          <span className="size-3 rounded-full bg-[#00685f]" />
                          <h3 className="font-bold text-sm sm:text-base text-[#141b2b]">
                            {section.Name}
                          </h3>
                        </div>

                        <div className="flex items-center gap-3 text-xs">
                          <span className="font-semibold text-[#00685f]">
                            ₹{Number(section.Price || 0).toLocaleString("en-IN")} / seat
                          </span>
                          <span className="text-[#565e74] bg-white px-2 py-0.5 rounded border border-[#bcc9c6]/40">
                            {sectionSeatCount} {sectionSeatCount === 1 ? "seat" : "seats"}
                          </span>
                        </div>
                      </div>

                      {/* Rows & Seats Grid */}
                      <div className="overflow-x-auto py-2">
                        <div className="flex flex-col gap-2.5 min-w-fit mx-auto items-center">
                          {section.Rows?.map((row, rIdx) => (
                            <div
                              key={row.ID || rIdx}
                              className="flex items-center gap-3 w-full justify-center"
                            >
                              {/* Left Row Label */}
                              <span className="w-12 text-right font-bold text-xs text-[#565e74] shrink-0 select-none">
                                Row {row.RowName}
                              </span>

                              {/* Seats in Row */}
                              <div className="flex items-center gap-1.5 flex-wrap justify-center">
                                {row.Seats?.map((seat, seatIdx) => {
                                  const isDisabled = seat.Status === "DISABLED";

                                  return (
                                    <div
                                      key={seat.ID || seatIdx}
                                      title={`Row ${row.RowName}, Seat ${seat.SeatNumber}${
                                        isDisabled
                                          ? " (Disabled)"
                                          : ` - ₹${Number(section.Price || 0).toLocaleString("en-IN")}`
                                      }`}
                                      className={`size-7 sm:size-8 rounded-lg flex items-center justify-center text-[10px] sm:text-xs font-semibold select-none transition-all ${
                                        isDisabled
                                          ? "bg-gray-100 text-gray-400 border border-gray-300 line-through opacity-60 cursor-not-allowed"
                                          : "bg-white text-[#141b2b] border border-[#bcc9c6] hover:border-[#00685f] hover:bg-[#00685f]/10 hover:text-[#00685f] shadow-2xs cursor-default"
                                      }`}
                                    >
                                      {seat.SeatNumber}
                                    </div>
                                  );
                                })}
                              </div>

                              {/* Right Row Label */}
                              <span className="w-12 text-left font-bold text-xs text-[#565e74] shrink-0 select-none">
                                Row {row.RowName}
                              </span>
                            </div>
                          ))}
                        </div>
                      </div>
                    </div>
                  );
                })}
              </div>
            </div>
          )}

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
                  {eventData.Rules || "Standard venue rules apply."}
                </p>
              </div>

              <div className="flex flex-col gap-1.5">
                <span className="font-semibold text-[#141b2b] uppercase tracking-wider text-[11px] text-[#565e74]">
                  Arrival & Check-in Notes
                </span>

                <p className="text-[#565e74] whitespace-pre-line leading-relaxed">
                  {eventData.AttendeeInformation ||
                    "Check-in instructions will be delivered with ticket confirmation."}
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
                    {formatDate(schedule?.EventDate)}
                  </span>

                  <span className="text-xs text-[#565e74] flex items-center gap-1 mt-0.5">
                    <ClockIcon className="size-3 text-[#565e74]" />

                    <span>
                      {formatTimeRange()}
                      {calculateDuration()
                        ? ` (${calculateDuration()})`
                        : ""}
                    </span>
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
                    Venue ({eventData.EventType || "PHYSICAL"})
                  </span>

                  <span className="text-sm font-bold text-[#141b2b] mt-0.5">
                    {venue?.Name ||
                      (eventData.EventType === "ONLINE"
                        ? "Online Event"
                        : "Venue unavailable")}
                  </span>

                  {venue?.Address && (
                    <span className="text-xs text-[#565e74] mt-0.5">
                      {venue.Address}
                    </span>
                  )}

                  {venue?.City && (
                    <span className="text-xs text-[#565e74]">
                      {venue.City}
                      {venue.State ? `, ${venue.State}` : ""}
                      {venue.PostalCode ? ` ${venue.PostalCode}` : ""}
                    </span>
                  )}

                  {venue?.Country && (
                    <span className="text-xs text-[#565e74]">
                      {venue.Country}
                    </span>
                  )}

                  {eventData.EventType === "ONLINE" &&
                    eventData.OnlineURL && (
                      <span className="text-xs text-[#00685f] mt-1 truncate">
                        Online event link configured
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

                <span className="font-semibold text-[#141b2b]">
                  {contact?.Name || "N/A"}
                </span>
              </div>

              <div className="flex justify-between py-1 border-b border-[#bcc9c6]/20">
                <span className="text-[#565e74]">Phone:</span>

                <span className="font-semibold text-[#141b2b]">
                  {contact?.Phone || "N/A"}
                </span>
              </div>

              <div className="flex justify-between py-1">
                <span className="text-[#565e74]">Email:</span>

                <span className="font-semibold text-[#00685f] truncate max-w-[170px]">
                  {contact?.Email || "N/A"}
                </span>
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
                  {getRefundPolicyLabel()}
                </span>
              </div>

              <div className="flex justify-between py-1 border-b border-[#bcc9c6]/20">
                <span className="text-[#565e74]">Deadline:</span>

                <span className="font-semibold text-[#141b2b]">
                  {getCancellationDeadline()}
                </span>
              </div>

              {cancellation?.RefundPercentage !== undefined &&
                cancellation?.RefundPolicy === "PARTIAL" && (
                  <div className="flex justify-between py-1">
                    <span className="text-[#565e74]">Refund Percentage:</span>

                    <span className="font-semibold text-[#141b2b]">
                      {cancellation.RefundPercentage}%
                    </span>
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
                onClick={() =>
                  alert(
                    `Downloading attendee report for ${eventData.Title}`
                  )
                }
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

      {/* Delete Event Confirmation Modal */}
      <DeleteEventConfirmationModal
        isOpen={showDeleteModal}
        onClose={() => {
          if (!isDeleting) {
            setShowDeleteModal(false);
            setDeleteError("");
          }
        }}
        onConfirm={handleDeleteConfirm}
        eventTitle={eventData?.Title}
        loading={isDeleting}
        error={deleteError}
      />

      {/* Publish Event Confirmation Modal */}
      <PublishEventConfirmationModal
        isOpen={showPublishModal}
        onClose={() => {
          if (!isPublishing) {
            setShowPublishModal(false);
            setPublishError("");
          }
        }}
        onConfirm={handlePublishConfirm}
        eventTitle={eventData?.Title}
        loading={isPublishing}
        error={publishError}
      />

      {/* Floating Toast Notification */}
      {toast && (
        <div className="fixed top-20 right-4 sm:right-8 z-50 max-w-md bg-slate-900 text-white px-5 py-3.5 rounded-xl shadow-xl flex items-start gap-3 border border-slate-700 animate-slide-in">
          <div
            className={`${
              toast.type === "error" ? "text-red-400" : "text-emerald-400"
            } shrink-0 mt-0.5`}
          >
            {toast.type === "error" ? (
              <svg
                width="18"
                height="18"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                strokeWidth="2"
                strokeLinecap="round"
                strokeLinejoin="round"
              >
                <circle cx="12" cy="12" r="10" />
                <line x1="12" y1="8" x2="12" y2="12" />
                <line x1="12" y1="16" x2="12.01" y2="16" />
              </svg>
            ) : (
              <svg
                width="18"
                height="18"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                strokeWidth="2.5"
                strokeLinecap="round"
                strokeLinejoin="round"
              >
                <path d="M20 6L9 17l-5-5" />
              </svg>
            )}
          </div>

          <div className="flex-1 text-xs sm:text-sm font-medium leading-snug">
            {toast.message}
          </div>

          <button
            type="button"
            onClick={() => setToast(null)}
            className="text-slate-400 hover:text-white transition-colors cursor-pointer ml-1 shrink-0"
            aria-label="Close notification"
          >
            <svg
              width="16"
              height="16"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              strokeWidth="2"
              strokeLinecap="round"
              strokeLinejoin="round"
            >
              <path d="M18 6L6 18M6 6l12 12" />
            </svg>
          </button>
        </div>
      )}
    </div>
  );
}