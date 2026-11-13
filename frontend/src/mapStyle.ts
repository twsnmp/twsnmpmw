import type * as maplibre from "maplibre-gl";

export const DEFAULT_MAP_STYLE = "https://tile.openstreetmap.jp/styles/osm-bright-ja/style.json";
export const MAP_STYLE_ERROR_MESSAGE = "スタイルは、URLかJSONで指定してください。";

export type MapStylePreset = {
  id: string;
  name: string;
  badge?: string;
  style: string;
};

export const MAP_STYLE_PRESETS: readonly MapStylePreset[] = [
  {
    id: "gsi-pale",
    name: "国土地理院 淡色地図",
    badge: "推奨",
    style: JSON.stringify(
      {
        version: 8,
        sources: {
          "gsi-pale": {
            type: "raster",
            tiles: [
              "https://cyberjapandata.gsi.go.jp/xyz/pale/{z}/{x}/{y}.png",
            ],
            tileSize: 256,
            attribution: "国土地理院 (地理院タイル 淡色地図)",
          },
        },
        layers: [
          {
            id: "gsi-pale",
            type: "raster",
            source: "gsi-pale",
            minzoom: 2,
            maxzoom: 18,
          },
        ],
      },
      null,
      2
    ),
  },
  {
    id: "carto-dark",
    name: "CARTO Dark Matter",
    badge: "ダーク推奨",
    style: JSON.stringify(
      {
        version: 8,
        sources: {
          "carto-dark": {
            type: "raster",
            tiles: [
              "https://a.basemaps.cartocdn.com/rastertiles/dark_all/{z}/{x}/{y}.png",
              "https://b.basemaps.cartocdn.com/rastertiles/dark_all/{z}/{x}/{y}.png",
              "https://c.basemaps.cartocdn.com/rastertiles/dark_all/{z}/{x}/{y}.png",
            ],
            tileSize: 256,
            attribution: "© OpenStreetMap contributors, © CARTO",
          },
        },
        layers: [
          {
            id: "carto-dark",
            type: "raster",
            source: "carto-dark",
            minzoom: 0,
            maxzoom: 19,
          },
        ],
      },
      null,
      2
    ),
  },
  {
    id: "osm-bright-ja",
    name: "OSM Bright (Japan)",
    badge: "デフォルト",
    style: DEFAULT_MAP_STYLE,
  },
  {
    id: "mierune-mono",
    name: "MIERUNE Mono",
    badge: "モノクロ",
    style: JSON.stringify(
      {
        version: 8,
        sources: {
          MIERUNEMAP: {
            type: "raster",
            tiles: [
              "https://tile.mierune.co.jp/mierune_mono/{z}/{x}/{y}.png",
            ],
            tileSize: 256,
            attribution:
              "Maptiles by MIERUNE, under CC BY. Data by OpenStreetMap contributors, under ODbL.",
          },
        },
        layers: [
          {
            id: "MIERUNEMAP",
            type: "raster",
            source: "MIERUNEMAP",
            minzoom: 0,
            maxzoom: 18,
          },
        ],
      },
      null,
      2
    ),
  },
  {
    id: "gsi-std",
    name: "国土地理院 標準地図",
    style: JSON.stringify(
      {
        version: 8,
        sources: {
          "gsi-std": {
            type: "raster",
            tiles: [
              "https://cyberjapandata.gsi.go.jp/xyz/std/{z}/{x}/{y}.png",
            ],
            tileSize: 256,
            attribution: "国土地理院 (地理院タイル 標準地図)",
          },
        },
        layers: [
          {
            id: "gsi-std",
            type: "raster",
            source: "gsi-std",
            minzoom: 2,
            maxzoom: 18,
          },
        ],
      },
      null,
      2
    ),
  },
  {
    id: "gsi-photo",
    name: "国土地理院 写真",
    badge: "航空写真",
    style: JSON.stringify(
      {
        version: 8,
        sources: {
          "gsi-photo": {
            type: "raster",
            tiles: [
              "https://cyberjapandata.gsi.go.jp/xyz/seamlessphoto/{z}/{x}/{y}.jpg",
            ],
            tileSize: 256,
            attribution: "国土地理院 (地理院タイル 写真)",
          },
        },
        layers: [
          {
            id: "gsi-photo",
            type: "raster",
            source: "gsi-photo",
            minzoom: 2,
            maxzoom: 18,
          },
        ],
      },
      null,
      2
    ),
  },
  {
    id: "carto-light",
    name: "CARTO Positron",
    badge: "ライト",
    style: JSON.stringify(
      {
        version: 8,
        sources: {
          "carto-light": {
            type: "raster",
            tiles: [
              "https://a.basemaps.cartocdn.com/rastertiles/light_all/{z}/{x}/{y}.png",
              "https://b.basemaps.cartocdn.com/rastertiles/light_all/{z}/{x}/{y}.png",
              "https://c.basemaps.cartocdn.com/rastertiles/light_all/{z}/{x}/{y}.png",
            ],
            tileSize: 256,
            attribution: "© OpenStreetMap contributors, © CARTO",
          },
        },
        layers: [
          {
            id: "carto-light",
            type: "raster",
            source: "carto-light",
            minzoom: 0,
            maxzoom: 19,
          },
        ],
      },
      null,
      2
    ),
  },
  {
    id: "osm",
    name: "OpenStreetMap 標準",
    style: JSON.stringify(
      {
        version: 8,
        sources: {
          osm: {
            type: "raster",
            tiles: ["https://tile.openstreetmap.org/{z}/{x}/{y}.png"],
            tileSize: 256,
            attribution: "© OpenStreetMap contributors",
          },
        },
        layers: [
          {
            id: "osm",
            type: "raster",
            source: "osm",
            minzoom: 0,
            maxzoom: 19,
          },
        ],
      },
      null,
      2
    ),
  },
  {
    id: "default-empty",
    name: "デフォルトに戻す",
    badge: "空欄",
    style: "",
  },
];

