<script lang="ts">
  import { goto } from "$app/navigation";
  import { page } from "$app/stores";
  import { useAlert } from "@rilldata/web-admin/features/alerts/selectors";
  import { mapQueryToDashboard } from "@rilldata/web-admin/features/dashboards/query-mappers/mapQueryToDashboard";
  import CtaButton from "@rilldata/web-common/components/calls-to-action/CTAButton.svelte";
  import CtaContentContainer from "@rilldata/web-common/components/calls-to-action/CTAContentContainer.svelte";
  import CtaLayoutContainer from "@rilldata/web-common/components/calls-to-action/CTALayoutContainer.svelte";
  import CtaMessage from "@rilldata/web-common/components/calls-to-action/CTAMessage.svelte";
  import Spinner from "@rilldata/web-common/features/entity-management/Spinner.svelte";
  import { EntityStatus } from "@rilldata/web-common/features/entity-management/types";
  import { runtime } from "@rilldata/web-common/runtime-client/runtime-store";

  $: organization = $page.params.organization;
  $: project = $page.params.project;
  $: alertId = $page.params.alert;
  $: executionTime = $page.url.searchParams.get("execution_time");

  $: alert = useAlert($runtime.instanceId, alertId);

  let dashboardStateForAlert: ReturnType<typeof mapQueryToDashboard>;
  $: dashboardStateForAlert = mapQueryToDashboard(
    $alert.data?.resource?.alert?.spec?.resolverProperties?.query_name ?? "",
    $alert.data?.resource?.alert?.spec?.resolverProperties?.query_args_json ??
      "",
    executionTime,
    $alert.data?.resource?.alert?.spec?.annotations ?? {},
  );

  $: if ($dashboardStateForAlert.data) {
    goto(
      `/${organization}/${project}/${$dashboardStateForAlert.data.metricsView}?state=${encodeURIComponent($dashboardStateForAlert.data.state)}`,
    );
  }
</script>

<CtaLayoutContainer>
  <CtaContentContainer>
    {#if $dashboardStateForAlert.isFetching}
      <div class="h-36 mt-10">
        <Spinner status={EntityStatus.Running} size="7rem" duration={725} />
      </div>
    {:else if $dashboardStateForAlert.error}
      <div class="flex flex-col gap-y-2">
        <h2 class="text-lg font-semibold">Unable to open Alert</h2>
        <CtaMessage>
          {$dashboardStateForAlert.error}
        </CtaMessage>
      </div>
      <CtaButton
        variant="secondary"
        href={`/${organization}/${project}/-/alerts/${$alert}`}
      >
        Go to Alerts page
      </CtaButton>
    {/if}
  </CtaContentContainer>
</CtaLayoutContainer>
