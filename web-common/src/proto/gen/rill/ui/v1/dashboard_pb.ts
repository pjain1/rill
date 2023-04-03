// @generated by protoc-gen-es v1.2.0 with parameter "target=ts"
// @generated from file rill/ui/v1/dashboard.proto (package rill.ui.v1, syntax proto3)
/* eslint-disable */
// @ts-nocheck

import type { BinaryReadOptions, FieldList, JsonReadOptions, JsonValue, PartialMessage, PlainMessage } from "@bufbuild/protobuf";
import { Message, proto3, Timestamp } from "@bufbuild/protobuf";
import { MetricsViewFilter } from "../../runtime/v1/queries_pb.js";
import { TimeGrain } from "../../runtime/v1/catalog_pb.js";

/**
 * DashboardState represents the dashboard as seen by the user
 *
 * @generated from message rill.ui.v1.DashboardState
 */
export class DashboardState extends Message<DashboardState> {
  /**
   * Selected time range
   *
   * @generated from field: rill.ui.v1.DashboardTimeRange time_range = 1;
   */
  timeRange?: DashboardTimeRange;

  /**
   * Dimension filters applied
   *
   * @generated from field: rill.runtime.v1.MetricsViewFilter filters = 2;
   */
  filters?: MetricsViewFilter;

  /**
   * Selected time granularity
   *
   * @generated from field: rill.runtime.v1.TimeGrain time_grain = 3;
   */
  timeGrain = TimeGrain.UNSPECIFIED;

  /**
   * Selected measure for the leaderboard
   *
   * @generated from field: optional string leaderboard_measure = 5;
   */
  leaderboardMeasure?: string;

  /**
   * Focused dimension
   *
   * @generated from field: optional string selected_dimension = 6;
   */
  selectedDimension?: string;

  constructor(data?: PartialMessage<DashboardState>) {
    super();
    proto3.util.initPartial(data, this);
  }

  static readonly runtime: typeof proto3 = proto3;
  static readonly typeName = "rill.ui.v1.DashboardState";
  static readonly fields: FieldList = proto3.util.newFieldList(() => [
    { no: 1, name: "time_range", kind: "message", T: DashboardTimeRange },
    { no: 2, name: "filters", kind: "message", T: MetricsViewFilter },
    { no: 3, name: "time_grain", kind: "enum", T: proto3.getEnumType(TimeGrain) },
    { no: 5, name: "leaderboard_measure", kind: "scalar", T: 9 /* ScalarType.STRING */, opt: true },
    { no: 6, name: "selected_dimension", kind: "scalar", T: 9 /* ScalarType.STRING */, opt: true },
  ]);

  static fromBinary(bytes: Uint8Array, options?: Partial<BinaryReadOptions>): DashboardState {
    return new DashboardState().fromBinary(bytes, options);
  }

  static fromJson(jsonValue: JsonValue, options?: Partial<JsonReadOptions>): DashboardState {
    return new DashboardState().fromJson(jsonValue, options);
  }

  static fromJsonString(jsonString: string, options?: Partial<JsonReadOptions>): DashboardState {
    return new DashboardState().fromJsonString(jsonString, options);
  }

  static equals(a: DashboardState | PlainMessage<DashboardState> | undefined, b: DashboardState | PlainMessage<DashboardState> | undefined): boolean {
    return proto3.util.equals(DashboardState, a, b);
  }
}

/**
 * @generated from message rill.ui.v1.DashboardTimeRange
 */
export class DashboardTimeRange extends Message<DashboardTimeRange> {
  /**
   * @generated from field: optional string name = 1;
   */
  name?: string;

  /**
   * @generated from field: optional google.protobuf.Timestamp time_start = 2;
   */
  timeStart?: Timestamp;

  /**
   * @generated from field: optional google.protobuf.Timestamp time_end = 3;
   */
  timeEnd?: Timestamp;

  constructor(data?: PartialMessage<DashboardTimeRange>) {
    super();
    proto3.util.initPartial(data, this);
  }

  static readonly runtime: typeof proto3 = proto3;
  static readonly typeName = "rill.ui.v1.DashboardTimeRange";
  static readonly fields: FieldList = proto3.util.newFieldList(() => [
    { no: 1, name: "name", kind: "scalar", T: 9 /* ScalarType.STRING */, opt: true },
    { no: 2, name: "time_start", kind: "message", T: Timestamp, opt: true },
    { no: 3, name: "time_end", kind: "message", T: Timestamp, opt: true },
  ]);

  static fromBinary(bytes: Uint8Array, options?: Partial<BinaryReadOptions>): DashboardTimeRange {
    return new DashboardTimeRange().fromBinary(bytes, options);
  }

  static fromJson(jsonValue: JsonValue, options?: Partial<JsonReadOptions>): DashboardTimeRange {
    return new DashboardTimeRange().fromJson(jsonValue, options);
  }

  static fromJsonString(jsonString: string, options?: Partial<JsonReadOptions>): DashboardTimeRange {
    return new DashboardTimeRange().fromJsonString(jsonString, options);
  }

  static equals(a: DashboardTimeRange | PlainMessage<DashboardTimeRange> | undefined, b: DashboardTimeRange | PlainMessage<DashboardTimeRange> | undefined): boolean {
    return proto3.util.equals(DashboardTimeRange, a, b);
  }
}

