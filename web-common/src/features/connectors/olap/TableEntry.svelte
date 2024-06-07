<script lang="ts">
  import { page } from "$app/stores";
  import ContextButton from "@rilldata/web-common/components/column-profile/ContextButton.svelte";
  import * as DropdownMenu from "@rilldata/web-common/components/dropdown-menu/";
  import MoreHorizontal from "@rilldata/web-common/components/icons/MoreHorizontal.svelte";
  import TableIcon from "../../../components/icons/TableIcon.svelte";
  import Tooltip from "../../../components/tooltip/Tooltip.svelte";
  import TooltipContent from "../../../components/tooltip/TooltipContent.svelte";
  import TableMenuItems from "./TableMenuItems.svelte";
  import TableSchema from "./TableSchema.svelte";
  import UnsupportedTypesIndicator from "./UnsupportedTypesIndicator.svelte";
  import {
    makeFullyQualifiedTableName,
    makeTablePreviewHref,
  } from "./olap-config";

  export let instanceId: string;
  export let driver: string;
  export let connector: string;
  export let database: string; // The backend interprets an empty string as the default database
  export let databaseSchema: string; // The backend interprets an empty string as the default schema
  export let table: string;
  export let hasUnsupportedDataTypes: boolean;

  let contextMenuOpen = false;
  let showSchema = false;

  $: fullyQualifiedTableName = makeFullyQualifiedTableName(
    driver,
    database,
    databaseSchema,
    table,
  );
  $: tableId = `${connector}-${fullyQualifiedTableName}`;
  $: href = makeTablePreviewHref(
    driver,
    connector,
    database,
    databaseSchema,
    table,
  );
  $: open = $page.url.pathname === href;
</script>

<li aria-label={tableId} class="table-entry group" class:open>
  <div class="table-entry-header {database ? 'pl-[58px]' : 'pl-[40px]'}">
    <TableIcon size="14px" className="shrink-0 text-gray-400" />
    <Tooltip alignment="start" location="right" distance={32}>
      <button
        class="clickable-text"
        on:click={() => (showSchema = !showSchema)}
      >
        <span class="truncate">
          {table}
        </span>
      </button>
      <TooltipContent slot="tooltip-content">
        {showSchema ? "Hide schema" : "Show schema"}
      </TooltipContent>
    </Tooltip>
    {#if hasUnsupportedDataTypes}
      <UnsupportedTypesIndicator
        {instanceId}
        {connector}
        {database}
        {databaseSchema}
        {table}
      />
    {/if}
    <DropdownMenu.Root bind:open={contextMenuOpen}>
      <DropdownMenu.Trigger asChild let:builder>
        <ContextButton
          id="more-actions-{tableId}"
          tooltipText="More actions"
          label="{tableId} actions menu trigger"
          builders={[builder]}
          suppressTooltip={contextMenuOpen}
        >
          <MoreHorizontal />
        </ContextButton>
      </DropdownMenu.Trigger>
      <DropdownMenu.Content
        class="border-none bg-gray-800 text-white min-w-60"
        align="start"
        side="right"
        sideOffset={16}
      >
        <TableMenuItems
          {driver}
          {connector}
          {database}
          {databaseSchema}
          {table}
        />
      </DropdownMenu.Content>
    </DropdownMenu.Root>
  </div>

  {#if showSchema}
    <TableSchema {connector} {database} {databaseSchema} {table} />
  {/if}
</li>

<style lang="postcss">
  .table-entry {
    @apply w-full justify-between;
    @apply flex flex-col;
  }

  .table-entry-header {
    @apply h-6 pr-2; /* left-padding is set dynamically above */
    @apply flex justify-between items-center gap-x-1;
  }

  .table-entry-header:hover {
    @apply bg-slate-100;
  }

  .open {
    @apply bg-slate-100;
  }

  .clickable-text {
    @apply select-none cursor-pointer;
    @apply w-fit flex grow items-center gap-x-2 truncate;
    @apply text-gray-900;
  }
  .clickable-text:hover {
    @apply text-gray-900;
  }
</style>