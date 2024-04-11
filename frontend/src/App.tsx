import 'flowbite';
import { Match, Switch, createSignal,onMount } from 'solid-js';
import { Site } from "../bindings/main/models"
import Map from './Map';
import List from './List';
import EditSite from './EditSite';
import Config  from './Config';
import {MWConfig} from './Config';
import { SaveDark,SaveMapStyle,IsDark,GetMapStyle } from '../bindings/main/Twsnmp';

const [page, setPage] = createSignal("list");
let site: Site;
let oldPage = "list";
let config :MWConfig = {
  dark:false,
  mapStyle:"",
};

const App = () => {
  const close = () => {
    setPage(oldPage);
  }
  onMount(async ()=>{
    config.dark = await IsDark();
    config.mapStyle  = await GetMapStyle();
    setDarkMode(config.dark);
  });
  const saveConfig = (c:MWConfig) => {
    SaveDark(c.dark);
    SaveMapStyle(c.mapStyle);
    config = c;
    setDarkMode(c.dark);
    setPage(oldPage);
  }
  const setDarkMode = (dark:boolean) => {
    if (dark) {
      document.documentElement.classList.add('dark');
      document.documentElement.style.backgroundColor = "rgba(19,19,19,0.8)";
    } else {
      document.documentElement.classList.remove('dark')
      document.documentElement.style.backgroundColor = "transparent";
    }
  }
  const editSite = (s: Site) => {
    site = s;
    oldPage = page();
    setPage("edit");
  }
  return (
    <Switch>
      <Match when={page() == "list"}>
        <List edit={editSite}></List>
        <BottomNav></BottomNav>
      </Match>
      <Match when={page() == "map"}>
        <Map mapStyle={config.mapStyle}></Map>
        <BottomNav></BottomNav>
      </Match>
      <Match when={page() == "edit"}>
        <EditSite close={close} site={site}></EditSite>
      </Match>
      <Match when={page() == "config"}>
        <Config save={saveConfig} close={close} config={config}></Config>
      </Match>
    </Switch>
  )
}

const BottomNav = () => {
  const addSite = () => {
    site = Site.createFrom({
      id: "",
      name: "test",
      url: "",
      user: "",
      password: "",
      state: "",
    });
    setPage("edit");
  }
  const showConfig = ()=> {
    oldPage = page();
    setPage("config");
  }
  const getTabColor = (p: string) => {
    return p == page() ? 
      "text-sm text-blue-500 dark:text-blue-400 group-hover:text-blue-600 dark:group-hover:text-blue-500"
      : "text-sm text-gray-500 dark:text-gray-400 group-hover:text-gray-600 dark:group-hover:text-gray-500"
  }
  return (
    <div class="fixed bottom-0 left-0 z-50 w-full h-16 bg-white border-t border-gray-200 dark:bg-gray-700 dark:border-gray-600">
      <div class="grid h-full max-w-lg grid-cols-4 mx-auto font-medium">
        <button onClick={() => setPage("list")} type="button" class="inline-flex flex-col items-center justify-center px-5 hover:bg-gray-50 dark:hover:bg-gray-800 group">
          <svg class="w-[32px] h-[32px] mb-1 text-gray-500 dark:text-gray-400" aria-hidden="true" xmlns="http://www.w3.org/2000/svg" width="24" height="24" fill="none" viewBox="0 0 24 24">
            <path stroke="currentColor" stroke-linecap="round" stroke-width="2" d="M9 8h10M9 12h10M9 16h10M4.99 8H5m-.02 4h.01m0 4H5" />
          </svg>
          <span class={getTabColor("list")}>一覧</span>
        </button>
        <button onClick={() => setPage("map")} type="button" class="inline-flex flex-col items-center justify-center px-5 hover:bg-gray-50 dark:hover:bg-gray-800 group">
          <svg class="w-[32px] h-[32px] mb-1 text-gray-500 dark:text-gray-400" aria-hidden="true" xmlns="http://www.w3.org/2000/svg" width="24" height="24" fill="currentColor" viewBox="0 0 24 24">
            <path fill-rule="evenodd" d="M11.906 1.994a8.002 8.002 0 0 1 8.09 8.421 7.996 7.996 0 0 1-1.297 3.957.996.996 0 0 1-.133.204l-.108.129c-.178.243-.37.477-.573.699l-5.112 6.224a1 1 0 0 1-1.545 0L5.982 15.26l-.002-.002a18.146 18.146 0 0 1-.309-.38l-.133-.163a.999.999 0 0 1-.13-.202 7.995 7.995 0 0 1 6.498-12.518ZM15 9.997a3 3 0 1 1-5.999 0 3 3 0 0 1 5.999 0Z" clip-rule="evenodd" />
          </svg>
          <span class={getTabColor("map")}>地図</span>
        </button>
        <button onClick={addSite} type="button" class="inline-flex flex-col items-center justify-center px-5 hover:bg-gray-50 dark:hover:bg-gray-800 group">
          <svg class="w-[32px] h-[32px] mb-1 text-blue-600" aria-hidden="true" xmlns="http://www.w3.org/2000/svg" width="24" height="24" fill="currentColor" viewBox="0 0 24 24">
            <path fill-rule="evenodd" d="M2 12C2 6.477 6.477 2 12 2s10 4.477 10 10-4.477 10-10 10S2 17.523 2 12Zm11-4.243a1 1 0 1 0-2 0V11H7.757a1 1 0 1 0 0 2H11v3.243a1 1 0 1 0 2 0V13h3.243a1 1 0 1 0 0-2H13V7.757Z" clip-rule="evenodd" />
          </svg>
          <span class="text-sm text-gray-500 dark:text-gray-400 group-hover:text-gray-600 dark:group-hover:text-gray-500">登録</span>
        </button>
        <button onClick={showConfig} type="button" class="inline-flex flex-col items-center justify-center px-5 hover:bg-gray-50 dark:hover:bg-gray-800 group">
          <svg class="w-[32px] h-[32px] mb-1 text-gray-500 dark:text-gray-400" aria-hidden="true" xmlns="http://www.w3.org/2000/svg" width="24" height="24" fill="currentColor" viewBox="0 0 24 24">
            <path fill-rule="evenodd" d="M9.586 2.586A2 2 0 0 1 11 2h2a2 2 0 0 1 2 2v.089l.473.196.063-.063a2.002 2.002 0 0 1 2.828 0l1.414 1.414a2 2 0 0 1 0 2.827l-.063.064.196.473H20a2 2 0 0 1 2 2v2a2 2 0 0 1-2 2h-.089l-.196.473.063.063a2.002 2.002 0 0 1 0 2.828l-1.414 1.414a2 2 0 0 1-2.828 0l-.063-.063-.473.196V20a2 2 0 0 1-2 2h-2a2 2 0 0 1-2-2v-.089l-.473-.196-.063.063a2.002 2.002 0 0 1-2.828 0l-1.414-1.414a2 2 0 0 1 0-2.827l.063-.064L4.089 15H4a2 2 0 0 1-2-2v-2a2 2 0 0 1 2-2h.09l.195-.473-.063-.063a2 2 0 0 1 0-2.828l1.414-1.414a2 2 0 0 1 2.827 0l.064.063L9 4.089V4a2 2 0 0 1 .586-1.414ZM8 12a4 4 0 1 1 8 0 4 4 0 0 1-8 0Z" clip-rule="evenodd" />
          </svg>
          <span class="text-sm text-gray-500 dark:text-gray-400 group-hover:text-gray-600 dark:group-hover:text-gray-500">設定</span>
        </button>
      </div>
    </div>
  )

}


export default App