import 'flowbite';
import { Match, Switch } from 'solid-js';
import Main from './Main';

const App = () => {
  const url = new URL(window.location.href);
  const params = url.searchParams;
  const page = params.get("page") || "main";
  const id = params.get("id");

  return (
    <>
      <Switch>
        <Match when={page == "main"} >
          <Main></Main>
        </Match>
        <Match when={page == "map"} >
          <p class="text-4xl text-blue-600">MAP Page id={id}</p>
        </Match>
      </Switch>
    </>
  )
}

export default App