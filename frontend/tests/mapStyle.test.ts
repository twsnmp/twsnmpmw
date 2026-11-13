import { describe, it } from "node:test";
import assert from "node:assert/strict";
import {
  DEFAULT_MAP_STYLE,
  MAP_STYLE_ERROR_MESSAGE,
  MAP_STYLE_PRESETS,
  isPresetActive,
  resolveMapStyle,
  validateMapStyle,
} from "../src/mapStyle.ts";

describe("mapStyle helpers", () => {
  it("exposes expected presets from twsnmpmv plus default reset", () => {
    const ids = MAP_STYLE_PRESETS.map((p) => p.id);
    assert.deepEqual(ids, [
      "gsi-pale",
      "carto-dark",
      "osm-bright-ja",
      "mierune-mono",
      "gsi-std",
      "gsi-photo",
      "carto-light",
      "osm",
      "default-empty",
    ]);

    for (const preset of MAP_STYLE_PRESETS) {
      assert.equal(validateMapStyle(preset.style), null, `preset ${preset.id} should be valid`);
    }
  });

  it("validates empty, URL, relative path, and JSON object styles", () => {
    assert.equal(validateMapStyle(""), null);
    assert.equal(validateMapStyle("   "), null);
    assert.equal(validateMapStyle("https://tile.openstreetmap.jp/styles/osm-bright-ja/style.json"), null);
    assert.equal(validateMapStyle("http://localhost:8080/style.json"), null);
    assert.equal(validateMapStyle("/styles/custom.json"), null);
    assert.equal(validateMapStyle('{"version": 8, "sources": {}, "layers": []}'), null);

    assert.equal(validateMapStyle("{ invalid_json "), MAP_STYLE_ERROR_MESSAGE);
    assert.equal(validateMapStyle("not-a-url"), MAP_STYLE_ERROR_MESSAGE);
    assert.equal(validateMapStyle("ftp://example.com/style.json"), MAP_STYLE_ERROR_MESSAGE);
    assert.equal(validateMapStyle("[]"), MAP_STYLE_ERROR_MESSAGE);
  });

  it("resolves empty, JSON object, and URL styles for MapLibre", () => {
    assert.equal(resolveMapStyle(""), DEFAULT_MAP_STYLE);
    assert.equal(resolveMapStyle("   "), DEFAULT_MAP_STYLE);
    assert.equal(resolveMapStyle(undefined), DEFAULT_MAP_STYLE);

    const gsiPreset = MAP_STYLE_PRESETS.find((p) => p.id === "gsi-pale");
    assert.ok(gsiPreset);
    const parsed = resolveMapStyle(gsiPreset.style);
    assert.equal(typeof parsed, "object");
    assert.equal((parsed as { version: number }).version, 8);

    assert.equal(
      resolveMapStyle("  https://example.com/style.json  "),
      "https://example.com/style.json"
    );

    // Invalid stored JSON safely falls back to default instead of crashing MapLibre.
    assert.equal(resolveMapStyle("{ broken"), DEFAULT_MAP_STYLE);
  });

  it("matches active preset even with different JSON formatting", () => {
    const gsiPreset = MAP_STYLE_PRESETS.find((p) => p.id === "gsi-pale");
    assert.ok(gsiPreset);
    const compactJson = JSON.stringify(JSON.parse(gsiPreset.style));
    assert.equal(isPresetActive(compactJson, gsiPreset.style), true);
    assert.equal(isPresetActive("  ", ""), true);
    assert.equal(isPresetActive(DEFAULT_MAP_STYLE, DEFAULT_MAP_STYLE), true);
    assert.equal(isPresetActive(DEFAULT_MAP_STYLE, gsiPreset.style), false);
  });
});
