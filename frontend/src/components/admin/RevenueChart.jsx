import { useState } from "react";

const data = [
  { month: "Jan", revenue: 80000 },
  { month: "Feb", revenue: 120000 },
  { month: "Mar", revenue: 95000 },
  { month: "Apr", revenue: 145000 },
  { month: "May", revenue: 200000 },
  { month: "Jun", revenue: 185000 },
  { month: "Jul", revenue: 240000 },
  { month: "Aug", revenue: 280000 },
];

export default function RevenueChart() {
  const [hoverIndex, setHoverIndex] = useState(null);

  // Dimensions
  const width = 500;
  const height = 180;
  const paddingLeft = 45;
  const paddingRight = 20;
  const paddingTop = 15;
  const paddingBottom = 25;

  const chartWidth = width - paddingLeft - paddingRight;
  const chartHeight = height - paddingTop - paddingBottom;

  const maxRevenue = 300000;
  const yTicks = [0, 100000, 200000, 300000];

  // Map data to coordinates
  const points = data.map((d, i) => {
    const x = paddingLeft + (i / (data.length - 1)) * chartWidth;
    const y = paddingTop + chartHeight - (d.revenue / maxRevenue) * chartHeight;
    return { ...d, x, y };
  });

  // Generate cubic bezier curve
  const generateSmoothPath = (pts) => {
    if (pts.length === 0) return "";
    let path = `M ${pts[0].x} ${pts[0].y}`;

    for (let i = 0; i < pts.length - 1; i++) {
      const current = pts[i];
      const next = pts[i + 1];
      const controlX = (current.x + next.x) / 2;
      path += ` C ${controlX} ${current.y}, ${controlX} ${next.y}, ${next.x} ${next.y}`;
    }
    return path;
  };

  const linePath = generateSmoothPath(points);
  const areaPath = `${linePath} L ${points[points.length - 1].x} ${
    paddingTop + chartHeight
  } L ${points[0].x} ${paddingTop + chartHeight} Z`;

  return (
    <div className="relative w-full h-[190px] select-none">
      <svg
        viewBox={`0 0 ${width} ${height}`}
        className="w-full h-full overflow-visible"
      >
        <defs>
          <linearGradient id="revenueAreaGrad" x1="0" y1="0" x2="0" y2="1">
            <stop offset="0%" stopColor="#00685f" stopOpacity={0.35} />
            <stop offset="100%" stopColor="#00685f" stopOpacity={0.02} />
          </linearGradient>
        </defs>

        {/* Horizontal grid lines and Y-axis labels */}
        {yTicks.map((tick) => {
          const y = paddingTop + chartHeight - (tick / maxRevenue) * chartHeight;
          return (
            <g key={tick}>
              <line
                x1={paddingLeft}
                y1={y}
                x2={width - paddingRight}
                y2={y}
                stroke="#e5e7eb"
                strokeDasharray="3 3"
                strokeWidth="1"
              />
              <text
                x={paddingLeft - 8}
                y={y + 3.5}
                textAnchor="end"
                className="text-[10px] fill-[#6b7280] font-sans"
              >
                {tick === 0 ? "₹0" : `₹${tick / 1000}k`}
              </text>
            </g>
          );
        })}

        {/* Gradient Area */}
        <path d={areaPath} fill="url(#revenueAreaGrad)" />

        {/* Stroke Line */}
        <path
          d={linePath}
          fill="none"
          stroke="#00685f"
          strokeWidth="2.5"
          strokeLinecap="round"
          strokeLinejoin="round"
        />

        {/* X-axis labels and points */}
        {points.map((pt, i) => (
          <g key={pt.month}>
            <text
              x={pt.x}
              y={height - 5}
              textAnchor="middle"
              className="text-[11px] fill-[#6b7280] font-sans font-medium"
            >
              {pt.month}
            </text>

            {/* Invisible hover area and interactive circle */}
            <circle
              cx={pt.x}
              cy={pt.y}
              r={hoverIndex === i ? 5 : 3.5}
              fill={hoverIndex === i ? "#00685f" : "#ffffff"}
              stroke="#00685f"
              strokeWidth="2"
              className="transition-all duration-150 cursor-pointer"
            />
            <rect
              x={pt.x - 15}
              y={paddingTop}
              width={30}
              height={chartHeight}
              fill="transparent"
              className="cursor-pointer"
              onMouseEnter={() => setHoverIndex(i)}
              onMouseLeave={() => setHoverIndex(null)}
            />
          </g>
        ))}
      </svg>

      {/* Floating Tooltip */}
      {hoverIndex !== null && (
        <div
          className="absolute pointer-events-none bg-white border border-[#dce2f7] shadow-lg rounded-lg px-2.5 py-1.5 text-xs transform -translate-x-1/2 -translate-y-full transition-all duration-100 z-10"
          style={{
            left: `${(points[hoverIndex].x / width) * 100}%`,
            top: `${(points[hoverIndex].y / height) * 100 - 8}%`,
          }}
        >
          <div className="text-[#3d4947] text-[10px] font-medium">
            {points[hoverIndex].month} 2026
          </div>
          <div className="text-[#00685f] font-bold text-xs">
            ₹{(points[hoverIndex].revenue / 1000).toFixed(0)}k
          </div>
        </div>
      )}
    </div>
  );
}
