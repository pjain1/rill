<script lang="ts">
  import WithTogglableFloatingElement from "@rilldata/web-common/components/floating-element/WithTogglableFloatingElement.svelte";
  import Add from "@rilldata/web-common/components/icons/Add.svelte";
  import SearchableFilterDropdown from "@rilldata/web-common/components/searchable-filter-menu/SearchableFilterDropdown.svelte";
  import type { SearchableFilterSelectableGroup } from "@rilldata/web-common/components/searchable-filter-menu/SearchableFilterSelectableItem";
  import Tooltip from "@rilldata/web-common/components/tooltip/Tooltip.svelte";
  import TooltipContent from "@rilldata/web-common/components/tooltip/TooltipContent.svelte";
  import { getDimensionDisplayName } from "@rilldata/web-common/features/dashboards/filters/getDisplayName";
  import { getStateManagers } from "../state-managers/state-managers";
  import { getMeasureDisplayName } from "./getDisplayName";

  const {
    selectors: {
      dimensions: { allDimensions },
      dimensionFilters: { dimensionHasFilter },
      measures: { filteredSimpleMeasures },
      measureFilters: { measureHasFilter },
    },
    actions: {
      filters: { setTemporaryFilterName },
    },
  } = getStateManagers();

  $: selectableGroups = [
    <SearchableFilterSelectableGroup>{
      name: "MEASURES",
      items:
        $filteredSimpleMeasures()
          ?.map((m) => ({
            name: m.name as string,
            label: getMeasureDisplayName(m),
          }))
          .filter((m) => !$measureHasFilter(m.name)) ?? [],
    },
    <SearchableFilterSelectableGroup>{
      name: "DIMENSIONS",
      items:
        $allDimensions
          ?.map((d) => ({
            name: (d.name || d.column) as string,
            label: getDimensionDisplayName(d),
          }))
          .filter((d) => !$dimensionHasFilter(d.name)) ?? [],
    },
  ];
</script>

<WithTogglableFloatingElement
  alignment="start"
  distance={8}
  let:active
  let:toggleFloatingElement
>
  <Tooltip distance={8} suppress={active}>
    <button class:active on:click={toggleFloatingElement}>
      <Add size="17px" />
    </button>
    <TooltipContent slot="tooltip-content">Add filter</TooltipContent>
  </Tooltip>

  <SearchableFilterDropdown
    allowMultiSelect={false}
    let:toggleFloatingElement
    on:click-outside={toggleFloatingElement}
    on:escape={toggleFloatingElement}
    on:focus
    on:hover
    on:item-clicked={(e) => {
      toggleFloatingElement();
      setTemporaryFilterName(e.detail.name);
    }}
    {selectableGroups}
    selectedItems={[]}
    slot="floating-element"
  />
</WithTogglableFloatingElement>

<style lang="postcss">
  button {
    @apply w-[34px] h-[26px] rounded-2xl;
    @apply flex items-center justify-center;
    @apply border border-dashed border-slate-300;
    @apply bg-white;
  }

  button:hover {
    @apply bg-slate-100;
  }

  button:active,
  .active {
    @apply bg-slate-200;
  }
</style>
