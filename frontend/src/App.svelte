<script>
  import logo from "./assets/appicon.png"
  import { Table, TableBody, TableBodyCell, TableBodyRow, TableHead, TableHeadCell } from 'flowbite-svelte';
  import { Navbar, NavBrand, DarkMode } from 'flowbite-svelte'
  import { Footer, FooterBrand, FooterCopyright, FooterIcon, FooterLink, FooterLinkGroup } from "flowbite-svelte"
  import * as Icon from 'svelte-heros-v2';
  let maps = [
    { state: "normal", name: 'Mac mini', url: 'http://localhost:3000', show: false },
    { state: "warn", name: 'miniPC2', url: 'http://192.168.14:8080', show: false },
  ];
  let version = "";

</script>

<Navbar>
  <NavBrand>
    <img src="{logo}" class="mr-3 h-10"alt="Logo" />
    <span class="self-center whitespace-nowrap text-xl font-semibold dark:text-white">
      TWSNMP FC Multi Window
    </span>
    <span class="text-sm">
      {version}
    </span>
  </NavBrand>
  <DarkMode></DarkMode>
</Navbar>

<Table hoverable={true}>
  <TableHead>
    <TableHeadCell>状態</TableHeadCell>
    <TableHeadCell>名前</TableHeadCell>
    <TableHeadCell>URL</TableHeadCell>
    <TableHeadCell>表示</TableHeadCell>
    <TableHeadCell>編集</TableHeadCell>
    <TableHeadCell>削除</TableHeadCell>
  </TableHead>
  <TableBody>
    {#each  maps as map}
    <TableBodyRow>
      <TableBodyCell>{map.state}</TableBodyCell>
      <TableBodyCell>{map.name}</TableBodyCell>
      <TableBodyCell>{map.url}</TableBodyCell>
      <TableBodyCell>
        <Icon.Eye on:click={()=>console.log('show')}></Icon.Eye>
      </TableBodyCell>
      <TableBodyCell>
        <Icon.Pencil></Icon.Pencil>
      </TableBodyCell>
      <TableBodyCell>
        <Icon.Trash color="#cc1111"></Icon.Trash>
      </TableBodyCell>
    </TableBodyRow>
    {/each }
  </TableBody>
</Table>

  <Footer class="absolute bottom-0 left-0 z-20 w-full">
    <FooterCopyright by="Masayuki Yamai™" year={2023} />
  </Footer> 