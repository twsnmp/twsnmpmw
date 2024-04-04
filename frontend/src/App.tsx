import 'flowbite';
import { Match, Switch } from 'solid-js';

function App() {
  const url = new URL(window.location.href);
  const params = url.searchParams;

  console.log(params.get("page"));
  console.log(params.get("id"));

  const page = params.get("page");
  const id = params.get("id");

  return (
    <>
      <Switch>
        <Match when={page == "main"} >
          <p class="text-4xl text-red-600">Main Page</p>
        </Match>
        <Match when={page == "map"} >
          <p class="text-4xl text-blue-600">MAP Page id={id}</p>
        </Match>
      </Switch>
    </>
  )
}

export default App