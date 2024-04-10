import 'flowbite';
import { createSignal, onMount, For,Component } from 'solid-js';
import { Site } from "../bindings/main/models"
import { GetSites, DeleteSite,OpenSiteMap } from "../bindings/main/Twsnmp"
import StateIcon from './StateIcon';

type Props = {
  edit: (s:Site) => void,
}

const List: Component<Props> = (props: Props) => {
  const [sites, setSites] = createSignal<Site[]>([]);
  onMount(async () => {
    const l = await GetSites() as any;
    setSites(l)
  });
  const delSite = async (id:string) => {
    await DeleteSite(id);
    // 再読み込み　しないように変更したほうがよい
    const l = await GetSites() as any;
    setSites(l)
  }
  return (
    <>
      <ul class="w-[98%] mt-2 mx-auto text-sm font-medium text-gray-900 bg-white border border-gray-200 rounded-lg dark:bg-gray-700 dark:border-gray-600 dark:text-white">
        <For each={sites()}>{(site) =>
          <li class="w-full px-4 py-2 border-b border-gray-200 dark:border-gray-600">
            <div class="flex">
              <div>
                <StateIcon state={site.state}></StateIcon>
              </div>
              <div class="grow m-1">
                {site.name}
              </div>
              <button onClick={()=>{OpenSiteMap(site.id)}} type="button" class="mr-1 p-1">
                <svg class="w-[18px] h-[18px] text-blue-500" aria-hidden="true" xmlns="http://www.w3.org/2000/svg" width="24" height="24" fill="currentColor" viewBox="0 0 24 24">
                  <path fill-rule="evenodd" d="M4.998 7.78C6.729 6.345 9.198 5 12 5c2.802 0 5.27 1.345 7.002 2.78a12.713 12.713 0 0 1 2.096 2.183c.253.344.465.682.618.997.14.286.284.658.284 1.04s-.145.754-.284 1.04a6.6 6.6 0 0 1-.618.997 12.712 12.712 0 0 1-2.096 2.183C17.271 17.655 14.802 19 12 19c-2.802 0-5.27-1.345-7.002-2.78a12.712 12.712 0 0 1-2.096-2.183 6.6 6.6 0 0 1-.618-.997C2.144 12.754 2 12.382 2 12s.145-.754.284-1.04c.153-.315.365-.653.618-.997A12.714 12.714 0 0 1 4.998 7.78ZM12 15a3 3 0 1 0 0-6 3 3 0 0 0 0 6Z" clip-rule="evenodd" />
                </svg>
              </button>
              <button onClick={()=>{props.edit(site)}} type="button" class="mr-1 p-1">
                <svg class="w-[18px] h-[18px] text-grey-800 dark:text-white" aria-hidden="true" xmlns="http://www.w3.org/2000/svg" width="24" height="24" fill="none" viewBox="0 0 24 24">
                  <path stroke="currentColor" stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10.779 17.779 4.36 19.918 6.5 13.5m4.279 4.279 8.364-8.643a3.027 3.027 0 0 0-2.14-5.165 3.03 3.03 0 0 0-2.14.886L6.5 13.5m4.279 4.279L6.499 13.5m2.14 2.14 6.213-6.504M12.75 7.04 17 11.28" />
                </svg>

              </button>
              <button onClick={()=>{delSite(site.id)}} type="button" class="mr-1 p-1">
                <svg class="w-[18px] h-[18px] text-red-500" aria-hidden="true" xmlns="http://www.w3.org/2000/svg" width="24" height="24" fill="currentColor" viewBox="0 0 24 24">
                  <path fill-rule="evenodd" d="M8.586 2.586A2 2 0 0 1 10 2h4a2 2 0 0 1 2 2v2h3a1 1 0 1 1 0 2v12a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V8a1 1 0 0 1 0-2h3V4a2 2 0 0 1 .586-1.414ZM10 6h4V4h-4v2Zm1 4a1 1 0 1 0-2 0v8a1 1 0 1 0 2 0v-8Zm4 0a1 1 0 1 0-2 0v8a1 1 0 1 0 2 0v-8Z" clip-rule="evenodd" />
                </svg>
              </button>
            </div>
          </li>
        }
        </For>
      </ul>
    </>
  )
}


export default List;

