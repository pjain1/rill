<script lang="ts">
  import MetadataLabel from "@rilldata/web-admin/features/scheduled-reports/metadata/MetadataLabel.svelte";
  import DimensionFilterReadOnlyChip from "@rilldata/web-common/features/dashboards/filters/dimension-filters/DimensionFilterReadOnlyChip.svelte";
  import { useDashboard } from "@rilldata/web-common/features/dashboards/selectors";
  import { getDimensionFilters } from "@rilldata/web-common/features/dashboards/state-managers/selectors/dimension-filters";
  import { getMapFromArray } from "@rilldata/web-common/lib/arrayUtils";
  import type {
    V1Expression,
    V1TimeRange,
  } from "@rilldata/web-common/runtime-client";
  import { runtime } from "@rilldata/web-common/runtime-client/runtime-store";
  import { flip } from "svelte/animate";
  import { fly } from "svelte/transition";
  import TimeRangeReadOnly from "@rilldata/web-common/features/dashboards/filters/TimeRangeReadOnly.svelte";

  export let metricsViewName: string;
  export let filters: V1Expression | undefined;
  export let timeRange: V1TimeRange | undefined;
  export let comparisonTimeRange: V1TimeRange | undefined;

  // time range could be an empty object sometimes
  $: hasTimeRange = timeRange && Object.keys(timeRange).length > 0;
  $: filtersLength =
    (filters?.cond?.exprs?.length ?? 0) + (hasTimeRange ? 1 : 0);

  $: dashboard = useDashboard($runtime.instanceId, metricsViewName);
  $: dimensions =
    $dashboard.data?.metricsView?.state?.validSpec?.dimensions ?? [];
  $: dimensionIdMap = getMapFromArray(
    dimensions,
    (dimension) => dimension.name,
  );
  $: currentDimensionFilters = getDimensionFilters(dimensionIdMap, filters);
</script>

<div class="flex flex-col gap-y-3">
  <MetadataLabel>Filters ({filtersLength})</MetadataLabel>
  <div class="flex flex-wrap gap-2">
    {#if filtersLength}
      {#if hasTimeRange}
        <TimeRangeReadOnly {timeRange} {comparisonTimeRange} />
      {/if}
      {#each currentDimensionFilters as { name, label, selectedValues, isInclude } (name)}
        {@const dimension = dimensions.find((d) => d.name === name)}
        <div animate:flip={{ duration: 200 }}>
          {#if dimension?.column}
            <DimensionFilterReadOnlyChip
              label={label ?? name}
              values={selectedValues}
              {isInclude}
            />
          {/if}
        </div>
      {/each}
    {:else}
      <div
        in:fly|local={{ duration: 200, x: 8 }}
        class="ui-copy-disabled grid items-center"
        style:min-height="26px"
      >
        No filters selected
      </div>
    {/if}
  </div>
</div>
