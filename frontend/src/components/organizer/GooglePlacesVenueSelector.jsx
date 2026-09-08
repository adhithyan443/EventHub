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
 * Extracts address components from Google Place object according to Google Places standard.
 */
function extractAddressComponents(place) {
  let city = "";
  let state = "";
  let country = "";
  let postal_code = "";

  if (Array.isArray(place.addressComponents)) {
    for (const component of place.addressComponents) {
      const types = component.types || [];
      if (types.includes("locality")) {
        city = component.longText || component.name || "";
      } else if (!city && (types.includes("sublocality") || types.includes("postal_town"))) {
        city = component.longText || component.name || "";
      }

      if (types.includes("administrative_area_level_1")) {
        state = component.longText || component.name || "";
      }

      if (types.includes("country")) {
        country = component.longText || component.name || "";
      }

      if (types.includes("postal_code")) {
        postal_code = component.longText || component.name || "";
      }
    }
  }

  return { city, state, country, postal_code };
}

export default function GooglePlacesVenueSelector() {
  const venue = useEventCreationStore((state) => state.venue);
  const setVenue = useEventCreationStore((state) => state.setVenue);

  const autocompleteRef = useRef(null);
  const mapRef = useRef(null);
  const mapInstanceRef = useRef(null);
  const markerRef = useRef(null);

  const [isLoadingGoogle, setIsLoadingGoogle] = useState(true);
  const [error, setError] = useState("");

  const apiKey = import.meta.env.VITE_GOOGLE_MAPS_API_KEY;
  const isApiKeyMissing = !apiKey;

  // Initialize or center Google Map
  const renderMap = useCallback(
    async (MapsLibrary, MarkerLibrary, lat, lng, title) => {
      if (!mapRef.current) return;

      const { Map } = MapsLibrary;
      const { AdvancedMarkerElement } = MarkerLibrary;

      if (!mapInstanceRef.current) {
        mapInstanceRef.current = new Map(mapRef.current, {
          center: { lat, lng },
          zoom: 15,
          mapId: "DEMO_MAP_ID",
          mapTypeControl: false,
          streetViewControl: false,
          fullscreenControl: true,
        });
      } else {
        mapInstanceRef.current.setCenter({ lat, lng });
        mapInstanceRef.current.setZoom(15);
      }

      // Update or create marker
      if (markerRef.current) {
        markerRef.current.map = null;
      }

      markerRef.current = new AdvancedMarkerElement({
        map: mapInstanceRef.current,
        position: { lat, lng },
        title: title || "Selected Venue",
      });
    },
    []
  );

  useEffect(() => {
    let isMounted = true;
    const autocompleteContainer = autocompleteRef.current;
    let autocompleteElement = null;
    let selectListener = null;

    if (isApiKeyMissing) {
      setIsLoadingGoogle(false);
      setError("Google Maps service is not configured. Please check environment configuration.");
      return;
    }

    const loadLibraries = async () => {
      try {
        setIsLoadingGoogle(true);
        configureGoogleMaps(apiKey);

        const PlacesLibrary = await importLibrary("places");
        const MapsLibrary = await importLibrary("maps");
        const MarkerLibrary = await importLibrary("marker");

        if (!isMounted) return;

        // Initialize Places Autocomplete Web Component
        autocompleteElement = new PlacesLibrary.PlaceAutocompleteElement();

        selectListener = async (event) => {
          const prediction = event.placePrediction;
          if (!prediction) {
            console.warn("GooglePlacesVenueSelector: Selection occurred without placePrediction.");
            return;
          }

          try {
            const place = prediction.toPlace();
            await place.fetchFields({
              fields: [
                "id",
                "displayName",
                "formattedAddress",
                "location",
                "addressComponents",
              ],
            });

            if (!isMounted) return;

            const latitude =
              typeof place.location?.lat === "function"
                ? place.location.lat()
                : place.location?.lat;

            const longitude =
              typeof place.location?.lng === "function"
                ? place.location.lng()
                : place.location?.lng;

            if (typeof latitude !== "number" || typeof longitude !== "number") {
              throw new Error("Selected place does not contain valid latitude and longitude coordinates.");
            }

            const { city, state, country, postal_code } = extractAddressComponents(place);

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

            setVenue(selectedVenue);
            setError("");

            // Move map to the newly selected venue
            await renderMap(MapsLibrary, MarkerLibrary, latitude, longitude, selectedVenue.name);
          } catch (err) {
            if (!isMounted) return;
            console.error("Failed to load details for selected place:", err);
            setError("Failed to retrieve details for the selected place. Please try again.");
          }
        };

        autocompleteElement.addEventListener("gmp-select", selectListener);

        if (autocompleteContainer) {
          autocompleteContainer.replaceChildren(autocompleteElement);
        }

        // If venue already selected, initialize map with existing location
        if (venue.latitude && venue.longitude) {
          await renderMap(MapsLibrary, MarkerLibrary, venue.latitude, venue.longitude, venue.name);
        }

        setIsLoadingGoogle(false);
      } catch (err) {
        if (!isMounted) return;
        console.error("Google Maps failed to load:", err);
        setError("Unable to load Google Places. Please verify network connectivity and configuration.");
        setIsLoadingGoogle(false);
      }
    };

    loadLibraries();

    return () => {
      isMounted = false;
      if (autocompleteElement && selectListener) {
        autocompleteElement.removeEventListener("gmp-select", selectListener);
      }
      if (autocompleteContainer) {
        autocompleteContainer.replaceChildren();
      }
    };
  }, [apiKey, isApiKeyMissing, renderMap, setVenue]); // Dependencies only for mount & API configuration

  const hasVenue = Boolean(venue.google_place_id || venue.name);

  return (
    <div className="bg-[#f9f9ff] border border-[#bcc9c6] rounded-xl p-6 flex flex-col gap-6" data-name="GooglePlacesVenueSelector">
      {/* Header Info */}
      <div className="flex flex-col gap-1">
        <h4 className="text-[14px] font-bold text-[#141b2b] uppercase tracking-wider">
          Search Venue with Google Places
        </h4>
        <p className="text-[13px] text-[#565e74]">
          Search for an established venue or location. The selected Google Place will be used as the authoritative event venue.
        </p>
      </div>

      {/* Google Places Autocomplete Container */}
      <div className="flex flex-col gap-2">
        <label className="text-[13px] font-semibold text-[#141b2b]">
          Venue Search <span className="text-[#ba1a1a]">*</span>
        </label>
        <div
          ref={autocompleteRef}
          className="min-h-[44px] w-full rounded-lg overflow-hidden bg-white border border-[#bcc9c6] shadow-xs"
        />
        {isLoadingGoogle && (
          <span className="text-xs text-[#00685f] flex items-center gap-1.5 mt-1">
            <span className="size-2 rounded-full bg-[#00685f] animate-pulse" />
            Connecting to Google Places...
          </span>
        )}
      </div>

      {/* Error Banner */}
      {error && (
        <div className="rounded-lg bg-red-50 border border-red-200 p-3.5 text-sm text-red-700 flex items-start gap-2.5">
          <svg className="size-5 shrink-0 text-red-500 mt-0.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
          </svg>
          <div className="flex flex-col">
            <span className="font-semibold text-xs text-red-800">Location Notice</span>
            <span className="text-xs">{error}</span>
          </div>
        </div>
      )}

      {/* Selected Venue Summary Card */}
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
              Place ID: {venue.google_place_id || "Selected"}
            </span>
          </div>

          <div className="grid grid-cols-1 md:grid-cols-2 gap-4 text-xs">
            <div>
              <span className="text-[#565e74] font-medium block">Venue Name:</span>
              <span className="text-[#141b2b] font-semibold text-[13px]">{venue.name || "—"}</span>
            </div>

            <div>
              <span className="text-[#565e74] font-medium block">Address:</span>
              <span className="text-[#141b2b]">{venue.address || "—"}</span>
            </div>

            <div>
              <span className="text-[#565e74] font-medium block">City / State:</span>
              <span className="text-[#141b2b]">
                {venue.city ? `${venue.city}, ` : ""}{venue.state || "—"}
              </span>
            </div>

            <div>
              <span className="text-[#565e74] font-medium block">Country / Postal Code:</span>
              <span className="text-[#141b2b]">
                {venue.country || "—"} {venue.postal_code ? `(${venue.postal_code})` : ""}
              </span>
            </div>

            {venue.latitude && venue.longitude && (
              <div className="col-span-full flex items-center gap-4 text-[11px] text-[#565e74] pt-1 border-t border-gray-50">
                <span>Latitude: <strong className="text-gray-700">{venue.latitude}</strong></span>
                <span>Longitude: <strong className="text-gray-700">{venue.longitude}</strong></span>
              </div>
            )}
          </div>
        </div>
      ) : (
        <div className="border border-dashed border-[#bcc9c6] rounded-xl p-4 text-center text-xs text-[#565e74] bg-white/50">
          Search and select a venue in the input above to view verified location details and preview the map.
        </div>
      )}

      {/* Google Map Visual Confirmation Container */}
      <div className="flex flex-col gap-2">
        <span className="text-[12px] font-semibold text-[#565e74] uppercase tracking-wider">
          Map Confirmation
        </span>
        <div
          ref={mapRef}
          className="h-[300px] w-full rounded-xl overflow-hidden border border-[#bcc9c6] bg-gray-100 shadow-inner relative"
        >
          {!hasVenue && !isLoadingGoogle && (
            <div className="absolute inset-0 flex items-center justify-center bg-gray-50/80 text-xs text-gray-500 pointer-events-none">
              Map preview will appear once a venue is selected.
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
