import 'flowbite';
import { Component, For, Show, createEffect, createSignal, on, onCleanup } from 'solid-js';
import { UpdateState, VersionInfo } from "../bindings/github.com/twsnmp/twsnmpmw";
import {
  MAP_STYLE_PRESETS,
  isPresetActive,
  validateMapStyle,
} from './mapStyle';

export type MWConfig = {
  dark: boolean,
  mapStyle: string
}

type Props = {
  close: () => void,
  save: (c: MWConfig) => void,
  config: MWConfig,
  version: VersionInfo,
  updateState: UpdateState,
  checkUpdate: () => Promise<void>,
  applyUpdate: () => Promise<void>,
}

const formatBytes = (bytes: number): string => {
  if (!bytes || bytes <= 0) {
    return "0 B";
  }
  const units = ["B", "KB", "MB", "GB"];
  let idx = 0;
  let value = bytes;
  while (value >= 1024 && idx < units.length - 1) {
    value /= 1024;
    idx++;
  }
  return `${value.toFixed(idx === 0 ? 0 : 1)} ${units[idx]}`;
};

const Config: Component<Props> = (props: Props) => {
  let mapStyleRef: HTMLTextAreaElement | undefined;
  let announceTimer: number | undefined;

  const [dark, setDark] = createSignal(props.config.dark);
  const [mapStyle, setMapStyle] = createSignal(props.config.mapStyle);
  const [styleError, setStyleError] = createSignal<string | null>(null);
  const [hasInteracted, setHasInteracted] = createSignal(false);
  const [announcement, setAnnouncement] = createSignal("");

  createEffect(on(() => props.config, (c) => {
    setDark(c.dark);
    setMapStyle(c.mapStyle);
    setStyleError(null);
    setHasInteracted(false);
    setAnnouncement("");
  }));

  createEffect(on(mapStyle, (val) => {
    if (mapStyleRef && mapStyleRef.value !== val) {
      mapStyleRef.value = val;
    }
  }));

  const clearAnnouncementTimer = () => {
    if (announceTimer !== undefined) {
      window.clearTimeout(announceTimer);
      announceTimer = undefined;
    }
  };

  onCleanup(() => {
    clearAnnouncementTimer();
  });

  const isBusy = () => {
    const st = props.updateState.state;
    return (
      st === "checking" ||
      st === "downloading" ||
      st === "verifying" ||
      st === "installing" ||
      st === "ready"
    );
  };

  const canApply = () => {
    return props.updateState.updateAvailable && !isBusy();
  };

  const applyPreset = (presetStyle: string) => {
    clearAnnouncementTimer();
    setHasInteracted(true);
    setMapStyle(presetStyle);
    setStyleError(null);
    setAnnouncement("");
  };

  const handleStyleInput = (value: string) => {
    setHasInteracted(true);
    setMapStyle(value);
    if (styleError()) {
      const err = validateMapStyle(value);
      if (!err) {
        clearAnnouncementTimer();
        setStyleError(null);
        setAnnouncement("");
      }
    }
  };

  const handleStyleBlur = () => {
    if (!hasInteracted()) {
      return;
    }
    const err = validateMapStyle(mapStyle());
    if (err) {
      setStyleError(err);
      clearAnnouncementTimer();
      setAnnouncement("");
      // Wait briefly so focus announcement on the next element is not interrupted.
      announceTimer = window.setTimeout(() => {
        setAnnouncement(`地図のスタイル: ${err}`);
      }, 500);
    } else {
      clearAnnouncementTimer();
      setStyleError(null);
      setAnnouncement("");
    }
  };

  const handleSubmit = (e: SubmitEvent) => {
    e.preventDefault();
    clearAnnouncementTimer();
    const err = validateMapStyle(mapStyle());
    if (err) {
      setHasInteracted(true);
      setStyleError(err);
      setAnnouncement("");
      mapStyleRef?.focus();
      return;
    }
    props.save({
      dark: dark(),
      mapStyle: mapStyle().trim(),
    });
  };

  return (
    <div class="max-w-2xl mx-auto px-4 mt-8 pb-10">
      <div aria-live="polite" class="visually-hidden">
        {announcement()}
      </div>

      <form onSubmit={handleSubmit} novalidate>
        <div class="mb-5">
          <label for="config-dark" class="inline-flex items-center cursor-pointer">
            <input
              id="config-dark"
              name="dark"
              type="checkbox"
              class="sr-only peer"
              checked={dark()}
              onChange={(e) => setDark(e.currentTarget.checked)}
            />
            <div class="relative w-11 h-6 bg-gray-200 rounded-full peer peer-focus-visible:ring-4 peer-focus-visible:ring-blue-300 dark:peer-focus-visible:ring-blue-800 dark:bg-gray-700 peer-checked:after:translate-x-full rtl:peer-checked:after:-translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-0.5 after:start-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all dark:border-gray-600 peer-checked:bg-blue-600"></div>
            <span class="ms-3 text-sm font-medium text-gray-900 dark:text-gray-300">Darkモード</span>
          </label>
        </div>

        <fieldset class="mb-4">
          <legend class="block mb-2 text-sm font-medium text-gray-900 dark:text-white">
            地図スタイル ショートカット
          </legend>
          <div class="grid grid-cols-2 sm:grid-cols-3 gap-2">
            <For each={MAP_STYLE_PRESETS}>
              {(preset) => {
                const active = () => isPresetActive(mapStyle(), preset.style);
                return (
                  <button
                    type="button"
                    data-testid={`map-style-preset-${preset.id}`}
                    aria-pressed={active()}
                    onClick={() => applyPreset(preset.style)}
                    class={
                      active()
                        ? "min-h-[38px] px-2.5 py-2 text-left text-xs font-medium rounded-lg border border-blue-600 bg-blue-50 text-blue-900 dark:border-blue-400 dark:bg-blue-950/60 dark:text-blue-100 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500 flex items-center justify-between gap-1.5 transition-colors"
                        : "min-h-[38px] px-2.5 py-2 text-left text-xs font-medium rounded-lg border border-gray-200 bg-white text-gray-800 hover:bg-gray-50 hover:border-gray-300 dark:border-gray-600 dark:bg-gray-800 dark:text-gray-200 dark:hover:bg-gray-700 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500 flex items-center justify-between gap-1.5 transition-colors"
                    }
                  >
                    <span class="truncate">{preset.name}</span>
                    <span class="inline-flex items-center gap-1 shrink-0">
                      <Show when={preset.badge}>
                        <span
                          class={
                            active()
                              ? "px-1.5 py-0.5 text-[10px] rounded bg-blue-100 text-blue-800 dark:bg-blue-900/70 dark:text-blue-200"
                              : "px-1.5 py-0.5 text-[10px] rounded bg-gray-100 text-gray-600 dark:bg-gray-700 dark:text-gray-300"
                          }
                        >
                          {preset.badge}
                        </span>
                      </Show>
                      <Show when={active()}>
                        <svg
                          class="w-3.5 h-3.5 text-blue-600 dark:text-blue-400 shrink-0"
                          aria-hidden="true"
                          xmlns="http://www.w3.org/2000/svg"
                          fill="none"
                          viewBox="0 0 24 24"
                        >
                          <path
                            stroke="currentColor"
                            stroke-linecap="round"
                            stroke-linejoin="round"
                            stroke-width="2.5"
                            d="M5 11.917 9.724 16.5 19 7.5"
                          />
                        </svg>
                      </Show>
                    </span>
                  </button>
                );
              }}
            </For>
          </div>
        </fieldset>

        <div class="mb-5">
          <label for="map-style" class="block mb-1 text-sm font-medium text-gray-900 dark:text-white">
            地図のスタイル (URL または JSON)
          </label>
          <p id="map-style-hint" class="mb-2 text-xs text-gray-500 dark:text-gray-400">
            MapLibre スタイルの URL または JSON を指定してください。空欄の場合はデフォルト（OSM Bright）を使用します。
          </p>
          <textarea
            ref={mapStyleRef}
            id="map-style"
            name="mapStyle"
            rows={6}
            value={props.config.mapStyle}
            onInput={(e) => handleStyleInput(e.currentTarget.value)}
            onBlur={handleStyleBlur}
            {...({ autocorrect: "off" } as any)}
            autocapitalize="none"
            spellcheck={false}
            aria-invalid={styleError() ? "true" : "false"}
            aria-describedby={styleError() ? "map-style-hint map-style-error" : "map-style-hint"}
            placeholder="https://... または { ... }"
            class={
              styleError()
                ? "bg-red-50 border border-red-500 text-gray-900 text-xs font-mono rounded-lg focus:ring-red-500 focus:border-red-500 block w-full p-2.5 min-h-[120px] resize-y dark:bg-red-950/30 dark:border-red-500 dark:placeholder-gray-400 dark:text-white dark:focus:ring-red-500 dark:focus:border-red-500"
                : "bg-gray-50 border border-gray-300 text-gray-900 text-xs font-mono rounded-lg focus:ring-blue-500 focus:border-blue-500 block w-full p-2.5 min-h-[120px] resize-y dark:bg-gray-700 dark:border-gray-600 dark:placeholder-gray-400 dark:text-white dark:focus:ring-blue-500 dark:focus:border-blue-500"
            }
          />
          <Show when={styleError()}>
            <div
              id="map-style-error"
              data-testid="map-style-error"
              class="mt-2 p-2.5 text-xs text-red-800 rounded-lg bg-red-50 border border-red-300 dark:bg-red-950/40 dark:border-red-800 dark:text-red-300 flex items-center gap-2"
            >
              <svg
                class="w-4 h-4 shrink-0"
                aria-hidden="true"
                xmlns="http://www.w3.org/2000/svg"
                fill="none"
                viewBox="0 0 24 24"
              >
                <path
                  stroke="currentColor"
                  stroke-linecap="round"
                  stroke-linejoin="round"
                  stroke-width="2"
                  d="M12 13V8m0 8h.01M21 12a9 9 0 1 1-18 0 9 9 0 0 1 18 0Z"
                />
              </svg>
              <span>{styleError()}</span>
            </div>
          </Show>
        </div>

        <button
          type="submit"
          data-testid="save-config-btn"
          class="px-3 py-1.5 text-xs font-medium text-center inline-flex items-center text-white bg-blue-700 rounded-lg hover:bg-blue-800 focus:ring-4 focus:outline-none focus:ring-blue-300 dark:bg-blue-600 dark:hover:bg-blue-700 dark:focus:ring-blue-800"
        >
          <svg class="w-5 h-5 text-white dark:text-white" aria-hidden="true" xmlns="http://www.w3.org/2000/svg" width="24" height="24" fill="none" viewBox="0 0 24 24">
            <path stroke="currentColor" stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 11.917 9.724 16.5 19 7.5" />
          </svg>
          保存
        </button>
        <button
          onClick={props.close}
          type="button"
          data-testid="cancel-config-btn"
          class="ml-2 px-3 py-1.5 text-xs font-medium text-center inline-flex items-center text-white bg-gray-400 rounded-lg hover:bg-gray-500 focus:ring-4 focus:outline-none focus:ring-blue-300 dark:bg-gray-600 dark:hover:bg-gray-700 dark:focus:ring-gray-800"
        >
          <svg class="w-5 h-5 text-white" aria-hidden="true" xmlns="http://www.w3.org/2000/svg" width="24" height="24" fill="none" viewBox="0 0 24 24">
            <path stroke="currentColor" stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18 17.94 6M18 18 6.06 6" />
          </svg>
          キャンセル
        </button>
      </form>

      <div class="mt-8 p-4 bg-white border border-gray-200 rounded-lg shadow-sm dark:bg-gray-800 dark:border-gray-700" data-testid="updater-section">
        <div class="flex items-center justify-between mb-2">
          <h3 class="text-sm font-semibold text-gray-900 dark:text-white">バージョン・アップデート</h3>
          <Show when={props.version.platform}>
            <span class="text-xs font-mono text-gray-500 dark:text-gray-400">
              {props.version.platform}/{props.version.arch}
            </span>
          </Show>
        </div>

        <p class="text-xs text-gray-700 dark:text-gray-300 mb-3">
          現在のバージョン:{" "}
          <span class="font-mono font-semibold text-gray-900 dark:text-white" data-testid="app-version">
            {props.version.display || props.updateState.currentVersion || "-"}
          </span>
        </p>

        <Show when={props.updateState.state === "checking"}>
          <p class="text-xs text-blue-600 dark:text-blue-400 mb-3">アップデートを確認中...</p>
        </Show>

        <Show when={props.updateState.state === "up-to-date"}>
          <p class="text-xs text-green-600 dark:text-green-400 mb-3">最新バージョンです。</p>
        </Show>

        <Show when={props.updateState.updateAvailable && props.updateState.state !== "error"}>
          <div class="mb-3 p-2.5 text-xs bg-blue-50 border border-blue-200 rounded-lg text-blue-900 dark:bg-blue-950/50 dark:border-blue-800 dark:text-blue-200">
            <div class="font-semibold">
              新しいバージョン {props.updateState.latestVersion} が利用可能です
            </div>
            <Show when={props.updateState.releaseNotes}>
              <pre class="mt-1.5 max-h-28 overflow-y-auto whitespace-pre-wrap font-sans text-[11px] text-blue-800 dark:text-blue-300">
                {props.updateState.releaseNotes}
              </pre>
            </Show>
          </div>
        </Show>

        <Show when={props.updateState.state === "downloading"}>
          <div class="mb-3">
            <div class="flex justify-between text-xs text-gray-700 dark:text-gray-300 mb-1">
              <span>ダウンロード中...</span>
              <span>
                {props.updateState.progressPercent}% ({formatBytes(props.updateState.progressWritten)}
                <Show when={props.updateState.progressTotal > 0}>
                  {" "}/ {formatBytes(props.updateState.progressTotal)}
                </Show>
                )
              </span>
            </div>
            <div class="w-full bg-gray-200 rounded-full h-2 dark:bg-gray-700">
              <div
                class="bg-blue-600 h-2 rounded-full transition-all"
                style={{ width: `${props.updateState.progressPercent}%` }}
              />
            </div>
          </div>
        </Show>

        <Show when={props.updateState.state === "verifying"}>
          <p class="text-xs text-blue-600 dark:text-blue-400 mb-3">署名・チェックサムを検証中...</p>
        </Show>

        <Show when={props.updateState.state === "installing" || props.updateState.state === "ready"}>
          <p class="text-xs text-blue-600 dark:text-blue-400 mb-3">アップデートを適用して再起動しています...</p>
        </Show>

        <Show when={props.updateState.state === "error" && props.updateState.error}>
          <div class="mb-3 p-2 text-xs text-red-700 bg-red-50 border border-red-200 rounded-lg dark:bg-red-950/40 dark:border-red-800 dark:text-red-300">
            {props.updateState.error}
          </div>
        </Show>

        <div class="flex flex-wrap gap-2">
          <button
            type="button"
            disabled={isBusy()}
            onClick={() => props.checkUpdate()}
            data-testid="check-update-btn"
            class="px-3 py-1.5 text-xs font-medium text-gray-900 bg-gray-100 border border-gray-300 rounded-lg hover:bg-gray-200 disabled:opacity-50 disabled:cursor-not-allowed dark:bg-gray-700 dark:text-white dark:border-gray-600 dark:hover:bg-gray-600"
          >
            アップデートの確認
          </button>

          <Show when={canApply()}>
            <button
              type="button"
              disabled={isBusy()}
              onClick={() => props.applyUpdate()}
              data-testid="apply-update-btn"
              class="px-3 py-1.5 text-xs font-medium text-white bg-green-600 rounded-lg hover:bg-green-700 focus:ring-4 focus:outline-none focus:ring-green-300 disabled:opacity-50 disabled:cursor-not-allowed dark:bg-green-600 dark:hover:bg-green-700 dark:focus:ring-green-800"
            >
              アップデートを実施
            </button>
          </Show>
        </div>
      </div>
    </div>
  )
}

export default Config;
