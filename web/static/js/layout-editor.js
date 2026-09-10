const WAITER_COLORS = [
  "#7ec8ff", "#ff8fab", "#7dffb3", "#ffd166", "#c084fc",
  "#fb923c", "#67e8f9", "#f472b6", "#86efac", "#facc15",
  "#818cf8", "#fb7185", "#2dd4bf", "#e879f9", "#94a3b8",
];
const SHADOW_WAITER_COLOR = "#e7d7a7";

const LAYOUT_PALETTE = [
  "#7ec8ff", "#ff8fab", "#7dffb3", "#ffd166", "#c084fc",
  "#fb923c", "#67e8f9", "#f472b6", "#86efac", "#facc15",
  "#818cf8", "#fb7185", "#2dd4bf", "#e879f9", "#ffffff",
  "#f2f2f2", "#d9d9d9", "#a6a6a6", "#737373", "#404040",
];

const LAYOUT_THEME = {
  floor: "#101010",
  ink: "#f5f5f5",
  muted: "#c4c4c4",
  glassStroke: "rgba(255,255,255,0.58)",
  selection: "rgba(255,255,255,0.92)",
  preview: "rgba(255,255,255,0.08)",
};

const DEFAULT_LAYOUT = { version: 2, width: 1400, height: 900, waiters: [], elements: [] };
const MOBILE_LAYOUT_QUERY = window.matchMedia("(max-width: 820px)");
const GRID_SIZE = 40;
const LAYOUT_HISTORY_LIMIT = 100;
const TABLE_SIZES = {
  small: {
    table_round: { width: GRID_SIZE, height: GRID_SIZE },
    table_rect: { width: GRID_SIZE * 2, height: GRID_SIZE },
  },
  medium: {
    table_round: { width: GRID_SIZE * 2, height: GRID_SIZE * 2 },
    table_rect: { width: GRID_SIZE * 3, height: GRID_SIZE * 2 },
  },
  large: {
    table_round: { width: GRID_SIZE * 3, height: GRID_SIZE * 3 },
    table_rect: { width: GRID_SIZE * 4, height: GRID_SIZE * 3 },
  },
};
const TABLE_ROUND_SIZE = TABLE_SIZES.medium.table_round.width;
const TABLE_RECT_WIDTH = TABLE_SIZES.medium.table_rect.width;
const TABLE_RECT_HEIGHT = TABLE_SIZES.medium.table_rect.height;
const TABLE_ROW_GAP = GRID_SIZE;
const WORLD_MIN = -6000;
const WORLD_MAX = 12000;
const SEATS_PER_TABLE = 8;
const MIN_VIEW_SCALE = 32 / TABLE_ROUND_SIZE;
const MAX_VIEW_SCALE = 120 / TABLE_ROUND_SIZE;
const TARGET_TABLE_PX = 58;

function layoutColorForWaiter(name, registry, options = {}) {
  const key = (name || "").trim().toLowerCase();
  if (!key) return "#9a9a9a";
  if (registry.has(key)) return registry.get(key);
  const color = options.shadow ? SHADOW_WAITER_COLOR : WAITER_COLORS[registry.size % WAITER_COLORS.length];
  registry.set(key, color);
  return color;
}

function layoutAccentForElement(item, registry) {
  if (!item || item.type === "marker") return "rgba(255,255,255,0.38)";
  if (item.waiter) return layoutColorForWaiter(item.waiter, registry);
  if (item.color) return item.color;
  return LAYOUT_THEME.glassStroke;
}

function hasWaiterColor(accent) {
  return Boolean(accent) && accent !== LAYOUT_THEME.glassStroke && accent !== "rgba(255,255,255,0.38)";
}

function tableLabelAttrs(extra = {}) {
  return {
    "text-anchor": "middle",
    "dominant-baseline": "middle",
    fill: LAYOUT_THEME.ink,
    stroke: "rgba(0,0,0,0.62)",
    "stroke-width": "3",
    "paint-order": "stroke",
    ...extra,
  };
}

function remapWaiterNameOnElements(elements, fromName, toName) {
  const previous = (fromName || "").trim();
  const next = (toName || "").trim();
  if (!previous || previous === next) return;
  elements.forEach((item) => {
    if ((item.waiter || "").trim() === previous) item.waiter = next;
  });
}

function svgEl(name, attrs) {
  const node = document.createElementNS("http://www.w3.org/2000/svg", name);
  Object.entries(attrs).forEach(([key, value]) => {
    if (value !== undefined && value !== null) node.setAttribute(key, String(value));
  });
  return node;
}

function appendGlassShape(group, item, registry, options = {}) {
  const ghost = Boolean(options.ghost);
  const accent = layoutAccentForElement({ ...item, waiter: options.waiter || item.waiter }, registry);
  const cx = item.x + item.width / 2;
  const cy = item.y + item.height / 2;
  const isRound = item.type === "table_round";
  const isMarker = item.type === "marker";

  if (isRound) {
    const radius = Math.min(item.width, item.height) / 2;
    group.append(svgEl("circle", {
      "data-layout-shape": "body",
      cx,
      cy,
      r: radius,
      fill: "url(#layout-glass-fill)",
      stroke: LAYOUT_THEME.glassStroke,
      "stroke-width": ghost ? "1.5" : "1.25",
      filter: ghost ? "" : "url(#layout-glass-shadow)",
      "fill-opacity": ghost ? "0.72" : "1",
    }));
    if (hasWaiterColor(accent)) {
      group.append(svgEl("circle", {
        "data-layout-shape": "wash",
        cx,
        cy,
        r: Math.max(6, radius - 1),
        fill: accent,
        "fill-opacity": ghost ? "0.42" : "0.78",
        "pointer-events": "none",
      }));
    }
    group.append(svgEl("ellipse", {
      "data-layout-shape": "highlight",
      cx,
      cy: cy - radius * 0.32,
      rx: radius * 0.62,
      ry: radius * 0.28,
      fill: "url(#layout-glass-highlight)",
      "pointer-events": "none",
    }));
    group.append(svgEl("circle", {
      "data-layout-shape": "ring",
      cx,
      cy,
      r: Math.max(6, radius - 4),
      fill: "none",
      stroke: accent,
      "stroke-width": hasWaiterColor(accent) ? "5" : "2.4",
      "pointer-events": "none",
    }));
    return;
  }

  group.append(svgEl("rect", {
    "data-layout-shape": "body",
    x: item.x,
    y: item.y,
    width: item.width,
    height: item.height,
    rx: isMarker ? "8" : "10",
    fill: isMarker ? "url(#layout-glass-fill-marker)" : "url(#layout-glass-fill)",
    "fill-opacity": ghost ? "0.72" : isMarker ? "0.95" : "1",
    stroke: isMarker ? "rgba(255,255,255,0.28)" : LAYOUT_THEME.glassStroke,
    "stroke-width": isMarker ? "1.4" : ghost ? "1.5" : "1.25",
    "stroke-dasharray": isMarker ? "7 5" : "",
    filter: ghost || isMarker ? "" : "url(#layout-glass-shadow)",
  }));
  if (!isMarker) {
    if (hasWaiterColor(accent)) {
      group.append(svgEl("rect", {
        "data-layout-shape": "wash",
        x: item.x + 1,
        y: item.y + 1,
        width: Math.max(0, item.width - 2),
        height: Math.max(0, item.height - 2),
        rx: "9",
        fill: accent,
        "fill-opacity": ghost ? "0.42" : "0.78",
        "pointer-events": "none",
      }));
    }
    group.append(svgEl("rect", {
      "data-layout-shape": "highlight",
      x: item.x + 8,
      y: item.y + 6,
      width: Math.max(0, item.width - 16),
      height: Math.max(10, item.height * 0.28),
      rx: "8",
      fill: "url(#layout-glass-highlight)",
      "pointer-events": "none",
    }));
    group.append(svgEl("rect", {
      "data-layout-shape": "ring",
      x: item.x + 4,
      y: item.y + 4,
      width: Math.max(0, item.width - 8),
      height: Math.max(0, item.height - 8),
      rx: "8",
      fill: "none",
      stroke: accent,
      "stroke-width": hasWaiterColor(accent) ? "5" : "2.2",
      "pointer-events": "none",
    }));
  }
}

function syncGlassGeometry(group, element) {
  const body = group.querySelector('[data-layout-shape="body"]');
  const wash = group.querySelector('[data-layout-shape="wash"]');
  const highlight = group.querySelector('[data-layout-shape="highlight"]');
  const ring = group.querySelector('[data-layout-shape="ring"]');
  const cx = element.x + element.width / 2;
  const cy = element.y + element.height / 2;
  if (element.type === "table_round") {
    const radius = Math.min(element.width, element.height) / 2;
    if (body) {
      body.setAttribute("cx", String(cx));
      body.setAttribute("cy", String(cy));
      body.setAttribute("r", String(radius));
    }
    if (wash) {
      wash.setAttribute("cx", String(cx));
      wash.setAttribute("cy", String(cy));
      wash.setAttribute("r", String(Math.max(6, radius - 1)));
    }
    if (ring) {
      ring.setAttribute("cx", String(cx));
      ring.setAttribute("cy", String(cy));
      ring.setAttribute("r", String(Math.max(6, radius - 4)));
    }
    if (highlight) {
      highlight.setAttribute("cx", String(cx));
      highlight.setAttribute("cy", String(cy - radius * 0.32));
      highlight.setAttribute("rx", String(radius * 0.62));
      highlight.setAttribute("ry", String(radius * 0.28));
    }
    return;
  }
  if (body) {
    body.setAttribute("x", String(element.x));
    body.setAttribute("y", String(element.y));
    body.setAttribute("width", String(element.width));
    body.setAttribute("height", String(element.height));
  }
  if (wash) {
    wash.setAttribute("x", String(element.x + 1));
    wash.setAttribute("y", String(element.y + 1));
    wash.setAttribute("width", String(Math.max(0, element.width - 2)));
    wash.setAttribute("height", String(Math.max(0, element.height - 2)));
  }
  if (ring) {
    ring.setAttribute("x", String(element.x + 4));
    ring.setAttribute("y", String(element.y + 4));
    ring.setAttribute("width", String(Math.max(0, element.width - 8)));
    ring.setAttribute("height", String(Math.max(0, element.height - 8)));
  }
  if (highlight) {
    highlight.setAttribute("x", String(element.x + 8));
    highlight.setAttribute("y", String(element.y + 6));
    highlight.setAttribute("width", String(Math.max(0, element.width - 16)));
    highlight.setAttribute("height", String(Math.max(10, element.height * 0.28)));
  }
}

function createElementId() {
  return `el-${Date.now().toString(36)}-${Math.random().toString(36).slice(2, 7)}`;
}

function parseLayoutState(raw) {
  try {
    const parsed = typeof raw === "string" ? JSON.parse(raw) : raw;
    if (!parsed || typeof parsed !== "object") return structuredClone(DEFAULT_LAYOUT);
    return {
      version: 2,
      width: Number(parsed.width) || 1400,
      height: Number(parsed.height) || 900,
      waiters: Array.isArray(parsed.waiters) ? parsed.waiters.filter(Boolean) : [],
      elements: Array.isArray(parsed.elements)
        ? parsed.elements.filter((item) => item?.type !== "zone" && item?.type !== "label")
        : [],
    };
  } catch (_error) {
    return structuredClone(DEFAULT_LAYOUT);
  }
}

function parseWaiterNames(raw) {
  try {
    const parsed = typeof raw === "string" ? JSON.parse(raw) : raw;
    if (!Array.isArray(parsed)) return [];
    return parsed.map((name) => String(name || "").trim()).filter(Boolean);
  } catch (_error) {
    return [];
  }
}

function tableCount(elements) {
  return elements.filter((item) => item.type === "table_round" || item.type === "table_rect").length;
}

function resizeWaiterRoster(names, count) {
  const size = Math.max(0, Math.min(30, Number(count) || 0));
  const source = Array.isArray(names) ? names : [];
  const next = [];
  for (let index = 0; index < size; index += 1) {
    next.push(source[index] || `Garçom ${index + 1}`);
  }
  return next;
}

function waiterNamesFromElements(elements) {
  const names = new Set();
  elements.forEach((item) => {
    const waiter = (item.waiter || "").trim();
    if (waiter) names.add(waiter);
  });
  return [...names];
}

function svgPoint(svg, clientX, clientY) {
  const point = svg.createSVGPoint();
  point.x = clientX;
  point.y = clientY;
  const matrix = svg.getScreenCTM();
  if (!matrix) return { x: 0, y: 0 };
  const transformed = point.matrixTransform(matrix.inverse());
  return { x: transformed.x, y: transformed.y };
}

function snapCoord(value) {
  return Math.round(value / GRID_SIZE) * GRID_SIZE;
}

function snapSize(value) {
  return Math.max(GRID_SIZE, Math.round(value / GRID_SIZE) * GRID_SIZE);
}

function snapPosition(x, y, width, height) {
  return {
    x: Math.max(WORLD_MIN, Math.min(WORLD_MAX - width, snapCoord(x))),
    y: Math.max(WORLD_MIN, Math.min(WORLD_MAX - height, snapCoord(y))),
  };
}

function clampWorld(value, size = 0) {
  return Math.max(WORLD_MIN, Math.min(WORLD_MAX - size, value));
}

function tableDimensions(type, size = "medium") {
  return TABLE_SIZES[size]?.[type] || TABLE_SIZES.medium[type];
}

function inferTableSize(element) {
  if (element.tableSize && TABLE_SIZES[element.tableSize]) return element.tableSize;
  if (element.type === "table_round") {
    if (element.width <= GRID_SIZE) return "small";
    if (element.width >= GRID_SIZE * 3) return "large";
    return "medium";
  }
  if (element.type === "table_rect") {
    if (element.width <= GRID_SIZE * 2) return "small";
    if (element.width >= GRID_SIZE * 4) return "large";
    return "medium";
  }
  return "medium";
}

function applyTableSize(element, sizeKey) {
  const dims = tableDimensions(element.type, sizeKey);
  if (!dims) return;
  const centerX = element.x + element.width / 2;
  const centerY = element.y + element.height / 2;
  element.width = dims.width;
  element.height = dims.height;
  element.tableSize = sizeKey;
  const snapped = snapPosition(centerX - dims.width / 2, centerY - dims.height / 2, dims.width, dims.height);
  element.x = snapped.x;
  element.y = snapped.y;
}

function syncColorSwatchSelection(container, color) {
  if (!container) return;
  const normalized = (color || "").trim().toLowerCase();
  container.querySelectorAll("[data-layout-color-swatch]").forEach((button) => {
    button.classList.toggle("is-selected", button.dataset.color.toLowerCase() === normalized);
  });
}

function setColorPickerOpen(form, open) {
  if (!form) return;
  const panel = form.querySelector("[data-layout-color-panel]");
  const toggle = form.querySelector("[data-layout-color-toggle]");
  if (panel) panel.hidden = !open;
  if (toggle) toggle.setAttribute("aria-expanded", open ? "true" : "false");
}

