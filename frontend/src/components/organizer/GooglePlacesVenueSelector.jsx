import { useEffect, useRef, useState, useCallback } from "react";
import { setOptions, importLibrary } from "@googlemaps/js-api-loader";
import useEventCreationStore from "../../store/eventCreationStore";

let isGoogleConfigured = false;

function configureGoogleMaps(apiKey) {
  if (!isGoogleConfigured && apiKey) {
    setOptions({
      key: apiKey,
      v: "weekly",
    });

    isGoogleConfigured = true;
  }
}

/**
 * Extract city, state, country and postal code
 * from Google Places address components.
 */
function extractAddressComponents(place) {
  let city = "";
  let state = "";
  let country = "";
  let postal_code = "";

  if (!Array.isArray(place.addressComponents)) {
    return {
      city,
      state,
      country,
      postal_code,
    };
  }

  for (const component of place.addressComponents) {
    const types = component.types || [];

    const value =
      component.longText ||
      component.shortText ||
      component.name ||
      "";

    if (types.includes("locality")) {
      city = value;
    } else if (
      !city &&
      (types.includes("sublocality") ||
        types.includes("sublocality_level_1") ||
        types.includes("postal_town"))
    ) {
      city = value;
    }

    if (types.includes("administrative_area_level_1")) {
      state = value;
    }

    if (types.includes("country")) {
      country = value;
    }

    if (types.includes("postal_code")) {
      postal_code = value;
    }
  }

  return {
    city,
    state,
    country,
    postal_code,
  };
}

