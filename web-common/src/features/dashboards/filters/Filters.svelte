<script lang="ts">
  import Button from "@rilldata/web-common/components/button/Button.svelte";
  import Filter from "@rilldata/web-common/components/icons/Filter.svelte";
  import { MeasureFilterEntry } from "@rilldata/web-common/features/dashboards/filters/measure-filters/measure-filter-entry";
  import MeasureFilter from "@rilldata/web-common/features/dashboards/filters/measure-filters/MeasureFilter.svelte";
  import { getMapFromArray } from "@rilldata/web-common/lib/arrayUtils";
  import { flip } from "svelte/animate";
  import { useMetricsView } from "../selectors/index";
  import { getStateManagers } from "../state-managers/state-managers";
  import FilterButton from "./FilterButton.svelte";
  import DimensionFilter from "./dimension-filters/DimensionFilter.svelte";
  import SuperPill from "../time-controls/super-pill/SuperPill.svelte";
  import { useTimeControlStore } from "../time-controls/time-control-store";
  import Calendar from "@rilldata/web-common/components/icons/Calendar.svelte";
  import { fly } from "svelte/transition";
  import ComparisonPill from "../time-controls/comparison-pill/ComparisonPill.svelte";
  import { useModelHasTimeSeries } from "../selectors";
  import { runtime } from "@rilldata/web-common/runtime-client/runtime-store";

  export let readOnly = false;

  /** the height of a row of chips */
  const ROW_HEIGHT = "26px";

  const StateManagers = getStateManagers();
  const {
    metricsViewName,
    actions: {
      dimensionsFilter: {
        toggleDimensionValueSelection,
        removeDimensionFilter,
      },
      measuresFilter: { setMeasureFilter, removeMeasureFilter },
      filters: { clearAllFilters },
    },
    selectors: {
      dimensionFilters: { getDimensionFilterItems, getAllDimensionFilterItems },
      measureFilters: { getMeasureFilterItems, getAllMeasureFilterItems },
    },
  } = StateManagers;

  const timeControlsStore = useTimeControlStore(StateManagers);
  const metricsView = useMetricsView(StateManagers);

  $: ({
    selectedTimeRange,
    allTimeRange,
    showTimeComparison,
    selectedComparisonTimeRange,
  } = $timeControlsStore);

  $: ({ instanceId } = $runtime);

  $: dimensions = $metricsView.data?.dimensions ?? [];
  $: dimensionIdMap = getMapFromArray(
    dimensions,
    (dimension) => (dimension.name || dimension.column) as string,
  );

  $: measures = $metricsView.data?.measures ?? [];
  $: measureIdMap = getMapFromArray(measures, (m) => m.name as string);

  $: currentDimensionFilters = $getDimensionFilterItems(dimensionIdMap);
  $: allDimensionFilters = $getAllDimensionFilterItems(
    currentDimensionFilters,
    dimensionIdMap,
  );

  $: currentMeasureFilters = $getMeasureFilterItems(measureIdMap);
  $: allMeasureFilters = $getAllMeasureFilterItems(
    currentMeasureFilters,
    measureIdMap,
  );

  // hasFilter only checks for complete filters and excludes temporary ones
  $: hasFilters =
    currentDimensionFilters.length > 0 || currentMeasureFilters.length > 0;
  $: metricTimeSeries = useModelHasTimeSeries(instanceId, $metricsViewName);
  $: hasTimeSeries = $metricTimeSeries.data;

  function handleMeasureFilterApply(
    dimension: string,
    measureName: string,
    oldDimension: string,
    filter: MeasureFilterEntry,
  ) {
    if (oldDimension && oldDimension !== dimension) {
      removeMeasureFilter(oldDimension, measureName);
    }
    setMeasureFilter(dimension, filter);
  }
</script>

<div class="flex flex-col gap-y-2 size-full">
  {#if hasTimeSeries}
    <div class="flex flex-row flex-wrap gap-x-2 gap-y-1.5 items-center">
      <Calendar size="16px" />
      {#if allTimeRange?.start && allTimeRange?.end}
        <SuperPill {allTimeRange} {selectedTimeRange} />
        <ComparisonPill
          {allTimeRange}
          {selectedTimeRange}
          showTimeComparison={!!showTimeComparison}
          {selectedComparisonTimeRange}
        />
      {/if}
    </div>
  {/if}

  <div class="relative flex flex-row gap-x-2 gap-y-2 items-start">
    {#if !readOnly}
      <Filter size="16px" className="ui-copy-icon flex-none mt-[5px]" />
    {/if}
    <div class="relative flex flex-row flex-wrap gap-x-2 gap-y-2">
      {#if !allDimensionFilters.length && !allMeasureFilters.length}
        <div
          in:fly={{ duration: 200, x: 8 }}
          class="ui-copy-disabled grid ml-1 items-center"
          style:min-height={ROW_HEIGHT}
        >
          No filters selected
        </div>
      {:else}
        {#each allDimensionFilters as { name, label, selectedValues } (name)}
          {@const dimension = dimensions.find(
            (d) => d.name === name || d.column === name,
          )}
          {@const dimensionName = dimension?.name || dimension?.column}
          <div animate:flip={{ duration: 200 }}>
            {#if dimensionName}
              <DimensionFilter
                {name}
                {label}
                {selectedValues}
                on:remove={() => removeDimensionFilter(name)}
                on:apply={(event) =>
                  toggleDimensionValueSelection(name, event.detail, true)}
              />
            {/if}
          </div>
        {/each}
        {#each allMeasureFilters as { name, label, dimensionName, filter } (name)}
          <div animate:flip={{ duration: 200 }}>
            <MeasureFilter
              {name}
              {label}
              {dimensionName}
              {filter}
              on:remove={() => removeMeasureFilter(dimensionName, name)}
              on:apply={({ detail: { dimension, oldDimension, filter } }) =>
                handleMeasureFilterApply(dimension, name, oldDimension, filter)}
            />
          </div>
        {/each}
      {/if}

      {#if !readOnly}
        <FilterButton />
        <!-- if filters are present, place a chip at the end of the flex container 
      that enables clearing all filters -->
        {#if hasFilters}
          <Button type="text" on:click={clearAllFilters}>Clear filters</Button>
        {/if}
      {/if}
    </div>
  </div>
</div>
