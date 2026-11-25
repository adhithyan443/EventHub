function classifyCameraError(error) {
  const name = error?.name || "";
  const msg = (error?.message || "").toLowerCase();

  // 1. Security & Insecure Context
  if (
    name === "SecurityError" ||
    name === "InsecureContextError" ||
    (typeof window !== "undefined" && !window.isSecureContext) ||
    msg.includes("secure connection") ||
    msg.includes("https")
  ) {
    return {
      type: "SECURITY",
      title: "HTTPS Connection Required",
      description:
        "Modern browsers only allow camera access over HTTPS or localhost. Please ensure you are connected securely, or use manual check-in below.",
      iconType: "LOCK",
      canRetry: false,
    };
  }

  // 2. Permission Denied
  if (
    name === "NotAllowedError" ||
    name === "PermissionDeniedError" ||
    msg.includes("permission") ||
    msg.includes("denied") ||
    msg.includes("not allowed")
  ) {
    return {
      type: "PERMISSION_DENIED",
      title: "Camera Permission Denied",
      description:
        "Camera access was blocked by your browser. Click the lock/settings icon next to your browser address bar and enable Camera permissions to scan tickets.",
      iconType: "BLOCKED",
      canRetry: true,
    };
  }

  // 3. No Camera Detected
  if (
    name === "NotFoundError" ||
    name === "DevicesNotFoundError" ||
    msg.includes("not found") ||
    msg.includes("devices not found") ||
    msg.includes("no camera")
  ) {
    return {
      type: "NOT_FOUND",
      title: "No Camera Detected",
      description:
        "We could not find an active camera on this device. You can connect an external webcam or switch to a smartphone with a camera.",
      iconType: "CAMERA_OFF",
      canRetry: true,
    };
  }

  // 4. Camera In Use / Hardware Busy
  if (
    name === "NotReadableError" ||
    name === "TrackStartError" ||
    msg.includes("in use") ||
    msg.includes("not readable") ||
    msg.includes("busy")
  ) {
    return {
      type: "NOT_READABLE",
      title: "Camera in Use by Another App",
      description:
        "The camera hardware could not be started. Another application or browser tab may currently be using it. Please close other camera apps and retry.",
      iconType: "CAMERA_OFF",
      canRetry: true,
    };
  }

  // 5. Overconstrained / Resolution or Facing Mode Mismatch
  if (
    name === "OverconstrainedError" ||
    name === "ConstraintNotSatisfiedError" ||
    msg.includes("constraint") ||
    msg.includes("overconstrained")
  ) {
    return {
      type: "OVERCONSTRAINED",
      title: "Camera Constraints Unsupported",
      description:
        "The requested camera mode is not supported on this device. Try retrying or selecting a different camera from the toolbar.",
      iconType: "CAMERA_OFF",
      canRetry: true,
    };
  }

  // 6. AbortError / Initialization Interrupted
  if (
    name === "AbortError" ||
    msg.includes("abort") ||
    msg.includes("interrupted")
  ) {
    return {
      type: "ABORT",
      title: "Camera Startup Interrupted",
      description:
        "Camera initialization was interrupted before it could complete. Please try starting the camera again.",
      iconType: "CAMERA_OFF",
      canRetry: true,
    };
  }

  // 7. Generic Fallback
  return {
    type: "GENERIC",
    title: "Camera Access Issue",
    description:
      "Could not initialize video stream. Please ensure camera permissions are granted and no other application is blocking access.",
    iconType: "GENERIC",
    canRetry: true,
  };
}

export default function CameraPermissionCard({
  error,
  onRetry,
  onManualEntry,
}) {
  const { title, description, iconType, canRetry } = classifyCameraError(error);

  const isLock = iconType === "LOCK";
  const isBlocked = iconType === "BLOCKED";

  return (
    <div
      role="alert"
      aria-live="assertive"
      aria-labelledby="camera-error-title"
      aria-describedby="camera-error-description"
      className="w-full max-w-md mx-auto my-4 bg-white rounded-2xl border border-[#bcc9c6]/60 p-6 shadow-sm flex flex-col items-center text-center animate-fade-in"
    >
      {/* Icon */}
      <div
        className={`size-14 rounded-full flex items-center justify-center mb-4 ${
          isLock
            ? "bg-amber-100 text-amber-800"
            : isBlocked
            ? "bg-[#ba1a1a]/10 text-[#ba1a1a]"
            : "bg-gray-100 text-[#565e74]"
        }`}
      >
        {isLock ? (
          <svg className="size-7" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z" />
          </svg>
        ) : isBlocked ? (
          <svg className="size-7" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M18.364 18.364A9 9 0 005.636 5.636m12.728 12.728A9 9 0 015.636 5.636m12.728 12.728L5.636 5.636" />
          </svg>
        ) : (
          <svg className="size-7" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 10l4.553-2.276A1 1 0 0121 8.618v6.764a1 1 0 01-1.447.894L15 14M5 18h8a2 2 0 002-2V8a2 2 0 00-2-2H5a2 2 0 00-2 2v8a2 2 0 002 2z" />
          </svg>
        )}
      </div>

      {/* Heading */}
      <h3 id="camera-error-title" className="text-base font-bold text-[#141b2b]">
        {title}
      </h3>

      {/* Explanation */}
      <p id="camera-error-description" className="text-xs text-[#565e74] mt-2 mb-4 leading-relaxed max-w-sm">
        {description}
      </p>

      {/* Action buttons */}
      <div className="flex flex-col sm:flex-row items-center gap-2.5 w-full">
        {canRetry && onRetry && (
          <button
            type="button"
            onClick={onRetry}
            aria-label="Retry camera initialization"
            className="w-full sm:flex-1 py-2.5 px-4 rounded-xl text-xs font-bold text-white bg-[#00685f] hover:bg-[#005550] transition-colors shadow-sm cursor-pointer flex items-center justify-center gap-1.5"
          >
            <svg className="size-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
            </svg>
            <span>Retry Camera</span>
          </button>
        )}

        {onManualEntry && (
          <button
            type="button"
            onClick={onManualEntry}
            aria-label="Open manual ticket code check-in"
            className="w-full sm:flex-1 py-2.5 px-4 rounded-xl text-xs font-semibold text-[#141b2b] bg-[#f9f9ff] hover:bg-gray-100 border border-[#bcc9c6]/60 transition-colors cursor-pointer flex items-center justify-center gap-1.5"
          >
            <svg className="size-4 text-[#00685f]" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z" />
            </svg>
            <span>Manual Check-in</span>
          </button>
        )}
      </div>
    </div>
  );
}