export default function GooglePlacesVenueSelector() {
  const venue = useEventCreationStore((state) => state.venue);
  const setVenue = useEventCreationStore((state) => state.setVenue);

  const autocompleteRef = useRef(null);
  const mapRef = useRef(null);

  const mapInstanceRef = useRef(null);
  const markerRef = useRef(null);

  const apiKey = import.meta.env.VITE_GOOGLE_MAPS_API_KEY;
  const isApiKeyMissing = !apiKey;

  /*
   * Initialize the state based on configuration.
   *
   * This avoids calling setState synchronously
   * inside useEffect when the API key is missing.
   */
  const [isLoadingGoogle, setIsLoadingGoogle] = useState(
    !isApiKeyMissing
  );

  const [error, setError] = useState(
    isApiKeyMissing
      ? "Google Maps service is not configured. Please check the environment configuration."
      : ""
  );

  /**
   * Create the map once and update its position
   * when a different venue is selected.
   */
  const renderMap = useCallback(
    (MapsLibrary, MarkerLibrary, latitude, longitude, title) => {
      if (!mapRef.current) {
        return;
      }

      const { Map } = MapsLibrary;
      const { AdvancedMarkerElement } = MarkerLibrary;

      const position = {
        lat: latitude,
        lng: longitude,
      };

      /*
       * Google Maps owns everything inside mapRef.
       *
       * React must not render conditional children
       * inside this element.
       */
      if (!mapInstanceRef.current) {
        mapInstanceRef.current = new Map(mapRef.current, {
          center: position,
          zoom: 15,
          mapId: "DEMO_MAP_ID",
          mapTypeControl: false,
          streetViewControl: false,
          fullscreenControl: true,
        });
      } else {
        mapInstanceRef.current.setCenter(position);
        mapInstanceRef.current.setZoom(15);
      }

      /*
       * Remove previous marker.
       */
      if (markerRef.current) {
        markerRef.current.map = null;
        markerRef.current = null;
      }

      /*
       * Create new marker.
       */
      markerRef.current = new AdvancedMarkerElement({
        map: mapInstanceRef.current,
        position,
        title: title || "Selected Venue",
      });
    },
    []
  );

  useEffect(() => {
    let isMounted = true;

    if (isApiKeyMissing) {
      return undefined;
    }

    const autocompleteContainer = autocompleteRef.current;

    let autocompleteElement = null;
    let selectListener = null;

    const initializeGoogleMaps = async () => {
      try {
        setIsLoadingGoogle(true);
        setError("");

        configureGoogleMaps(apiKey);

        /*
         * Load Google Maps libraries.
         */
        const PlacesLibrary = await importLibrary("places");
        const MapsLibrary = await importLibrary("maps");
        const MarkerLibrary = await importLibrary("marker");

        if (!isMounted) {
          return;
        }

        /*
         * Create Google Places Autocomplete Web Component.
         */
        autocompleteElement =
          new PlacesLibrary.PlaceAutocompleteElement();

        autocompleteElement.placeholder = "Search for a venue...";
        autocompleteElement.style.width = "100%";

        /*
         * Handle Google Place selection.
         */
        selectListener = async ({ placePrediction }) => {
          console.log("GOOGLE PLACE SELECTED");

          if (!placePrediction) {
            console.warn(
              "GooglePlacesVenueSelector: No placePrediction received."
            );

            return;
          }

          try {
            console.log("FETCHING PLACE DETAILS");

            /*
             * Convert prediction into a Place object.
             */
            const place = placePrediction.toPlace();

            /*
             * Fetch only the fields required by EventHub.
             */
            await place.fetchFields({
              fields: [
                "id",
                "displayName",
                "formattedAddress",
                "location",
                "addressComponents",
              ],
            });

            console.log("PLACE DETAILS RECEIVED", place);

            if (!isMounted) {
              return;
            }

            /*
             * Extract coordinates.
             */
            const latitude =
              typeof place.location?.lat === "function"
                ? place.location.lat()
                : place.location?.lat;

            const longitude =
              typeof place.location?.lng === "function"
                ? place.location.lng()
                : place.location?.lng;

            if (
              typeof latitude !== "number" ||
              typeof longitude !== "number" ||
              !Number.isFinite(latitude) ||
              !Number.isFinite(longitude)
            ) {
              throw new Error(
                "Selected place does not contain valid coordinates."
              );
            }

            /*
             * Extract address information.
             */
            const {
              city,
              state,
              country,
              postal_code,
            } = extractAddressComponents(place);

            /*
             * Keep the venue object aligned
             * with the backend VenueRequest structure.
             */
            const selectedVenue = {
              google_place_id: place.id || "",
              name: place.displayName || "",
              address: place.formattedAddress || "",
              city,
              state,
              country,
              postal_code,
              latitude,
              longitude,
            };

            console.log(
              "SELECTED VENUE:",
              selectedVenue
            );

            /*
             * Save selected venue to Zustand.
             */
            setVenue(selectedVenue);

            setError("");

            /*
             * Update Google Map.
             */
            renderMap(
              MapsLibrary,
              MarkerLibrary,
              latitude,
              longitude,
              selectedVenue.name
            );
          } catch (err) {
            if (!isMounted) {
              return;
            }

            console.error(
              "Failed to retrieve Google Place details:",
              err
            );

            setError(
              "Failed to retrieve details for the selected place. Please try again."
            );
          }
        };

        /*
         * Listen for Google Place selection.
         */
        autocompleteElement.addEventListener(
          "gmp-select",
          selectListener
        );

        /*
         * Mount the Google autocomplete element.
         *
         * This container is intentionally empty from React's
         * point of view. Google owns the inserted element.
         */
        if (autocompleteContainer) {
          autocompleteContainer.appendChild(
            autocompleteElement
          );
        }

        /*
         * Restore previously selected venue when the user
         * navigates back to Step 1.
         */
        if (
          typeof venue.latitude === "number" &&
          typeof venue.longitude === "number" &&
          Number.isFinite(venue.latitude) &&
          Number.isFinite(venue.longitude)
        ) {
          renderMap(
            MapsLibrary,
            MarkerLibrary,
            venue.latitude,
            venue.longitude,
            venue.name
          );
        }

        if (isMounted) {
          setIsLoadingGoogle(false);
        }
      } catch (err) {
        if (!isMounted) {
          return;
        }

        console.error(
          "Google Maps initialization failed:",
          err
        );

        setError(
          "Unable to load Google Places. Please verify your Google Maps configuration."
        );

        setIsLoadingGoogle(false);
      }
    };

    initializeGoogleMaps();

    return () => {
      isMounted = false;

      /*
       * Remove Google Places event listener.
       */
      if (autocompleteElement && selectListener) {
        autocompleteElement.removeEventListener(
          "gmp-select",
          selectListener
        );
      }

      /*
       * Remove the Google autocomplete element.
       *
       * Do not use replaceChildren() here because React and
       * Google can both be involved in DOM mutations.
       */
      if (autocompleteElement) {
        autocompleteElement.remove();
      }

      /*
       * Remove marker.
       */
      if (markerRef.current) {
        markerRef.current.map = null;
        markerRef.current = null;
      }

      /*
       * Release map reference.
       *
       * We intentionally don't manually remove Google's
       * internal map DOM children.
       */
      mapInstanceRef.current = null;
    };
  }, [
    apiKey,
    isApiKeyMissing,
    renderMap,
    setVenue,
  ]);

  const hasVenue = Boolean(
    venue.google_place_id || venue.name
  );

  const hasCoordinates =
    typeof venue.latitude === "number" &&
    typeof venue.longitude === "number" &&
    Number.isFinite(venue.latitude) &&
    Number.isFinite(venue.longitude);

  return (
    <div
      className="bg-[#f9f9ff] border border-[#bcc9c6] rounded-xl p-6 flex flex-col gap-6 overflow-visible"
      data-name="GooglePlacesVenueSelector"
    >
      {/* Header */}
      <div className="flex flex-col gap-1">
        <h4 className="text-[14px] font-bold text-[#141b2b] uppercase tracking-wider">
          Search Venue with Google Places
        </h4>

        <p className="text-[13px] text-[#565e74]">
          Search for an established venue or location. The
          selected Google Place will be used as the
          authoritative event venue.
        </p>
      </div>

      {/* Google Places Search */}
      <div className="flex flex-col gap-2 relative z-50">
        <label className="text-[13px] font-semibold text-[#141b2b]">
          Venue Search{" "}
          <span className="text-[#ba1a1a]">*</span>
        </label>

        <div
          ref={autocompleteRef}
          className="min-h-[44px] w-full rounded-lg bg-white border border-[#bcc9c6] shadow-xs"
        />

        {isLoadingGoogle && (
          <span className="text-xs text-[#00685f] flex items-center gap-1.5 mt-1">
            <span className="size-2 rounded-full bg-[#00685f] animate-pulse" />
            Connecting to Google Places...
          </span>
        )}
      </div>

      {/* Error */}
      {error && (
        <div className="rounded-lg bg-red-50 border border-red-200 p-3.5 text-sm text-red-700 flex items-start gap-2.5">
          <svg
            className="size-5 shrink-0 text-red-500 mt-0.5"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth="2"
              d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"
            />
          </svg>

          <div className="flex flex-col">
            <span className="font-semibold text-xs text-red-800">
              Location Notice
            </span>

            <span className="text-xs">
              {error}
            </span>
          </div>
        </div>
      )}

      {/* Selected Venue */}
      {hasVenue ? (
        <div className="bg-white border border-[#bcc9c6]/80 rounded-xl p-5 shadow-sm flex flex-col gap-4">
          <div className="flex items-center justify-between border-b border-gray-100 pb-3">
            <div className="flex items-center gap-2">
              <span className="size-2 rounded-full bg-[#00685f]" />

              <h5 className="font-bold text-[14px] text-[#141b2b]">
                Selected Venue Details
              </h5>
            </div>

            <span className="text-[11px] font-mono bg-gray-100 text-gray-600 px-2 py-0.5 rounded">
              Place ID:{" "}
              {venue.google_place_id || "Selected"}
            </span>
          </div>

          <div className="grid grid-cols-1 md:grid-cols-2 gap-4 text-xs">
            {/* Venue Name */}
            <div>
              <span className="text-[#565e74] font-medium block">
                Venue Name:
              </span>

              <span className="text-[#141b2b] font-semibold text-[13px]">
                {venue.name || "—"}
              </span>
            </div>

            {/* Address */}
            <div>
              <span className="text-[#565e74] font-medium block">
                Address:
              </span>

              <span className="text-[#141b2b]">
                {venue.address || "—"}
              </span>
            </div>

            {/* City / State */}
            <div>
              <span className="text-[#565e74] font-medium block">
                City / State:
              </span>

              <span className="text-[#141b2b]">
                {venue.city
                  ? `${venue.city}, `
                  : ""}
                {venue.state || "—"}
              </span>
            </div>

            {/* Country / Postal */}
            <div>
              <span className="text-[#565e74] font-medium block">
                Country / Postal Code:
              </span>

              <span className="text-[#141b2b]">
                {venue.country || "—"}{" "}
                {venue.postal_code
                  ? `(${venue.postal_code})`
                  : ""}
              </span>
            </div>

            {/* Coordinates */}
            {hasCoordinates && (
              <div className="col-span-full flex items-center gap-4 text-[11px] text-[#565e74] pt-1 border-t border-gray-50">
                <span>
                  Latitude:{" "}
                  <strong className="text-gray-700">
                    {venue.latitude}
                  </strong>
                </span>

                <span>
                  Longitude:{" "}
                  <strong className="text-gray-700">
                    {venue.longitude}
                  </strong>
                </span>
              </div>
            )}
          </div>
        </div>
      ) : (
        <div className="border border-dashed border-[#bcc9c6] rounded-xl p-4 text-center text-xs text-[#565e74] bg-white/50">
          Search and select a venue in the input above to view
          verified location details and preview the map.
        </div>
      )}

      {/* Map */}
      <div className="flex flex-col gap-2">
        <span className="text-[12px] font-semibold text-[#565e74] uppercase tracking-wider">
          Map Confirmation
        </span>

        {/*
         * IMPORTANT:
         *
         * Google Maps gets its own completely isolated
         * container. React must NOT render children inside
         * this div.
         */}
        <div className="relative h-[300px] w-full rounded-xl overflow-hidden border border-[#bcc9c6] bg-gray-100 shadow-inner">
          <div
            ref={mapRef}
            className="absolute inset-0"
          />

          {/*
           * React owns this overlay, not the Google Maps
           * container. This prevents React's removeChild
           * error when Google modifies the map DOM.
           */}
          {!hasVenue && !isLoadingGoogle && (
            <div className="absolute inset-0 flex items-center justify-center bg-gray-50/80 text-xs text-gray-500 pointer-events-none">
              Map preview will appear once a venue is
              selected.
            </div>
          )}
        </div>
      </div>
    </div>
  );
}