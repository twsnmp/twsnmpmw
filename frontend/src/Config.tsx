import 'flowbite';
import { Component, Show } from 'solid-js';
import { UpdateState, VersionInfo } from "../bindings/github.com/twsnmp/twsnmpmw";

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
  let dark: HTMLInputElement | undefined;
  let mapStyle: HTMLInputElement | undefined;

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

  const save = () => {
    props.save({
      dark: dark ? dark.checked : false,
      mapStyle: mapStyle ? mapStyle.value : "",
    });
  }

  return (
    <div class="max-w-sm mx-auto mt-10 pb-10">
      <form>
        <div class="mb-5">
          <label class="inline-flex items-center cursor-pointer">
            <input ref={dark} type="checkbox" class="sr-only peer" checked={props.config.dark} />
            <div class="relative w-11 h-6 bg-gray-200 rounded-full peer peer-focus:ring-4 peer-focus:ring-blue-300 dark:peer-focus:ring-blue-800 dark:bg-gray-700 peer-checked:after:translate-x-full rtl:peer-checked:after:-translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-0.5 after:start-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all dark:border-gray-600 peer-checked:bg-blue-600"></div>
            <span class="ms-3 text-sm font-medium text-gray-900 dark:text-gray-300">Darkモード</span>
          </label>
        </div>
        <div class="mb-5">
          <label class="block mb-2 text-sm font-medium text-gray-900 dark:text-white">地図のスタイル
            <input ref={mapStyle} value={props.config.mapStyle} type="url" class="bg-gray-50 border border-gray-300 text-gray-900 text-sm rounded-lg focus:ring-blue-500 focus:border-blue-500 block w-full p-2.5 dark:bg-gray-700 dark:border-gray-600 dark:placeholder-gray-400 dark:text-white dark:focus:ring-blue-500 dark:focus:border-blue-500" placeholder="スタイルのURL" />
          </label>
        </div>
        <button onClick={save} type="button" class="px-2 py-1 text-xs font-medium text-center inline-flex items-center text-white bg-blue-700 rounded-lg hover:bg-blue-800 focus:ring-4 focus:outline-none focus:ring-blue-300 dark:bg-blue-600 dark:hover:bg-blue-700 dark:focus:ring-blue-800">
          <svg class="w-5 h-5 text-white dark:text-white" aria-hidden="true" xmlns="http://www.w3.org/2000/svg" width="24" height="24" fill="none" viewBox="0 0 24 24">
            <path stroke="currentColor" stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 11.917 9.724 16.5 19 7.5" />
          </svg>
          保存
        </button>
        <button onClick={props.close} type="button" class="ml-2 px-2 py-1 text-xs font-medium text-center inline-flex items-center text-white bg-gray-400 rounded-lg hover:bg-gray-500 focus:ring-4 focus:outline-none focus:ring-blue-300 dark:bg-gray-600 dark:hover:bg-gray-700 dark:focus:ring-gray-800">
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
