<script lang="ts">
  import { page } from "$app/stores";
  import Rill from "@rilldata/web-common/components/icons/Rill.svelte";
  import type { PathOption } from "@rilldata/web-common/components/navigation/breadcrumbs/Breadcrumbs.svelte";
  import Breadcrumbs from "@rilldata/web-common/components/navigation/breadcrumbs/Breadcrumbs.svelte";
  import { useValidDashboards } from "@rilldata/web-common/features/dashboards/selectors.js";
  import StateManagersProvider from "@rilldata/web-common/features/dashboards/state-managers/StateManagersProvider.svelte";
  import DashboardCtAs from "@rilldata/web-common/features/dashboards/workspace/DashboardCTAs.svelte";
  import { useProjectTitle } from "@rilldata/web-common/features/project/selectors";
  import { runtime } from "@rilldata/web-common/runtime-client/runtime-store";

  $: ({ instanceId } = $runtime);

  $: ({
    params: { name: dashboardName },
    route,
  } = $page);

  $: dashboardsQuery = useValidDashboards(instanceId);
  $: projectTitleQuery = useProjectTitle(instanceId);

  $: projectTitle = $projectTitleQuery.data ?? "Untitled Rill Project";

  $: dashboards = $dashboardsQuery.data ?? [];

  $: dashboardOptions = dashboards.reduce((map, dimension) => {
    const label = dimension.metricsView?.state?.validSpec?.title ?? "";
    const name = dimension.meta?.name?.name ?? "";

    if (label && name)
      map.set(name.toLowerCase(), { label, section: "dashboard", depth: 0 });

    return map;
  }, new Map<string, PathOption>());

  $: projectPath = <PathOption>{
    label: projectTitle,
    section: "project",
    depth: -1,
    href: "/",
  };

  $: pathParts = [
    new Map([[projectTitle.toLowerCase(), projectPath]]),
    dashboardOptions,
  ];

  $: currentPath = [projectTitle, dashboardName.toLowerCase()];

  $: currentDashboard = dashboards.find(
    (d) => d.meta?.name?.name?.toLowerCase() === dashboardName.toLowerCase(),
  );

  $: metricsViewName = currentDashboard?.meta?.name?.name;
</script>

<div class="flex flex-col size-full">
  <header class="py-3 w-full bg-white flex gap-x-2 items-center px-4 border-b">
    {#if $dashboardsQuery.data}
      <Breadcrumbs {pathParts} {currentPath}>
        <a href="/" slot="icon">
          <Rill />
        </a>
      </Breadcrumbs>
    {/if}
    <span class="rounded-full px-2 border text-gray-800 bg-gray-50">
      PREVIEW
    </span>
    {#if route.id?.includes("dashboard") && metricsViewName}
      <StateManagersProvider {metricsViewName}>
        <DashboardCtAs metricViewName={metricsViewName} />
      </StateManagersProvider>
    {/if}
  </header>
  <slot />
</div>
