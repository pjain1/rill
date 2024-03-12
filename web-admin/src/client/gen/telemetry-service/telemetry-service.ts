/**
 * Generated by orval v6.12.0 🍺
 * Do not edit manually.
 * rill/admin/v1/ai.proto
 * OpenAPI spec version: version not set
 */
import { createMutation } from "@tanstack/svelte-query";
import type {
  CreateMutationOptions,
  MutationFunction,
} from "@tanstack/svelte-query";
import type {
  V1RecordEventsResponse,
  RpcStatus,
  V1RecordEventsRequest,
} from "../index.schemas";
import { httpClient } from "../../http-client";

type AwaitedInput<T> = PromiseLike<T> | T;

type Awaited<O> = O extends AwaitedInput<infer T> ? T : never;

/**
 * @summary RecordEvents sends a batch of telemetry events.
The events must conform to the schema described in rill/runtime/pkg/activity/README.md.
 */
export const telemetryServiceRecordEvents = (
  v1RecordEventsRequest: V1RecordEventsRequest,
) => {
  return httpClient<V1RecordEventsResponse>({
    url: `/v1/telemetry/events`,
    method: "post",
    headers: { "Content-Type": "application/json" },
    data: v1RecordEventsRequest,
  });
};

export type TelemetryServiceRecordEventsMutationResult = NonNullable<
  Awaited<ReturnType<typeof telemetryServiceRecordEvents>>
>;
export type TelemetryServiceRecordEventsMutationBody = V1RecordEventsRequest;
export type TelemetryServiceRecordEventsMutationError = RpcStatus;

export const createTelemetryServiceRecordEvents = <
  TError = RpcStatus,
  TContext = unknown,
>(options?: {
  mutation?: CreateMutationOptions<
    Awaited<ReturnType<typeof telemetryServiceRecordEvents>>,
    TError,
    { data: V1RecordEventsRequest },
    TContext
  >;
}) => {
  const { mutation: mutationOptions } = options ?? {};

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof telemetryServiceRecordEvents>>,
    { data: V1RecordEventsRequest }
  > = (props) => {
    const { data } = props ?? {};

    return telemetryServiceRecordEvents(data);
  };

  return createMutation<
    Awaited<ReturnType<typeof telemetryServiceRecordEvents>>,
    TError,
    { data: V1RecordEventsRequest },
    TContext
  >(mutationFn, mutationOptions);
};