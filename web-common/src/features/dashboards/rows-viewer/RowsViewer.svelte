<script lang="ts">
  import type { VirtualizedTableColumns } from "@rilldata/web-common/components/virtualized-table/types";
  import { getStateManagers } from "@rilldata/web-common/features/dashboards/state-managers/state-managers";
  import { sanitiseExpression } from "@rilldata/web-common/features/dashboards/stores/filter-utils";
  import { useTimeControlStore } from "@rilldata/web-common/features/dashboards/time-controls/time-control-store";
  import { createQueryServiceMetricsViewRows } from "@rilldata/web-common/runtime-client";
  import { runtime } from "@rilldata/web-common/runtime-client/runtime-store";
  import { writable } from "svelte/store";
  import { useDashboardStore } from "web-common/src/features/dashboards/stores/dashboard-stores";
  import { PreviewTable } from "../../../components/preview-table";
  import ReconcilingSpinner from "../../entity-management/ReconcilingSpinner.svelte";

  export let metricViewName = "";
  export let height: number;

  const SAMPLE_SIZE = 10000;
  const FALLBACK_SAMPLE_SIZE = 1000;

  $: dashboardStore = useDashboardStore(metricViewName);
  const timeControlsStore = useTimeControlStore(getStateManagers());

  let limit = writable(SAMPLE_SIZE);

  $: tableQuery = createQueryServiceMetricsViewRows(
    $runtime?.instanceId,
    metricViewName,
    {
      limit: $limit,
      where: sanitiseExpression($dashboardStore.whereFilter, undefined),
      timeStart: $timeControlsStore.timeStart,
      timeEnd: $timeControlsStore.timeEnd,
    },
    {
      query: {
        enabled: $timeControlsStore.ready && !!$dashboardStore?.whereFilter,
      },
    },
  );

  // If too much date is requested, limit the query to 1000 rows
  $: if (
    // @ts-ignore
    $tableQuery?.error?.response?.data?.code === 8 &&
    $limit > FALLBACK_SAMPLE_SIZE
  ) {
    // SK: Have to set the limit on the next tick or the tableQuery does not update. Not sure why, seems like a svelte-query issue.
    setTimeout(() => {
      limit.set(FALLBACK_SAMPLE_SIZE);
    });
  }

  let rows;
  let tableColumns: VirtualizedTableColumns[];
  $: {
    if ($tableQuery.isSuccess) {
      rows = $tableQuery.data.data;
      tableColumns = $tableQuery.data.meta as VirtualizedTableColumns[];
    }
  }
</script>

<div class="overflow-hidden border-t" style:height="{height}px">
  {#if rows}
    <PreviewTable
      {rows}
      columnNames={tableColumns}
      rowHeight={32}
      name={metricViewName}
    />
  {:else}
    <ReconcilingSpinner />
  {/if}
</div>