function syncColorPickerUI(form, element, waiterRegistry) {
  if (!form) return;
  const preview = form.querySelector("[data-layout-color-preview]");
  const label = form.querySelector("[data-layout-color-label]");
  const clearBtn = form.querySelector("[data-layout-color-clear]");
  const swatches = form.querySelector("[data-layout-color-swatches]");
  const customColor = (element?.color || "").trim();
  const effectiveColor = customColor || layoutColorForWaiter(element?.waiter || "", waiterRegistry) || "#f5f5f5";

  if (preview) {
    preview.style.backgroundColor = effectiveColor;
    preview.classList.toggle("is-custom", Boolean(customColor));
  }
  if (label) label.textContent = customColor ? "Cor personalizada" : "Usar cor do garçom";
  if (clearBtn) clearBtn.hidden = !customColor;
  syncColorSwatchSelection(swatches, customColor || effectiveColor);
  setColorPickerOpen(form, false);
}

function sortWaiterNames(names) {
  return [...names].sort((a, b) => a.localeCompare(b, "pt-BR", { numeric: true, sensitivity: "base" }));
}

const MIN_RESIZE = GRID_SIZE * 2;
const RESIZE_HANDLES = [
  { id: "nw", cursor: "nwse-resize" },
  { id: "n", cursor: "ns-resize" },
  { id: "ne", cursor: "nesw-resize" },
  { id: "e", cursor: "ew-resize" },
  { id: "se", cursor: "nwse-resize" },
  { id: "s", cursor: "ns-resize" },
  { id: "sw", cursor: "nesw-resize" },
  { id: "w", cursor: "ew-resize" },
];

function isResizableElement(element) {
  return element?.type === "marker";
}

function computeResizeBounds(state, dx, dy) {
  const { handle, originX, originY, originWidth, originHeight } = state;
  const min = MIN_RESIZE;

  let x = originX;
  let y = originY;
  let width = originWidth;
  let height = originHeight;

  if (handle.includes("e")) width = originWidth + dx;
  if (handle.includes("w")) {
    x = originX + dx;
    width = originWidth - dx;
  }
  if (handle.includes("s")) height = originHeight + dy;
  if (handle.includes("n")) {
    y = originY + dy;
    height = originHeight - dy;
  }

  if (width < min) {
    if (handle.includes("w")) x -= min - width;
    width = min;
  }
  if (height < min) {
    if (handle.includes("n")) y -= min - height;
    height = min;
  }

  width = snapSize(width);
  height = snapSize(height);

  if (handle.includes("w")) x = snapCoord(originX + originWidth - width);
  else x = snapCoord(x);

  if (handle.includes("n")) y = snapCoord(originY + originHeight - height);
  else y = snapCoord(y);

  width = Math.max(min, snapSize(handle.includes("w") ? originX + originWidth - x : width));
  height = Math.max(min, snapSize(handle.includes("n") ? originY + originHeight - y : height));

  if (handle.includes("w")) x = snapCoord(originX + originWidth - width);
  if (handle.includes("n")) y = snapCoord(originY + originHeight - height);

  return {
    x: clampWorld(x, width),
    y: clampWorld(y, height),
    width,
    height,
  };
}

function ensureColorSwatches(container, onSelect) {
  if (!container) return;
  if (container.childElementCount === 0) {
    LAYOUT_PALETTE.forEach((color) => {
      const button = document.createElement("button");
      button.type = "button";
      button.className = "layout-color-swatch";
      button.dataset.layoutColorSwatch = "1";
      button.dataset.color = color;
      button.style.backgroundColor = color;
      button.title = color;
      button.setAttribute("aria-label", `Cor ${color}`);
      container.append(button);
    });
  }
  container.querySelectorAll("[data-layout-color-swatch]").forEach((button) => {
    button.onclick = () => onSelect(button.dataset.color, container);
  });
}

function layoutExportBounds(state, padding = 60) {
  if (!state.elements.length) {
    return { x: 0, y: 0, width: state.width, height: state.height };
  }
  let minX = Infinity;
  let minY = Infinity;
  let maxX = -Infinity;
  let maxY = -Infinity;
  state.elements.forEach((item) => {
    minX = Math.min(minX, item.x);
    minY = Math.min(minY, item.y);
    maxX = Math.max(maxX, item.x + item.width);
    maxY = Math.max(maxY, item.y + item.height);
  });
  const x = snapCoord(minX - padding);
  const y = snapCoord(minY - padding);
  const width = Math.max(GRID_SIZE, snapCoord(maxX - minX + padding * 2));
  const height = Math.max(GRID_SIZE, snapCoord(maxY - minY + padding * 2));
  return { x, y, width, height };
}

async function svgToCanvas(svg, bounds) {
  const clone = svg.cloneNode(true);
  clone.querySelector('[data-layout-layer="selection"]')?.replaceChildren();
  clone.querySelector('[data-layout-layer="preview"]')?.replaceChildren();
  clone.querySelectorAll("[data-layout-export-hide]").forEach((node) => node.remove());

  [clone.querySelector("[data-layout-bg-fill]"), clone.querySelector("[data-layout-bg-grid]")].forEach((node) => {
    if (!node) return;
    node.setAttribute("x", String(bounds.x));
    node.setAttribute("y", String(bounds.y));
    node.setAttribute("width", String(bounds.width));
    node.setAttribute("height", String(bounds.height));
  });

  const maxSide = 2400;
  const scale = Math.min(maxSide / bounds.width, maxSide / bounds.height, 2.5);
  const pixelWidth = Math.max(1, Math.round(bounds.width * scale));
  const pixelHeight = Math.max(1, Math.round(bounds.height * scale));

  clone.setAttribute("xmlns", "http://www.w3.org/2000/svg");
  clone.setAttribute("viewBox", `${bounds.x} ${bounds.y} ${bounds.width} ${bounds.height}`);
  clone.setAttribute("width", String(pixelWidth));
  clone.setAttribute("height", String(pixelHeight));
  clone.setAttribute("preserveAspectRatio", "xMidYMid meet");

  const serialized = new XMLSerializer().serializeToString(clone);
  const dataUrl = `data:image/svg+xml;charset=utf-8,${encodeURIComponent(serialized)}`;
  const image = await new Promise((resolve, reject) => {
    const img = new Image();
    img.onload = () => resolve(img);
    img.onerror = () => reject(new Error("Falha ao renderizar SVG"));
    img.src = dataUrl;
  });
  const canvas = document.createElement("canvas");
  canvas.width = pixelWidth;
  canvas.height = pixelHeight;
  const ctx = canvas.getContext("2d");
  if (!ctx) throw new Error("Canvas indisponível");
  ctx.fillStyle = LAYOUT_THEME.floor;
  ctx.fillRect(0, 0, pixelWidth, pixelHeight);
  ctx.drawImage(image, 0, 0, pixelWidth, pixelHeight);
  return canvas;
}

function exportWaiterLegendEntries(configuredWaiters, elements, registry) {
  const names = sortWaiterNames([...new Set([...configuredWaiters, ...waiterNamesFromElements(elements)])].filter(Boolean));
  return names.map((name) => ({
    name,
    color: layoutColorForWaiter(name, registry),
  }));
}

function exportLegendMetrics(layoutWidth, entryCount) {
  const scale = Math.max(1, layoutWidth / 900);
  const padding = Math.round(20 * scale);
  const swatchSize = Math.round(14 * scale);
  const rowHeight = Math.round(28 * scale);
  const titleHeight = Math.round(30 * scale);
  const fontSize = Math.round(13 * scale);
  const titleFontSize = Math.round(15 * scale);
  const cols = entryCount <= 1 ? 1 : layoutWidth >= 1400 ? 4 : layoutWidth >= 900 ? 3 : 2;
  const rows = Math.ceil(entryCount / cols);
  const height = padding + titleHeight + rows * rowHeight + padding;
  return {
    padding,
    swatchSize,
    rowHeight,
    titleHeight,
    fontSize,
    titleFontSize,
    cols,
    rows,
    height,
  };
}

function fillTextEllipsis(ctx, text, x, y, maxWidth) {
  if (maxWidth <= 0) return;
  if (ctx.measureText(text).width <= maxWidth) {
    ctx.fillText(text, x, y);
    return;
  }
  let truncated = text;
  while (truncated.length > 1 && ctx.measureText(`${truncated}…`).width > maxWidth) {
    truncated = truncated.slice(0, -1);
  }
  ctx.fillText(`${truncated}…`, x, y);
}

function composeExportCanvas(layoutCanvas, legendEntries) {
  if (!legendEntries.length) return layoutCanvas;

  const metrics = exportLegendMetrics(layoutCanvas.width, legendEntries.length);
  const canvas = document.createElement("canvas");
  canvas.width = layoutCanvas.width;
  canvas.height = layoutCanvas.height + metrics.height;
  const ctx = canvas.getContext("2d");
  if (!ctx) return layoutCanvas;

  ctx.fillStyle = LAYOUT_THEME.floor;
  ctx.fillRect(0, 0, canvas.width, canvas.height);
  ctx.drawImage(layoutCanvas, 0, 0);

  const legendTop = layoutCanvas.height;
  ctx.fillStyle = "#0a0a0a";
  ctx.fillRect(0, legendTop, canvas.width, metrics.height);
  ctx.strokeStyle = "rgba(255,255,255,0.16)";
  ctx.lineWidth = Math.max(1, Math.round(canvas.width / 900));
  ctx.beginPath();
  ctx.moveTo(0, legendTop + 0.5);
  ctx.lineTo(canvas.width, legendTop + 0.5);
  ctx.stroke();

  const contentTop = legendTop + metrics.padding;
  ctx.fillStyle = LAYOUT_THEME.ink;
  ctx.font = `700 ${metrics.titleFontSize}px Inter, ui-sans-serif, system-ui, sans-serif`;
  ctx.fillText("Legenda dos garçons", metrics.padding, contentTop + metrics.titleFontSize);

  const colWidth = (canvas.width - metrics.padding * 2) / metrics.cols;
  const labelMaxWidth = colWidth - metrics.swatchSize - Math.round(12 * Math.max(1, canvas.width / 900));

  legendEntries.forEach((entry, index) => {
    const col = index % metrics.cols;
    const row = Math.floor(index / metrics.cols);
    const x = metrics.padding + col * colWidth;
    const y = contentTop + metrics.titleHeight + row * metrics.rowHeight;

    ctx.fillStyle = entry.color;
    ctx.fillRect(x, y, metrics.swatchSize, metrics.swatchSize);
    ctx.strokeStyle = "rgba(255,255,255,0.28)";
    ctx.lineWidth = Math.max(1, Math.round(canvas.width / 1200));
    ctx.strokeRect(x + 0.5, y + 0.5, metrics.swatchSize - 1, metrics.swatchSize - 1);

    ctx.fillStyle = LAYOUT_THEME.ink;
    ctx.font = `600 ${metrics.fontSize}px Inter, ui-sans-serif, system-ui, sans-serif`;
    fillTextEllipsis(ctx, entry.name, x + metrics.swatchSize + Math.round(8 * Math.max(1, canvas.width / 900)), y + metrics.swatchSize - 2, labelMaxWidth);
  });

  return canvas;
}

function downloadBlob(blob, filename) {
  if (!blob) throw new Error("Arquivo vazio");
  const link = document.createElement("a");
  const url = URL.createObjectURL(blob);
  link.href = url;
  link.download = filename;
  document.body.append(link);
  link.click();
  link.remove();
  setTimeout(() => URL.revokeObjectURL(url), 1000);
}

function buildPdfFromJpeg(jpegBytes, width, height, title) {
  const encoder = new TextEncoder();
  const parts = [];
  const offsets = [];
  const write = (text) => parts.push(encoder.encode(text));
  const writeBinary = (bytes) => parts.push(bytes);
  const currentLength = () => parts.reduce((sum, part) => sum + part.length, 0);

  write("%PDF-1.4\n");
  offsets[1] = currentLength();
  write("1 0 obj\n<< /Type /Catalog /Pages 2 0 R >>\nendobj\n");
  offsets[2] = currentLength();
  write("2 0 obj\n<< /Type /Pages /Kids [3 0 R] /Count 1 >>\nendobj\n");
  offsets[3] = currentLength();
  write(`3 0 obj\n<< /Type /Page /Parent 2 0 R /MediaBox [0 0 ${width} ${height}] /Resources << /XObject << /Im1 4 0 R >> /Font << /F1 5 0 R >> >> /Contents 6 0 R >>\nendobj\n`);
  offsets[4] = currentLength();
  write(`4 0 obj\n<< /Type /XObject /Subtype /Image /Width ${width} /Height ${height} /ColorSpace /DeviceRGB /BitsPerComponent 8 /Filter /DCTDecode /Length ${jpegBytes.length} >>\nstream\n`);
  writeBinary(jpegBytes);
  write("\nendstream\nendobj\n");
  offsets[5] = currentLength();
  write("5 0 obj\n<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>\nendobj\n");
  offsets[6] = currentLength();
  const safeTitle = (title || "Layout do salão").replace(/[()\\]/g, " ");
  const content = `BT /F1 14 Tf 36 ${height - 36} Td (${safeTitle}) Tj ET q ${width} 0 0 ${height - 56} 0 0 cm /Im1 Do Q`;
  write(`6 0 obj\n<< /Length ${content.length} >>\nstream\n${content}\nendstream\nendobj\n`);
  const xrefOffset = currentLength();
  write("xref\n0 7\n");
  write("0000000000 65535 f \n");
  for (let index = 1; index <= 6; index += 1) {
    write(`${String(offsets[index]).padStart(10, "0")} 00000 n \n`);
  }
  write(`trailer\n<< /Size 7 /Root 1 0 R >>\nstartxref\n${xrefOffset}\n%%EOF`);
  return new Blob(parts, { type: "application/pdf" });
}

