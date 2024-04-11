import 'flowbite';
import { Component } from 'solid-js';

export type MWConfig = {
  dark: boolean,
  mapStyle: string
}

type Props = {
  close: () => void,
  save: (c: MWConfig) => void,
  config: MWConfig
}

const Config: Component<Props> = (props: Props) => {
  let dark: HTMLInputElement | undefined;
  let mapStyle: HTMLInputElement | undefined;

  const save = () => {
    props.save({
      dark: dark ? dark.checked : false,
      mapStyle: mapStyle ? mapStyle.value : "",
    });
  }
  return (
    <form class="max-w-sm mx-auto mt-10">
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
      <button onClick={props.close} type="button" class="ml-2 px-2 py-1 text-xs font-medium text-center inline-flex items-center text-white bg-gray-400 rounded-lg hover:bg-gray-500 focus:ring-4 focus:outline-none focus:ring-blue-300 dark:bg-gary-600 dark:hover:bg-gray-700 dark:focus:ring-gray-800">
        <svg class="w-5 h-5 text-white" aria-hidden="true" xmlns="http://www.w3.org/2000/svg" width="24" height="24" fill="none" viewBox="0 0 24 24">
          <path stroke="currentColor" stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18 17.94 6M18 18 6.06 6" />
        </svg>
        キャンセル
      </button>
    </form>
  )
}

export default Config; 