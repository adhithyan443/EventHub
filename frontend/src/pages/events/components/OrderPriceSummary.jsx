export default function OrderPriceSummary({
  subtotal = "₹2,998",
  platformFee = "₹99",
  taxes = "₹180",
  total = "₹3,277",
  className = "",
}) {
  return (
    <div className={`space-y-3 text-sm ${className}`}>
      <div className="flex justify-between text-[#565e74]">
        <span>Subtotal</span>
        <span className="font-medium text-ink">{subtotal}</span>
      </div>
      <div className="flex justify-between text-[#565e74]">
        <span>Platform fee</span>
        <span className="font-medium text-ink">{platformFee}</span>
      </div>
      <div className="flex justify-between text-[#565e74]">
        <span>Taxes</span>
        <span className="font-medium text-ink">{taxes}</span>
      </div>
      <div className="flex items-end justify-between border-t border-[#bcc9c6] pt-4">
        <strong className="text-base text-ink font-semibold">Total</strong>
        <strong className="text-2xl font-bold text-[#00796d] font-display">{total}</strong>
      </div>
    </div>
  );
}