function initializeLayoutEditor(root = document) {
  const editor = root.querySelector?.("[data-layout-editor]") || (root.matches?.("[data-layout-editor]") ? root : null);
  if (!editor || editor.dataset.initialized === "true") return;
  editor.dataset.initialized = "true";

  const mode = editor.dataset.layoutMode || "event";
  const saveForm = editor.querySelector("[data-layout-form]");
  if (mode === "standalone" && saveForm?.dataset.layoutKey === "standalone:new") {
    saveForm.dataset.layoutKey = `standalone:new:${crypto.randomUUID?.() || Date.now()}`;
  }
  const layoutDraftKey = saveForm?.dataset.layoutKey || "";
  const hiddenInput = editor.querySelector("[data-layout-json]");
  const stage = editor.querySelector("[data-layout-stage]");
  const svg = editor.querySelector("[data-layout-canvas]");
  const viewport = editor.querySelector("[data-layout-viewport]");
  const propsPanel = editor.querySelector("[data-layout-props]");
  const propsForm = editor.querySelector("[data-layout-props-form]");
  const legendList = editor.querySelector("[data-layout-legend-list]");
  const statTables = editor.querySelector('[data-layout-stat="tables"]');
  const statWaiters = editor.querySelector('[data-layout-stat="waiters"]');
  const statGuests = editor.querySelector('[data-layout-stat="guests"]');
  const statConfiguredWaiters = editor.querySelector('[data-layout-stat="configured-waiters"]');
  const floatSheet = editor.querySelector("[data-layout-sheet]");
  const floatSheetBody = editor.querySelector("[data-layout-sheet-body]");
  const floatSheetTitle = editor.querySelector("[data-layout-sheet-title]");
  const floatTop = editor.querySelector("[data-layout-float-top]");
  const metaForm = editor.querySelector("[data-layout-meta-form]");
  const waiterCountInput = editor.querySelector("[data-layout-waiter-count]");
  const waiterNameGrid = editor.querySelector("[data-layout-waiter-name-grid]");
  const waiterSelect = propsForm?.querySelector('[data-layout-prop="waiter"]');
  const zoomLabel = editor.querySelector("[data-layout-zoom-label]");
  const bgFill = svg.querySelector("[data-layout-bg-fill]");
  const bgGrid = svg.querySelector("[data-layout-bg-grid]");
  const placementHint = document.createElement("div");
  placementHint.className = "layout-placement-hint";
  placementHint.hidden = true;
  viewport?.append(placementHint);
  const layers = {
    tables: editor.querySelector('[data-layout-layer="tables"]'),
    markers: editor.querySelector('[data-layout-layer="markers"]'),
    preview: editor.querySelector('[data-layout-layer="preview"]'),
    selection: editor.querySelector('[data-layout-layer="selection"]'),
  };

  let state = parseLayoutState(hiddenInput?.value || DEFAULT_LAYOUT);
  let configuredWaiters = [...state.waiters];
  if (mode === "standalone") {
    configuredWaiters = parseWaiterNames(editor.dataset.waiterNames || "[]");
    if (!configuredWaiters.length && waiterCountInput) {
      configuredWaiters = Array.from({ length: Number(waiterCountInput.value) || 0 }, (_, index) => `Garçom ${index + 1}`);
    }
  } else if (!configuredWaiters.length) {
    const eventWaiters = Math.max(0, Number(editor.dataset.waiterCount) || 0);
    configuredWaiters = Array.from({ length: eventWaiters }, (_, index) => `Garçom ${index + 1}`);
  }
  state.waiters = [...configuredWaiters];
  if (waiterCountInput) {
    if (configuredWaiters.length) waiterCountInput.value = String(configuredWaiters.length);
    else configuredWaiters = resizeWaiterRoster([], Number(waiterCountInput.value) || 0);
    editor.dataset.waiterCount = String(configuredWaiters.length);
  }
  state.waiters = [...configuredWaiters];
  let includeCoLeader = false;

  let selectedIds = new Set();
  let placementWaiter = "";
  let activeTool = "select";
  let dragState = null;
  let resizeState = null;
  let marqueeState = null;
  let pendingRow = null;
  let rowPreview = null;
  let rowDragState = null;
  let fullscreenActive = false;
  let view = { x: 0, y: 0, scale: 1 };
  let panState = null;
  let pinchState = null;
  let draftReady = false;
  let draftTimer = null;
  let undoStack = [];
  let redoStack = [];
  let historyCurrent = JSON.stringify(state);
  let applyingHistory = false;
  const waiterRegistry = new Map();
  function registerWaiterColors() {
    waiterRegistry.clear();
    const shadow = shadowWaiterName();
    configuredWaiters.forEach((name) => layoutColorForWaiter(name, waiterRegistry, { shadow: name === shadow }));
    state.elements.forEach((item) => {
      if (item.waiter) layoutColorForWaiter(item.waiter, waiterRegistry, { shadow: item.waiter === shadow });
    });
  }

  function shadowWaiterName() {
    const names = configuredWaiters.filter(Boolean);
    return names.length ? names[names.length - 1] : "";
  }

  function staffCounts() {
    return {
      waiters: configuredWaiters.length || Number(waiterCountInput?.value) || Number(editor.dataset.waiterCount) || 0,
      coordinators: Number(editor.dataset.coordinatorCount) || 0,
      leaders: Number(editor.dataset.leaderCount) || 0,
      coleaders: Number(editor.dataset.coleaderCount) || 0,
    };
  }

  function guestCountValue() {
    if (metaForm) return Math.max(0, Number(metaForm.querySelector('[name="guest_count"]')?.value) || 0);
    return Math.max(0, Number(editor.dataset.guestCount) || 0);
  }

  function divisionTableCount() {
    const drawn = tableCount(state.elements);
    if (drawn > 0) return { tables: drawn, estimated: false };
    const guests = guestCountValue();
    if (guests <= 0) return { tables: 0, estimated: false };
    return { tables: Math.max(1, Math.ceil(guests / SEATS_PER_TABLE)), estimated: true };
  }

  function currentDivision() {
    const suggest = window.emenysSuggestFloorWaiterDivision;
    if (!suggest) return null;
    const staff = staffCounts();
    const tables = divisionTableCount();
    const plan = suggest({
      tables: tables.tables,
      waiters: staff.waiters,
      coordinators: staff.coordinators,
      leaders: staff.leaders,
      coleaders: staff.coleaders,
      includeCoLeader,
    });
    return { ...plan, estimated: tables.estimated, guestCount: guestCountValue() };
  }

  function servingNamesForDivision(plan) {
    const waiters = configuredWaiters.filter(Boolean);
    const serving = waiters.slice(0, Math.max(0, waiters.length - (plan?.shadowWaiters || 0)));
    const coleaders = window.emenysColeaderNames?.(plan?.usedCoLeaders || 0) || [];
    return [...serving, ...coleaders];
  }

  registerWaiterColors();

  function isMobileLayout() {
    return MOBILE_LAYOUT_QUERY.matches;
  }

  function syncViewportSize() {
    if (!viewport) return;
    const rect = viewport.getBoundingClientRect();
    svg.setAttribute("width", String(Math.max(1, Math.round(rect.width))));
    svg.setAttribute("height", String(Math.max(1, Math.round(rect.height))));
  }

  function updateInfiniteBackground(viewWidth, viewHeight) {
    const pad = Math.max(viewWidth, viewHeight, 800);
    const x = view.x - pad;
    const y = view.y - pad;
    const width = viewWidth + pad * 2;
    const height = viewHeight + pad * 2;
    [bgFill, bgGrid].forEach((node) => {
      if (!node) return;
      node.setAttribute("x", String(x));
      node.setAttribute("y", String(y));
      node.setAttribute("width", String(width));
      node.setAttribute("height", String(height));
    });
  }

  function viewportRect() {
    return viewport?.getBoundingClientRect() || svg.getBoundingClientRect();
  }

  function clampScale(value) {
    const scale = Number(value) || 1;
    return Math.max(MIN_VIEW_SCALE, Math.min(MAX_VIEW_SCALE, scale));
  }

  function comfortableScale() {
    const target = isMobileLayout() ? TARGET_TABLE_PX : 72;
    return clampScale(target / TABLE_ROUND_SIZE);
  }

  function clientToWorld(clientX, clientY) {
    const rect = viewportRect();
    return {
      x: view.x + (clientX - rect.left) / view.scale,
      y: view.y + (clientY - rect.top) / view.scale,
    };
  }

  function setScaleAroundClient(clientX, clientY, nextScale) {
    const world = clientToWorld(clientX, clientY);
    view.scale = clampScale(nextScale);
    const rect = viewportRect();
    view.x = world.x - (clientX - rect.left) / view.scale;
    view.y = world.y - (clientY - rect.top) / view.scale;
    updateViewBox();
  }

  function visibleWorldSize() {
    const rect = viewportRect();
    return {
      width: Math.max(1, rect.width) / view.scale,
      height: Math.max(1, rect.height) / view.scale,
    };
  }

  function updateViewBox() {
    syncViewportSize();
    view.scale = clampScale(view.scale);
    const size = visibleWorldSize();
    svg.setAttribute("viewBox", `${view.x} ${view.y} ${size.width} ${size.height}`);
    svg.setAttribute("preserveAspectRatio", "none");
    updateInfiniteBackground(size.width, size.height);
    if (zoomLabel) zoomLabel.textContent = `${Math.round((view.scale * TABLE_ROUND_SIZE / TARGET_TABLE_PX) * 100)}%`;
  }

  function clampView() {
    // Pan livre — o grid infinito acompanha a área visível.
  }

  function fitViewToContent() {
    const rect = viewportRect();
    if (!rect.width || !rect.height) {
      view.scale = comfortableScale();
      view.x = 0;
      view.y = 0;
      updateViewBox();
      return;
    }
    if (!state.elements.length) {
      view.scale = comfortableScale();
      view.x = -GRID_SIZE;
      view.y = -GRID_SIZE * 0.5;
      updateViewBox();
      return;
    }
    const bounds = contentBounds(isMobileLayout() ? 48 : 80);
    const scale = clampScale(Math.min(rect.width / Math.max(bounds.width, 1), rect.height / Math.max(bounds.height, 1)));
    view.scale = scale;
    const size = {
      width: rect.width / view.scale,
      height: rect.height / view.scale,
    };
    view.x = bounds.x + bounds.width / 2 - size.width / 2;
    view.y = bounds.y + bounds.height / 2 - size.height / 2;
    updateViewBox();
  }

  function updateSelectionOutline(element) {
    if (!element || !layers.selection) return;
    const pad = 8;
    const outline = layers.selection.querySelector(`[data-layout-selection-id="${element.id}"]`)
      || layers.selection.querySelector("[data-layout-selection-outline]");
    if (outline) {
      outline.setAttribute("x", String(element.x - pad));
      outline.setAttribute("y", String(element.y - pad));
      outline.setAttribute("width", String(element.width + pad * 2));
      outline.setAttribute("height", String(element.height + pad * 2));
    }
    layers.selection.querySelectorAll("[data-layout-resize-handle]").forEach((hit) => {
      const handleId = hit.getAttribute("data-layout-resize-handle");
      if (!handleId) return;
      const [cx, cy] = resizeHandlePosition(element, handleId, pad);
      hit.setAttribute("cx", String(cx));
      hit.setAttribute("cy", String(cy));
    });
  }

  function contentBounds(padding = 100) {
    if (!state.elements.length) {
      return {
        x: -padding,
        y: -padding,
        width: state.width + padding * 2,
        height: state.height + padding * 2,
      };
    }
    let minX = Infinity;
    let minY = Infinity;
    let maxX = -Infinity;
    let maxY = -Infinity;
    state.elements.forEach((item) => {
      minX = Math.min(minX, item.x);
      minY = Math.min(minY, item.y);
      maxX = Math.max(maxX, item.x + item.width);
      maxY = Math.max(maxY, item.y + item.height);
    });
    return {
      x: minX - padding,
      y: minY - padding,
      width: maxX - minX + padding * 2,
      height: maxY - minY + padding * 2,
    };
  }

  function fitInitialView() {
    fitViewToContent();
  }

  function resetView() {
    fitInitialView();
  }

  function clearPlacementHint() {
    if (!placementHint) return;
    placementHint.hidden = true;
    placementHint.classList.remove("is-interactive");
    placementHint.replaceChildren();
  }

  function setPlacementHint(message) {
    if (!placementHint) return;
    if (!message) {
      clearPlacementHint();
      return;
    }
    placementHint.hidden = false;
    placementHint.classList.remove("is-interactive");
    placementHint.replaceChildren();
    placementHint.textContent = message;
  }

  function showRowPreviewHint() {
    if (!placementHint) return;
    placementHint.hidden = false;
    placementHint.classList.add("is-interactive");
    placementHint.replaceChildren();
    const text = document.createElement("p");
    text.className = "layout-placement-text";
    text.textContent = "Arraste a fileira para posicionar. Toque em Confirmar quando estiver no lugar certo.";
    const actions = document.createElement("div");
    actions.className = "layout-placement-actions";
    const confirm = document.createElement("button");
    confirm.type = "button";
    confirm.className = "layout-placement-btn primary";
    confirm.textContent = "Confirmar fileira";
    confirm.addEventListener("click", (event) => {
      event.stopPropagation();
      confirmRowPreview();
    });
    const cancel = document.createElement("button");
    cancel.type = "button";
    cancel.className = "layout-placement-btn";
    cancel.textContent = "Cancelar";
    cancel.addEventListener("click", (event) => {
      event.stopPropagation();
      cancelRowPreview();
    });
    actions.append(confirm, cancel);
    placementHint.append(text, actions);
  }

  function viewportCenterPoint() {
    if (!viewport) return { x: state.width / 2, y: state.height / 2 };
    const rect = viewport.getBoundingClientRect();
    return svgPoint(svg, rect.left + rect.width / 2, rect.top + rect.height / 2);
  }

  function rowLayoutMetrics(config) {
    const type = config.type === "table_rect" ? "table_rect" : "table_round";
    const count = Math.max(1, Math.min(30, Number(config.count) || 5));
    const horizontal = config.direction !== "vertical";
    const gap = TABLE_ROW_GAP;
    const { width, height } = tableDimensions(type, "medium");
    const rowWidth = horizontal ? count * width + (count - 1) * gap : width;
    const rowHeight = horizontal ? height : count * height + (count - 1) * gap;
    return { type, count, horizontal, gap, width, height, rowWidth, rowHeight };
  }

  function rowStartFromPoint(point, metrics) {
    return snapPosition(point.x - metrics.rowWidth / 2, point.y - metrics.rowHeight / 2, metrics.rowWidth, metrics.rowHeight);
  }

  function buildRowTableSpecs(startX, startY, config, labelOffset = 0) {
    const metrics = rowLayoutMetrics(config);
    const specs = [];
    for (let index = 0; index < metrics.count; index += 1) {
      const x = metrics.horizontal ? startX + index * (metrics.width + metrics.gap) : startX;
      const y = metrics.horizontal ? startY : startY + index * (metrics.height + metrics.gap);
      if (x + metrics.width > WORLD_MAX || y + metrics.height > WORLD_MAX) break;
      specs.push({
        type: metrics.type,
        x,
        y,
        width: metrics.width,
        height: metrics.height,
        label: `Mesa ${labelOffset + index + 1}`,
      });
    }
    return specs;
  }

  function beginRowPlacement(config) {
    pendingRow = config;
    const metrics = rowLayoutMetrics(config);
    const start = rowStartFromPoint(viewportCenterPoint(), metrics);
    rowPreview = { startX: start.x, startY: start.y };
    setActiveTool("place_row");
    closeSheet();
    showRowPreviewHint();
    render();
  }

  function cancelRowPreview() {
    pendingRow = null;
    rowPreview = null;
    rowDragState = null;
    clearPlacementHint();
    setActiveTool("select");
    render();
  }

  function confirmRowPreview() {
    if (!pendingRow || !rowPreview) return;
    commitTableRow(rowPreview.startX, rowPreview.startY, pendingRow);
    pendingRow = null;
    rowPreview = null;
    rowDragState = null;
    clearPlacementHint();
    setActiveTool("select");
    render();
  }

  function beginRowDrag(event) {
    if (!pendingRow || !rowPreview) return;
    rowDragState = {
      pointerId: event.pointerId,
      startX: event.clientX,
      startY: event.clientY,
      originStartX: rowPreview.startX,
      originStartY: rowPreview.startY,
    };
    event.currentTarget.setPointerCapture(event.pointerId);
  }

  function updateRowDrag(event) {
    if (!rowDragState || !pendingRow || !rowPreview) return;
    const matrix = svg.getScreenCTM();
    if (!matrix) return;
    const dx = (event.clientX - rowDragState.startX) / matrix.a;
    const dy = (event.clientY - rowDragState.startY) / matrix.d;
    const metrics = rowLayoutMetrics(pendingRow);
    const snapped = snapPosition(
      rowDragState.originStartX + dx,
      rowDragState.originStartY + dy,
      metrics.rowWidth,
      metrics.rowHeight,
    );
    rowPreview.startX = snapped.x;
    rowPreview.startY = snapped.y;
    renderRowPreview();
  }

  function finishRowDrag(event) {
    if (!rowDragState) return;
    if (event?.pointerId === rowDragState.pointerId) {
      event.currentTarget?.releasePointerCapture?.(event.pointerId);
    }
    rowDragState = null;
  }

  function sortedElements() {
    return [...state.elements].sort((a, b) => (a.zIndex || 0) - (b.zIndex || 0));
  }

  function selectedElements() {
    return state.elements.filter((item) => selectedIds.has(item.id));
  }

  function selectedElement() {
    const items = selectedElements();
    return items[items.length - 1] || null;
  }

  function isTableElement(item) {
    return item?.type === "table_round" || item?.type === "table_rect";
  }

  function isTablePlacementTool(tool = activeTool) {
    return tool === "table_round" || tool === "table_rect";
  }

  function sharedValue(items, getter) {
    if (!items.length) return { mixed: false, value: "" };
    const values = items.map(getter);
    const first = values[0];
    return { mixed: values.some((value) => value !== first), value: first };
  }

  function normalizeRect(x1, y1, x2, y2) {
    const x = Math.min(x1, x2);
    const y = Math.min(y1, y2);
    return { x, y, width: Math.abs(x2 - x1), height: Math.abs(y2 - y1) };
  }

  function rectsOverlap(a, b) {
    return a.x < b.x + b.width && a.x + a.width > b.x && a.y < b.y + b.height && a.y + a.height > b.y;
  }

  function appendDisabledOption(select, value, label) {
    if (!select || [...select.options].some((option) => option.value === value)) return;
    const option = document.createElement("option");
    option.value = value;
    option.textContent = label;
    option.disabled = true;
    select.insertBefore(option, select.firstChild);
  }

  function isWaiterNameGridEditing() {
    const active = document.activeElement;
    return Boolean(active?.closest("[data-layout-waiter-name-grid]"));
  }

  function applyConfiguredWaiters() {
    state.waiters = configuredWaiters.filter(Boolean);
    registerWaiterColors();
    refreshWaiterSelect();
    syncStandaloneHiddenFields();
    if (statConfiguredWaiters) statConfiguredWaiters.textContent = String(configuredWaiters.length);
    updateLegend();
    refreshDivisionPanel();
  }

  function syncConfiguredWaiters(options = {}) {
    applyConfiguredWaiters();
    const rebuildGrid = options.rebuildGrid ?? !isWaiterNameGridEditing();
    if (rebuildGrid) renderWaiterNameGrid();
    if (options.render !== false) render();
  }

  function bindWaiterNameInput(input, index, previousName) {
    input.dataset.waiterIndex = String(index);
    const commit = () => {
      const next = input.value.trim() || previousName || `Garçom ${index + 1}`;
      const previous = configuredWaiters[index];
      configuredWaiters[index] = next;
      remapWaiterNameOnElements(state.elements, previous, next);
      syncConfiguredWaiters({ rebuildGrid: true });
    };
    input.addEventListener("blur", commit);
    input.addEventListener("keydown", (event) => {
      if (event.key === "Enter") {
        event.preventDefault();
        input.blur();
      }
      if (event.key === "Escape") {
        event.preventDefault();
        input.value = previousName;
        input.blur();
      }
    });
  }

  function startWaiterNameEdit(button, index) {
    const previous = configuredWaiters[index] || `Garçom ${index + 1}`;
    const input = document.createElement("input");
    input.type = "text";
    input.className = "layout-waiter-name-input";
    input.value = "";
    input.placeholder = "Nome do garçom";
    input.setAttribute("aria-label", `Nome do garçom ${index + 1}`);
    bindWaiterNameInput(input, index, previous);
    button.replaceWith(input);
    input.focus();
  }

  function fillWaiterNameGrid(grid) {
    if (!grid) return;
    grid.hidden = false;
    grid.replaceChildren();
    const shadow = shadowWaiterName();
    configuredWaiters.forEach((name, index) => {
      const card = document.createElement("div");
      card.className = "layout-waiter-name-card";
      const title = document.createElement("span");
      title.className = "layout-waiter-name-title";
      const swatch = document.createElement("span");
      swatch.className = "layout-legend-swatch";
      swatch.style.background = layoutColorForWaiter(name, waiterRegistry, { shadow: name === shadow });
      title.append(swatch, document.createTextNode(name === shadow ? `Garçom ${index + 1} · sombra` : `Garçom ${index + 1}`));
      const button = document.createElement("button");
      button.type = "button";
      button.className = "layout-waiter-name-value";
      button.textContent = name;
      button.setAttribute("aria-label", `Trocar nome do ${name}`);
      button.addEventListener("click", () => startWaiterNameEdit(button, index));
      card.append(title, button);
      grid.append(card);
    });
  }

  function renderWaiterNameGrid() {
    editor.querySelectorAll("[data-layout-waiter-name-grid]").forEach((grid) => fillWaiterNameGrid(grid));
  }
  function refreshWaiterSelect(select) {
    if (select) {
      fillWaiterSelect(select);
      return;
    }
    fillWaiterSelect(waiterSelect);
    editor.querySelectorAll("[data-layout-place-waiter], [data-layout-prop='waiter']").forEach((node) => fillWaiterSelect(node));
  }

  function fillWaiterSelect(select) {
    if (!select) return;
    const current = select.value === "__mixed__" ? "" : select.value;
    const keepPlacement = select.hasAttribute("data-layout-place-waiter") ? placementWaiter : current;
    select.replaceChildren();
    const empty = document.createElement("option");
    empty.value = "";
    empty.textContent = "Sem garçom";
    select.append(empty);
    configuredWaiters.filter(Boolean).forEach((name) => {
      const option = document.createElement("option");
      option.value = name;
      option.textContent = name;
      select.append(option);
    });
    [...new Set([...configuredWaiters, ...waiterNamesFromElements(state.elements)])].filter(Boolean).forEach((name) => {
      if ([...select.options].some((option) => option.value === name)) return;
      const option = document.createElement("option");
      option.value = name;
      option.textContent = name;
      select.append(option);
    });
    if (select.hasAttribute("data-layout-place-waiter")) {
      select.value = [...select.options].some((option) => option.value === placementWaiter) ? placementWaiter : "";
    } else {
      select.value = [...select.options].some((option) => option.value === keepPlacement) ? keepPlacement : "";
    }
  }

  function syncWaiterCountInputs(count) {
    editor.querySelectorAll("[data-layout-waiter-count]").forEach((input) => {
      if (document.activeElement === input) return;
      input.value = String(count);
    });
    if (waiterCountInput && document.activeElement !== waiterCountInput) {
      waiterCountInput.value = String(count);
    }
    editor.dataset.waiterCount = String(count);
  }

  function rebuildWaiterNamesFromCount(rawCount) {
    const source = rawCount ?? editor.querySelector("[data-layout-waiter-count]:focus")?.value ?? waiterCountInput?.value ?? configuredWaiters.length;
    const count = Math.max(0, Math.min(30, Number(source) || 0));
    if (waiterCountInput) waiterCountInput.value = String(count);
    syncWaiterCountInputs(count);
    configuredWaiters = resizeWaiterRoster(configuredWaiters, count);
    syncConfiguredWaiters({ rebuildGrid: true, render: true });
    updateStats();
  }

  function stepWaiterCount(delta, fromInput) {
    const current = Number(fromInput?.value ?? waiterCountInput?.value ?? configuredWaiters.length) || 0;
    const next = Math.max(0, Math.min(30, current + delta));
    if (fromInput) fromInput.value = String(next);
    rebuildWaiterNamesFromCount(next);
  }

  function knownVenues() {
    try {
      const parsed = JSON.parse(editor.dataset.knownVenues || "[]");
      return Array.isArray(parsed) ? parsed.map((name) => String(name || "").trim()).filter(Boolean) : [];
    } catch (_error) {
      return [];
    }
  }

  function venueValue(root = metaForm) {
    return (root?.querySelector('[name="venue"]')?.value || editor.dataset.venue || "").trim();
  }

  function syncLayoutNameFromVenue(root = metaForm) {
    const venue = venueValue(root);
    root?.querySelectorAll('[name="name"]').forEach((input) => {
      input.value = venue;
    });
    if (metaForm && root !== metaForm) {
      const originalVenue = metaForm.querySelector('[name="venue"]');
      const originalName = metaForm.querySelector('[name="name"]');
      if (originalVenue) originalVenue.value = venue;
      if (originalName) originalName.value = venue;
    }
    if (venue) {
      editor.dataset.exportTitle = venue;
      editor.dataset.venue = venue;
      const floatTitle = editor.querySelector("[data-layout-float-title]");
      if (floatTitle) floatTitle.textContent = venue;
    }
    syncStandaloneHiddenFields();
  }

  function filterKnownVenues(query) {
    const needle = (query || "").trim().toLowerCase();
    const names = knownVenues();
    if (!needle) return names.slice(0, 8);
    return names.filter((name) => name.toLowerCase().includes(needle)).slice(0, 8);
  }

  function bindVenueSearch(root = metaForm) {
    if (!root) return;
    const input = root.querySelector("[data-layout-venue-search], [name='venue']");
    const list = root.querySelector("[data-layout-venue-results]");
    if (!input) return;
    const renderMatches = () => {
      if (!list) return;
      const matches = filterKnownVenues(input.value);
      list.replaceChildren();
      if (!matches.length || matches.some((name) => name.toLowerCase() === input.value.trim().toLowerCase())) {
        list.hidden = true;
        return;
      }
      matches.forEach((name) => {
        const item = document.createElement("button");
        item.type = "button";
        item.className = "layout-venue-option";
        item.textContent = name;
        item.addEventListener("mousedown", (event) => event.preventDefault());
        item.addEventListener("click", () => {
          input.value = name;
          list.hidden = true;
          syncLayoutNameFromVenue(root);
          updateStats();
        });
        list.append(item);
      });
      list.hidden = false;
    };
    if (input.dataset.venueBound === "1") return;
    input.dataset.venueBound = "1";
    input.addEventListener("input", () => {
      syncLayoutNameFromVenue(root);
      updateStats();
      renderMatches();
    });
    input.addEventListener("focus", renderMatches);
    input.addEventListener("blur", () => window.setTimeout(() => {
      if (list) list.hidden = true;
    }, 120));
  }

  function requireVenueToContinue() {
    if (mode !== "standalone") {
      if (!(editor.dataset.venue || "").trim()) {
        window.alert("Informe o local do evento antes de montar o layout.");
        return false;
      }
      return true;
    }
    if (venueValue()) {
      syncLayoutNameFromVenue();
      return true;
    }
    const setupOpen = floatSheet && !floatSheet.hidden && floatSheet.dataset.open === "setup";
    if (!setupOpen) openSetupSheet();
    const input = floatSheet?.querySelector("[data-layout-venue-search], [name='venue']") || metaForm?.querySelector("[name='venue']");
    input?.focus();
    window.alert("Informe o local do evento. Se não estiver na lista, preencha o nome do espaço.");
    return false;
  }

  function syncStandaloneHiddenFields() {
    if (mode !== "standalone" || !saveForm) return;
    const set = (field, value) => {
      const input = saveForm.querySelector(`[data-layout-field="${field}"]`);
      if (input) input.value = value;
    };
    if (metaForm) {
      set("name", venueValue() || metaForm.querySelector('[name="name"]')?.value || "");
      set("venue", venueValue());
      set("guest_count", metaForm.querySelector('[name="guest_count"]')?.value || "0");
      set("waiter_count", waiterCountInput?.value || metaForm.querySelector('[name="waiter_count"]')?.value || "0");
    }
    set("waiter_names_json", JSON.stringify(configuredWaiters.filter(Boolean)));
  }

  function persistHiddenInput() {
    state.waiters = configuredWaiters.filter(Boolean);
    if (hiddenInput) hiddenInput.value = JSON.stringify(state);
    syncStandaloneHiddenFields();
  }

  function updateStats() {
    if (statTables) statTables.textContent = String(tableCount(state.elements));
    if (statWaiters) statWaiters.textContent = String(waiterNamesFromElements(state.elements).length);
    if (statGuests && metaForm) statGuests.textContent = metaForm.querySelector('[name="guest_count"]')?.value || "0";
    if (statConfiguredWaiters) statConfiguredWaiters.textContent = String(configuredWaiters.length || Number(waiterCountInput?.value) || 0);
    refreshDivisionPanel();
  }

  function updateLegend() {
    if (!legendList) return;
    legendList.replaceChildren();
    const shadow = shadowWaiterName();
    const names = sortWaiterNames([...new Set([...configuredWaiters, ...servingNamesForDivision(currentDivision()), ...waiterNamesFromElements(state.elements)])].filter(Boolean));
    names.forEach((name) => {
      const li = document.createElement("li");
      const swatch = document.createElement("span");
      swatch.className = "layout-legend-swatch";
      swatch.style.background = layoutColorForWaiter(name, waiterRegistry, { shadow: name === shadow });
      const label = document.createElement("span");
      label.textContent = name === shadow ? `${name} · sombra dos noivos` : name;
      li.append(swatch, label);
      legendList.append(li);
    });
  }

  function formatTablesPerPerson(value) {
    if (!Number.isFinite(value) || value <= 0) return "—";
    return Number.isInteger(value) ? String(value) : value.toFixed(1).replace(".", ",");
  }

  function refreshDivisionPanel(root = editor) {
    const plan = currentDivision();
    if (!plan) return;
    root.querySelectorAll("[data-layout-division-summary]").forEach((node) => {
      const parts = [
        plan.estimated
          ? `${plan.tableCount} ${plan.tableCount === 1 ? "mesa prevista" : "mesas previstas"}`
          : `${plan.tableCount} ${plan.tableCount === 1 ? "mesa" : "mesas"}`,
        plan.guestCount ? `${plan.guestCount} convidados` : null,
        `${plan.waiterCount} ${plan.waiterCount === 1 ? "garçom" : "garçons"}`,
        plan.shadowWaiters ? "1 sombra dos noivos" : null,
        "coordenação e líder sem mesa",
      ].filter(Boolean);
      if (plan.servingPeople > 0 && plan.tableCount > 0) {
        const per = formatTablesPerPerson(plan.tablesPerPerson);
        parts.push(`${per} ${per === "1" ? "mesa" : "mesas"} por pessoa na pista`);
      } else if (plan.tableCount > 0) {
        parts.push("ninguém na pista até incluir o colíder ou mais garçons");
      }
      node.textContent = parts.join(" · ");
    });
    root.querySelectorAll("[data-layout-division-coleader-row]").forEach((row) => {
      row.hidden = !plan.offerCoLeader;
    });
    root.querySelectorAll("[data-layout-include-coleader]").forEach((input) => {
      input.checked = includeCoLeader && plan.offerCoLeader;
    });
    root.querySelectorAll("[data-layout-division-shares]").forEach((list) => {
      list.replaceChildren();
      const names = servingNamesForDivision(plan);
      names.forEach((name, index) => {
        const item = document.createElement("li");
        const swatch = document.createElement("span");
        swatch.className = "layout-legend-swatch";
        swatch.style.background = layoutColorForWaiter(name, waiterRegistry);
        const label = document.createElement("span");
        const tables = plan.shares[index] || 0;
        label.textContent = `${name} · ${tables} ${tables === 1 ? "mesa" : "mesas"}`;
        item.append(swatch, label);
        list.append(item);
      });
      if (plan.shadowWaiters) {
        const shadow = shadowWaiterName();
        if (shadow) {
          const item = document.createElement("li");
          const swatch = document.createElement("span");
          swatch.className = "layout-legend-swatch";
          swatch.style.background = layoutColorForWaiter(shadow, waiterRegistry, { shadow: true });
          const label = document.createElement("span");
          label.textContent = `${shadow} · sombra dos noivos`;
          item.append(swatch, label);
          list.append(item);
        }
      }
    });
  }

  function applySuggestedDivision() {
    const plan = currentDivision();
    if (!plan) return;
    const names = servingNamesForDivision(plan);
    if (!names.length) {
      window.alert("Não há quem assuma mesa. Inclua o colíder ou cadastre mais garçons.");
      return;
    }
    names.forEach((name) => layoutColorForWaiter(name, waiterRegistry));
    const tables = state.elements
      .filter((item) => item.type === "table_round" || item.type === "table_rect")
      .sort((left, right) => left.y - right.y || left.x - right.x);
    let offset = 0;
    names.forEach((name, index) => {
      const count = plan.shares[index] || 0;
      tables.slice(offset, offset + count).forEach((item) => {
        item.waiter = name;
        item.color = "";
      });
      offset += count;
    });
    if (plan.usedCoLeaders) {
      state.waiters = [...new Set([...configuredWaiters.filter(Boolean), ...names])];
    }
    render();
    persistHiddenInput();
  }

  function bindDivisionControls(root = editor) {
    root.querySelectorAll("[data-layout-include-coleader]").forEach((input) => {
      input.addEventListener("change", () => {
        includeCoLeader = input.checked;
        refreshDivisionPanel();
        if (root !== editor) refreshDivisionPanel(root);
        updateLegend();
      });
    });
    root.querySelectorAll("[data-layout-apply-division]").forEach((button) => {
      if (button.dataset.bound === "1") return;
      button.dataset.bound = "1";
      button.addEventListener("click", applySuggestedDivision);
    });
  }

  function renderGhostTable(spec, waiterName = "") {
    const group = document.createElementNS("http://www.w3.org/2000/svg", "g");
    const waiter = (waiterName || "").trim();
    const centerX = spec.x + spec.width / 2;
    const centerY = spec.y + spec.height / 2;
    appendGlassShape(group, spec, waiterRegistry, { ghost: true, waiter });
    const label = svgEl("text", {
      x: centerX,
      y: centerY - (waiter ? 6 : 0),
      "text-anchor": "middle",
      "dominant-baseline": "middle",
      fill: LAYOUT_THEME.ink,
      "font-size": "13",
      "font-weight": "700",
    });
    label.textContent = spec.label;
    group.append(label);
    if (waiter) {
      const waiterLabel = svgEl("text", {
        x: centerX,
        y: centerY + 12,
        "text-anchor": "middle",
        "dominant-baseline": "middle",
        fill: layoutColorForWaiter(waiter, waiterRegistry),
        "font-size": "11",
        "font-weight": "600",
      });
      waiterLabel.textContent = waiter;
      group.append(waiterLabel);
    }
    return group;
  }

  function renderRowPreview() {
    if (!layers.preview) return;
    layers.preview.replaceChildren();
    if (activeTool !== "place_row" || !pendingRow || !rowPreview) return;

    const metrics = rowLayoutMetrics(pendingRow);
    const specs = buildRowTableSpecs(rowPreview.startX, rowPreview.startY, pendingRow, tableCount(state.elements));
    const group = document.createElementNS("http://www.w3.org/2000/svg", "g");
    group.style.cursor = "grab";

    const outline = document.createElementNS("http://www.w3.org/2000/svg", "rect");
    outline.setAttribute("x", String(rowPreview.startX - 8));
    outline.setAttribute("y", String(rowPreview.startY - 8));
    outline.setAttribute("width", String(metrics.rowWidth + 16));
    outline.setAttribute("height", String(metrics.rowHeight + 16));
    outline.setAttribute("fill", LAYOUT_THEME.preview);
    outline.setAttribute("stroke", LAYOUT_THEME.selection);
    outline.setAttribute("stroke-width", "2");
    outline.setAttribute("stroke-dasharray", "10 6");
    outline.setAttribute("rx", "10");
    group.append(outline);

    const hitArea = document.createElementNS("http://www.w3.org/2000/svg", "rect");
    hitArea.setAttribute("x", String(rowPreview.startX - 8));
    hitArea.setAttribute("y", String(rowPreview.startY - 8));
    hitArea.setAttribute("width", String(metrics.rowWidth + 16));
    hitArea.setAttribute("height", String(metrics.rowHeight + 16));
    hitArea.setAttribute("fill", "transparent");
    hitArea.style.cursor = "grab";
    hitArea.addEventListener("pointerdown", (event) => {
      event.stopPropagation();
      beginRowDrag(event);
    });
    hitArea.addEventListener("pointermove", updateRowDrag);
    hitArea.addEventListener("pointerup", finishRowDrag);
    hitArea.addEventListener("pointercancel", finishRowDrag);
    group.append(hitArea);

    specs.forEach((spec) => group.append(renderGhostTable(spec, pendingRow.waiter || "")));
    layers.preview.append(group);
  }

  function resizeHandlePosition(element, handleId, pad = 8) {
    const left = element.x - pad;
    const right = element.x + element.width + pad;
    const top = element.y - pad;
    const bottom = element.y + element.height + pad;
    const cx = element.x + element.width / 2;
    const cy = element.y + element.height / 2;
    const positions = {
      nw: [left, top],
      n: [cx, top],
      ne: [right, top],
      e: [right, cy],
      se: [right, bottom],
      s: [cx, bottom],
      sw: [left, bottom],
      w: [left, cy],
    };
    return positions[handleId] || [cx, cy];
  }

  function updateResizableVisual(element) {
    const group = findElementGroup(element.id);
    if (!group) return;
    syncGlassGeometry(group, element);
    const texts = group.querySelectorAll("text:not([data-layout-export-hide])");
    const centerX = element.x + element.width / 2;
    const centerY = element.y + element.height / 2;
    if (texts[0]) {
      texts[0].setAttribute("x", String(centerX));
      texts[0].setAttribute("y", String(centerY - (texts[1] ? 6 : 0)));
    }
    if (texts[1]) {
      texts[1].setAttribute("x", String(centerX));
      texts[1].setAttribute("y", String(centerY + 12));
    }
  }

  function beginResize(event, elementId, handle) {
    event.stopPropagation();
    event.preventDefault();
    if (activeTool !== "select") return;
    const element = state.elements.find((item) => item.id === elementId);
    if (!element || !isResizableElement(element)) return;
    selectElement(elementId);
    dragState = null;
    rowDragState = null;
    resizeState = {
      id: elementId,
      handle,
      startX: event.clientX,
      startY: event.clientY,
      originX: element.x,
      originY: element.y,
      originWidth: element.width,
      originHeight: element.height,
      moved: false,
      pointerId: event.pointerId,
    };
    closeSheet();
    svg.setPointerCapture(event.pointerId);
  }

  function updateResize(event) {
    if (!resizeState) return;
    const matrix = svg.getScreenCTM();
    if (!matrix) return;
    const dx = (event.clientX - resizeState.startX) / matrix.a;
    const dy = (event.clientY - resizeState.startY) / matrix.d;
    if (!resizeState.moved && Math.hypot(dx, dy) >= 2) resizeState.moved = true;
    const element = state.elements.find((item) => item.id === resizeState.id);
    if (!element) return;
    const bounds = computeResizeBounds(resizeState, dx, dy);
    resizeState.pending = bounds;
    const preview = { ...element, ...bounds };
    updateResizableVisual(preview);
    updateSelectionOutline(preview);
  }

  function finishResize(event) {
    if (!resizeState) return;
    if (!event || event.pointerId === resizeState.pointerId) {
      svg.releasePointerCapture?.(resizeState.pointerId);
    }
    const element = state.elements.find((item) => item.id === resizeState.id);
    if (element && resizeState.moved && resizeState.pending) {
      Object.assign(element, resizeState.pending);
    }
    resizeState = null;
    render();
  }

  function renderSelection(elements) {
    layers.selection.replaceChildren();
    const items = Array.isArray(elements) ? elements : selectedElements();
    if (!items.length) return;
    const pad = 8;
    const showHandles = items.length === 1 && isResizableElement(items[0]) && activeTool === "select";
    items.forEach((element) => {
      const rect = document.createElementNS("http://www.w3.org/2000/svg", "rect");
      rect.setAttribute("data-layout-selection-outline", "1");
      rect.setAttribute("data-layout-selection-id", element.id);
      rect.setAttribute("x", String(element.x - pad));
      rect.setAttribute("y", String(element.y - pad));
      rect.setAttribute("width", String(element.width + pad * 2));
      rect.setAttribute("height", String(element.height + pad * 2));
      rect.setAttribute("fill", "none");
      rect.setAttribute("stroke", LAYOUT_THEME.selection);
      rect.setAttribute("stroke-width", "2");
      rect.setAttribute("stroke-dasharray", "6 4");
      rect.setAttribute("rx", "10");
      layers.selection.append(rect);
    });

    if (showHandles) {
      const element = items[0];
      RESIZE_HANDLES.forEach((handle) => {
        const [cx, cy] = resizeHandlePosition(element, handle.id, pad);
        const hit = document.createElementNS("http://www.w3.org/2000/svg", "circle");
        hit.setAttribute("cx", String(cx));
        hit.setAttribute("cy", String(cy));
        hit.setAttribute("r", "11");
        hit.setAttribute("fill", "#111111");
        hit.setAttribute("stroke", LAYOUT_THEME.selection);
        hit.setAttribute("stroke-width", "2");
        hit.setAttribute("data-layout-resize-handle", handle.id);
        hit.style.cursor = handle.cursor;
        hit.addEventListener("pointerdown", (event) => beginResize(event, element.id, handle.id));
        layers.selection.append(hit);
      });
    }
  }

  function renderMarquee() {
    if (!layers.preview || !marqueeState) return;
    layers.preview.replaceChildren();
    const box = normalizeRect(marqueeState.startX, marqueeState.startY, marqueeState.currentX, marqueeState.currentY);
    if (box.width < 2 && box.height < 2) return;
    const rect = document.createElementNS("http://www.w3.org/2000/svg", "rect");
    rect.setAttribute("x", String(box.x));
    rect.setAttribute("y", String(box.y));
    rect.setAttribute("width", String(box.width));
    rect.setAttribute("height", String(box.height));
    rect.setAttribute("fill", LAYOUT_THEME.preview);
    rect.setAttribute("stroke", LAYOUT_THEME.selection);
    rect.setAttribute("stroke-width", "1.5");
    rect.setAttribute("stroke-dasharray", "8 5");
    rect.setAttribute("rx", "4");
    layers.preview.append(rect);
  }

  function updateMarquee(event) {
    if (!marqueeState) return;
    const world = clientToWorld(event.clientX, event.clientY);
    marqueeState.currentX = world.x;
    marqueeState.currentY = world.y;
    const box = normalizeRect(marqueeState.startX, marqueeState.startY, marqueeState.currentX, marqueeState.currentY);
    selectedIds = new Set(marqueeState.baseline);
    state.elements.forEach((item) => {
      if (rectsOverlap(item, box)) selectedIds.add(item.id);
    });
    renderMarquee();
    renderSelection();
  }

  function finishMarquee() {
    if (!marqueeState) return;
    const box = normalizeRect(marqueeState.startX, marqueeState.startY, marqueeState.currentX, marqueeState.currentY);
    const moved = box.width >= 6 || box.height >= 6;
    if (!moved && !marqueeState.keepOnClick) selectedIds.clear();
    marqueeState = null;
    if (layers.preview && activeTool !== "place_row") layers.preview.replaceChildren();
    renderSelection();
    syncPropsPanel();
  }

  function renderElement(item) {
    const group = document.createElementNS("http://www.w3.org/2000/svg", "g");
    group.dataset.layoutId = item.id;
    group.style.cursor = activeTool === "select" ? "grab" : "crosshair";
    appendGlassShape(group, item, waiterRegistry);

    const title = (item.label || "").trim();
    const waiter = (item.waiter || "").trim();
    const centerX = item.x + item.width / 2;
    const centerY = item.y + item.height / 2;

    if (title) {
      const label = svgEl("text", item.type === "marker"
        ? {
            x: centerX,
            y: centerY - (waiter ? 6 : 0),
            "text-anchor": "middle",
            "dominant-baseline": "middle",
            fill: LAYOUT_THEME.muted,
            "font-size": "12",
            "font-weight": "600",
          }
        : tableLabelAttrs({
            x: centerX,
            y: centerY - (waiter ? 6 : 0),
            "font-size": "13",
            "font-weight": "700",
          }));
      label.textContent = title;
      group.append(label);
    }

    if (waiter && item.type !== "marker") {
      const waiterLabel = svgEl("text", tableLabelAttrs({
        x: centerX,
        y: centerY + 12,
        fill: LAYOUT_THEME.ink,
        "font-size": "11",
        "font-weight": "600",
      }));
      waiterLabel.textContent = waiter;
      group.append(waiterLabel);
    }

    if ((item.type === "table_round" || item.type === "table_rect") && item.seats) {
      const seats = svgEl("text", {
        "data-layout-export-hide": "1",
        x: item.x + 6,
        y: item.y + 14,
        fill: LAYOUT_THEME.muted,
        "font-size": "10",
      });
      seats.textContent = `${item.seats} lug.`;
      group.append(seats);
    }

    group.addEventListener("pointerdown", (event) => {
      if (activeTool !== "select") return;
      event.stopPropagation();
      const already = selectedIds.has(item.id);
      if (!already) {
        selectedIds.clear();
        selectedIds.add(item.id);
      }
      const selected = selectedElements();
      const origins = {};
      selected.forEach((element) => {
        origins[element.id] = { x: element.x, y: element.y };
      });
      dragState = {
        id: item.id,
        ids: selected.map((element) => element.id),
        startX: event.clientX,
        startY: event.clientY,
        originX: item.x,
        originY: item.y,
        origins,
        pendingOffsetX: 0,
        pendingOffsetY: 0,
        moved: false,
        toggleOnClick: false,
      };
      renderSelection();
      syncPropsPanel();
      group.setPointerCapture(event.pointerId);
    });

    group.addEventListener("pointermove", handlePointerMove);
    group.addEventListener("pointerup", finishDrag);
    group.addEventListener("pointercancel", finishDrag);

    return group;
  }

  function scheduleLayoutDraftSave() {
    if (!draftReady || !layoutDraftKey || !window.BuffetFlowOffline?.saveLayoutDraft) return;
    clearTimeout(draftTimer);
    draftTimer = setTimeout(async () => {
      persistHiddenInput();
      const draft = {
        layout_json: hiddenInput?.value || JSON.stringify(state),
        base_version: Number(saveForm?.dataset.version || 0),
      };
      if (mode === "standalone" && metaForm) {
        draft.name = metaForm.querySelector('[name="name"]')?.value || "";
        draft.venue = metaForm.querySelector('[name="venue"]')?.value || "";
        draft.guest_count = metaForm.querySelector('[name="guest_count"]')?.value || "0";
        draft.waiter_count = metaForm.querySelector('[name="waiter_count"]')?.value || "0";
        draft.waiter_names_json = JSON.stringify(configuredWaiters.filter(Boolean));
      }
      await window.BuffetFlowOffline.saveLayoutDraft(layoutDraftKey, draft);
    }, 1500);
  }

  function applyLayoutDraft(draft) {
    if (!draft?.layout_json) return false;
    const parsed = parseLayoutState(draft.layout_json);
    if (!parsed?.elements) return false;
    state = parsed;
    if (mode === "standalone") {
      configuredWaiters = parseWaiterNames(draft.waiter_names_json || JSON.stringify(parsed.waiters || []));
      if (metaForm) {
        if (draft.name != null) metaForm.querySelector('[name="name"]').value = draft.name;
        if (draft.venue != null) metaForm.querySelector('[name="venue"]').value = draft.venue;
        if (draft.guest_count != null) metaForm.querySelector('[name="guest_count"]').value = draft.guest_count;
        if (draft.waiter_count != null && waiterCountInput) waiterCountInput.value = draft.waiter_count;
      }
      state.waiters = [...configuredWaiters];
    }
    registerWaiterColors();
    return true;
  }

  async function restoreLayoutDraftIfNeeded() {
    if (!layoutDraftKey || !window.BuffetFlowOffline?.loadLayoutDraft) return;
    const draft = await window.BuffetFlowOffline.loadLayoutDraft(layoutDraftKey);
    if (!draft?.layout_json) return;
    const shouldRestore = !navigator.onLine || (draft.updated_at && draft.updated_at > (saveForm?.dataset.loadedAt || ""));
    if (!shouldRestore && navigator.onLine) return;
    if (navigator.onLine && !window.confirm("Encontramos um rascunho local deste layout. Deseja restaurá-lo?")) return;
    if (applyLayoutDraft(draft)) {
      window.BuffetFlowOffline.showLayoutOfflineNotice?.("Rascunho local restaurado.");
    }
  }

  function layoutHistorySnapshot() {
    state.waiters = configuredWaiters.filter(Boolean);
    return JSON.stringify(state);
  }

  function updateHistoryButtons() {
    editor.querySelectorAll('[data-layout-history="undo"]').forEach((button) => {
      button.disabled = undoStack.length === 0;
      button.setAttribute("aria-disabled", button.disabled ? "true" : "false");
    });
    editor.querySelectorAll('[data-layout-history="redo"]').forEach((button) => {
      button.disabled = redoStack.length === 0;
      button.setAttribute("aria-disabled", button.disabled ? "true" : "false");
    });
  }

  function captureLayoutHistory() {
    const next = layoutHistorySnapshot();
    if (applyingHistory) {
      historyCurrent = next;
      updateHistoryButtons();
      return;
    }
    if (next === historyCurrent) return;
    undoStack.push(historyCurrent);
    if (undoStack.length > LAYOUT_HISTORY_LIMIT) undoStack.shift();
    redoStack = [];
    historyCurrent = next;
    updateHistoryButtons();
  }

  function restoreLayoutHistory(snapshot) {
    applyingHistory = true;
    try {
      state = parseLayoutState(snapshot);
      configuredWaiters = [...state.waiters];
      selectedIds.clear();
      activeTool = "select";
      dragState = null;
      resizeState = null;
      marqueeState = null;
      pendingRow = null;
      rowPreview = null;
      rowDragState = null;
      clearPlacementHint();
      closeSheet();
      syncWaiterCountInputs(configuredWaiters.length);
      registerWaiterColors();
      refreshWaiterSelect();
      renderWaiterNameGrid();
      historyCurrent = JSON.stringify(state);
      render();
    } finally {
      applyingHistory = false;
      updateHistoryButtons();
    }
  }

  function undoLayout() {
    if (!undoStack.length) return;
    const snapshot = undoStack.pop();
    redoStack.push(historyCurrent);
    if (redoStack.length > LAYOUT_HISTORY_LIMIT) redoStack.shift();
    restoreLayoutHistory(snapshot);
  }

  function redoLayout() {
    if (!redoStack.length) return;
    const snapshot = redoStack.pop();
    undoStack.push(historyCurrent);
    if (undoStack.length > LAYOUT_HISTORY_LIMIT) undoStack.shift();
    restoreLayoutHistory(snapshot);
  }

  function render() {
    captureLayoutHistory();
    layers.tables.replaceChildren();
    layers.markers.replaceChildren();
    sortedElements().forEach((item) => {
      const node = renderElement(item);
      if (item.type === "marker") layers.markers.append(node);
      else layers.tables.append(node);
    });
    renderSelection();
    renderRowPreview();
    updateStats();
    updateLegend();
    persistHiddenInput();
    syncPropsPanel();
    syncToolButtons();
    scheduleLayoutDraftSave();
  }

  function syncToolButtons() {
    editor.querySelectorAll("[data-layout-tool]").forEach((button) => {
      button.classList.toggle("active", button.dataset.layoutTool === activeTool);
    });
  }

  function syncPlacementWaiterUI() {
    const show = isTablePlacementTool();
    editor.querySelectorAll("[data-layout-place-waiter-wrap]").forEach((node) => {
      node.hidden = !show;
    });
    const bar = editor.querySelector("[data-layout-place-bar]");
    if (bar) bar.hidden = !show || !isMobileLayout();
    editor.querySelectorAll("[data-layout-place-waiter]").forEach((select) => fillWaiterSelect(select));
  }

  function bindPlacementWaiter() {
    editor.querySelectorAll("[data-layout-place-waiter]").forEach((select) => {
      if (select.dataset.bound === "true") return;
      select.dataset.bound = "true";
      select.addEventListener("change", () => {
        placementWaiter = select.value || "";
        editor.querySelectorAll("[data-layout-place-waiter]").forEach((other) => {
          if (other !== select) other.value = placementWaiter;
        });
      });
    });
  }

  function setActiveTool(tool) {
    activeTool = tool;
    if (tool !== "place_row") {
      pendingRow = null;
      rowPreview = null;
      rowDragState = null;
      clearPlacementHint();
    }
    syncToolButtons();
    syncPlacementWaiterUI();
    renderRowPreview();
  }

  function selectElement(id) {
    if (id == null) {
      selectedIds.clear();
    } else {
      selectedIds.clear();
      selectedIds.add(id);
    }
    if (id && !isMobileLayout() && fullscreenActive) closeSheet();
    renderSelection();
    syncPropsPanel();
  }

  function findElementGroup(id) {
    return editor.querySelector(`[data-layout-id="${id}"]`);
  }

  function finishDrag() {
    if (!dragState) return;
    const { id, ids, moved, pendingOffsetX, pendingOffsetY, origins, toggleOnClick } = dragState;
    (ids || [id]).forEach((itemId) => {
      const group = findElementGroup(itemId);
      if (group) group.removeAttribute("transform");
    });
    if (moved) {
      (ids || [id]).forEach((itemId) => {
        const element = state.elements.find((item) => item.id === itemId);
        const origin = origins?.[itemId];
        if (!element || !origin) return;
        element.x = origin.x + pendingOffsetX;
        element.y = origin.y + pendingOffsetY;
      });
      render();
    } else if (toggleOnClick) {
      selectedIds.delete(id);
      renderSelection();
      syncPropsPanel();
    } else {
      renderSelection();
    }
    dragState = null;
  }

  function handlePointerMove(event) {
    if (!dragState) return;
    const dx = event.clientX - dragState.startX;
    const dy = event.clientY - dragState.startY;
    if (!dragState.moved && Math.hypot(dx, dy) >= 8) {
      dragState.moved = true;
      closeSheet();
    }
    const matrix = svg.getScreenCTM();
    if (!matrix) return;
    const element = state.elements.find((item) => item.id === dragState.id);
    if (!element) return;
    const snapped = snapPosition(dragState.originX + dx / matrix.a, dragState.originY + dy / matrix.d, element.width, element.height);
    const pendingOffsetX = snapped.x - dragState.originX;
    const pendingOffsetY = snapped.y - dragState.originY;
    dragState.pendingOffsetX = pendingOffsetX;
    dragState.pendingOffsetY = pendingOffsetY;
    (dragState.ids || [dragState.id]).forEach((itemId) => {
      const group = findElementGroup(itemId);
      if (group) group.setAttribute("transform", `translate(${pendingOffsetX} ${pendingOffsetY})`);
    });
    renderSelection(selectedElements().map((item) => ({
      ...item,
      x: item.x + pendingOffsetX,
      y: item.y + pendingOffsetY,
    })));
  }

  function populatePropsForm(targetForm = propsForm) {
    if (!targetForm) return;
    const items = selectedElements();
    if (!items.length) return;
    const element = items[items.length - 1];
    const multi = items.length > 1;
    const labelInput = targetForm.querySelector('[data-layout-prop="label"]');
    if (labelInput) {
      labelInput.value = multi ? "" : (element.label || "");
      const labelField = labelInput.closest("[data-layout-name-field]");
      if (labelField) {
        const caption = labelField.querySelector("[data-layout-name-caption]");
        if (caption) caption.textContent = element.type === "marker" ? "Identificação (opcional)" : "Nome / número";
        labelInput.placeholder = element.type === "marker" ? "Ex.: Bar, pista, buffet..." : "Ex.: Mesa 12";
      }
    }
    const select = targetForm.querySelector('[data-layout-prop="waiter"]');
    refreshWaiterSelect(select);
    if (select) {
      const waiter = sharedValue(items, (item) => item.waiter || "");
      if (waiter.mixed) {
        appendDisabledOption(select, "__mixed__", "Vários valores");
        select.value = "__mixed__";
      } else {
        select.value = waiter.value;
      }
    }
    const color = sharedValue(items, (item) => item.color || "");
    const colorInput = targetForm.querySelector('[data-layout-prop="color"]');
    if (colorInput) colorInput.value = color.mixed ? "" : color.value;
    syncColorPickerUI(targetForm, color.mixed ? { ...element, color: "" } : element, waiterRegistry);
    const seatsInput = targetForm.querySelector('[data-layout-prop="seats"]');
    if (seatsInput) {
      const seats = sharedValue(items, (item) => String(item.seats || 8));
      seatsInput.value = seats.mixed ? "" : seats.value;
      seatsInput.placeholder = seats.mixed ? "Vários" : "";
    }
    const sizeSelect = targetForm.querySelector('[data-layout-prop="tableSize"]');
    if (sizeSelect) {
      const size = sharedValue(items, (item) => inferTableSize(item));
      if (size.mixed) {
        appendDisabledOption(sizeSelect, "__mixed__", "Vários valores");
        sizeSelect.value = "__mixed__";
      } else {
        sizeSelect.value = size.value;
      }
    }
    const widthInput = targetForm.querySelector('[data-layout-prop="width"]');
    const heightInput = targetForm.querySelector('[data-layout-prop="height"]');
    const width = sharedValue(items, (item) => String(Math.round(item.width)));
    const height = sharedValue(items, (item) => String(Math.round(item.height)));
    if (widthInput) {
      widthInput.value = width.mixed ? "" : width.value;
      widthInput.placeholder = width.mixed ? "Vários" : "";
    }
    if (heightInput) {
      heightInput.value = height.mixed ? "" : height.value;
      heightInput.placeholder = height.mixed ? "Vários" : "";
    }
    updatePropsFieldVisibility(targetForm, items);
  }

  function updatePropsFieldVisibility(targetForm, elements) {
    const items = Array.isArray(elements) ? elements : elements ? [elements] : [];
    if (!targetForm || !items.length) return;
    const multi = items.length > 1;
    const allTables = items.every(isTableElement);
    const allMarkers = items.every((item) => item.type === "marker");
    const anyWaiter = items.some((item) => (item.waiter || "").trim());
    targetForm.querySelectorAll("[data-layout-name-field]").forEach((node) => {
      node.hidden = multi;
    });
    targetForm.querySelectorAll("[data-layout-table-field]").forEach((node) => {
      node.hidden = !allTables;
    });
    targetForm.querySelectorAll("[data-layout-color-field]").forEach((node) => {
      node.hidden = !allTables || anyWaiter;
    });
    targetForm.querySelectorAll("[data-layout-marker-field]").forEach((node) => {
      node.hidden = !allMarkers;
    });
  }

  function isPropsFormEditing() {
    const active = document.activeElement;
    if (!active?.matches("input, select, textarea")) return false;
    return Boolean(active.closest("[data-layout-props-form]"));
  }

  function syncPropsPanel() {
    const selectedTableCount = selectedElements().filter(isTableElement).length;
    editor.querySelectorAll("[data-layout-table-action]").forEach((button) => {
      button.disabled = selectedTableCount === 0;
      button.setAttribute("aria-disabled", button.disabled ? "true" : "false");
    });
    if (propsPanel) propsPanel.hidden = !selectedIds.size || isMobileLayout();
    if (isPropsFormEditing()) return;
    populatePropsForm(propsForm);
    if (floatSheet && !floatSheet.hidden && floatSheet.dataset.open === "props" && isMobileLayout()) {
      openSheet("props");
    }
  }

  function applyProp(name, value) {
    if (value === "__mixed__") return;
    const items = selectedElements();
    if (!items.length) return;
    items.forEach((element) => {
      if (name === "label") {
        if (items.length > 1) return;
        element.label = value;
        return;
      }
      if (name === "seats") {
        if (!isTableElement(element) || value === "") return;
        element.seats = Math.max(1, Number(value) || 8);
        return;
      }
      if (name === "tableSize") {
        if (!isTableElement(element)) return;
        applyTableSize(element, value);
        return;
      }
      if (name === "width" && element.type === "marker") {
        if (value === "") return;
        element.width = snapSize(Number(value) || element.width);
        return;
      }
      if (name === "height" && element.type === "marker") {
        if (value === "") return;
        element.height = snapSize(Number(value) || element.height);
        return;
      }
      if (name === "waiter") {
        if (!isTableElement(element)) return;
        element.waiter = value;
        element.color = "";
        return;
      }
      if (name === "color") {
        if ((element.waiter || "").trim()) return;
        element.color = value;
        return;
      }
      element[name] = value;
    });
    render();
  }

  function createTableElement(type, x, y, labelNumber, size = "medium", waiter = placementWaiter) {
    const dims = tableDimensions(type, size);
    const width = dims.width;
    const height = dims.height;
    const snapped = snapPosition(x, y, width, height);
    return {
      id: createElementId(),
      type,
      x: snapped.x,
      y: snapped.y,
      width,
      height,
      tableSize: size,
      label: `Mesa ${labelNumber}`,
      waiter: (waiter || "").trim(),
      color: "",
      seats: 8,
      zIndex: state.elements.length + 1,
    };
  }

  function commitTableRow(startX, startY, config) {
    const labelNumber = tableCount(state.elements);
    const specs = buildRowTableSpecs(startX, startY, config, labelNumber);
    const waiter = (config.waiter || "").trim();
    const created = specs.map((spec, index) => {
      const table = createTableElement(spec.type, spec.x, spec.y, labelNumber + index + 1, "medium", waiter);
      table.x = spec.x;
      table.y = spec.y;
      table.zIndex = state.elements.length + index + 1;
      return table;
    });
    state.elements.push(...created);
    if (created.length) selectedIds = new Set(created.map((table) => table.id));
  }

  function readRowFormValues(form) {
    if (!form) return null;
    return {
      count: form.querySelector('[name="count"]')?.value,
      type: form.querySelector('[name="type"]')?.value,
      direction: form.querySelector('[name="direction"]')?.value,
      waiter: form.querySelector('[name="waiter"]')?.value,
    };
  }

  function bindRowForm(form) {
    if (!form || form.dataset.bound === "true") return;
    form.dataset.bound = "true";
    refreshWaiterSelect(form.querySelector('[name="waiter"]'));
    form.addEventListener("submit", (event) => {
      event.preventDefault();
      const values = readRowFormValues(form);
      if (!values) return;
      beginRowPlacement(values);
    });
  }

  function openDesktopEditPanel() {
    closeSheet();
    if (!selectedIds.size) {
      showSheetNotice("Selecione um item", "Toque em uma mesa ou área para selecionar. No computador, clique e arraste no desenho para selecionar várias.");
      return;
    }
    if (propsPanel) {
      propsPanel.hidden = false;
      propsPanel.scrollIntoView({ block: "nearest", behavior: "smooth" });
    }
    populatePropsForm(propsForm);
  }

  function openRowSheet() {
    if (!floatSheet || !floatSheetBody) return;
    const rect = viewportRect();
    floatSheet.style.setProperty("--layout-modal-center-x", `${rect.left + rect.width / 2}px`);
    floatSheet.style.setProperty("--layout-modal-center-y", `${rect.top + rect.height / 2}px`);
    floatSheet.hidden = false;
    floatSheet.dataset.open = "row";
    document.body.classList.add("layout-sheet-open");
    if (floatSheetTitle) floatSheetTitle.textContent = "Fileira de mesas";
    floatSheetBody.replaceChildren();
    const form = document.createElement("form");
    form.className = "layout-row-form";
    form.innerHTML = `
      <label>Quantidade de mesas<input type="number" min="1" max="30" name="count" value="5" required></label>
      <label>Tipo de mesa
        <select name="type">
          <option value="table_round">Redonda</option>
          <option value="table_rect">Retangular</option>
        </select>
      </label>
      <label>Direção da fileira
        <select name="direction">
          <option value="horizontal">Horizontal</option>
          <option value="vertical">Vertical</option>
        </select>
      </label>
      <label>Garçom responsável (opcional)
        <select name="waiter"><option value="">Sem garçom</option></select>
      </label>
      <p class="layout-sheet-hint">A fileira será criada no centro do desenho. Depois, você poderá arrastá-la antes de confirmar.</p>
      <button class="button primary full" type="submit">Preparar fileira</button>
    `;
    bindRowForm(form);
    floatSheetBody.append(form);
    window.requestAnimationFrame(() => form.querySelector('[name="count"]')?.focus());
  }

  function addElement(type, point) {
    const tableCountSoFar = tableCount(state.elements);
    if (type === "table_round" || type === "table_rect") {
      const width = type === "table_round" ? TABLE_ROUND_SIZE : TABLE_RECT_WIDTH;
      const height = type === "table_round" ? TABLE_ROUND_SIZE : TABLE_RECT_HEIGHT;
      const snapped = snapPosition(point.x - width / 2, point.y - height / 2, width, height);
      const table = createTableElement(type, snapped.x, snapped.y, tableCountSoFar + 1);
      state.elements.push(table);
      setActiveTool("select");
      selectElement(table.id);
      render();
      return;
    }
    if (type !== "marker") return;
    const base = {
      id: createElementId(),
      type: "marker",
      x: 0,
      y: 0,
      width: GRID_SIZE * 3,
      height: GRID_SIZE * 2,
      label: "",
      waiter: "",
      color: "",
      seats: 8,
      zIndex: 10,
    };
    const snapped = snapPosition(point.x - base.width / 2, point.y - base.height / 2, base.width, base.height);
    base.x = snapped.x;
    base.y = snapped.y;
    state.elements.push(base);
    selectElement(base.id);
    setActiveTool("select");
    render();
  }

  function deleteSelected() {
    if (!selectedIds.size) return;
    state.elements = state.elements.filter((item) => !selectedIds.has(item.id));
    selectedIds.clear();
    if (propsPanel) propsPanel.hidden = true;
    closeSheet();
    render();
  }

  function duplicateSelected() {
    const items = selectedElements();
    if (!items.length) return;
    const copies = items.map((element, index) => {
      const copy = { ...element, id: createElementId(), x: element.x + GRID_SIZE, y: element.y + GRID_SIZE, zIndex: state.elements.length + index + 1 };
      const snapped = snapPosition(copy.x, copy.y, copy.width, copy.height);
      copy.x = snapped.x;
      copy.y = snapped.y;
      return copy;
    });
    state.elements.push(...copies);
    selectedIds = new Set(copies.map((copy) => copy.id));
    render();
  }

  function deleteSelectedTables() {
    const tableIds = new Set(selectedElements().filter(isTableElement).map((table) => table.id));
    if (!tableIds.size) return;
    state.elements = state.elements.filter((item) => !tableIds.has(item.id));
    tableIds.forEach((id) => selectedIds.delete(id));
    if (!selectedIds.size && propsPanel) propsPanel.hidden = true;
    render();
  }

  function duplicateSelectedTables() {
    const tables = selectedElements().filter(isTableElement);
    if (!tables.length) return;
    const copies = tables.map((table, index) => {
      const copy = { ...table, id: createElementId(), x: table.x + GRID_SIZE, y: table.y + GRID_SIZE, zIndex: state.elements.length + index + 1 };
      const snapped = snapPosition(copy.x, copy.y, copy.width, copy.height);
      copy.x = snapped.x;
      copy.y = snapped.y;
      return copy;
    });
    state.elements.push(...copies);
    selectedIds = new Set(copies.map((copy) => copy.id));
    render();
  }

  function openSetupSheet() {
    if (!floatSheet || !floatSheetBody) return;
    floatSheet.hidden = false;
    floatSheet.dataset.open = "setup";
    document.body.classList.add("layout-sheet-open");
    if (floatSheetTitle) floatSheetTitle.textContent = "Layout e garçons";
    floatSheetBody.replaceChildren();
    if (metaForm) {
      const clone = metaForm.cloneNode(true);
      clone.classList.remove("layout-desktop-meta");
      const syncFromClone = (input) => {
        const original = metaForm.querySelector(`[name="${input.name}"]`);
        if (original) original.value = input.value;
        syncStandaloneHiddenFields();
        updateStats();
        if (editor.dataset.exportTitle !== undefined && (input.name === "name" || input.name === "venue")) {
          syncLayoutNameFromVenue(clone);
        }
      };
      clone.querySelectorAll("input").forEach((input) => {
        input.addEventListener("input", () => syncFromClone(input));
        input.addEventListener("change", () => syncFromClone(input));
      });
      bindVenueSearch(clone);
      floatSheetBody.append(clone);
    }
    const roster = editor.querySelector("[data-layout-waiter-roster]");
    if (roster) {
      const rosterClone = roster.cloneNode(true);
      rosterClone.classList.remove("layout-desktop-meta");
      floatSheetBody.append(rosterClone);
      renderWaiterNameGrid();
    }
    const divisionClone = editor.querySelector("[data-layout-division]")?.cloneNode(true);
    if (divisionClone) {
      bindDivisionControls(divisionClone);
      refreshDivisionPanel(divisionClone);
      floatSheetBody.append(divisionClone);
    }
    const done = document.createElement("button");
    done.type = "button";
    done.className = "button primary full";
    done.textContent = "Continuar desenhando";
    done.addEventListener("click", () => {
      if (!requireVenueToContinue()) return;
      closeSheet();
    });
    floatSheetBody.append(done);
  }

  function openDivisionSheet() {
    if (!floatSheet || !floatSheetBody) return;
    floatSheet.hidden = false;
    floatSheet.dataset.open = "division";
    document.body.classList.add("layout-sheet-open");
    if (floatSheetTitle) floatSheetTitle.textContent = "Divisão de garçons";
    floatSheetBody.replaceChildren();
    const panel = editor.querySelector("[data-layout-division]")?.cloneNode(true);
    if (panel) {
      panel.classList.remove("layout-desktop-stats");
      bindDivisionControls(panel);
      refreshDivisionPanel(panel);
      floatSheetBody.append(panel);
    }
    const apply = floatSheetBody.querySelector("[data-layout-apply-division]");
    if (apply) {
      apply.addEventListener("click", () => {
        applySuggestedDivision();
        closeSheet();
      });
    }
  }

  function openSheet(kind) {
    if (kind === "setup") {
      openSetupSheet();
      return;
    }
    if (kind === "division") {
      openDivisionSheet();
      return;
    }
    if (kind === "row") {
      openRowSheet();
      return;
    }
    if (kind === "props") {
      if (!isMobileLayout()) {
        openDesktopEditPanel();
        return;
      }
      if (!floatSheet || !floatSheetBody) return;
      if (!selectedIds.size) {
        showSheetNotice("Selecione um item", "Toque em uma mesa ou área para selecionar. No computador, clique e arraste no desenho para selecionar várias.");
        return;
      }
      floatSheet.hidden = false;
      floatSheet.dataset.open = kind;
      document.body.classList.add("layout-sheet-open");
      floatSheetBody.replaceChildren();
      if (floatSheetTitle) floatSheetTitle.textContent = selectedIds.size > 1 ? `Editar ${selectedIds.size} itens` : "Editar item";
      if (propsForm) {
        const clone = propsForm.cloneNode(true);
        bindPropsForm(clone);
        floatSheetBody.append(clone);
        populatePropsForm(clone);
      }
      if (legendList?.parentElement) {
        const legendClone = legendList.parentElement.cloneNode(true);
        floatSheetBody.append(legendClone);
      }
      return;
    }
    if (!floatSheet || !floatSheetBody) return;
    floatSheet.hidden = false;
    floatSheet.dataset.open = kind;
    document.body.classList.add("layout-sheet-open");
    floatSheetBody.replaceChildren();

    if (floatSheetTitle) floatSheetTitle.textContent = "Opções";
    const menu = document.createElement("div");
    menu.className = "layout-sheet-menu";
    if (mode === "event") {
      const standaloneLink = document.createElement("a");
      standaloneLink.className = "button secondary full";
      standaloneLink.href = "/layouts/new";
      standaloneLink.textContent = "Criar layout avulso (sem evento)";
      menu.append(standaloneLink);
    }
    if (mode === "standalone") {
      const setup = document.createElement("button");
      setup.type = "button";
      setup.className = "button secondary full";
      setup.textContent = "Dados do layout e garçons";
      setup.addEventListener("click", () => openSheet("setup"));
      menu.append(setup);
    }
    const division = document.createElement("button");
    division.type = "button";
    division.className = "button secondary full";
    division.textContent = "Divisão de garçons";
    division.addEventListener("click", () => openSheet("division"));
    menu.append(division);
    [
      ["row", "Adicionar fileira de mesas"],
      ["duplicate", "Duplicar seleção"],
      ["delete", "Remover seleção"],
      ["png", "Exportar PNG"],
      ["pdf", "Exportar PDF"],
      ["fullscreen", fullscreenActive ? "Sair da tela cheia" : "Entrar em tela cheia"],
    ].forEach(([action, label]) => {
      const button = document.createElement("button");
      button.type = "button";
      button.className = "button secondary full";
      button.textContent = label;
      button.addEventListener("click", () => {
        if (action === "row") openRowSheet();
        if (action === "duplicate") duplicateSelected();
        if (action === "delete") deleteSelected();
        if (action === "png" || action === "pdf") {
          exportLayout(action).catch(() => window.alert("Não foi possível exportar o layout. Tente novamente."));
        }
        if (action === "fullscreen") toggleFullscreen();
        if (action !== "fullscreen") closeSheet();
      });
      menu.append(button);
    });

    floatSheetBody.append(menu);
  }

  let noticeDismissTimer = null;

  function showSheetNotice(title, message) {
    if (!floatSheet || !floatSheetBody) return;
    if (noticeDismissTimer) {
      clearTimeout(noticeDismissTimer);
      noticeDismissTimer = null;
    }
    floatSheet.hidden = false;
    floatSheet.classList.remove("is-leaving");
    floatSheet.classList.add("is-notice");
    floatSheet.dataset.open = "notice";
    document.body.classList.remove("layout-sheet-open");
    if (floatSheetTitle) floatSheetTitle.textContent = title;
    floatSheetBody.replaceChildren();
    const hint = document.createElement("p");
    hint.className = "layout-sheet-hint";
    hint.textContent = message;
    floatSheetBody.append(hint);
    noticeDismissTimer = window.setTimeout(dismissSheetNotice, 2000);
  }

  function dismissSheetNotice() {
    if (!floatSheet?.classList.contains("is-notice")) return;
    if (noticeDismissTimer) {
      clearTimeout(noticeDismissTimer);
      noticeDismissTimer = null;
    }
    floatSheet.classList.add("is-leaving");
    noticeDismissTimer = window.setTimeout(() => {
      floatSheet.hidden = true;
      floatSheet.classList.remove("is-notice", "is-leaving");
      delete floatSheet.dataset.open;
      noticeDismissTimer = null;
    }, 320);
  }

  function closeSheet() {
    if (noticeDismissTimer) {
      clearTimeout(noticeDismissTimer);
      noticeDismissTimer = null;
    }
    if (!floatSheet) return;
    floatSheet.hidden = true;
    floatSheet.classList.remove("is-notice", "is-leaving");
    delete floatSheet.dataset.open;
    floatSheet.style.removeProperty("--layout-modal-center-x");
    floatSheet.style.removeProperty("--layout-modal-center-y");
    document.body.classList.remove("layout-sheet-open");
    if (!isWaiterNameGridEditing()) renderWaiterNameGrid();
  }

  function toggleFullscreen(force) {
    fullscreenActive = typeof force === "boolean" ? force : !fullscreenActive;
    document.body.classList.toggle("layout-fullscreen-active", fullscreenActive);
    editor.classList.toggle("is-fullscreen", fullscreenActive);
    if (floatTop) floatTop.hidden = !fullscreenActive;
    if (!fullscreenActive) closeSheet();
    if (fullscreenActive) {
      viewport?.scrollTo({ top: 0, left: 0 });
    }
    window.requestAnimationFrame(() => {
      window.requestAnimationFrame(() => fitInitialView());
    });
  }

  async function exportLayout(kind) {
    const exportTitle = editor.dataset.exportTitle || "layout";
    const scope = editor.dataset.eventId || editor.dataset.layoutId || "layout";
    const filenameBase = `layout-${exportTitle.replace(/\s+/g, "-").toLowerCase()}-${scope}`;
    const bounds = layoutExportBounds(state);
    const layoutCanvas = await svgToCanvas(svg, bounds);
    const legendEntries = exportWaiterLegendEntries(configuredWaiters, state.elements, waiterRegistry);
    const canvas = composeExportCanvas(layoutCanvas, legendEntries);
    if (kind === "png") {
      const blob = await new Promise((resolve, reject) => {
        canvas.toBlob((result) => {
          if (result) resolve(result);
          else reject(new Error("Falha ao gerar PNG"));
        }, "image/png");
      });
      downloadBlob(blob, `${filenameBase}.png`);
      return;
    }
    const dataUrl = canvas.toDataURL("image/jpeg", 0.92);
    const binary = Uint8Array.from(atob(dataUrl.split(",")[1]), (char) => char.charCodeAt(0));
    downloadBlob(buildPdfFromJpeg(binary, canvas.width, canvas.height + 56, `Layout — ${exportTitle}`), `${filenameBase}.pdf`);
  }

  editor.querySelectorAll("[data-layout-tool]").forEach((button) => {
    button.addEventListener("click", () => {
      setActiveTool(button.dataset.layoutTool);
    });
  });

  editor.querySelectorAll("[data-layout-action]").forEach((button) => {
    button.addEventListener("click", () => {
      if (button.dataset.layoutAction === "delete") deleteSelected();
      if (button.dataset.layoutAction === "duplicate") duplicateSelected();
    });
  });

  editor.querySelectorAll("[data-layout-table-action]").forEach((button) => {
    button.addEventListener("click", () => {
      if (button.dataset.layoutTableAction === "delete") deleteSelectedTables();
      if (button.dataset.layoutTableAction === "duplicate") duplicateSelectedTables();
    });
  });

  function bindPropsForm(targetForm) {
    if (!targetForm) return;
    ensureColorSwatches(targetForm.querySelector("[data-layout-color-swatches]"), (color, container) => {
      const input = targetForm.querySelector('[data-layout-prop="color"]');
      if (input) input.value = color;
      syncColorSwatchSelection(container, color);
      applyProp("color", color);
      syncColorPickerUI(targetForm, selectedElement(), waiterRegistry);
    });
    targetForm.querySelector("[data-layout-color-toggle]")?.addEventListener("click", () => {
      const panel = targetForm.querySelector("[data-layout-color-panel]");
      setColorPickerOpen(targetForm, Boolean(panel?.hidden));
    });
    targetForm.querySelector("[data-layout-color-clear]")?.addEventListener("click", () => {
      const input = targetForm.querySelector('[data-layout-prop="color"]');
      if (input) input.value = "";
      applyProp("color", "");
      syncColorPickerUI(targetForm, selectedElement(), waiterRegistry);
    });
    targetForm.querySelectorAll("[data-layout-prop]").forEach((input) => {
      if (input.dataset.layoutProp === "color") return;
      const handler = () => applyProp(input.dataset.layoutProp, input.value);
      input.addEventListener("input", handler);
      input.addEventListener("change", handler);
    });
  }

  bindPropsForm(propsForm);

  svg.addEventListener("pointerdown", (event) => {
    if (pinchState || event.isPrimary === false) return;
    if (activeTool === "place_row" && pendingRow && rowPreview) {
      beginRowDrag(event);
      return;
    }
    const isBackground = event.target === svg || event.target.hasAttribute("data-layout-bg-fill") || event.target.hasAttribute("data-layout-bg-grid");
    if (activeTool === "select" && isBackground) {
      if (!isMobileLayout()) {
        const world = clientToWorld(event.clientX, event.clientY);
        marqueeState = {
          startX: world.x,
          startY: world.y,
          currentX: world.x,
          currentY: world.y,
          baseline: new Set(),
          keepOnClick: false,
          pointerId: event.pointerId,
        };
        svg.setPointerCapture(event.pointerId);
        renderMarquee();
        return;
      }
      panState = { startX: event.clientX, startY: event.clientY, originX: view.x, originY: view.y };
      svg.setPointerCapture(event.pointerId);
      selectElement(null);
      return;
    }
    if (activeTool === "select") {
      if (isBackground) selectElement(null);
      return;
    }
    addElement(activeTool, clientToWorld(event.clientX, event.clientY));
  });

  svg.addEventListener("pointermove", (event) => {
    if (resizeState) {
      updateResize(event);
      return;
    }
    if (rowDragState) {
      updateRowDrag(event);
      return;
    }
    if (marqueeState) {
      updateMarquee(event);
      return;
    }
    if (panState) {
      view.x = panState.originX - (event.clientX - panState.startX) / view.scale;
      view.y = panState.originY - (event.clientY - panState.startY) / view.scale;
      clampView();
      updateViewBox();
    }
  });

  svg.addEventListener("pointerup", (event) => {
    finishResize(event);
    finishRowDrag(event);
    finishMarquee();
    panState = null;
  });

  svg.addEventListener("pointercancel", (event) => {
    finishResize(event);
    finishRowDrag(event);
    finishMarquee();
    panState = null;
  });

  function touchDistance(first, second) {
    return Math.hypot(first.clientX - second.clientX, first.clientY - second.clientY);
  }

  function touchMidpoint(first, second) {
    return { x: (first.clientX + second.clientX) / 2, y: (first.clientY + second.clientY) / 2 };
  }

  viewport?.addEventListener("touchstart", (event) => {
    if (event.touches.length < 2) return;
    event.preventDefault();
    const [first, second] = event.touches;
    pinchState = {
      distance: Math.max(1, touchDistance(first, second)),
      mid: touchMidpoint(first, second),
      scale: view.scale,
    };
    panState = null;
    dragState = null;
    resizeState = null;
    marqueeState = null;
  }, { passive: false });

  viewport?.addEventListener("touchmove", (event) => {
    if (!pinchState || event.touches.length < 2) return;
    event.preventDefault();
    const [first, second] = event.touches;
    const dist = Math.max(1, touchDistance(first, second));
    const mid = touchMidpoint(first, second);
    setScaleAroundClient(mid.x, mid.y, pinchState.scale * (dist / pinchState.distance));
  }, { passive: false });

  const endPinch = () => {
    pinchState = null;
  };
  viewport?.addEventListener("touchend", endPinch);
  viewport?.addEventListener("touchcancel", endPinch);

  viewport?.addEventListener("wheel", (event) => {
    event.preventDefault();
    const factor = event.deltaY > 0 ? 0.9 : 1.1;
    setScaleAroundClient(event.clientX, event.clientY, view.scale * factor);
  }, { passive: false });

  bindDivisionControls(editor);

  document.addEventListener("keydown", (event) => {
    if (!editor.isConnected) return;
    const active = document.activeElement;
    const isEditingField = active?.matches?.("input, textarea, select, [contenteditable='true']");
    const shortcutKey = event.key.toLowerCase();
    if ((event.ctrlKey || event.metaKey) && !event.altKey && !isEditingField && shortcutKey === "z") {
      event.preventDefault();
      if (event.shiftKey) redoLayout();
      else undoLayout();
      return;
    }
    if ((event.ctrlKey || event.metaKey) && !event.altKey && !isEditingField && shortcutKey === "y") {
      event.preventDefault();
      redoLayout();
      return;
    }
    if (event.key === "Escape" && activeTool === "place_row") {
      cancelRowPreview();
      return;
    }
    if (event.key === "Escape" && fullscreenActive) {
      toggleFullscreen(false);
      return;
    }
    if (event.key === "Delete" || event.key === "Backspace") {
      const tag = document.activeElement?.tagName;
      if (tag === "INPUT" || tag === "TEXTAREA" || tag === "SELECT") return;
      event.preventDefault();
      deleteSelected();
    }
  });

  editor.querySelectorAll("[data-layout-history]").forEach((button) => {
    button.addEventListener("click", () => {
      if (button.dataset.layoutHistory === "undo") undoLayout();
      if (button.dataset.layoutHistory === "redo") redoLayout();
    });
  });
  updateHistoryButtons();

  editor.querySelectorAll("[data-layout-export]").forEach((button) => {
    button.addEventListener("click", async () => {
      button.disabled = true;
      try {
        await exportLayout(button.dataset.layoutExport);
      } catch (_error) {
        window.alert("Não foi possível exportar o layout. Tente novamente.");
      } finally {
        button.disabled = false;
      }
    });
  });

  editor.querySelectorAll("[data-layout-fullscreen], [data-layout-fullscreen-exit]").forEach((button) => {
    button.addEventListener("click", () => toggleFullscreen(button.hasAttribute("data-layout-fullscreen-exit") ? false : undefined));
  });

  editor.querySelectorAll("[data-layout-sheet-open]").forEach((button) => {
    button.addEventListener("click", () => openSheet(button.dataset.layoutSheetOpen));
  });

  editor.querySelectorAll("[data-layout-sheet-close]").forEach((button) => {
    button.addEventListener("click", closeSheet);
  });

  metaForm?.querySelectorAll("input").forEach((input) => {
    input.addEventListener("input", () => {
      if (input.name === "venue" || input.name === "name") syncLayoutNameFromVenue();
      syncStandaloneHiddenFields();
      updateStats();
    });
  });
  bindVenueSearch(metaForm);

  editor.addEventListener("input", (event) => {
    const countInput = event.target.closest("[data-layout-waiter-count]");
    if (!countInput) return;
    rebuildWaiterNamesFromCount(countInput.value);
  });
  editor.addEventListener("click", (event) => {
    const step = event.target.closest("[data-layout-waiter-step]");
    if (!step) return;
    event.preventDefault();
    const input = step.closest(".layout-stepper")?.querySelector("[data-layout-waiter-count]");
    stepWaiterCount(Number(step.dataset.layoutWaiterStep) || 0, input);
  });

  saveForm?.addEventListener("layout-persist", () => {
    persistHiddenInput();
  });

  saveForm?.addEventListener("submit", (event) => {
    persistHiddenInput();
    if (!requireVenueToContinue()) {
      event.preventDefault();
      event.stopPropagation();
      return;
    }
    cleanupLayoutEditorState(editor);
  }, { capture: true });

  saveForm && (saveForm.dataset.loadedAt = new Date().toISOString());

  MOBILE_LAYOUT_QUERY.addEventListener("change", () => {
    if (propsPanel) propsPanel.hidden = !selectedIds.size || isMobileLayout();
    syncPlacementWaiterUI();
    closeSheet();
  });

  bindPlacementWaiter();

  syncConfiguredWaiters({ rebuildGrid: true });
  restoreLayoutDraftIfNeeded().finally(() => {
    draftReady = true;
    render();
  });

  if (viewport && typeof ResizeObserver !== "undefined") {
    const resizeObserver = new ResizeObserver(() => updateViewBox());
    resizeObserver.observe(viewport);
  } else {
    window.addEventListener("resize", updateViewBox);
  }

  const justSaved = new URLSearchParams(window.location.search).has("message");
  if (isMobileLayout() && !justSaved) {
    window.requestAnimationFrame(() => {
      toggleFullscreen(true);
      window.requestAnimationFrame(() => {
        fitInitialView();
        if (mode === "standalone" && editor.dataset.layoutNew === "1") openSetupSheet();
      });
    });
  } else {
    fitInitialView();
  }
}

