import 'flowbite';
import { Component } from 'solid-js';
import { Site } from "../bindings/main/models";
import { UpdateSite } from "../bindings/main/Twsnmp";

type Props = {
  site: Site,
  close: () => void,
}

const EditSite: Component<Props> = (props:Props) => {
  const site = props.site;
  let name: HTMLInputElement | undefined;
  let url : HTMLInputElement | undefined;
  let user : HTMLInputElement | undefined;
  let password : HTMLInputElement | undefined;
  const save = async () => {
    site.name = name ? name.value : "";
    site.url = url ? url.value : "";
    site.user = user ? user.value : "";
    site.password = password ? password.value : "";
    await UpdateSite(site.id,site.name,site.url,site.user,site.password);
    props.close();
  };
  return (
    <>
      <form class="max-w-sm mx-auto mt-5">
        <div class="mb-5">
          <label class="block mb-2 text-sm font-medium text-gray-900 dark:text-white">名前
            <input ref={name} value={site.name} class="bg-gray-50 border border-gray-300 text-gray-900 text-sm rounded-lg focus:ring-blue-500 focus:border-blue-500 block w-full p-2.5 dark:bg-gray-700 dark:border-gray-600 dark:placeholder-gray-400 dark:text-white dark:focus:ring-blue-500 dark:focus:border-blue-500" placeholder="TWSNMP FCの名前" />
          </label>
        </div>
        <div class="mb-5">
          <label class="block mb-2 text-sm font-medium text-gray-900 dark:text-white">URL
            <input ref={url} value={site.url} type="url" class="bg-gray-50 border border-gray-300 text-gray-900 text-sm rounded-lg focus:ring-blue-500 focus:border-blue-500 block w-full p-2.5 dark:bg-gray-700 dark:border-gray-600 dark:placeholder-gray-400 dark:text-white dark:focus:ring-blue-500 dark:focus:border-blue-500" placeholder="TWSNMP FCのURL" />
          </label>
        </div>
        <div class="mb-5">
          <label class="block mb-2 text-sm font-medium text-gray-900 dark:text-white">ユーザーID
            <input ref={user} value={site.user} class="bg-gray-50 border border-gray-300 text-gray-900 text-sm rounded-lg focus:ring-blue-500 focus:border-blue-500 block w-full p-2.5 dark:bg-gray-700 dark:border-gray-600 dark:placeholder-gray-400 dark:text-white dark:focus:ring-blue-500 dark:focus:border-blue-500" placeholder="ユーザーID" />
          </label>
        </div>
        <div class="mb-5">
          <label class="block mb-2 text-sm font-medium text-gray-900 dark:text-white">パスワード
            <input ref={password} value={site.password} type="password" class="bg-gray-50 border border-gray-300 text-gray-900 text-sm rounded-lg focus:ring-blue-500 focus:border-blue-500 block w-full p-2.5 dark:bg-gray-700 dark:border-gray-600 dark:placeholder-gray-400 dark:text-white dark:focus:ring-blue-500 dark:focus:border-blue-500" />
          </label>
        </div>
        <button onClick={save} type="button" class="px-2 py-1 text-xs font-medium text-center inline-flex items-center text-white bg-blue-700 rounded-lg hover:bg-blue-800 focus:ring-4 focus:outline-none focus:ring-blue-300 dark:bg-blue-600 dark:hover:bg-blue-700 dark:focus:ring-blue-800">
          <svg class="w-5 h-5 text-white dark:text-white" aria-hidden="true" xmlns="http://www.w3.org/2000/svg" width="24" height="24" fill="none" viewBox="0 0 24 24">
            <path stroke="currentColor" stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 11.917 9.724 16.5 19 7.5" />
          </svg>
          保存
        </button>
        <button onClick={props.close} type="button" class="ml-2 px-2 py-1 text-xs font-medium text-center inline-flex items-center text-white bg-gray-400 rounded-lg hover:bg-gray-500 focus:ring-4 focus:outline-none focus:ring-blue-300 dark:bg-blue-600 dark:hover:bg-blue-700 dark:focus:ring-gray-800">
          <svg class="w-5 h-5 text-white dark:text-white" aria-hidden="true" xmlns="http://www.w3.org/2000/svg" width="24" height="24" fill="none" viewBox="0 0 24 24">
            <path stroke="currentColor" stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18 17.94 6M18 18 6.06 6" />
          </svg>
          キャンセル
        </button>
      </form>
    </>
  )
}

export default EditSite;