/**
 * Validates a user-entered map style string (empty, http(s)/relative URL, or JSON object).
 * Returns an error message when invalid, or null when valid.
 */
export function validateMapStyle(raw: string): string | null {
  const trimmed = raw.trim();
  if (!trimmed) {
    return null;
  }
  if (trimmed.startsWith("{")) {
    try {
      const parsed = JSON.parse(trimmed);
      if (parsed && typeof parsed === "object" && !Array.isArray(parsed)) {
        return null;
      }
    } catch {
      return MAP_STYLE_ERROR_MESSAGE;
    }
    return MAP_STYLE_ERROR_MESSAGE;
  }
  if (trimmed.startsWith("/")) {
    return null;
  }
  try {
    const parsedUrl = new URL(trimmed);
    if (parsedUrl.protocol === "http:" || parsedUrl.protocol === "https:") {
      return null;
    }
  } catch {
    // Fall through to validation error.
  }
  return MAP_STYLE_ERROR_MESSAGE;
}

/**
 * Resolves a stored map style string to a MapLibre style URL or StyleSpecification object,
 * falling back to DEFAULT_MAP_STYLE when empty or invalid JSON.
 */
export function resolveMapStyle(raw?: string): string | maplibre.StyleSpecification {
  const trimmed = (raw ?? "").trim();
  if (!trimmed) {
    return DEFAULT_MAP_STYLE;
  }
  if (trimmed.startsWith("{")) {
    try {
      const parsed = JSON.parse(trimmed);
      if (parsed && typeof parsed === "object" && !Array.isArray(parsed)) {
        return parsed as maplibre.StyleSpecification;
      }
    } catch (err) {
      console.error("Failed to parse map style JSON, falling back to default:", err);
      return DEFAULT_MAP_STYLE;
    }
  }
  return trimmed;
}

function normalizeStyleForComparison(raw: string): string {
  const trimmed = raw.trim();
  if (!trimmed) {
    return "";
  }
  if (trimmed.startsWith("{")) {
    try {
      const parsed = JSON.parse(trimmed);
      if (parsed && typeof parsed === "object" && !Array.isArray(parsed)) {
        return JSON.stringify(parsed);
      }
    } catch {
      return trimmed;
    }
  }
  return trimmed;
}

/**
 * Checks whether the current style string matches a given preset style.
 */
export function isPresetActive(currentRaw: string, presetStyle: string): boolean {
  return normalizeStyleForComparison(currentRaw) === normalizeStyleForComparison(presetStyle);
}
