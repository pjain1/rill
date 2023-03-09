<script lang="ts">
  import { outline } from "@rilldata/web-common/components/data-graphic/actions/outline";
  import Body from "@rilldata/web-common/components/data-graphic/elements/Body.svelte";
  import SimpleDataGraphic from "@rilldata/web-common/components/data-graphic/elements/SimpleDataGraphic.svelte";
  import WithBisector from "@rilldata/web-common/components/data-graphic/functional-components/WithBisector.svelte";
  import {
    Axis,
    Grid,
    TimeSeriesMouseover,
  } from "@rilldata/web-common/components/data-graphic/guides";
  import { ChunkedLine } from "@rilldata/web-common/components/data-graphic/marks";
  import { previousValueStore } from "@rilldata/web-local/lib/store-utils";
  import { extent } from "d3-array";
  import { cubicOut } from "svelte/easing";
  import { writable } from "svelte/store";
  import { fade, fly } from "svelte/transition";
  export let width: number = undefined;
  export let height: number = undefined;
  export let xMin;
  export let xMax;
  export let yMin;
  export let yMax;
  export let data;
  export let xAccessor = "ts";
  export let yAccessor = "value";
  export let mouseoverValue;
  export let hovered = false;
  export let mouseoverFormat: (d: number) => string = (v) => v.toString();
  export let mouseoverTimeFormat: (d: number) => string = (v) => v.toString();

  export let tweenProps = { duration: 400, easing: cubicOut };

  // control point for scrub functionality.
  export let scrubbing = false;
  export let scrubStart = undefined;
  export let scrubEnd = undefined;

  $: [xExtentMin, xExtentMax] = extent(data, (d) => d[xAccessor]);
  $: [yExtentMin, yExtentMax] = extent(data, (d) => d[yAccessor]);
  $: internalXMin = xMin || xExtentMin;
  $: internalXMax = xMax || xExtentMax;
  /** we'll set the inflation amount here. */
  let inflate = 5 / 6;

  $: internalYMin = yExtentMin >= 0 ? 0 : yExtentMin;

  $: internalYMax = yMax
    ? yMax
    : yExtentMin == yExtentMax
    ? yExtentMax / inflate
    : yExtentMax / inflate;

  // we delay the tween if previousYMax < yMax
  let yMaxStore = writable(yExtentMax);
  let previousYMax = previousValueStore(yMaxStore);

  $: yMaxStore.set(yExtentMax);
  const timeRangeKey = writable(`${xMin}-${xMax}`);

  const previousTimeRangeKey = previousValueStore(timeRangeKey);

  /** reset the keys to trigger animations on time range changes */
  let syncTimeRangeKey;
  $: {
    timeRangeKey.set(`${xMin}-${xMax}`);
    if ($previousTimeRangeKey !== $timeRangeKey) {
      if (syncTimeRangeKey) clearTimeout(syncTimeRangeKey);
      syncTimeRangeKey = setTimeout(() => {
        previousTimeRangeKey.set($timeRangeKey);
      }, 400);
    }
  }

  $: delay =
    $previousTimeRangeKey === $timeRangeKey && $previousYMax < yExtentMax
      ? 100
      : 0;

  function alwaysBetween(min, max, value) {
    // note: must work with dates
    if (value < min) return min;
    if (value > max) return max;
    return value;
  }

  $: if (scrubbing) {
    scrubEnd = alwaysBetween(internalXMin, internalXMax, mouseoverValue);
  }
</script>

<SimpleDataGraphic
  overflowHidden={false}
  yMin={internalYMin * inflate}
  yMax={internalYMax}
  shareYScale={false}
  yType="number"
  xType="date"
  {width}
  {height}
  top={4}
  left={12}
  right={50}
  bind:mouseoverValue
  bind:hovered
  let:config
  yMinTweenProps={tweenProps}
  yMaxTweenProps={tweenProps}
  xMaxTweenProps={tweenProps}
  xMinTweenProps={tweenProps}
>
  <Axis side="right" format={mouseoverFormat} />
  <Grid />
  <Body>
    <!-- key on the time range itself to prevent weird tweening animations.
    We'll need to migrate this to a more robust solution once we've figured out
    the right way to "tile" together a time series with multiple pages of data.
    -->
    {#key $timeRangeKey}
      <ChunkedLine
        delay={$timeRangeKey !== $previousTimeRangeKey ? 0 : delay}
        duration={$timeRangeKey !== $previousTimeRangeKey ? 0 : 200}
        {data}
        {xAccessor}
        {yAccessor}
        key={$timeRangeKey}
      />
    {/key}
  </Body>
  {#if !scrubbing && mouseoverValue?.x}
    <g transition:fade|local={{ duration: 100 }}>
      <TimeSeriesMouseover
        {data}
        {mouseoverValue}
        {xAccessor}
        {yAccessor}
        format={mouseoverFormat}
      />
    </g>
    <WithBisector
      {data}
      callback={(d) => d[xAccessor]}
      value={mouseoverValue.x}
      let:point
    >
      <g transition:fly|local={{ duration: 100, x: -4 }}>
        <text
          use:outline
          class="fill-gray-600"
          x={config.plotLeft + config.bodyBuffer + 6}
          y={config.plotTop + 10 + config.bodyBuffer}
        >
          {mouseoverTimeFormat(point[xAccessor])}
        </text>
      </g></WithBisector
    >
  {/if}
</SimpleDataGraphic>