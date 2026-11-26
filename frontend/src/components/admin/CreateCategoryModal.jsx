import { useState } from "react";

export default function CreateCategoryModal({ isOpen, onClose, onSubmit, loading = false }) {
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [error, setError] = useState("");

  if (!isOpen) return null;

  const handleSubmit = async (e) => {
    e.preventDefault();
    const trimmedName = name.trim();
    if (!trimmedName) {
      setError("Category name is required.");
      return;
    }
    if (trimmedName.length < 2) {
      setError("Category name must be at least 2 characters.");
      return;
    }
    if (trimmedName.length > 100) {
      setError("Category name cannot exceed 100 characters.");
      return;
    }
    if (description.trim().length > 500) {
      setError("Description cannot exceed 500 characters.");
      return;
    }

    setError("");
    try {
      await onSubmit({ name: trimmedName, description: description.trim() });
      setName("");
      setDescription("");
    } catch (err) {
      const message =
        err.response?.data?.message || err.message || "Failed to create category.";
      setError(message);
    }
  };

  const handleClose = () => {
    if (!loading) {
      setName("");
      setDescription("");
      setError("");
      onClose();
    }
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
      {/* Backdrop */}
      <div
        className="absolute inset-0 bg-black/40 backdrop-blur-[1px] transition-opacity"
        onClick={handleClose}
      />

      {/* Modal Card */}
      <div className="relative bg-white rounded-2xl shadow-2xl w-full max-w-[500px] overflow-hidden animate-in fade-in zoom-in-95 duration-150 border border-[#dce2f7]">
        <form onSubmit={handleSubmit}>
          {/* Header */}
          <div className="p-4 sm:p-6 border-b border-[#dce2f7] flex items-center justify-between">
            <div className="flex items-center gap-3">
              <div className="w-10 h-10 rounded-xl bg-[#e6f4f2] text-[#00685f] flex items-center justify-center shrink-0">
                <svg width="20" height="20" viewBox="0 0 16 17" fill="none">
                  <path
                    d="M1.5 7H6.5C7.33 7 8 6.33 8 5.5V1.5C8 0.67 7.33 0 6.5 0H1.5C0.67 0 0 0.67 0 1.5V5.5C0 6.33 0.67 7 1.5 7ZM9.5 7H14.5C15.33 7 16 6.33 16 5.5V1.5C16 0.67 15.33 0 14.5 0H9.5C8.67 0 8 0.67 8 1.5V5.5C8 6.33 8.67 7 9.5 7ZM1.5 17H6.5C7.33 17 8 16.33 8 15.5V11.5C8 10.67 7.33 10 6.5 10H1.5C0.67 10 0 10.67 0 11.5V15.5C0 16.33 0.67 17 1.5 17ZM9.5 17H14.5C15.33 17 16 16.33 16 15.5V11.5C16 10.67 15.33 10 14.5 10H9.5C8.67 10 8 10.67 8 11.5V15.5C8 16.33 8.67 17 9.5 17Z"
                    fill="currentColor"
                  />
                </svg>
              </div>
              <div>
                <h2 className="text-[#141b2b] text-lg font-bold leading-6">
                  Add New Category
                </h2>
                <p className="text-[#3d4947] text-xs">
                  Create a new platform category for events.
                </p>
              </div>
            </div>
            <button
              type="button"
              onClick={handleClose}
              disabled={loading}
              className="text-[#6d7a77] hover:text-[#141b2b] transition-colors p-1 rounded-lg disabled:opacity-50"
            >
              <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M6 18L18 6M6 6l12 12" />
              </svg>
            </button>
          </div>

          {/* Body */}
          <div className="p-4 sm:p-6 space-y-4">
            {error && (
              <div className="flex items-center justify-between rounded-lg border border-[#ffdad6] bg-[#fff5f4] px-4 py-3 text-xs text-[#93000a] font-medium">
                <span>{error}</span>
                <button
                  type="button"
                  onClick={() => setError("")}
                  className="font-bold underline ml-2 cursor-pointer"
                >
                  Dismiss
                </button>
              </div>
            )}

            {/* Name Input */}
            <div>
              <label className="block text-xs font-bold text-[#141b2b] uppercase tracking-wider mb-1.5">
                Category Name <span className="text-[#ba1a1a]">*</span>
              </label>
              <input
                type="text"
                value={name}
                onChange={(e) => setName(e.target.value)}
                placeholder="e.g. Comedy, Music, Tech Conferences"
                maxLength={100}
                disabled={loading}
                className="w-full bg-[#f9f9ff] border border-[#bcc9c6] rounded-xl px-4 py-2.5 text-sm text-[#141b2b] placeholder-[#6d7a77] focus:outline-none focus:border-[#00685f] focus:ring-1 focus:ring-[#00685f] transition-all disabled:opacity-60"
              />
              <div className="flex justify-between items-center mt-1">
                <span className="text-[11px] text-[#6d7a77]">Must be unique across the platform</span>
                <span className="text-[11px] text-[#6d7a77]">{name.length}/100</span>
              </div>
            </div>

            {/* Description Input */}
            <div>
              <label className="block text-xs font-bold text-[#141b2b] uppercase tracking-wider mb-1.5">
                Description <span className="text-[#6d7a77] font-normal lowercase">(optional)</span>
              </label>
              <textarea
                value={description}
                onChange={(e) => setDescription(e.target.value)}
                placeholder="Brief summary of events included in this category..."
                rows={3}
                maxLength={500}
                disabled={loading}
                className="w-full bg-[#f9f9ff] border border-[#bcc9c6] rounded-xl px-4 py-2.5 text-sm text-[#141b2b] placeholder-[#6d7a77] focus:outline-none focus:border-[#00685f] focus:ring-1 focus:ring-[#00685f] transition-all resize-none disabled:opacity-60"
              />
              <div className="flex justify-end mt-1">
                <span className="text-[11px] text-[#6d7a77]">{description.length}/500</span>
              </div>
            </div>
          </div>

          {/* Footer */}
          <div className="p-4 sm:p-6 bg-[#f9f9ff] border-t border-[#dce2f7] flex items-center justify-end gap-3">
            <button
              type="button"
              onClick={handleClose}
              disabled={loading}
              className="px-4 py-2.5 text-sm font-semibold text-[#141b2b] hover:bg-[#eef0fb] rounded-xl transition-colors cursor-pointer disabled:opacity-50"
            >
              Cancel
            </button>
            <button
              type="submit"
              disabled={loading || !name.trim()}
              className="px-5 py-2.5 text-sm font-semibold text-white bg-[#00685f] hover:bg-[#00574f] rounded-xl transition-colors shadow-sm disabled:opacity-50 cursor-pointer flex items-center gap-2"
            >
              {loading ? (
                <>
                  <svg className="animate-spin h-4 w-4 text-white" viewBox="0 0 24 24" fill="none">
                    <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
                    <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
                  </svg>
                  <span>Creating...</span>
                </>
              ) : (
                <span>Create Category</span>
              )}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}
