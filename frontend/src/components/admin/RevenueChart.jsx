import { useState, useMemo } from "react";
import { formatCurrency } from "../../utils/formatters.js";

const formatPeriodMonth = (periodStr) => {
  if (!periodStr) return "";
  const parts = periodStr.split("-");
  if (parts.length === 2) {
    const monthIndex = parseInt(parts[1], 10) - 1;
    const monthNames = [
      "Jan", "Feb", "Mar", "Apr", "May", "Jun",
      "Jul", "Aug", "Sep", "Oct", "Nov", "Dec",
    ];
    if (monthIndex >= 0 && monthIndex < 12) {
      return monthNames[monthIndex];
    }
  }
  return periodStr;
};

const formatYTick = (tick) => {
  if (tick <= 0) return "₹0";
  if (tick >= 10000000) return `₹${(tick / 10000000).toFixed(1).replace(/\.0$/, "")}Cr`;
  if (tick >= 100000) return `₹${(tick / 100000).toFixed(1).replace(/\.0$/, "")}L`;
  if (tick >= 1000) return `₹${(tick / 1000).toFixed(0)}k`;
  return `₹${Math.round(tick)}`;
};

/**
 * RevenueChart component displaying dynamic monthly or quarterly revenue trends.
 *
 * @param {Object} props
 * @param {Array} [props.trends=[]] - Array of { period: string, gross_volume: string|number, platform_fees: string|number }
 * @param {string} [props.mode="monthly"] - "monthly" or "quarterly"
 */
export default function RevenueChart({ trends = [], mode = "monthly" }) {
  const [hoverIndex, setHoverIndex] = useState(null);

  // Dimensions
  const width = 500;
  const height = 180;
  const paddingLeft = 48;
  const paddingRight = 20;
  const paddingTop = 15;
  const paddingBottom = 25;

  const chartWidth = width - paddingLeft - paddingRight;
  const chartHeight = height - paddingTop - paddingBottom;

  // Transform and aggregate trend points based on mode
  const chartData = useMemo(() => {
    if (!Array.isArray(trends) || trends.length === 0) {
      return [];
    }

    if (mode === "quarterly") {
      // Group by year and quarter
      const quarters = {};
      trends.forEach((t) => {
        const period = t.period || "";
        const parts = period.split("-");
        const year = parts[0] || "";
        const monthNum = parseInt(parts[1], 10) || 1;
        const qNum = Math.ceil(monthNum / 3);
        const qKey = year ? `${year} Q${qNum}` : `Q${qNum}`;

        const fee = parseFloat(t.platform_fees) || 0;
        const gross = parseFloat(t.gross_volume) || 0;

        if (!quarters[qKey]) {
          quarters[qKey] = {
            month: `Q${qNum}`,
            fullLabel: qKey,
            revenue: 0,
            grossVolume: 0,
          };
        }
        quarters[qKey].revenue += fee;
        quarters[qKey].grossVolume += gross;
      });

      return Object.values(quarters);
    }

    // Default: monthly
    return trends.map((t) => {
      const fee = parseFloat(t.platform_fees) || 0;
      const gross = parseFloat(t.gross_volume) || 0;
      return {
        month: formatPeriodMonth(t.period),
        fullLabel: t.period,
        revenue: fee,
        grossVolume: gross,
      };
    });
  }, [trends, mode]);

  // Compute maximum revenue with padding for nice scale
  const maxRevenue = useMemo(() => {
    if (chartData.length === 0) return 10000;
    const maxVal = Math.max(...chartData.map((d) => d.revenue), 0);
    if (maxVal <= 0) return 10000;
    // Scale up by 25% so the top line isn't clipped
    const target = maxVal * 1.25;
    const magnitude = Math.pow(10, Math.floor(Math.log10(target)));
    return Math.ceil(target / magnitude) * magnitude;
  }, [chartData]);

  // Generate 4 clean Y-axis ticks
  const yTicks = useMemo(() => {
    return [
      0,
      maxRevenue * 0.33,
      maxRevenue * 0.66,
      maxRevenue,
    ];
  }, [maxRevenue]);

  // Map data items to chart coordinates
  const points = useMemo(() => {
    if (chartData.length === 0) return [];
    if (chartData.length === 1) {
      const d = chartData[0];
      const x = paddingLeft + chartWidth / 2;
      const y = paddingTop + chartHeight - (d.revenue / maxRevenue) * chartHeight;
      return [{ ...d, x, y }];
    }
    return chartData.map((d, i) => {
      const x = paddingLeft + (i / (chartData.length - 1)) * chartWidth;
      const y = paddingTop + chartHeight - (d.revenue / maxRevenue) * chartHeight;
      return { ...d, x, y };
    });
  }, [chartData, maxRevenue, paddingLeft, chartWidth, paddingTop, chartHeight]);

  // Generate cubic bezier curve
  const generateSmoothPath = (pts) => {
    if (pts.length === 0) return "";
    if (pts.length === 1) {
      return `M ${paddingLeft} ${pts[0].y} L ${width - paddingRight} ${pts[0].y}`;
    }

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
  const areaPath =
    points.length >= 2
      ? `${linePath} L ${points[points.length - 1].x} ${
          paddingTop + chartHeight
        } L ${points[0].x} ${paddingTop + chartHeight} Z`
      : "";

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
        {yTicks.map((tick, idx) => {
          const y = paddingTop + chartHeight - (tick / maxRevenue) * chartHeight;
          return (
            <g key={idx}>
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
                {formatYTick(tick)}
              </text>
            </g>
          );
        })}

        {/* Gradient Area for multi-point series */}
        {areaPath && <path d={areaPath} fill="url(#revenueAreaGrad)" />}

        {/* Stroke Line */}
        {linePath && (
          <path
            d={linePath}
            fill="none"
            stroke="#00685f"
            strokeWidth="2.5"
            strokeLinecap="round"
            strokeLinejoin="round"
          />
        )}

        {/* Empty state overlay when no points */}
        {points.length === 0 && (
          <text
            x={width / 2}
            y={height / 2}
            textAnchor="middle"
            className="text-xs fill-[#6d7a77] font-medium"
          >
            No revenue trend data available
          </text>
        )}

        {/* X-axis labels and interactive points */}
        {points.map((pt, i) => (
          <g key={i}>
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
      {hoverIndex !== null && points[hoverIndex] && (
        <div
          className="absolute pointer-events-none bg-white border border-[#dce2f7] shadow-lg rounded-lg px-2.5 py-1.5 text-xs transform -translate-x-1/2 -translate-y-full transition-all duration-100 z-10"
          style={{
            left: `${(points[hoverIndex].x / width) * 100}%`,
            top: `${(points[hoverIndex].y / height) * 100 - 8}%`,
          }}
        >
          <div className="text-[#3d4947] text-[10px] font-medium">
            {points[hoverIndex].fullLabel || points[hoverIndex].month}
          </div>
          <div className="text-[#00685f] font-bold text-xs">
            Fee Revenue: {formatCurrency(points[hoverIndex].revenue)}
          </div>
          {points[hoverIndex].grossVolume > 0 && (
            <div className="text-[#6d7a77] text-[10px]">
              Gross: {formatCurrency(points[hoverIndex].grossVolume)}
            </div>
          )}
        </div>
      )}
    </div>
  );
}
