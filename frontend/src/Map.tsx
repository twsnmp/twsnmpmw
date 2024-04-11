import { Component, createSignal,onMount,onCleanup,For } from "solid-js";
import MapGL, { Viewport,Marker } from "solid-map-gl";
import * as maplibre from 'maplibre-gl'
import { Site } from "../bindings/main/models"
import { GetSites,LoadViewport,SaveViewport,UpdateSiteLoc,OpenSiteMap } from "../bindings/main/Twsnmp"
import 'maplibre-gl/dist/maplibre-gl.css'

type Props = {
  mapStyle: string
}

const Map: Component<Props> = (props:Props) => {
  const mapStyle = props.mapStyle || "https://tile.openstreetmap.jp/styles/osm-bright-ja/style.json"; 
  const [viewport, setViewport] = createSignal({
    center: [139.75,35.68],
    zoom: 2,
  } as Viewport);
  const [sites, setSites] = createSignal<Site[]>([]);
  onMount(async () => {
    const l = await GetSites() as any;
    setSites(l)
    const vp = await LoadViewport() as string;
    const a = vp.split(",");
    if (a && a.length == 3 ) {
      setViewport({
        zoom: Number(a[0]),
        center: [Number(a[1]),Number(a[2])],
      })
    }
  });
  onCleanup(async()=> {
    const vp = viewport()
    await SaveViewport(vp.zoom + "," + vp.center[0] + "," + vp.center[1]);
  })
  const mapIcon  = (s: Site) :Element => {
    let c = "bg-gray-500"
    switch (s.state){
    case "high":
      c = "bg-red-500";
      break;
    case "low":
      c = "bg-rose-400";
      break;
    case "warn":
      c = "bg-yellow-400";
      break;
    case "normal":
      c = "bg-green-400";
      break;
    }
    const tempEl = document.createElement('div');
    tempEl.innerHTML = `
<div class="${c} w-[24px]">
  <svg class="w-[24px] h-[24px] p-1  text-white" aria-hidden="true" xmlns="http://www.w3.org/2000/svg" width="24" height="24" fill="currentColor" viewBox="0 0 24 24">
    <path fill-rule="evenodd" d="M4.857 3A1.857 1.857 0 0 0 3 4.857v4.286C3 10.169 3.831 11 4.857 11h4.286A1.857 1.857 0 0 0 11 9.143V4.857A1.857 1.857 0 0 0 9.143 3H4.857Zm10 0A1.857 1.857 0 0 0 13 4.857v4.286c0 1.026.831 1.857 1.857 1.857h4.286A1.857 1.857 0 0 0 21 9.143V4.857A1.857 1.857 0 0 0 19.143 3h-4.286Zm-10 10A1.857 1.857 0 0 0 3 14.857v4.286C3 20.169 3.831 21 4.857 21h4.286A1.857 1.857 0 0 0 11 19.143v-4.286A1.857 1.857 0 0 0 9.143 13H4.857Zm10 0A1.857 1.857 0 0 0 13 14.857v4.286c0 1.026.831 1.857 1.857 1.857h4.286A1.857 1.857 0 0 0 21 19.143v-4.286A1.857 1.857 0 0 0 19.143 13h-4.286Z" clip-rule="evenodd"/>
  </svg>
</div>
<div style="font-size: 12px;text-align: center;">
${s.name}
</div>
    `;
    return tempEl;
  }
  let lastLoc :any;
  const locToLngLat = (loc:string) :[number,number] => {
    const a = loc.split(",");
    if (a.length < 2) {
      return [139.75,35.68];
    }
    return [Number(a[0]), Number(a[1])];
  }
  const saveSiteLoc = async (id:string) => {
    if (lastLoc && lastLoc.length == 2) {
      await UpdateSiteLoc(id,lastLoc[0] + "," + lastLoc[1]);
    }
  }
  return (
    <MapGL
      mapLib={maplibre}
      options={{ 
        style: mapStyle
      }}
      viewport={viewport()}
      onViewportChange={(evt: Viewport) => setViewport(evt)}
    >
      <For each={sites()}>{(site) => 
        <Marker 
          lngLat={locToLngLat(site.loc)} 
          options={{ 
            element: mapIcon(site),
          }} 
          draggable={true}
          onDrag={(loc)=>lastLoc =loc}
          onDragEnd={()=> saveSiteLoc(site.id)}
        >
          <button onClick={()=>OpenSiteMap(site.id)}>{site.url}</button>
        </Marker>
      }
      </For>
    </MapGL>
  );
};

export default Map;
