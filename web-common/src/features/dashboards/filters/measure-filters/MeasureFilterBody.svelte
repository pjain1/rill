<!-- @component
  renders the body content of a filter set chip:
  - a label for the current measure
  - a short hand notation of the filter criteria
-->
<script lang="ts">
  import IconSpaceFixer from "@rilldata/web-common/components/button/IconSpaceFixer.svelte";
  import CaretDownIcon from "@rilldata/web-common/components/icons/CaretDownIcon.svelte";
  import { MeasureFilterEntry } from "@rilldata/web-common/features/dashboards/filters/measure-filters/measure-filter-entry";
  import {
    AllMeasureFilterOperationOptions,
    AllMeasureFilterTypeOptions,
    MeasureFilterOperation,
    MeasureFilterType,
  } from "@rilldata/web-common/features/dashboards/filters/measure-filters/measure-filter-options";

  export let dimensionName: string;
  export let label: string | undefined;
  export let filter: MeasureFilterEntry | undefined;
  export let labelMaxWidth = "160px";
  export let active = false;
  export let readOnly = false;
  export let comparisonLabel = "";

  let typeLabel: string | undefined;
  let shortLabel: string | undefined;
  $: if (filter) {
    const typeOption = AllMeasureFilterTypeOptions.find(
      (o) => o.value === filter?.type,
    );
    typeLabel = typeOption?.shortLabel;

    if (
      filter.type === MeasureFilterType.AbsoluteChange ||
      filter.type === MeasureFilterType.PercentChange
    ) {
      typeLabel += ` from ${comparisonLabel}`;
    }

    switch (filter.operation) {
      case MeasureFilterOperation.GreaterThan:
      case MeasureFilterOperation.GreaterThanOrEquals:
      case MeasureFilterOperation.LessThan:
      case MeasureFilterOperation.LessThanOrEquals:
      case MeasureFilterOperation.Equals:
      case MeasureFilterOperation.NotEquals:
        shortLabel =
          AllMeasureFilterOperationOptions.find(
            (o) =>
              // svelte-check is throwing an error here stating `filter could be undefined` so we need this
              o.value === filter?.operation ??
              MeasureFilterOperation.GreaterThan,
          )?.shortLabel +
          " " +
          filter.value1 +
          (filter.type === MeasureFilterType.PercentChange ? "%" : "");
        break;
      case MeasureFilterOperation.Between:
        shortLabel = `(${filter.value1},${filter.value2})`;
        break;
      case MeasureFilterOperation.NotBetween:
        shortLabel = `!(${filter.value1},${filter.value2})`;
        break;
    }
  }
</script>

<div class="flex gap-x-2">
  <div
    class="font-bold text-ellipsis overflow-hidden whitespace-nowrap"
    style:max-width={labelMaxWidth}
  >
    {label}
    {#if dimensionName}
      <!-- span needed to make sure the space before the `for` is not removed by prettier -->
      <span> for {dimensionName}</span>
    {/if}
    {#if typeLabel}
      <span>{typeLabel}</span>
    {/if}
  </div>
  <div class="flex flex-wrap flex-row items-baseline gap-y-1">
    {#if shortLabel}
      {shortLabel}
    {/if}
    {#if !readOnly}
      <IconSpaceFixer className="pl-2" pullRight>
        <div class="transition-transform" class:-rotate-180={active}>
          <CaretDownIcon className="inline" size="10px" />
        </div>
      </IconSpaceFixer>
    {/if}
  </div>
</div>
