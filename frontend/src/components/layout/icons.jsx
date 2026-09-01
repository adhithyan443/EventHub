export function SearchIcon(props) {
    return (
        <svg width="16" height="16" viewBox="0 0 16 16" fill="none" {...props}>
            <circle cx="7" cy="7" r="5.5" stroke="currentColor" strokeWidth="1.5" />
            <path d="M11.5 11.5L15 15" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" />
        </svg>
    );
}

export function BellIcon(props) {
    return (
        <svg width="16" height="18" viewBox="0 0 16 18" fill="none" {...props}>
            <path d="M2 14V8C2 5.24 4.24 3 7 3H9C11.76 3 14 5.24 14 8V14L15.5 15.5H0.5L2 14Z" stroke="currentColor" strokeWidth="1.5" strokeLinejoin="round" />
            <path d="M6 15.5C6 16.6 6.9 17.5 8 17.5C9.1 17.5 10 16.6 10 15.5" stroke="currentColor" strokeWidth="1.5" />
        </svg>
    );
}

export function LocationIcon(props) {
    return (
        <svg width="16" height="16" viewBox="0 0 16 16" fill="none" {...props}>
            <path d="M8 14.5C8 14.5 13 10.5 13 6.5C13 3.7 10.8 1.5 8 1.5C5.2 1.5 3 3.7 3 6.5C3 10.5 8 14.5 8 14.5Z" stroke="currentColor" strokeWidth="1.5" strokeLinejoin="round" />
            <circle cx="8" cy="6.5" r="1.8" stroke="currentColor" strokeWidth="1.5" />
        </svg>
    );
}

export function CalendarIcon(props) {
    return (
        <svg width="16" height="16" viewBox="0 0 16 16" fill="none" {...props}>
            <rect x="1.5" y="3" width="13" height="11.5" rx="1.5" stroke="currentColor" strokeWidth="1.5" />
            <path d="M1.5 6.5H14.5" stroke="currentColor" strokeWidth="1.5" />
            <path d="M4.5 1.5V4" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" />
            <path d="M11.5 1.5V4" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" />
        </svg>
    );
}


export function GridIcon(props) {
  return (
    <svg width="16" height="16" viewBox="0 0 16 16" fill="none" {...props}>
      <rect x="1" y="1" width="6" height="6" rx="1" stroke="currentColor" strokeWidth="1.5" />
      <rect x="9" y="1" width="6" height="6" rx="1" stroke="currentColor" strokeWidth="1.5" />
      <rect x="1" y="9" width="6" height="6" rx="1" stroke="currentColor" strokeWidth="1.5" />
      <rect x="9" y="9" width="6" height="6" rx="1" stroke="currentColor" strokeWidth="1.5" />
    </svg>
  );
}

export function TagIcon(props) {
  return (
    <svg width="16" height="16" viewBox="0 0 16 16" fill="none" {...props}>
      <path d="M8.5 1.5H2.5C1.95 1.5 1.5 1.95 1.5 2.5V8.5C1.5 8.77 1.61 9.02 1.79 9.21L8.29 15.71C8.68 16.1 9.32 16.1 9.71 15.71L14.71 10.71C15.1 10.32 15.1 9.68 14.71 9.29L8.21 2.79C8.02 2.61 7.77 2.5 7.5 2.5" stroke="currentColor" strokeWidth="1.5" strokeLinejoin="round" />
      <circle cx="5" cy="5" r="1" fill="currentColor" />
    </svg>
  );
}