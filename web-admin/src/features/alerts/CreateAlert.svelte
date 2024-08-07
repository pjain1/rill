<script lang="ts">
  import Tooltip from "@rilldata/web-common/components/tooltip/Tooltip.svelte";
  import TooltipContent from "@rilldata/web-common/components/tooltip/TooltipContent.svelte";
  import { useMetricsView } from "@rilldata/web-common/features/dashboards/selectors";
  import { getStateManagers } from "@rilldata/web-common/features/dashboards/state-managers/state-managers";
  import { Button } from "@rilldata/web-common/components/button";
  import { BellPlusIcon } from "lucide-svelte";
  import CreateAlertForm from "@rilldata/web-common/features/alerts/CreateAlertForm.svelte";
  import {
    DialogContent,
    DialogTrigger,
  } from "@rilldata/web-common/components/dialog-v2/index";
  import GuardedDialog from "@rilldata/web-common/components/dialog-v2/GuardedDialog.svelte";

  const {
    selectors: {
      timeRangeSelectors: { isCustomTimeRange },
    },
    runtime,
    metricsViewName,
  } = getStateManagers();

  $: metricsView = useMetricsView($runtime?.instanceId, $metricsViewName);
  $: hasTimeDimension = !!$metricsView?.data?.timeDimension;

  let open = false;
</script>

{#if hasTimeDimension}
  <GuardedDialog
    title="Close without saving?"
    description="You haven’t saved changes to this alert yet, so closing this window will lose your work."
    confirmLabel="Close"
    cancelLabel="Keep editing"
    bind:open
    let:onCancel
    let:onClose
  >
    <DialogTrigger asChild let:builder>
      <Tooltip distance={8} location="top" suppress={!$isCustomTimeRange}>
        <Button
          compact
          disabled={$isCustomTimeRange}
          type="secondary"
          builders={[builder]}
        >
          <BellPlusIcon class="inline-flex" size="16px" />
        </Button>
        <TooltipContent slot="tooltip-content">
          To create an alert, set a non-custom time range.
        </TooltipContent>
      </Tooltip>
    </DialogTrigger>
    <DialogContent class="p-0 m-0 w-[802px] max-w-fit rounded-md" noClose>
      <CreateAlertForm on:cancel={onCancel} on:close={onClose} />
    </DialogContent>
  </GuardedDialog>
{/if}
