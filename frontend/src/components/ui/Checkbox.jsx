export default function Checkbox({ name, checked, onChange, error, children }) {
  return (
    <div className="flex flex-col gap-1 w-full">
      <label className="flex items-start gap-2 cursor-pointer">
        <input
          type="checkbox"
          name={name}
          checked={checked}
          onChange={onChange}
          className="mt-0.5 h-4 w-4 rounded border-border text-primary focus:ring-primary/40"
        />
        <span className="text-sm text-ink/70">{children}</span>
      </label>
      {error && <p className="text-xs text-red-500">{error}</p>}
    </div>
  );
}