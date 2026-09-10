import { useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";

import EventCreationStepper from "../../../components/organizer/EventCreationStepper";
import EventCreationFooter from "../../../components/organizer/EventCreationFooter";
import GooglePlacesVenueSelector from "../../../components/organizer/GooglePlacesVenueSelector";
import useEventCreationStore from "../../../store/eventCreationStore";
import { getCategories } from "../../../api/organizerApi";

import {
  LOCATION_TYPES,
  ORGANIZER_ROUTES,
  isPhysicalEvent,
  isOnlineEvent,
  isHybridEvent,
} from "../../../constants/eventConstants";

export default function CreateEventStep1BasicInfoPage() {
  const navigate = useNavigate();

  const basicInformation = useEventCreationStore(
    (state) => state.basicInformation
  );
  const updateBasicInformation = useEventCreationStore(
    (state) => state.updateBasicInformation
  );

  const locationType = useEventCreationStore(
    (state) => state.locationType
  );
  const setLocationType = useEventCreationStore(
    (state) => state.setLocationType
  );

  const venue = useEventCreationStore((state) => state.venue);
  const onlineUrl = useEventCreationStore((state) => state.onlineUrl);
  const setOnlineUrl = useEventCreationStore(
    (state) => state.setOnlineUrl
  );

  const [validationError, setValidationError] = useState("");
  const [categories, setCategories] = useState([]);
  const [categoryLoading, setCategoryLoading] = useState(true);
  const [categoryError, setCategoryError] = useState("");

  useEffect(() => {
    const loadCategories = async () => {
      try {
        setCategoryLoading(true);
        setCategoryError("");

        const response = await getCategories();
        const fetchedCategories = response.data?.categories ?? [];

        setCategories(fetchedCategories);

        // Preserve the existing category name from the UI
        // and resolve its corresponding backend category UUID.
        if (!basicInformation.categoryId && basicInformation.category) {
          const matchingCategory = fetchedCategories.find(
            (category) =>
              category.Name?.toLowerCase() ===
              basicInformation.category.toLowerCase()
          );

          if (matchingCategory) {
            updateBasicInformation({
              categoryId: matchingCategory.ID,
              category: matchingCategory.Name,
            });
          }
        }
      } catch (error) {
        console.error("Failed to load categories:", error);
        setCategoryError("Unable to load event categories.");
      } finally {
        setCategoryLoading(false);
      }
    };

    loadCategories();
  }, [
    basicInformation.category,
    basicInformation.categoryId,
    updateBasicInformation,
  ]);

  const languages = [
    "English",
    "Spanish",
    "French",
    "German",
    "Hindi",
    "Other",
  ];

  const ageRestrictions = [
    "All Ages (Family Friendly)",
    "13+",
    "16+",
    "18+",
    "21+",
  ];

  const handleCategoryChange = (event) => {
    const categoryId = event.target.value;

    const selectedCategory = categories.find(
      (category) => category.ID === categoryId
    );

    updateBasicInformation({
      categoryId,
      category: selectedCategory?.Name ?? "",
    });
  };

  const handleContinue = () => {
    if (!basicInformation.title || !basicInformation.title.trim()) {
      setValidationError(
        "Please enter an event title before continuing."
      );
      return;
    }

    if (!basicInformation.categoryId) {
      setValidationError(
        "Please select an event category before continuing."
      );
      return;
    }

    if (
      !basicInformation.description ||
      !basicInformation.description.trim()
    ) {
      setValidationError(
        "Please enter an event description before continuing."
      );
      return;
    }

    if (isPhysicalEvent(locationType)) {
      if (!venue || (!venue.name && !venue.address)) {
        setValidationError(
          "Please select a physical venue for your physical event."
        );
        return;
      }
    } else if (isOnlineEvent(locationType)) {
      if (!onlineUrl || !onlineUrl.trim()) {
        setValidationError(
          "Please provide a live stream or meeting link for your online event."
        );
        return;
      }
    } else if (isHybridEvent(locationType)) {
      if (!venue || (!venue.name && !venue.address)) {
        setValidationError(
          "Please select a physical venue for your hybrid event."
        );
        return;
      }

      if (!onlineUrl || !onlineUrl.trim()) {
        setValidationError(
          "Please provide a virtual stream link for online attendees of your hybrid event."
        );
        return;
      }
    }

    setValidationError("");
    navigate(ORGANIZER_ROUTES.CREATE_STEP_2);
  };

  return (
    <div className="min-h-full flex flex-col justify-between">
      <div className="p-8 max-w-4xl mx-auto w-full flex flex-col gap-6">
        {/* Stepper Progress Bar */}
        <EventCreationStepper currentStep={1} />

        {/* Form Container Card */}
        <div className="bg-white border border-[#bcc9c6]/50 rounded-2xl shadow-sm overflow-hidden flex flex-col">
          {/* Header */}
          <div className="p-6 border-b border-[#bcc9c6]/30 bg-[#f9f9ff]">
            <h1 className="text-xl font-bold text-[#141b2b]">
              Basic Event Information
            </h1>

            <p className="text-xs text-[#565e74] mt-1">
              Provide essential information and set the physical or
              digital venue for your event.
            </p>
          </div>

          <div className="p-8 flex flex-col gap-8">
            {/* Validation Error */}
            {validationError && (
              <div className="rounded-lg border border-[#ba1a1a]/30 bg-[#ba1a1a]/5 px-4 py-3 text-sm text-[#ba1a1a]">
                {validationError}
              </div>
            )}

            {/* Category API Error */}
            {categoryError && (
              <div className="rounded-lg border border-[#ba1a1a]/30 bg-[#ba1a1a]/5 px-4 py-3 text-sm text-[#ba1a1a]">
                {categoryError}
              </div>
            )}

            {/* Event Title */}
            <div className="flex flex-col gap-2">
              <label className="text-[13px] font-semibold text-[#141b2b]">
                Event Title{" "}
                <span className="text-[#ba1a1a]">*</span>
              </label>

              <input
                type="text"
                value={basicInformation.title}
                onChange={(e) =>
                  updateBasicInformation({
                    title: e.target.value,
                  })
                }
                placeholder="e.g. Sunfield Summer Music Festival"
                className="w-full bg-[#f9f9ff] border border-[#bcc9c6] rounded-lg px-4 py-2.5 text-sm text-[#141b2b] focus:outline-none focus:border-[#00685f] focus:ring-1 focus:ring-[#00685f]/20 transition-all"
              />
            </div>

            {/* Category & Language Row */}
            <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
              {/* Category */}
              <div className="flex flex-col gap-2">
                <label className="text-[13px] font-semibold text-[#141b2b]">
                  Category{" "}
                  <span className="text-[#ba1a1a]">*</span>
                </label>

                <select
                  value={basicInformation.categoryId || ""}
                  onChange={handleCategoryChange}
                  disabled={categoryLoading}
                  className="w-full bg-[#f9f9ff] border border-[#bcc9c6] rounded-lg px-4 py-2.5 text-sm text-[#141b2b] focus:outline-none focus:border-[#00685f] disabled:opacity-60"
                >
                  {categoryLoading ? (
                    <option value="">
                      Loading categories...
                    </option>
                  ) : (
                    <>
                      <option value="">
                        Select a category
                      </option>

                      {categories.map((category) => (
                        <option
                          key={category.ID}
                          value={category.ID}
                        >
                          {category.Name}
                        </option>
                      ))}
                    </>
                  )}
                </select>
              </div>

              {/* Primary Language */}
              <div className="flex flex-col gap-2">
                <label className="text-[13px] font-semibold text-[#141b2b]">
                  Primary Language
                </label>

                <select
                  value={basicInformation.language}
                  onChange={(e) =>
                    updateBasicInformation({
                      language: e.target.value,
                    })
                  }
                  className="w-full bg-[#f9f9ff] border border-[#bcc9c6] rounded-lg px-4 py-2.5 text-sm text-[#141b2b] focus:outline-none focus:border-[#00685f]"
                >
                  <option value="">
                    Select primary language
                  </option>

                  {languages.map((language) => (
                    <option key={language} value={language}>
                      {language}
                    </option>
                  ))}
                </select>
              </div>
            </div>

            {/* Age Restriction */}
            <div className="flex flex-col gap-2">
              <label className="text-[13px] font-semibold text-[#141b2b]">
                Age Restriction
              </label>

              <select
                value={basicInformation.ageRestriction}
                onChange={(e) =>
                  updateBasicInformation({
                    ageRestriction: e.target.value,
                  })
                }
                className="w-full md:w-1/2 bg-[#f9f9ff] border border-[#bcc9c6] rounded-lg px-4 py-2.5 text-sm text-[#141b2b] focus:outline-none focus:border-[#00685f]"
              >
                <option value="">
                  Select age restriction
                </option>
                
                {ageRestrictions.map((restriction) => (
                  <option
                    key={restriction}
                    value={restriction}
                  >
                    {restriction}
                  </option>
                ))}
              </select>
            </div>

            {/* Description */}
            <div className="flex flex-col gap-2">
              <label className="text-[13px] font-semibold text-[#141b2b]">
                Event Description{" "}
                <span className="text-[#ba1a1a]">*</span>
              </label>

              <textarea
                rows={4}
                value={basicInformation.description}
                onChange={(e) =>
                  updateBasicInformation({
                    description: e.target.value,
                  })
                }
                placeholder="Give attendees an exciting summary of what to expect at your event..."
                className="w-full bg-[#f9f9ff] border border-[#bcc9c6] rounded-lg p-4 text-sm text-[#141b2b] focus:outline-none focus:border-[#00685f] focus:ring-1 focus:ring-[#00685f]/20 transition-all resize-y"
              />
            </div>

            {/* Banner Upload Box */}
            <div className="flex flex-col gap-2">
              <label className="text-[13px] font-semibold text-[#141b2b]">
                Event Banner Image
              </label>

              <div className="border-2 border-dashed border-[#bcc9c6] hover:border-[#00685f] bg-[#f9f9ff] rounded-xl p-8 flex flex-col items-center justify-center text-center cursor-pointer transition-colors">
                <svg
                  className="size-10 text-[#565e74] mb-2"
                  fill="none"
                  stroke="currentColor"
                  viewBox="0 0 24 24"
                >
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    strokeWidth="1.5"
                    d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z"
                  />
                </svg>

                <span className="text-xs font-semibold text-[#00685f]">
                  Click or drag image to upload banner
                </span>

                <span className="text-[11px] text-[#565e74] mt-1">
                  Recommended 16:9 ratio (PNG, JPG, max 10MB)
                </span>
              </div>
            </div>

            {/* Divider */}
            <hr className="border-t border-[#bcc9c6]/40 my-2" />

            {/* Section: Event Location */}
            <div className="flex flex-col gap-4">
              <div className="flex flex-col">
                <h3 className="font-bold text-[14px] text-[#3d4947] uppercase tracking-wider">
                  Event Location
                </h3>

                <span className="text-xs text-[#565e74]">
                  Choose whether attendees will join at a physical
                  venue, online, or a combination.
                </span>
              </div>

              {/* Location Type Pills */}
              <div className="flex flex-wrap items-center gap-3">
                {/* Physical */}
                <button
                  type="button"
                  onClick={() =>
                    setLocationType(LOCATION_TYPES.PHYSICAL)
                  }
                  className={`flex items-center gap-2 px-5 py-2.5 rounded-full text-xs font-semibold transition-all cursor-pointer ${locationType === LOCATION_TYPES.PHYSICAL
                    ? "bg-[#00685f]/15 border-2 border-[#00685f] text-[#00685f]"
                    : "bg-white border border-[#bcc9c6] text-[#141b2b] hover:bg-gray-50"
                    }`}
                >
                  <svg
                    className="size-4"
                    fill="none"
                    stroke="currentColor"
                    viewBox="0 0 24 24"
                  >
                    <path
                      strokeLinecap="round"
                      strokeLinejoin="round"
                      strokeWidth="2"
                      d="M17.657 16.657L13.414 20.9a1.998 1.998 0 01-2.827 0l-4.244-4.243a8 8 0 1111.314 0z"
                    />
                  </svg>

                  <span>Physical Event</span>
                </button>

                {/* Online */}
                <button
                  type="button"
                  onClick={() =>
                    setLocationType(LOCATION_TYPES.ONLINE)
                  }
                  className={`flex items-center gap-2 px-5 py-2.5 rounded-full text-xs font-semibold transition-all cursor-pointer ${locationType === LOCATION_TYPES.ONLINE
                    ? "bg-[#00685f]/15 border-2 border-[#00685f] text-[#00685f]"
                    : "bg-white border border-[#bcc9c6] text-[#141b2b] hover:bg-gray-50"
                    }`}
                >
                  <svg
                    className="size-4"
                    fill="none"
                    stroke="currentColor"
                    viewBox="0 0 24 24"
                  >
                    <path
                      strokeLinecap="round"
                      strokeLinejoin="round"
                      strokeWidth="2"
                      d="M15 10l4.553-2.276A1 1 0 0121 8.618v6.764a1 1 0 01-1.447.894L15 14M5 18h8a2 2 0 002-2V8a2 2 0 00-2-2H5a2 2 0 00-2 2v8a2 2 0 002 2z"
                    />
                  </svg>

                  <span>Online Event</span>
                </button>

                {/* Hybrid */}
                <button
                  type="button"
                  onClick={() =>
                    setLocationType(LOCATION_TYPES.HYBRID)
                  }
                  className={`flex items-center gap-2 px-5 py-2.5 rounded-full text-xs font-semibold transition-all cursor-pointer ${locationType === LOCATION_TYPES.HYBRID
                    ? "bg-[#00685f]/15 border-2 border-[#00685f] text-[#00685f]"
                    : "bg-white border border-[#bcc9c6] text-[#141b2b] hover:bg-gray-50"
                    }`}
                >
                  <svg
                    className="size-4"
                    fill="none"
                    stroke="currentColor"
                    viewBox="0 0 24 24"
                  >
                    <path
                      strokeLinecap="round"
                      strokeLinejoin="round"
                      strokeWidth="2"
                      d="M9.75 17L9 20l-1 1h8l-1-1-.75-3M3 13h18M5 17h14a2 2 0 002-2V5a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z"
                    />
                  </svg>

                  <span>Hybrid Event</span>
                </button>
              </div>

              {/* Physical Event */}
              {locationType === LOCATION_TYPES.PHYSICAL && (
                <div className="mt-2">
                  <GooglePlacesVenueSelector />
                </div>
              )}

              {/* Online Event */}
              {locationType === LOCATION_TYPES.ONLINE && (
                <div className="bg-[#f9f9ff] border border-[#bcc9c6] rounded-xl p-6 flex flex-col gap-4 mt-2">
                  <div className="flex flex-col gap-1">
                    <h4 className="text-[14px] font-bold text-[#141b2b]">
                      Online Event Configuration
                    </h4>

                    <p className="text-xs text-[#565e74]">
                      Provide the live stream or meeting URL where
                      online attendees will participate.
                    </p>
                  </div>

                  <div className="flex flex-col gap-2">
                    <label className="text-[13px] font-semibold text-[#141b2b]">
                      Live Stream or Meeting Link{" "}
                      <span className="text-[#ba1a1a]">*</span>
                    </label>

                    <input
                      type="url"
                      value={onlineUrl}
                      onChange={(e) =>
                        setOnlineUrl(e.target.value)
                      }
                      placeholder="https://zoom.us/j/... or https://youtube.com/live/..."
                      className="w-full bg-white border border-[#bcc9c6] rounded-lg px-4 py-2.5 text-sm text-[#141b2b] focus:outline-none focus:border-[#00685f]"
                    />

                    <span className="text-[11px] text-[#565e74]">
                      Access link will only be shared with verified
                      ticket holders.
                    </span>
                  </div>
                </div>
              )}

              {/* Hybrid Event */}
              {locationType === LOCATION_TYPES.HYBRID && (
                <div className="flex flex-col gap-6 mt-2">
                  {/* Physical Venue Selector */}
                  <GooglePlacesVenueSelector />

                  {/* Online Stream Link */}
                  <div className="bg-[#f9f9ff] border border-[#bcc9c6] rounded-xl p-6 flex flex-col gap-4">
                    <div className="flex flex-col gap-1">
                      <h4 className="text-[14px] font-bold text-[#141b2b]">
                        Virtual Stream Link (For Hybrid Attendees)
                      </h4>

                      <p className="text-xs text-[#565e74]">
                        Broadcast URL for attendees joining the hybrid
                        stream remotely.
                      </p>
                    </div>

                    <div className="flex flex-col gap-2">
                      <label className="text-[13px] font-semibold text-[#141b2b]">
                        Online Stream Link
                      </label>

                      <input
                        type="url"
                        value={onlineUrl}
                        onChange={(e) =>
                          setOnlineUrl(e.target.value)
                        }
                        placeholder="https://zoom.us/j/... or https://eventhub.live/..."
                        className="w-full bg-white border border-[#bcc9c6] rounded-lg px-4 py-2.5 text-sm text-[#141b2b] focus:outline-none focus:border-[#00685f]"
                      />
                    </div>
                  </div>
                </div>
              )}
            </div>
          </div>
        </div>
      </div>

      {/* Persistent Footer */}
      <EventCreationFooter
        isFirstStep={true}
        onContinue={handleContinue}
      />
    </div>
  );
}