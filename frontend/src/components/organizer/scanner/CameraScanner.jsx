import { useEffect, useRef, useState, useCallback, useId } from "react";
import { Html5Qrcode } from "html5-qrcode";

export default function CameraScanner({
  isPaused = false,
  onScan,
  onError,
  className = "",
}) {
  const reactId = useId();
  const containerId = `eventhub-qr-reader-${reactId.replace(/:/g, "")}`;

  // Scanner and lifecycle tracking refs
  const scannerRef = useRef(null);
  const mountGenRef = useRef(0);
  const isRunningRef = useRef(false);
  const activeCameraIdRef = useRef(null);
  const targetCameraIdRef = useRef(null);

  // Serialized execution queue for all camera and video constraint operations
  const opQueueRef = useRef(Promise.resolve());

  // Stable callback and prop refs
  const onScanRef = useRef(onScan);
  const onErrorRef = useRef(onError);
  const isPausedRef = useRef(isPaused);

  useEffect(() => {
    onScanRef.current = onScan;
  }, [onScan]);

  useEffect(() => {
    onErrorRef.current = onError;
  }, [onError]);

  useEffect(() => {
    isPausedRef.current = isPaused;
  }, [isPaused]);

  // UI state
  const [cameras, setCameras] = useState([]);
  const [selectedCameraId, setSelectedCameraId] = useState("");
  const [isReady, setIsReady] = useState(false);
  const [isTorchOn, setIsTorchOn] = useState(false);
  const [hasTorch, setHasTorch] = useState(false);

  // Enqueue an async operation to guarantee serial execution
  const enqueue = useCallback((op) => {
    const next = opQueueRef.current
      .catch(() => {})
      .then(async () => {
        return op();
      });
    opQueueRef.current = next;
    return next;
  }, []);

  // Safely synchronize paused/resumed state on active scanner
  const applyPauseState = useCallback((scanner) => {
    if (!scanner || !isRunningRef.current) return;

    try {
      const state =
        typeof scanner.getState === "function" ? scanner.getState() : null;
      // Html5QrcodeScannerState: 1 = NOT_STARTED, 2 = SCANNING, 3 = PAUSED
      if (isPausedRef.current) {
        if (state === 2 || state === null) {
          scanner.pause(true);
        }
      } else {
        if (state === 3) {
          scanner.resume();
        }
      }
    } catch {
      // Ignore transition state race
    }
  }, []);

  // Idempotent stop & clear targeting a specific Html5Qrcode instance
  const stopSpecificScanner = useCallback(
    async (scanner) => {
      if (!scanner) return true;

      let stopSucceeded = true;
      try {
        const isScanning = Boolean(scanner.isScanning);
        const state =
          typeof scanner.getState === "function" ? scanner.getState() : null;
        // 1 = NOT_STARTED, 2 = SCANNING, 3 = PAUSED
        if (isScanning || state === 2 || state === 3) {
          await scanner.stop();
        }
      } catch (err) {
        // Inspect state after stop attempt failure
        const stateAfterErr =
          typeof scanner.getState === "function" ? scanner.getState() : null;
        const isStillActive =
          Boolean(scanner.isScanning) || stateAfterErr === 2 || stateAfterErr === 3;

        if (isStillActive) {
          stopSucceeded = false;
          onErrorRef.current?.(err);
          return false;
        }
      }

      // Do not clear the DOM container while the camera may still be active
      if (stopSucceeded) {
        try {
          scanner.clear();
        } catch {
          // Ignore DOM already clear
        }

        if (scannerRef.current === scanner) {
          isRunningRef.current = false;
          activeCameraIdRef.current = null;
          setIsReady(false);
          setHasTorch(false);
          setIsTorchOn(false);
        }
      }

      return stopSucceeded;
    },
    []
  );

  // Start or transition camera instance with generation safety and explicit status return
  const syncCamera = useCallback(
    async (scanner, targetId, gen) => {
      if (!scanner || mountGenRef.current !== gen) return false;

      // Already running this exact camera -> verify pause state only
      if (isRunningRef.current && activeCameraIdRef.current === targetId) {
        applyPauseState(scanner);
        return true;
      }

      // If running with a different camera, stop first
      if (isRunningRef.current) {
        const stopped = await stopSpecificScanner(scanner);
        if (!stopped) {
          return false; // Stop failed, avoid retry loop
        }
      }

      if (mountGenRef.current !== gen) return false;

      const cameraConfig = targetId
        ? { deviceId: { exact: targetId } }
        : { facingMode: "environment" };

      const qrConfig = {
        fps: 10,
        qrbox: (viewfinderWidth, viewfinderHeight) => {
          const minEdge = Math.min(viewfinderWidth, viewfinderHeight);
          const edge = Math.max(180, Math.floor(minEdge * 0.72));
          return { width: edge, height: edge };
        },
        aspectRatio: 1.0,
      };

      try {
        await scanner.start(
          cameraConfig,
          qrConfig,
          (decodedText) => {
            if (mountGenRef.current === gen && !isPausedRef.current) {
              onScanRef.current?.(decodedText);
            }
          },
          () => {
            // Frame parse miss: normal search state
          }
        );

        // If unmounted or superseded while start was awaiting, shut down immediately
        if (mountGenRef.current !== gen) {
          await stopSpecificScanner(scanner);
          return false;
        }

        isRunningRef.current = true;
        activeCameraIdRef.current = targetId;
        setIsReady(true);

        // Sync pause state in case isPaused was toggled while starting
        applyPauseState(scanner);

        // Inspect torch capabilities on the newly active video track
        try {
          const capabilities = scanner.getRunningTrackCapabilities?.();
          const torchSupported = Boolean(capabilities && "torch" in capabilities);
          setHasTorch(torchSupported);
          if (!torchSupported) {
            setIsTorchOn(false);
          }
        } catch {
          setHasTorch(false);
          setIsTorchOn(false);
        }

        return true;
      } catch (startErr) {
        // Rollback partial startup using the unified stopSpecificScanner
        await stopSpecificScanner(scanner);

        if (mountGenRef.current === gen) {
          onErrorRef.current?.(startErr);
        }

        return false;
      }
    },
    [applyPauseState, stopSpecificScanner]
  );

  // Request a camera sync through the serialized queue with loop termination on failure
  const requestCameraSync = useCallback(
    (desiredCameraId) => {
      targetCameraIdRef.current = desiredCameraId;
      const gen = mountGenRef.current;
      const scanner = scannerRef.current;
      if (!scanner) return;

      enqueue(async () => {
        while (
          mountGenRef.current === gen &&
          (!isRunningRef.current ||
            activeCameraIdRef.current !== targetCameraIdRef.current)
        ) {
          const nextTarget = targetCameraIdRef.current;
          const success = await syncCamera(scanner, nextTarget, gen);

          // Stop reconciliation loop if transition failed, aborted, or could not start
          if (!success) {
            break;
          }
        }

        if (mountGenRef.current === gen && scannerRef.current === scanner) {
          applyPauseState(scanner);
        }
      });
    },
    [enqueue, syncCamera, applyPauseState]
  );

  // Synchronize pause/resume prop changes
  useEffect(() => {
    isPausedRef.current = isPaused;
    enqueue(async () => {
      if (scannerRef.current && isRunningRef.current) {
        applyPauseState(scannerRef.current);
      }
    });
  }, [isPaused, enqueue, applyPauseState]);

  // Mount scanner, enumerate devices, and start preferred camera
  useEffect(() => {
    const currentGen = ++mountGenRef.current;

    // Verify browser secure context
    if (typeof window !== "undefined" && !window.isSecureContext) {
      const secureErr = new Error(
        "Camera access requires a secure connection (HTTPS or localhost)."
      );
      secureErr.name = "InsecureContextError";
      onErrorRef.current?.(secureErr);
      return;
    }

    // Capture the exact Html5Qrcode instance created by this effect
    const scannerInstance = new Html5Qrcode(containerId);
    scannerRef.current = scannerInstance;

    // Enumerate available video input devices
    Html5Qrcode.getCameras()
      .then((devices) => {
        if (mountGenRef.current !== currentGen) return;

        if (devices && devices.length > 0) {
          setCameras(devices);
          const backCam = devices.find(
            (c) =>
              c.label.toLowerCase().includes("back") ||
              c.label.toLowerCase().includes("rear") ||
              c.label.toLowerCase().includes("environment")
          );
          const defaultId = backCam ? backCam.id : devices[0].id;
          setSelectedCameraId(defaultId);
          requestCameraSync(defaultId);
        } else {
          requestCameraSync(null);
        }
      })
      .catch((err) => {
        if (mountGenRef.current !== currentGen) return;

        // Report camera enumeration error
        onErrorRef.current?.(err);

        // Fallback to default environment-facing camera
        requestCameraSync(null);
      });

    return () => {
      // Invalidate current generation so pending async operations are discarded
      if (mountGenRef.current === currentGen) {
        mountGenRef.current += 1;
      }

      // Clean up the exact instance created in this effect
      enqueue(async () => {
        await stopSpecificScanner(scannerInstance);
      });
    };
  }, [containerId, requestCameraSync, enqueue, stopSpecificScanner]);

  // Handle switching camera selection in dropdown
  const handleCameraChange = (e) => {
    const newId = e.target.value;
    setSelectedCameraId(newId);
    setIsTorchOn(false);
    requestCameraSync(newId);
  };

  // Toggle flashlight / torch safely inside the serialized execution queue
  const toggleTorch = useCallback(() => {
    const gen = mountGenRef.current;
    const scanner = scannerRef.current;

    enqueue(async () => {
      if (
        !scanner ||
        mountGenRef.current !== gen ||
        !isRunningRef.current ||
        !hasTorch ||
        isPausedRef.current
      ) {
        return;
      }

      try {
        const nextState = !isTorchOn;
        await scanner.applyVideoConstraints({
          advanced: [{ torch: nextState }],
        });
        if (mountGenRef.current === gen && isRunningRef.current) {
          setIsTorchOn(nextState);
        }
      } catch {
        if (mountGenRef.current === gen) {
          setIsTorchOn(false);
        }
      }
    });
  }, [enqueue, hasTorch, isTorchOn]);

  return (
    <div
      className={`relative flex flex-col items-center w-full max-w-md mx-auto ${className}`}
    >
      {/* Viewport Frame */}
      <div className="relative w-full aspect-square bg-black rounded-2xl overflow-hidden shadow-md border-2 border-[#bcc9c6]/40">
        <div id={containerId} className="w-full h-full" />

        {/* Viewfinder Target Guidelines */}
        {!isPaused && isReady && (
          <div className="pointer-events-none absolute inset-0 flex items-center justify-center p-8">
            <div className="relative w-3/4 h-3/4 border-2 border-dashed border-[#00685f]/80 rounded-xl flex items-center justify-center">
              {/* Corner accents */}
              <div className="absolute -top-1 -left-1 size-5 border-t-3 border-l-3 border-[#008378] rounded-tl-sm" />
              <div className="absolute -top-1 -right-1 size-5 border-t-3 border-r-3 border-[#008378] rounded-tr-sm" />
              <div className="absolute -bottom-1 -left-1 size-5 border-b-3 border-l-3 border-[#008378] rounded-bl-sm" />
              <div className="absolute -bottom-1 -right-1 size-5 border-b-3 border-r-3 border-[#008378] rounded-br-sm" />

              {/* Scanning laser animation bar */}
              <div className="w-full h-0.5 bg-gradient-to-r from-transparent via-[#008378] to-transparent animate-pulse" />
            </div>
          </div>
        )}

        {/* Paused state blur overlay */}
        {isPaused && (
          <div className="absolute inset-0 bg-black/50 backdrop-blur-xs flex items-center justify-center">
            <div className="flex flex-col items-center gap-2 text-white">
              <span className="size-8 border-3 border-white/30 border-t-white rounded-full animate-spin" />
              <span className="text-xs font-semibold tracking-wide">
                Processing Ticket...
              </span>
            </div>
          </div>
        )}
      </div>

      {/* Camera Toolbar (Switch camera & Torch) */}
      <div className="w-full flex items-center justify-between gap-2 mt-3 px-1 text-xs text-[#565e74]">
        {cameras.length > 1 ? (
          <div className="flex items-center gap-1.5 min-w-0">
            <svg
              className="size-4 shrink-0 text-[#00685f]"
              fill="none"
              viewBox="0 0 24 24"
              stroke="currentColor"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"
              />
            </svg>
            <select
              value={selectedCameraId}
              onChange={handleCameraChange}
              disabled={isPaused}
              aria-label="Select Camera"
              className="bg-white border border-[#bcc9c6]/60 rounded-lg px-2 py-1 text-xs text-[#141b2b] truncate focus:outline-none focus:border-[#00685f] cursor-pointer"
            >
              {cameras.map((c, i) => (
                <option key={c.id} value={c.id}>
                  {c.label || `Camera ${i + 1}`}
                </option>
              ))}
            </select>
          </div>
        ) : (
          <span className="text-[11px] text-[#565e74]">
            Align QR inside the box
          </span>
        )}

        {hasTorch && (
          <button
            type="button"
            onClick={toggleTorch}
            disabled={isPaused}
            className={`flex items-center gap-1.5 px-3 py-1 rounded-lg border text-xs font-semibold transition-colors cursor-pointer ${
              isTorchOn
                ? "bg-amber-100 border-amber-300 text-amber-900"
                : "bg-white border-[#bcc9c6]/60 text-[#3d4947] hover:bg-gray-50"
            }`}
            title="Toggle Flashlight"
          >
            <svg
              className="size-3.5"
              fill="none"
              viewBox="0 0 24 24"
              stroke="currentColor"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M13 10V3L4 14h7v7l9-11h-7z"
              />
            </svg>
            <span>{isTorchOn ? "Torch On" : "Torch"}</span>
          </button>
        )}
      </div>
    </div>
  );
}
