import { useState, useEffect, useRef, useCallback } from "react";
import { useParams, useNavigate, Link } from "react-router-dom";
import {
  getOrganizerEvents,
  getOrganizerEventById,
  checkInOrganizerTicket,
} from "../../api/organizerApi";
import {
  ORGANIZER_ROUTES,
  getEventDetailsRoute,
} from "../../constants/eventConstants";
import CameraScanner from "../../components/organizer/scanner/CameraScanner";
import ScanResultOverlay from "../../components/organizer/scanner/ScanResultOverlay";
import ManualCodeModal from "../../components/organizer/scanner/ManualCodeModal";
import CameraPermissionCard from "../../components/organizer/scanner/CameraPermissionCard";
import RecentScansList from "../../components/organizer/scanner/RecentScansList";
import notify from "../../utils/toast";

export default function ScanTicketsPage() {
  const { eventId: routeEventId } = useParams();
  const navigate = useNavigate();

  // Selected event state
  const [selectedEventId, setSelectedEventId] = useState(routeEventId || "");
  const [eventDetails, setEventDetails] = useState(null);
  const [loadingEvent, setLoadingEvent] = useState(Boolean(routeEventId));
  const [eventError, setEventError] = useState("");

  // Event list for selector (when no event is chosen or when changing event)
  const [eventsList, setEventsList] = useState([]);
  const [loadingEventsList, setLoadingEventsList] = useState(false);
  const [eventsListError, setEventsListError] = useState("");
  const [isChangingEvent, setIsChangingEvent] = useState(!routeEventId);

  // Scanner state
  const [isPaused, setIsPaused] = useState(false);
  const [cameraError, setCameraError] = useState(null);
  const [scanResult, setScanResult] = useState(null);
  const [recentScansByEvent, setRecentScansByEvent] = useState({});
  const [cameraRetryKey, setCameraRetryKey] = useState(0);

  // Manual entry modal state
  const [isManualModalOpen, setIsManualModalOpen] = useState(false);
  const [isSubmittingManual, setIsSubmittingManual] = useState(false);

  // Audio mute preference (default on)
  const [isAudioEnabled, setIsAudioEnabled] = useState(true);

  // Lock ref to prevent rapid concurrent duplicate scan submissions
  const isProcessingRef = useRef(false);

  // Stale response prevention: tracks generation of check-in requests & active event ID
  const checkInGenRef = useRef(0);
  const selectedEventIdRef = useRef(selectedEventId);

  useEffect(() => {
    selectedEventIdRef.current = selectedEventId;
  }, [selectedEventId]);

  // Audio tone synthesizer for gate validation feedback with explicit AudioContext lifecycle cleanup
  const playFeedbackTone = useCallback(
    async (isSuccess) => {
      if (!isAudioEnabled) return;
      try {
        const AudioCtx = window.AudioContext || window.webkitAudioContext;
        if (!AudioCtx) return;
        const ctx = new AudioCtx();

        if (ctx.state === "suspended") {
          await ctx.resume().catch(() => { });
        }

        const osc = ctx.createOscillator();
        const gain = ctx.createGain();
        osc.connect(gain);
        gain.connect(ctx.destination);

        const duration = isSuccess ? 0.25 : 0.35;
        if (isSuccess) {
          osc.type = "sine";
          osc.frequency.setValueAtTime(880, ctx.currentTime);
          osc.frequency.setValueAtTime(1320, ctx.currentTime + 0.1);
          gain.gain.setValueAtTime(0.2, ctx.currentTime);
          gain.gain.exponentialRampToValueAtTime(0.01, ctx.currentTime + duration);
        } else {
          osc.type = "sawtooth";
          osc.frequency.setValueAtTime(220, ctx.currentTime);
          gain.gain.setValueAtTime(0.25, ctx.currentTime);
          gain.gain.exponentialRampToValueAtTime(0.01, ctx.currentTime + duration);
        }

        osc.onended = () => {
          ctx.close().catch(() => { });
        };

        osc.start(ctx.currentTime);
        osc.stop(ctx.currentTime + duration);

        // Fallback cleanup timer in case onended does not fire
        setTimeout(() => {
          if (ctx.state !== "closed") {
            ctx.close().catch(() => { });
          }
        }, Math.ceil((duration + 0.15) * 1000));
      } catch {
        // Audio policy or device restrictions handled gracefully
      }
    },
    [isAudioEnabled]
  );

  // Sync route param changes safely and reset stale scan states
  useEffect(() => {
    if (routeEventId) {
      if (routeEventId !== selectedEventId) {
        setSelectedEventId(routeEventId);
        setIsChangingEvent(false);
        setScanResult(null);
        setIsPaused(false);
        isProcessingRef.current = false;
        checkInGenRef.current += 1;
      }
    } else {
      setSelectedEventId("");
      setEventDetails(null);
      setIsChangingEvent(true);
      setScanResult(null);
      setIsPaused(false);
      isProcessingRef.current = false;
      checkInGenRef.current += 1;
    }
  }, [routeEventId, selectedEventId]);

  // Fetch organizer's events list for selector
  useEffect(() => {
    let cancelled = false;

    const loadOrganizerEvents = async () => {
      setLoadingEventsList(true);
      setEventsListError("");

      try {
        const res = await getOrganizerEvents({
          page: 1,
          limit: 100,
          status: "",
        });

        if (cancelled) return;
        const events = res?.data?.events ?? [];
        setEventsList(events);
      } catch (err) {
        if (cancelled) return;
        setEventsListError("Failed to load organizer events.");
      } finally {
        if (!cancelled) setLoadingEventsList(false);
      }
    };

    loadOrganizerEvents();

    return () => {
      cancelled = true;
    };
  }, []);

  // Fetch details of currently selected event
  const fetchEventDetails = useCallback(async (eventId) => {
    if (!eventId) {
      setEventDetails(null);
      setLoadingEvent(false);
      return;
    }

    setLoadingEvent(true);
    setEventError("");

    try {
      const res = await getOrganizerEventById(eventId);
      const raw = res?.data?.Event || res?.data?.event || res?.data;
      if (!raw) {
        throw new Error("Event details could not be found.");
      }
      setEventDetails(raw);
    } catch (err) {
      const msg =
        err?.response?.data?.message ||
        err?.response?.data?.error ||
        err?.message ||
        "Could not retrieve event details.";
      setEventError(msg);
      notify.error(msg);
    } finally {
      setLoadingEvent(false);
    }
  }, []);

  useEffect(() => {
    let cancelled = false;

    if (!selectedEventId) {
      setEventDetails(null);
      setLoadingEvent(false);
      return;
    }

    const load = async () => {
      setLoadingEvent(true);
      setEventError("");

      try {
        const res = await getOrganizerEventById(selectedEventId);
        if (cancelled) return;

        const raw = res?.data?.Event || res?.data?.event || res?.data;
        if (!raw) {
          throw new Error("Event details could not be found.");
        }
        setEventDetails(raw);
      } catch (err) {
        if (cancelled) return;
        const msg =
          err?.response?.data?.message ||
          err?.response?.data?.error ||
          err?.message ||
          "Could not retrieve event details.";
        setEventError(msg);
        notify.error(msg);
      } finally {
        if (!cancelled) setLoadingEvent(false);
      }
    };

    load();

    return () => {
      cancelled = true;
    };
  }, [selectedEventId]);

  // Handler when organizer selects an event from selector
  const handleSelectEvent = (id) => {
    if (!id) return;
    checkInGenRef.current += 1;
    setSelectedEventId(id);
    setIsChangingEvent(false);
    setScanResult(null);
    setIsPaused(false);
    isProcessingRef.current = false;
    navigate(`/organizer/events/${id}/scan`, { replace: true });
  };

  // Check-in execution logic with stale-response protection
  const processCheckIn = useCallback(
    async (payload) => {
      const targetEventId = selectedEventIdRef.current;
      if (!targetEventId) {
        notify.error("Please select an event before scanning tickets.");
        return false;
      }

      const trimmedPayload = payload?.trim();
      if (!trimmedPayload) return false;

      // Processing lock: prevent concurrent overlapping requests
      if (isProcessingRef.current) return false;
      isProcessingRef.current = true;
      setIsPaused(true);

      const currentGen = ++checkInGenRef.current;

      try{

      const response = await checkInOrganizerTicket(
        targetEventId,
        trimmedPayload
      );

      // Stale response guard: ignore if event was switched during API transit
      if (
        checkInGenRef.current !== currentGen ||
        selectedEventIdRef.current !== targetEventId
      ) {
        return true;
      }

      const ticketData = response?.data;
      playFeedbackTone(true);

      const isBooking = ticketData?.type === "BOOKING";
      const successResult = {
        status: "SUCCESS",
        title: isBooking
          ? `Booking Pass Verified (${ticketData?.total_count || 1} Tickets)`
          : "Ticket Verified",
        message:
          response?.message ||
          (isBooking
            ? `Booking checked in successfully (${ticketData?.total_count || 1} tickets)`
            : "Ticket checked in successfully"),
        data: ticketData,
        timestamp: new Date(),
      };

      setScanResult(successResult);
      setRecentScansByEvent((prev) => ({
        ...prev,
        [targetEventId]: [
          {
            id: ticketData?.ticket_id || `tc-${Date.now()}`,
            ticketCode: isBooking
              ? `BOOKING (${ticketData?.total_count || 1} TIX)`
              : ticketData?.ticket_code || "Checked In",
            status: "SUCCESS",
            message: isBooking
              ? `Booking Checked In (${ticketData?.total_count || 1} tickets)`
              : "Ticket Valid",
            timestamp: new Date(),
          },
          ...(prev[targetEventId] || []).slice(0, 49),
        ],
      }));

      notify.success(
        isBooking
          ? `Booking pass verified! ${ticketData?.total_count || 1} tickets checked in.`
          : `Ticket ${ticketData?.ticket_code || "verified"} checked in!`
      );
      return true;
    } catch (err) {
      // Stale response guard: ignore if event was switched during API transit
      if (
        checkInGenRef.current !== currentGen ||
        selectedEventIdRef.current !== targetEventId
      ) {
        return true;
      }

      playFeedbackTone(false);
      const status = err?.response?.status;
      const serverMsg =
        err?.response?.data?.message ||
        err?.response?.data?.error ||
        err?.message ||
        "Check-in verification failed.";

      let resultState = "ERROR";
      let displayTitle = "Scan Failed";

      if (status === 409) {
        resultState = "CONFLICT";
        displayTitle = "Ticket Already Used or Invalid";
      } else if (status === 400) {
        resultState = "BAD_REQUEST";
        displayTitle = "Invalid QR Payload";
      } else if (status === 403) {
        resultState = "UNAUTHORIZED";
        displayTitle = "Not Authorized";
      } else if (status === 404) {
        resultState = "NOT_FOUND";
        displayTitle = "Event or Ticket Not Found";
      }

      const failResult = {
        status: resultState,
        title: displayTitle,
        message: serverMsg,
        timestamp: new Date(),
      };

      setScanResult(failResult);
      setRecentScansByEvent((prev) => ({
        ...prev,
        [targetEventId]: [
          {
            id: `err-${Date.now()}`,
            ticketCode: "Scan Failed",
            status: resultState,
            message: serverMsg,
            timestamp: new Date(),
          },
          ...(prev[targetEventId] || []).slice(0, 49),
        ],
      }));

      notify.error(serverMsg);
      return true;
    }
},
[playFeedbackTone]
  );

// Resume scanning for next attendee
const handleScanNext = useCallback(() => {
  setScanResult(null);
  isProcessingRef.current = false;
  setIsPaused(false);
}, []);

// Handle camera errors from child component
const handleCameraError = useCallback((err) => {
  setCameraError(err);
}, []);

// Retry camera initialization
const handleRetryCamera = () => {
  setCameraError(null);
  setCameraRetryKey((k) => k + 1);
};

// Submit manual code modal: only close when submission is accepted for processing
const handleManualCodeSubmit = async (payload) => {
  setIsSubmittingManual(true);
  try {
    const accepted = await processCheckIn(payload);
    if (accepted) {
      setIsManualModalOpen(false);
    }
  } finally {
    setIsSubmittingManual(false);
  }
};

// Clear scans for active event only
const handleClearRecentScans = () => {
  if (!selectedEventId) return;
  setRecentScansByEvent((prev) => ({
    ...prev,
    [selectedEventId]: [],
  }));
};

const activeEventScans =
  (selectedEventId && recentScansByEvent[selectedEventId]) || [];

const eventTitle =
  eventDetails?.title || eventDetails?.Title || "Event";
const eventStatus =
  eventDetails?.status || eventDetails?.Status || "";

return (
  <div className="p-4 sm:p-6 lg:p-8 max-w-6xl mx-auto flex flex-col gap-6">
    {/* Top Header */}
    <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3">
      <div>
        <div className="flex items-center gap-2">
          <span className="text-xs font-bold uppercase tracking-wider text-[#00685f]">
            Organizer Portal
          </span>
          <span className="text-xs text-[#bcc9c6]">•</span>
          <span className="text-xs font-semibold text-[#565e74]">
            Entry Management
          </span>
        </div>
        <h1 className="text-2xl sm:text-3xl font-extrabold text-[#141b2b] tracking-tight mt-0.5">
          Scan & Check-In Tickets
        </h1>
        <p className="text-xs sm:text-sm text-[#565e74] mt-1">
          Validate attendee ticket QR passes instantly at event entry gates.
        </p>
      </div>

      {/* Global Toolbar */}
      <div className="flex items-center gap-2 self-start sm:self-auto">
        {/* Audio toggle button */}
        <button
          type="button"
          onClick={() => setIsAudioEnabled((prev) => !prev)}
          className={`p-2 rounded-xl border text-xs font-semibold transition-colors cursor-pointer flex items-center gap-1.5 ${isAudioEnabled
              ? "bg-white border-[#bcc9c6]/60 text-[#00685f] hover:bg-gray-50"
              : "bg-gray-100 border-[#bcc9c6]/40 text-[#565e74]"
            }`}
          title={isAudioEnabled ? "Sound Effects Enabled" : "Sound Effects Muted"}
        >
          {isAudioEnabled ? (
            <svg className="size-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15.536 8.464a5 5 0 010 7.072m2.828-9.9a9 9 0 010 12.728M5.586 15H4a1 1 0 01-1-1v-4a1 1 0 011-1h1.586l4.707-4.707C10.923 3.663 12 4.109 12 5v14c0 .891-1.077 1.337-1.707.707L5.586 15z" />
            </svg>
          ) : (
            <svg className="size-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5.586 15H4a1 1 0 01-1-1v-4a1 1 0 011-1h1.586l4.707-4.707C10.923 3.663 12 4.109 12 5v14c0 .891-1.077 1.337-1.707.707L5.586 15z" />
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M17 14l2-2m0 0l2-2m-2 2l-2-2m2 2l2 2" />
            </svg>
          )}
          <span className="hidden sm:inline">
            {isAudioEnabled ? "Sound On" : "Muted"}
          </span>
        </button>

        {/* Manual Entry Button */}
        <button
          type="button"
          onClick={() => setIsManualModalOpen(true)}
          disabled={!selectedEventId || loadingEvent || Boolean(eventError)}
          className="px-3.5 py-2 rounded-xl border border-[#bcc9c6]/60 bg-white hover:bg-gray-50 text-[#141b2b] text-xs font-bold transition-colors cursor-pointer disabled:opacity-50 disabled:cursor-not-allowed flex items-center gap-1.5 shadow-xs"
        >
          <svg className="size-4 text-[#00685f]" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z" />
          </svg>
          <span>Manual Entry</span>
        </button>
      </div>
    </div>

    {/* Main Container: Event Picker VS Loaded Event Scanner */}
    {!selectedEventId || isChangingEvent ? (
      /* Event Selection Selector Panel */
      <div className="bg-white rounded-2xl border border-[#bcc9c6]/50 p-6 shadow-xs flex flex-col gap-4">
        <div className="flex items-center justify-between pb-3 border-b border-[#bcc9c6]/30">
          <div>
            <h2 className="text-base font-bold text-[#141b2b]">
              Select an Event to Scan
            </h2>
            <p className="text-xs text-[#565e74]">
              Choose which event attendees are checking in for.
            </p>
          </div>
          {selectedEventId && (
            <button
              type="button"
              onClick={() => setIsChangingEvent(false)}
              className="text-xs font-semibold text-[#565e74] hover:text-[#141b2b] cursor-pointer"
            >
              Cancel
            </button>
          )}
        </div>

        {loadingEventsList ? (
          <div className="p-8 flex flex-col items-center justify-center gap-2 text-[#565e74]">
            <span className="size-6 border-2 border-[#00685f]/30 border-t-[#00685f] rounded-full animate-spin" />
            <span className="text-xs">Loading organizer events...</span>
          </div>
        ) : eventsListError ? (
          <div className="p-4 bg-[#ba1a1a]/10 border border-[#ba1a1a]/30 rounded-xl text-xs text-[#ba1a1a]">
            {eventsListError}
          </div>
        ) : eventsList.length === 0 ? (
          <div className="p-8 text-center text-xs text-[#565e74]">
            You don&apos;t have any active events.{" "}
            <Link
              to={ORGANIZER_ROUTES.CREATE_STEP_1}
              className="text-[#00685f] font-bold hover:underline"
            >
              Create an event
            </Link>{" "}
            first.
          </div>
        ) : (
          <div className="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 gap-3 max-h-72 overflow-y-auto p-1">
            {eventsList.map((ev) => {
              const id = ev.id || ev.ID;
              const title = ev.title || ev.Title || "Untitled Event";
              const status = ev.status || ev.Status || "";
              const isCurrent = id === selectedEventId;

              return (
                <button
                  key={id}
                  type="button"
                  onClick={() => handleSelectEvent(id)}
                  className={`p-3.5 rounded-xl border text-left transition-all cursor-pointer flex flex-col justify-between gap-2 ${isCurrent
                      ? "border-[#00685f] bg-[#00685f]/5 ring-1 ring-[#00685f]"
                      : "border-[#bcc9c6]/50 bg-[#f9f9ff] hover:bg-white hover:border-[#00685f]/60 shadow-2xs"
                    }`}
                >
                  <div className="flex items-start justify-between gap-2">
                    <span className="font-bold text-xs text-[#141b2b] line-clamp-2">
                      {title}
                    </span>
                    {status && (
                      <span
                        className={`text-[9px] font-bold px-1.5 py-0.5 rounded uppercase tracking-wider shrink-0 ${status === "PUBLISHED"
                            ? "bg-[#00685f]/10 text-[#00685f]"
                            : "bg-gray-200 text-gray-700"
                          }`}
                      >
                        {status}
                      </span>
                    )}
                  </div>
                  <span className="text-[11px] font-semibold text-[#00685f]">
                    {isCurrent ? "Active Scanner Event ✓" : "Select Event →"}
                  </span>
                </button>
              );
            })}
          </div>
        )}
      </div>
    ) : loadingEvent ? (
      /* Event Loading State */
      <div className="bg-white rounded-2xl border border-[#bcc9c6]/50 p-12 shadow-xs flex flex-col items-center justify-center gap-3 text-center">
        <span className="size-8 border-3 border-[#00685f]/30 border-t-[#00685f] rounded-full animate-spin" />
        <span className="text-sm font-semibold text-[#141b2b]">
          Loading event details...
        </span>
        <span className="text-xs text-[#565e74]">
          Preparing ticket verification for this event
        </span>
      </div>
    ) : eventError || !eventDetails ? (
      /* Event Loading Error State */
      <div className="bg-white rounded-2xl border border-red-200 p-8 shadow-xs flex flex-col items-center justify-center gap-4 text-center">
        <div className="size-12 rounded-full bg-red-100 text-red-600 flex items-center justify-center">
          <svg className="size-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
          </svg>
        </div>
        <div className="flex flex-col gap-1 max-w-md">
          <h3 className="text-base font-bold text-[#141b2b]">
            Unable to Load Event
          </h3>
          <p className="text-xs text-[#565e74]">
            {eventError || "Event details could not be retrieved."}
          </p>
        </div>
        <div className="flex items-center gap-3">
          <button
            type="button"
            onClick={() => fetchEventDetails(selectedEventId)}
            className="px-4 py-2 rounded-xl text-xs font-bold text-white bg-[#00685f] hover:bg-[#005550] transition-colors cursor-pointer"
          >
            Retry Loading
          </button>
          <button
            type="button"
            onClick={() => setIsChangingEvent(true)}
            className="px-4 py-2 rounded-xl text-xs font-semibold text-[#141b2b] bg-gray-100 hover:bg-gray-200 transition-colors cursor-pointer"
          >
            Choose Different Event
          </button>
        </div>
      </div>
    ) : (
      /* Event Loaded Successfully -> Render Scanner Workspace */
      <>
        {/* Selected Event Card */}
        <div className="bg-white rounded-2xl border border-[#bcc9c6]/50 p-4 sm:p-5 shadow-xs flex flex-col sm:flex-row sm:items-center justify-between gap-4">
          <div className="flex items-start sm:items-center gap-3.5 min-w-0">
            <div className="size-12 rounded-xl bg-[#00685f]/10 text-[#00685f] flex items-center justify-center shrink-0">
              <svg className="size-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 7V3m8 4V3m-9 8h10M5 21h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z" />
              </svg>
            </div>
            <div className="flex flex-col min-w-0">
              <div className="flex items-center gap-2">
                <span className="text-[11px] font-bold text-[#565e74] uppercase tracking-wider">
                  Target Event
                </span>
                {eventStatus && (
                  <span
                    className={`text-[10px] font-bold px-2 py-0.5 rounded-full uppercase tracking-wider ${eventStatus === "PUBLISHED"
                        ? "bg-[#00685f]/10 text-[#00685f]"
                        : eventStatus === "COMPLETED"
                          ? "bg-blue-100 text-blue-800"
                          : "bg-amber-100 text-amber-800"
                      }`}
                  >
                    {eventStatus}
                  </span>
                )}
              </div>
              <h2 className="text-base sm:text-lg font-bold text-[#141b2b] truncate mt-0.5">
                {eventTitle}
              </h2>
            </div>
          </div>

          <div className="flex items-center gap-2 shrink-0 self-end sm:self-auto">
            <Link
              to={getEventDetailsRoute(selectedEventId)}
              className="text-xs font-semibold text-[#565e74] hover:text-[#00685f] px-3 py-1.5 rounded-lg hover:bg-gray-50 transition-colors"
            >
              Event Details
            </Link>
            <button
              type="button"
              onClick={() => setIsChangingEvent(true)}
              className="px-3.5 py-1.5 rounded-xl border border-[#bcc9c6]/60 text-xs font-bold text-[#00685f] hover:bg-[#00685f]/10 transition-colors cursor-pointer"
            >
              Change Event
            </button>
          </div>
        </div>

        {/* Main Scanner Section */}
        <div className="grid grid-cols-1 lg:grid-cols-12 gap-6 items-start">
          {/* Left Column: Camera Viewport & Result Card */}
          <div className="lg:col-span-7 flex flex-col items-center">
            {cameraError ? (
              <CameraPermissionCard
                error={cameraError}
                onRetry={handleRetryCamera}
                onManualEntry={() => setIsManualModalOpen(true)}
              />
            ) : (
              <div className="w-full flex flex-col items-center">
                <CameraScanner
                  key={`scanner-${cameraRetryKey}-${selectedEventId}`}
                  isPaused={isPaused}
                  onScan={processCheckIn}
                  onError={handleCameraError}
                />

                {/* Scan Result Overlay card */}
                {scanResult && (
                  <ScanResultOverlay
                    result={scanResult}
                    onScanNext={handleScanNext}
                  />
                )}
              </div>
            )}
          </div>

          {/* Right Column: Event Info Summary & Session History */}
          <div className="lg:col-span-5 flex flex-col gap-4">
            <RecentScansList
              scans={activeEventScans}
              onClear={handleClearRecentScans}
            />

            {/* Gate Instructions Card */}
            <div className="bg-white rounded-2xl border border-[#bcc9c6]/50 p-4 shadow-xs text-xs text-[#565e74] flex flex-col gap-2">
              <div className="flex items-center gap-2 font-bold text-[#141b2b]">
                <svg className="size-4 text-[#00685f]" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
                </svg>
                <span>Gate Scanner Tips</span>
              </div>
              <ul className="list-disc list-inside space-y-1 text-[11px] leading-relaxed">
                <li>Ask attendees to increase screen brightness on their phones.</li>
                <li>Hold the phone flat ~15–25 cm in front of the camera lens.</li>
                <li>
                  Individual ticket QR passes are required. Booking overview passes cannot be used for individual entry.
                </li>
                <li>If the attendee&apos;s screen is cracked, use <strong>Manual Entry</strong> to enter the code.</li>
              </ul>
            </div>
          </div>
        </div>
      </>
    )}

    {/* Manual Code Entry Modal */}
    <ManualCodeModal
      isOpen={isManualModalOpen}
      onClose={() => setIsManualModalOpen(false)}
      onSubmit={handleManualCodeSubmit}
      isSubmitting={isSubmittingManual}
    />
  </div>
);
}
