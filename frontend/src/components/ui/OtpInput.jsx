import { useRef } from "react";

export default function OtpInput({ length = 6, value, onChange }) {
  const inputsRef = useRef([]);

  function handleChange(e, index) {
    const digit = e.target.value.replace(/[^0-9]/g, "").slice(-1);
    const chars = value.split("");
    chars[index] = digit || "";
    onChange(chars.join("").slice(0, length));

    if (digit && index < length - 1) {
      inputsRef.current[index + 1]?.focus();
    }
  }

  function handleKeyDown(e, index) {
    if (e.key === "Backspace" && !value[index] && index > 0) {
      inputsRef.current[index - 1]?.focus();
    }
  }

  function handlePaste(e) {
    e.preventDefault();
    const pasted = e.clipboardData.getData("text").replace(/[^0-9]/g, "").slice(0, length);
    onChange(pasted);
    inputsRef.current[Math.min(pasted.length, length - 1)]?.focus();
  }

  return (
    <div className="flex items-center justify-center gap-1.5 sm:gap-2.5 w-full" onPaste={handlePaste}>
      {Array.from({ length }).map((_, i) => (
        <input
          key={i}
          ref={(el) => (inputsRef.current[i] = el)}
          type="text"
          inputMode="numeric"
          maxLength={1}
          value={value[i] || ""}
          onChange={(e) => handleChange(e, i)}
          onKeyDown={(e) => handleKeyDown(e, i)}
          className="h-11 w-9 xs:h-12 xs:w-10 sm:h-14 sm:w-12 rounded-md border border-border bg-white text-center text-base sm:text-lg font-semibold text-ink focus:outline-none focus:ring-2 focus:ring-primary/40 focus:border-primary shrink-0"
        />
      ))}
    </div>
  );
}