import { useEffect, useRef, useState } from "react";
import { setOptions, importLibrary } from "@googlemaps/js-api-loader";

let isOptionsConfigured = false;

function configureGoogleMaps(apiKey) {
  if (!isOptionsConfigured) {
    setOptions({
      key: apiKey,
      v: "weekly",
    });

    isOptionsConfigured = true;
  }
}

const API_KEY = import.meta.env.VITE_GOOGLE_MAPS_API_KEY;
const IS_API_KEY_MISSING = !API_KEY;

function VenueDemoPage() {
  const autocompleteRef = useRef(null);
  const mapRef = useRef(null);
  const markerRef = useRef(null);

  const [selectedVenue, setSelectedVenue] = useState(null);
  const [error, setError] = useState("");

  const displayError = IS_API_KEY_MISSING
    ? "Google Maps API key is not configured."
    : error;

  useEffect(() => {
    let isMounted = true;

    const autocompleteContainer = autocompleteRef.current;
    const mapContainer = mapRef.current;

    let autocompleteElement = null;
    let selectListener = null;

    if (IS_API_KEY_MISSING) {
      console.error(
        "VenueDemoPage: Google Maps API key is not configured in environment variables."
      );

      return;
    }

    const loadGoogleMaps = async () => {
      try {
        configureGoogleMaps(API_KEY);

        /*
         * Load Google Maps libraries.
         *
         * places:
         * - Provides PlaceAutocompleteElement
         *
         * maps:
         * - Provides Map
         *
         * marker:
         * - Provides AdvancedMarkerElement
         */
        let PlacesLibrary;
        let MapsLibrary;
        let MarkerLibrary;

        try {
          PlacesLibrary = await importLibrary("places");
        } catch (placesErr) {
          if (!isMounted) return;

          console.error(
            "VenueDemoPage: Places library failed to load:",
            placesErr
          );

          setError("Places library failed to load.");
          return;
        }

        if (!isMounted) return;

        try {
          MapsLibrary = await importLibrary("maps");
        } catch (mapsErr) {
          if (!isMounted) return;

          console.error(
            "VenueDemoPage: Maps library failed to load:",
            mapsErr
          );

          setError("Google Maps failed to load.");
          return;
        }

        if (!isMounted) return;

        try {
          MarkerLibrary = await importLibrary("marker");
        } catch (markerErr) {
          if (!isMounted) return;

          console.error(
            "VenueDemoPage: Marker library failed to load:",
            markerErr
          );

          setError("Google Maps marker failed to load.");
          return;
        }

        if (!isMounted) return;

        /*
         * Create the autocomplete element.
         */
        try {
          const { PlaceAutocompleteElement } = PlacesLibrary;

          if (!PlaceAutocompleteElement) {
            throw new Error(
              "PlaceAutocompleteElement not found in Places library"
            );
          }

          autocompleteElement = new PlaceAutocompleteElement();

          autocompleteElement.placeholder = "Search for a venue...";
        } catch (elementErr) {
          if (!isMounted) return;

          console.error(
            "VenueDemoPage: Autocomplete element failed to initialize:",
            elementErr
          );

          setError("Autocomplete element failed to initialize.");
          return;
        }

        /*
         * Create the Google Map.
         */
        try {
          const { Map } = MapsLibrary;

          if (!Map) {
            throw new Error("Google Maps Map class not found.");
          }

          if (!mapContainer) {
            throw new Error("Map container not found.");
          }

          new Map(mapContainer, {
            center: {
              lat: 10.0261,
              lng: 76.3125,
            },
            zoom: 12,
            mapId: "DEMO_MAP_ID",
          });
        } catch (mapErr) {
          if (!isMounted) return;

          console.error(
            "VenueDemoPage: Google Map failed to initialize:",
            mapErr
          );

          setError("Google Maps failed to initialize.");
          return;
        }

        /*
         * Handle Google Places selection.
         *
         * gmp-select fires when the organizer
         * selects a place from autocomplete.
         */
        selectListener = async (event) => {
          const prediction = event.placePrediction;

          if (!prediction) {
            console.warn(
              "VenueDemoPage: gmp-select fired without a place prediction."
            );

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

            if (
              typeof latitude !== "number" ||
              typeof longitude !== "number"
            ) {
              throw new Error(
                "Selected place does not contain valid coordinates."
              );
            }

            const venue = {
              id: place.id || "",
              name: place.displayName || "",
              address: place.formattedAddress || "",
              latitude,
              longitude,
            };

            setSelectedVenue(venue);
            setError("");

            /*
             * Move the map to the selected venue.
             */
            const { Map } = MapsLibrary;

            const map = new Map(mapContainer, {
              center: {
                lat: latitude,
                lng: longitude,
              },
              zoom: 16,
              mapId: "DEMO_MAP_ID",
            });

            /*
             * Create a marker at the selected venue.
             */
            const { AdvancedMarkerElement } = MarkerLibrary;

            if (markerRef.current) {
              markerRef.current.map = null;
            }

            markerRef.current = new AdvancedMarkerElement({
              map,
              position: {
                lat: latitude,
                lng: longitude,
              },
              title: venue.name,
            });
          } catch (detailsErr) {
            if (!isMounted) return;

            console.error(
              "VenueDemoPage: Failed to load selected venue details:",
              detailsErr
            );

            setError("Failed to load selected venue details.");
          }
        };

        /*
         * Attach gmp-select listener.
         */
        autocompleteElement.addEventListener(
          "gmp-select",
          selectListener
        );

        /*
         * Add autocomplete element to the page.
         */
        if (autocompleteContainer) {
          autocompleteContainer.replaceChildren(autocompleteElement);
        }
      } catch (err) {
        if (!isMounted) return;

        console.error(
          "VenueDemoPage: Google Maps failed to load:",
          err
        );

        setError("Google Maps failed to load.");
      }
    };

    loadGoogleMaps();

    return () => {
      isMounted = false;

      /*
       * Remove gmp-select listener.
       */
      if (autocompleteElement && selectListener) {
        autocompleteElement.removeEventListener(
          "gmp-select",
          selectListener
        );
      }

      /*
       * Remove autocomplete element.
       */
      if (autocompleteContainer) {
        autocompleteContainer.replaceChildren();
      }

      /*
       * Remove marker.
       */
      if (markerRef.current) {
        markerRef.current.map = null;
        markerRef.current = null;
      }

      /*
       * Clear map container.
       */
      if (mapContainer) {
        mapContainer.replaceChildren();
      }
    };
  }, []);

  return (
    <div className="min-h-screen bg-gray-50 p-8">
      <div className="mx-auto max-w-4xl">
        <h1 className="text-2xl font-semibold text-gray-900">
          Venue Demo
        </h1>

        <p className="mt-2 text-sm text-gray-500">
          Search for an existing Google Place and select it as the
          event venue.
        </p>

        {/* Search */}
        <div className="mt-8">
          <label className="block text-sm font-medium text-gray-700">
            Search for a venue
          </label>

          <div
            ref={autocompleteRef}
            className="mt-2"
          />
        </div>

        {/* Error */}
        {displayError && (
          <div className="mt-4 rounded-lg border border-red-200 bg-red-50 p-4 text-sm text-red-600">
            {displayError}
          </div>
        )}

        {/* Map */}
        <div className="mt-8">
          <h2 className="text-lg font-semibold text-gray-900">
            Venue Location
          </h2>

          <div
            ref={mapRef}
            className="mt-3 h-[400px] w-full overflow-hidden rounded-xl border border-gray-200 bg-gray-100"
          />
        </div>

        {/* Selected Venue */}
        <div className="mt-8 rounded-lg border border-gray-200 bg-white p-6">
          <h2 className="text-lg font-semibold text-gray-900">
            Selected Venue
          </h2>

          {!selectedVenue ? (
            <p className="mt-4 text-sm text-gray-500">
              Search and select a venue to see its details.
            </p>
          ) : (
            <div className="mt-4 space-y-4 text-sm">
              <div>
                <span className="font-medium text-gray-700">
                  Google Place ID:
                </span>

                <p className="mt-1 break-all text-gray-600">
                  {selectedVenue.id}
                </p>
              </div>

              <div>
                <span className="font-medium text-gray-700">
                  Name:
                </span>

                <p className="mt-1 text-gray-600">
                  {selectedVenue.name}
                </p>
              </div>

              <div>
                <span className="font-medium text-gray-700">
                  Address:
                </span>

                <p className="mt-1 text-gray-600">
                  {selectedVenue.address}
                </p>
              </div>

              <div>
                <span className="font-medium text-gray-700">
                  Latitude:
                </span>

                <p className="mt-1 text-gray-600">
                  {selectedVenue.latitude}
                </p>
              </div>

              <div>
                <span className="font-medium text-gray-700">
                  Longitude:
                </span>

                <p className="mt-1 text-gray-600">
                  {selectedVenue.longitude}
                </p>
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}

export default VenueDemoPage;