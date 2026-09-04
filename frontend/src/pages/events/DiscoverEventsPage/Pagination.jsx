export default function Pagination({ currentPage, totalPages, onPageChange }) {
  const pages = [1, 2, 3, "...", totalPages];

  return (
    <div className="flex items-center justify-center gap-2 pt-4">
      <button
        onClick={() => onPageChange(Math.max(1, currentPage - 1))}
        className="h-10 w-10 flex items-center justify-center rounded-lg border border-border text-ink/70 hover:bg-background"
        aria-label="Previous page"
      >
        ‹
      </button>
      {pages.map((page, i) =>
        page === "..." ? (
          <span key={`ellipsis-${i}`} className="px-2 text-ink/50">…</span>
        ) : (
          <button
            key={page}
            onClick={() => onPageChange(page)}
            className={`h-10 w-10 flex items-center justify-center rounded-lg border text-sm ${
              currentPage === page
                ? "border-primary bg-primary text-white"
                : "border-border text-ink/70 hover:bg-background"
            }`}
          >
            {page}
          </button>
        )
      )}
      <button
        onClick={() => onPageChange(Math.min(totalPages, currentPage + 1))}
        className="h-10 w-10 flex items-center justify-center rounded-lg border border-border text-ink/70 hover:bg-background"
        aria-label="Next page"
      >
        ›
      </button>
    </div>
  );
}