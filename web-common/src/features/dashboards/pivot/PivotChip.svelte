<script context="module" lang="ts">
  import { Chip } from "@rilldata/web-common/components/chip";
  import type { ChipColors } from "@rilldata/web-common/components/chip/chip-types";
  import {
    defaultChipColors,
    measureChipColors,
    timeChipColors,
  } from "@rilldata/web-common/components/chip/chip-types";
  import { createEventDispatcher } from "svelte";
  import type { PivotChipData } from "./types";
  import { PivotChipType } from "./types";

  const colors: Record<PivotChipType, ChipColors> = {
    time: timeChipColors,
    measure: measureChipColors,
    dimension: defaultChipColors,
  };
</script>

<script lang="ts">
  export let item: PivotChipData;
  export let removable = false;
  export let grab = false;
  export let slideDuration = 150;
  export let active = false;

  const dispatch = createEventDispatcher();
</script>

<Chip
  {grab}
  on:mousedown
  on:click
  outline
  supressTooltip
  {active}
  {slideDuration}
  {removable}
  {...colors[item.type]}
  extraPadding={false}
  extraRounded={item.type !== PivotChipType.Measure}
  label={item.title}
  on:remove={() => {
    dispatch("remove", item);
  }}
>
  <div slot="body" class="flex gap-x-1 items-center">
    {#if item.type === PivotChipType.Time}
      <b>Time</b>
      <p>{item.title}</p>
    {:else}
      <p class="font-semibold">{item.title}</p>
    {/if}
  </div>
</Chip>
