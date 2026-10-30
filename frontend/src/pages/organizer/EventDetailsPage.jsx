import { useEffect, useState } from "react";

import { useParams, useNavigate, useLocation, Link } from "react-router-dom";

import { ORGANIZER_ROUTES, getEditEventRoute } from "../../constants/eventConstants";

import {
  ArrowLeftIcon,
  EditIcon,
  ShareIcon,
  TrashIcon,
  GlobeIcon,
} from "../../components/organizer/events/OrganizerEventsIcons";

import EventDetailsContent from "../../components/organizer/events/EventDetailsContent";

import {
  getOrganizerEventById,
  deleteOrganizerEvent,
  publishOrganizerEvent,
} from "../../api/organizerApi";

import notify from "../../utils/toast";
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

  useEffect(() => {
    if (location.state?.successMessage) {
      notify.success(location.state.successMessage);
      window.history.replaceState({}, document.title);
    }
  }, [location.state]);

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

  const fetchEventDetails = async () => {
    if (!eventId) return;

    try {
      setError("");

      const response = await getOrganizerEventById(eventId);

      if (!response?.success || !response?.data) {
        throw new Error("Unable to load event details.");
      }

      setEvent(response.data);
      return response.data;
    } catch (err) {
      console.error("Failed to fetch organizer event details:", err);
      throw err;
    }
  };

  useEffect(() => {
    let isMounted = true;
    if (eventId) {
      getOrganizerEventById(eventId)
        .then((response) => {
          if (!isMounted) return;
          if (!response?.success || !response?.data) {
            throw new Error("Unable to load event details.");
          }
          setEvent(response.data);
          setLoading(false);
        })
        .catch((err) => {
          if (!isMounted) return;
          console.error("Failed to fetch organizer event details:", err);
          setError(
            err?.response?.data?.message ||
              err?.response?.data?.error ||
              err?.message ||
              "Unable to load event details."
          );
          setLoading(false);
        });
    }
    return () => {
      isMounted = false;
    };
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

      notify.success(
        response?.message ||
          `Event "${event?.Event?.Title || "Event"}" was published successfully!`
      );
    } catch (err) {
      console.error("Failed to publish organizer event:", err);
      const errMsg =
        err?.response?.data?.message ||
        err?.response?.data?.error ||
        err?.message ||
        "Failed to publish event. Please try again.";
      setPublishError(errMsg);
      notify.error(errMsg);
    } finally {
      setIsPublishing(false);
    }
  };

  const handleShare = () => {
    if (navigator.clipboard) {
      navigator.clipboard.writeText(window.location.href);

      setCopySuccess(true);
      notify.success("Event link copied to clipboard!");

      setTimeout(() => setCopySuccess(false), 2000);
    }
  };

  const eventData = event?.Event;

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

      {/* Event Details Content */}
      <EventDetailsContent
        event={event}
        rightColumnExtra={
          <div className="bg-white border border-[#bcc9c6] rounded-xl p-5 sm:p-6 shadow-2xs flex flex-col gap-3">
            <h3 className="text-base font-bold text-[#141b2b] pb-2 border-b border-[#bcc9c6]/40">
              Event Management
            </h3>

            <div className="flex flex-col gap-2">
              <button
                type="button"
                onClick={() =>
                  notify.info(
                    `Downloading attendee report for ${eventData?.Title || "event"}...`
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
        }
      />

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
    </div>
  );
}