function cleanupLayoutEditorState(editor) {
  document.body.classList.remove("layout-fullscreen-active", "layout-sheet-open");
  if (editor) editor.classList.remove("is-fullscreen");
  document.querySelector("[data-layout-float-top]")?.setAttribute("hidden", "");
  document.querySelector("[data-layout-sheet]")?.setAttribute("hidden", "");
}

function cleanupAllLayoutEditors() {
  document.body.classList.remove("layout-fullscreen-active", "layout-sheet-open");
  document.querySelectorAll("[data-layout-editor]").forEach((editor) => {
    editor.classList.remove("is-fullscreen");
    editor.querySelector("[data-layout-float-top]")?.setAttribute("hidden", "");
    editor.querySelector("[data-layout-sheet]")?.setAttribute("hidden", "");
    editor.querySelector(".layout-placement-hint")?.setAttribute("hidden", "");
  });
}

document.addEventListener("DOMContentLoaded", () => initializeLayoutEditor(document));
document.addEventListener("htmx:afterSwap", (event) => initializeLayoutEditor(event.target));
window.addEventListener("pagehide", cleanupAllLayoutEditors);
document.addEventListener("click", (event) => {
  const link = event.target.closest("a[href]");
  if (!link || !document.querySelector("[data-layout-editor]")) return;
  const href = link.getAttribute("href");
  if (!href || href.startsWith("#")) return;
  cleanupAllLayoutEditors();
});
