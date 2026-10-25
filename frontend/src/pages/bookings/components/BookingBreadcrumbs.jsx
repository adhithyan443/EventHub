import { Link } from "react-router-dom";
import { ChevronRightIcon } from "./BookingIcons";

export default function BookingBreadcrumbs({ items = [] }) {
  if (!items || items.length === 0) return null;

  return (
    <nav
      aria-label="Breadcrumb"
      className="mb-6 flex flex-wrap items-center gap-2 text-xs sm:text-sm text-[#565e74]"
    >
      {items.map((item, index) => {
        const isLast = index === items.length - 1;
        return (
          <div key={item.label || index} className="flex items-center gap-2">
            {index > 0 && (
              <span className="text-[#8c94a8]" aria-hidden="true">
                <ChevronRightIcon className="w-2.5 h-2.5" />
              </span>
            )}
            {isLast || !item.to ? (
              <span className="font-semibold text-[#141b2b]">
                {item.label}
              </span>
            ) : (
              <Link
                to={item.to}
                className="transition-colors hover:text-[#00685f] hover:underline"
              >
                {item.label}
              </Link>
            )}
          </div>
        );
      })}
    </nav>
  );
}
