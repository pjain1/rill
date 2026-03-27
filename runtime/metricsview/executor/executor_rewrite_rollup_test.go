package executor

import (
	"testing"
	"time"

	runtimev1 "github.com/rilldata/rill/proto/gen/rill/runtime/v1"
	"github.com/rilldata/rill/runtime/metricsview"
	"github.com/stretchr/testify/require"
)

func TestRewriteQueryForRollup_BasicMatch(t *testing.T) {
	e := &Executor{
		metricsView: &runtimev1.MetricsViewSpec{
			Table:         "base_table",
			TimeDimension: "timestamp",
			Dimensions: []*runtimev1.MetricsViewSpec_Dimension{
				{Name: "publisher", Column: "publisher"},
				{Name: "domain", Column: "domain"},
			},
			Measures: []*runtimev1.MetricsViewSpec_Measure{
				{Name: "total_impressions", Expression: `SUM("impressions")`},
				{Name: "total_clicks", Expression: `SUM("clicks")`},
			},
			Rollups: []*runtimev1.MetricsViewSpec_RollupTable{
				{
					Table:      "daily_rollup",
					TimeGrain:  runtimev1.TimeGrain_TIME_GRAIN_DAY,
					Dimensions: []string{"publisher", "domain"},
					Measures: []*runtimev1.MetricsViewSpec_RollupMeasure{
						{Name: "total_impressions", Expression: `SUM("impressions_sum")`},
						{Name: "total_clicks", Expression: `SUM("clicks_sum")`},
					},
				},
			},
		},
	}

	qry := &metricsview.Query{
		Dimensions: []metricsview.Dimension{
			{Name: "publisher"},
			{Compute: &metricsview.DimensionCompute{TimeFloor: &metricsview.DimensionComputeTimeFloor{Dimension: "timestamp", Grain: metricsview.TimeGrainDay}}},
		},
		Measures: []metricsview.Measure{
			{Name: "total_impressions"},
		},
		TimeRange: &metricsview.TimeRange{
			Start: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			End:   time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC),
		},
	}

	result := e.rewriteQueryForRollup(qry)
	require.NotNil(t, result)
	require.NotNil(t, result.spec)
	require.Equal(t, "daily_rollup", result.spec.Table)
	require.Empty(t, result.spec.Model)
	require.Nil(t, result.spec.Rollups)

	// Check measure expression was rewritten
	for _, m := range result.spec.Measures {
		if m.Name == "total_impressions" {
			require.Equal(t, `SUM("impressions_sum")`, m.Expression)
		}
	}
}

func TestRewriteQueryForRollup_NoRollups(t *testing.T) {
	e := &Executor{
		metricsView: &runtimev1.MetricsViewSpec{
			Table: "base_table",
		},
	}

	qry := &metricsview.Query{
		Measures: []metricsview.Measure{{Name: "count"}},
	}

	result := e.rewriteQueryForRollup(qry)
	require.Nil(t, result)
}

func TestRewriteQueryForRollup_RawRows(t *testing.T) {
	e := &Executor{
		metricsView: &runtimev1.MetricsViewSpec{
			Table: "base_table",
			Rollups: []*runtimev1.MetricsViewSpec_RollupTable{
				{Table: "rollup", TimeGrain: runtimev1.TimeGrain_TIME_GRAIN_DAY},
			},
		},
	}

	qry := &metricsview.Query{Rows: true}
	result := e.rewriteQueryForRollup(qry)
	require.Nil(t, result)
}

func TestRewriteQueryForRollup_Spine(t *testing.T) {
	e := &Executor{
		metricsView: &runtimev1.MetricsViewSpec{
			Table: "base_table",
			Rollups: []*runtimev1.MetricsViewSpec_RollupTable{
				{Table: "rollup", TimeGrain: runtimev1.TimeGrain_TIME_GRAIN_DAY},
			},
		},
	}

	qry := &metricsview.Query{
		Spine:    &metricsview.Spine{},
		Measures: []metricsview.Measure{{Name: "count"}},
	}
	result := e.rewriteQueryForRollup(qry)
	require.Nil(t, result)
}

func TestRewriteQueryForRollup_MissingDimension(t *testing.T) {
	e := &Executor{
		metricsView: &runtimev1.MetricsViewSpec{
			Table:         "base_table",
			TimeDimension: "timestamp",
			Dimensions: []*runtimev1.MetricsViewSpec_Dimension{
				{Name: "publisher", Column: "publisher"},
				{Name: "domain", Column: "domain"},
			},
			Measures: []*runtimev1.MetricsViewSpec_Measure{
				{Name: "total_impressions", Expression: `SUM("impressions")`},
			},
			Rollups: []*runtimev1.MetricsViewSpec_RollupTable{
				{
					Table:      "daily_rollup",
					TimeGrain:  runtimev1.TimeGrain_TIME_GRAIN_DAY,
					Dimensions: []string{"publisher"}, // missing "domain"
					Measures: []*runtimev1.MetricsViewSpec_RollupMeasure{
						{Name: "total_impressions", Expression: `SUM("impressions_sum")`},
					},
				},
			},
		},
	}

	qry := &metricsview.Query{
		Dimensions: []metricsview.Dimension{
			{Name: "domain"}, // not in rollup
		},
		Measures: []metricsview.Measure{
			{Name: "total_impressions"},
		},
	}

	result := e.rewriteQueryForRollup(qry)
	require.Nil(t, result)
}

func TestRewriteQueryForRollup_MissingMeasure(t *testing.T) {
	e := &Executor{
		metricsView: &runtimev1.MetricsViewSpec{
			Table: "base_table",
			Measures: []*runtimev1.MetricsViewSpec_Measure{
				{Name: "total_impressions", Expression: `SUM("impressions")`},
				{Name: "total_clicks", Expression: `SUM("clicks")`},
			},
			Rollups: []*runtimev1.MetricsViewSpec_RollupTable{
				{
					Table:     "daily_rollup",
					TimeGrain: runtimev1.TimeGrain_TIME_GRAIN_DAY,
					Measures: []*runtimev1.MetricsViewSpec_RollupMeasure{
						{Name: "total_impressions", Expression: `SUM("impressions_sum")`},
						// missing total_clicks
					},
				},
			},
		},
	}

	qry := &metricsview.Query{
		Measures: []metricsview.Measure{
			{Name: "total_clicks"}, // not in rollup
		},
	}

	result := e.rewriteQueryForRollup(qry)
	require.Nil(t, result)
}

func TestRewriteQueryForRollup_GrainNotDerivable(t *testing.T) {
	e := &Executor{
		metricsView: &runtimev1.MetricsViewSpec{
			Table:         "base_table",
			TimeDimension: "timestamp",
			Measures: []*runtimev1.MetricsViewSpec_Measure{
				{Name: "total_impressions", Expression: `SUM("impressions")`},
			},
			Rollups: []*runtimev1.MetricsViewSpec_RollupTable{
				{
					Table:     "weekly_rollup",
					TimeGrain: runtimev1.TimeGrain_TIME_GRAIN_WEEK,
					Measures: []*runtimev1.MetricsViewSpec_RollupMeasure{
						{Name: "total_impressions", Expression: `SUM("impressions_sum")`},
					},
				},
			},
		},
	}

	// Query for month grain; month is not derivable from week
	qry := &metricsview.Query{
		Dimensions: []metricsview.Dimension{
			{Compute: &metricsview.DimensionCompute{TimeFloor: &metricsview.DimensionComputeTimeFloor{Dimension: "timestamp", Grain: metricsview.TimeGrainMonth}}},
		},
		Measures: []metricsview.Measure{
			{Name: "total_impressions"},
		},
	}

	result := e.rewriteQueryForRollup(qry)
	require.Nil(t, result)
}

func TestRewriteQueryForRollup_TimeRangeNotAligned(t *testing.T) {
	e := &Executor{
		metricsView: &runtimev1.MetricsViewSpec{
			Table:         "base_table",
			TimeDimension: "timestamp",
			Measures: []*runtimev1.MetricsViewSpec_Measure{
				{Name: "total_impressions", Expression: `SUM("impressions")`},
			},
			Rollups: []*runtimev1.MetricsViewSpec_RollupTable{
				{
					Table:     "daily_rollup",
					TimeGrain: runtimev1.TimeGrain_TIME_GRAIN_DAY,
					Measures: []*runtimev1.MetricsViewSpec_RollupMeasure{
						{Name: "total_impressions", Expression: `SUM("impressions_sum")`},
					},
				},
			},
		},
	}

	qry := &metricsview.Query{
		Measures: []metricsview.Measure{
			{Name: "total_impressions"},
		},
		TimeRange: &metricsview.TimeRange{
			Start: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC), // not aligned to day
			End:   time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC),
		},
	}

	result := e.rewriteQueryForRollup(qry)
	require.Nil(t, result)
}

func TestRewriteQueryForRollup_WhereDimensionMissing(t *testing.T) {
	e := &Executor{
		metricsView: &runtimev1.MetricsViewSpec{
			Table: "base_table",
			Dimensions: []*runtimev1.MetricsViewSpec_Dimension{
				{Name: "publisher", Column: "publisher"},
				{Name: "domain", Column: "domain"},
			},
			Measures: []*runtimev1.MetricsViewSpec_Measure{
				{Name: "total_impressions", Expression: `SUM("impressions")`},
			},
			Rollups: []*runtimev1.MetricsViewSpec_RollupTable{
				{
					Table:      "daily_rollup",
					TimeGrain:  runtimev1.TimeGrain_TIME_GRAIN_DAY,
					Dimensions: []string{"publisher"}, // no "domain"
					Measures: []*runtimev1.MetricsViewSpec_RollupMeasure{
						{Name: "total_impressions", Expression: `SUM("impressions_sum")`},
					},
				},
			},
		},
	}

	qry := &metricsview.Query{
		Measures: []metricsview.Measure{
			{Name: "total_impressions"},
		},
		Where: &metricsview.Expression{
			Condition: &metricsview.Condition{
				Operator: metricsview.OperatorEq,
				Expressions: []*metricsview.Expression{
					{Name: "domain"},
					{Value: "example.com"},
				},
			},
		},
	}

	result := e.rewriteQueryForRollup(qry)
	require.Nil(t, result)
}

func TestRewriteQueryForRollup_PreferCoarsestGrain(t *testing.T) {
	e := &Executor{
		metricsView: &runtimev1.MetricsViewSpec{
			Table:         "base_table",
			TimeDimension: "timestamp",
			Measures: []*runtimev1.MetricsViewSpec_Measure{
				{Name: "total_impressions", Expression: `SUM("impressions")`},
			},
			Rollups: []*runtimev1.MetricsViewSpec_RollupTable{
				{
					Table:     "hourly_rollup",
					TimeGrain: runtimev1.TimeGrain_TIME_GRAIN_HOUR,
					Measures: []*runtimev1.MetricsViewSpec_RollupMeasure{
						{Name: "total_impressions", Expression: `SUM("impressions_sum")`},
					},
				},
				{
					Table:     "daily_rollup",
					TimeGrain: runtimev1.TimeGrain_TIME_GRAIN_DAY,
					Measures: []*runtimev1.MetricsViewSpec_RollupMeasure{
						{Name: "total_impressions", Expression: `SUM("impressions_sum")`},
					},
				},
			},
		},
	}

	// Query for month grain; both hourly and daily are eligible, but daily is coarser
	qry := &metricsview.Query{
		Dimensions: []metricsview.Dimension{
			{Compute: &metricsview.DimensionCompute{TimeFloor: &metricsview.DimensionComputeTimeFloor{Dimension: "timestamp", Grain: metricsview.TimeGrainMonth}}},
		},
		Measures: []metricsview.Measure{
			{Name: "total_impressions"},
		},
		TimeRange: &metricsview.TimeRange{
			Start: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			End:   time.Date(2024, 4, 1, 0, 0, 0, 0, time.UTC),
		},
	}

	result := e.rewriteQueryForRollup(qry)
	require.NotNil(t, result)
	require.NotNil(t, result.spec)
	require.Equal(t, "daily_rollup", result.spec.Table)
}

func TestRewriteQueryForRollup_ComparisonTimeRange(t *testing.T) {
	e := &Executor{
		metricsView: &runtimev1.MetricsViewSpec{
			Table: "base_table",
			Measures: []*runtimev1.MetricsViewSpec_Measure{
				{Name: "total_impressions", Expression: `SUM("impressions")`},
			},
			Rollups: []*runtimev1.MetricsViewSpec_RollupTable{
				{
					Table:     "daily_rollup",
					TimeGrain: runtimev1.TimeGrain_TIME_GRAIN_DAY,
					Measures: []*runtimev1.MetricsViewSpec_RollupMeasure{
						{Name: "total_impressions", Expression: `SUM("impressions_sum")`},
					},
				},
			},
		},
	}

	qry := &metricsview.Query{
		Measures: []metricsview.Measure{
			{Name: "total_impressions"},
		},
		ComparisonTimeRange: &metricsview.TimeRange{
			Start: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
			End:   time.Date(2023, 2, 1, 0, 0, 0, 0, time.UTC),
		},
	}

	result := e.rewriteQueryForRollup(qry)
	require.Nil(t, result)
}

func TestRewriteQueryForRollup_DerivedMeasure(t *testing.T) {
	e := &Executor{
		metricsView: &runtimev1.MetricsViewSpec{
			Table: "base_table",
			Measures: []*runtimev1.MetricsViewSpec_Measure{
				{Name: "total_impressions", Expression: `SUM("impressions")`, Type: runtimev1.MetricsViewSpec_MEASURE_TYPE_SIMPLE},
				{Name: "derived_measure", Expression: `total_impressions * 2`, Type: runtimev1.MetricsViewSpec_MEASURE_TYPE_DERIVED},
			},
			Rollups: []*runtimev1.MetricsViewSpec_RollupTable{
				{
					Table:     "daily_rollup",
					TimeGrain: runtimev1.TimeGrain_TIME_GRAIN_DAY,
					Measures: []*runtimev1.MetricsViewSpec_RollupMeasure{
						{Name: "total_impressions", Expression: `SUM("impressions_sum")`},
					},
				},
			},
		},
	}

	qry := &metricsview.Query{
		Measures: []metricsview.Measure{
			{Name: "derived_measure"},
		},
	}

	result := e.rewriteQueryForRollup(qry)
	require.Nil(t, result)
}

func TestRewriteQueryForRollup_NoTimeGrainQuery(t *testing.T) {
	// Query without time grain (pure aggregation) should still match rollup
	e := &Executor{
		metricsView: &runtimev1.MetricsViewSpec{
			Table: "base_table",
			Dimensions: []*runtimev1.MetricsViewSpec_Dimension{
				{Name: "publisher", Column: "publisher"},
			},
			Measures: []*runtimev1.MetricsViewSpec_Measure{
				{Name: "total_impressions", Expression: `SUM("impressions")`},
			},
			Rollups: []*runtimev1.MetricsViewSpec_RollupTable{
				{
					Table:      "daily_rollup",
					TimeGrain:  runtimev1.TimeGrain_TIME_GRAIN_DAY,
					Dimensions: []string{"publisher"},
					Measures: []*runtimev1.MetricsViewSpec_RollupMeasure{
						{Name: "total_impressions", Expression: `SUM("impressions_sum")`},
					},
				},
			},
		},
	}

	qry := &metricsview.Query{
		Dimensions: []metricsview.Dimension{
			{Name: "publisher"},
		},
		Measures: []metricsview.Measure{
			{Name: "total_impressions"},
		},
	}

	result := e.rewriteQueryForRollup(qry)
	require.NotNil(t, result)
	require.NotNil(t, result.spec)
	require.Equal(t, "daily_rollup", result.spec.Table)
}

func TestCollectWhereDimensions(t *testing.T) {
	expr := &metricsview.Expression{
		Condition: &metricsview.Condition{
			Operator: metricsview.OperatorAnd,
			Expressions: []*metricsview.Expression{
				{
					Condition: &metricsview.Condition{
						Operator: metricsview.OperatorEq,
						Expressions: []*metricsview.Expression{
							{Name: "publisher"},
							{Value: "google"},
						},
					},
				},
				{
					Condition: &metricsview.Condition{
						Operator: metricsview.OperatorIn,
						Expressions: []*metricsview.Expression{
							{Name: "domain"},
							{Value: "a.com"},
							{Value: "b.com"},
						},
					},
				},
			},
		},
	}

	dims := collectWhereDimensions(expr)
	require.True(t, dims["publisher"])
	require.True(t, dims["domain"])
	require.Len(t, dims, 2)
}

func TestCollectWhereDimensions_Nil(t *testing.T) {
	dims := collectWhereDimensions(nil)
	require.Empty(t, dims)
}

func TestRewriteQueryForRollup_TimezoneMatching(t *testing.T) {
	baseMV := func(rollupTZ string) *runtimev1.MetricsViewSpec {
		return &runtimev1.MetricsViewSpec{
			Table:         "base_table",
			TimeDimension: "timestamp",
			Measures: []*runtimev1.MetricsViewSpec_Measure{
				{Name: "total_impressions", Expression: `SUM("impressions")`},
			},
			Rollups: []*runtimev1.MetricsViewSpec_RollupTable{
				{
					Table:     "daily_rollup",
					TimeGrain: runtimev1.TimeGrain_TIME_GRAIN_DAY,
					Timezone:  rollupTZ,
					Measures: []*runtimev1.MetricsViewSpec_RollupMeasure{
						{Name: "total_impressions", Expression: `SUM("impressions_sum")`},
					},
				},
			},
		}
	}

	baseQuery := func(tz string) *metricsview.Query {
		return &metricsview.Query{
			Measures: []metricsview.Measure{
				{Name: "total_impressions"},
			},
			TimeZone: tz,
			TimeRange: &metricsview.TimeRange{
				Start: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
				End:   time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC),
			},
		}
	}

	t.Run("day rollup UTC, query tz New York: falls back", func(t *testing.T) {
		e := &Executor{metricsView: baseMV("")}
		result := e.rewriteQueryForRollup(baseQuery("America/New_York"))
		require.Nil(t, result)
	})

	t.Run("day rollup New York, query tz New York: routes", func(t *testing.T) {
		e := &Executor{metricsView: baseMV("America/New_York")}
		// Use time range aligned to New York day boundaries
		ny, _ := time.LoadLocation("America/New_York")
		qry := baseQuery("America/New_York")
		qry.TimeRange = &metricsview.TimeRange{
			Start: time.Date(2024, 1, 1, 0, 0, 0, 0, ny),
			End:   time.Date(2024, 2, 1, 0, 0, 0, 0, ny),
		}
		result := e.rewriteQueryForRollup(qry)
		require.NotNil(t, result)
		require.NotNil(t, result.spec)
		require.Equal(t, "daily_rollup", result.spec.Table)
	})

	t.Run("day rollup unset, query tz UTC: routes", func(t *testing.T) {
		e := &Executor{metricsView: baseMV("")}
		result := e.rewriteQueryForRollup(baseQuery("UTC"))
		require.NotNil(t, result)
		require.NotNil(t, result.spec)
		require.Equal(t, "daily_rollup", result.spec.Table)
	})

	t.Run("day rollup unset, query tz empty: routes", func(t *testing.T) {
		e := &Executor{metricsView: baseMV("")}
		result := e.rewriteQueryForRollup(baseQuery(""))
		require.NotNil(t, result)
		require.NotNil(t, result.spec)
		require.Equal(t, "daily_rollup", result.spec.Table)
	})

	t.Run("hour rollup, query tz New York: routes (sub-day safe)", func(t *testing.T) {
		mv := baseMV("")
		mv.Rollups[0].TimeGrain = runtimev1.TimeGrain_TIME_GRAIN_HOUR
		e := &Executor{metricsView: mv}
		qry := baseQuery("America/New_York")
		// Align to hour boundaries
		qry.TimeRange = &metricsview.TimeRange{
			Start: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			End:   time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC),
		}
		result := e.rewriteQueryForRollup(qry)
		require.NotNil(t, result)
		require.NotNil(t, result.spec)
		require.Equal(t, "daily_rollup", result.spec.Table)
	})
}

func TestNormalizeTimezone(t *testing.T) {
	require.Equal(t, "UTC", normalizeTimezone(""))
	require.Equal(t, "UTC", normalizeTimezone("UTC"))
	require.Equal(t, "UTC", normalizeTimezone("Etc/UTC"))
	require.Equal(t, "UTC", normalizeTimezone("utc"))
	require.Equal(t, "America/New_York", normalizeTimezone("America/New_York"))
}

func TestRewriteQueryForRollup_TimeColumn(t *testing.T) {
	e := &Executor{
		metricsView: &runtimev1.MetricsViewSpec{
			Table:         "base_table",
			TimeDimension: "timestamp",
			Dimensions: []*runtimev1.MetricsViewSpec_Dimension{
				{Name: "publisher", Column: "publisher"},
			},
			Measures: []*runtimev1.MetricsViewSpec_Measure{
				{Name: "total_impressions", Expression: `SUM("impressions")`},
			},
			Rollups: []*runtimev1.MetricsViewSpec_RollupTable{
				{
					Table:      "daily_rollup",
					TimeGrain:  runtimev1.TimeGrain_TIME_GRAIN_DAY,
					TimeColumn: "day_ts",
					Dimensions: []string{"publisher"},
					Measures: []*runtimev1.MetricsViewSpec_RollupMeasure{
						{Name: "total_impressions", Expression: `SUM("impressions_sum")`},
					},
				},
			},
		},
	}

	qry := &metricsview.Query{
		Dimensions: []metricsview.Dimension{
			{Name: "publisher"},
		},
		Measures: []metricsview.Measure{
			{Name: "total_impressions"},
		},
		TimeRange: &metricsview.TimeRange{
			Start: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			End:   time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC),
		},
	}

	result := e.rewriteQueryForRollup(qry)
	require.NotNil(t, result)
	require.NotNil(t, result.spec)
	require.Equal(t, "daily_rollup", result.spec.Table)

	// Verify the time dimension column was overridden
	var timeDim *runtimev1.MetricsViewSpec_Dimension
	for _, d := range result.spec.Dimensions {
		if d.Name == "timestamp" {
			timeDim = d
			break
		}
	}
	require.NotNil(t, timeDim)
	require.Equal(t, "day_ts", timeDim.Column)
	require.Empty(t, timeDim.Expression)
}

func TestRewriteQueryForRollup_TimeColumnImplicitTimeDim(t *testing.T) {
	// Time dimension is not in the dimensions list; verify a new entry is added
	e := &Executor{
		metricsView: &runtimev1.MetricsViewSpec{
			Table:         "base_table",
			TimeDimension: "timestamp",
			Dimensions: []*runtimev1.MetricsViewSpec_Dimension{
				{Name: "publisher", Column: "publisher"},
				// No "timestamp" dimension in the list
			},
			Measures: []*runtimev1.MetricsViewSpec_Measure{
				{Name: "total_impressions", Expression: `SUM("impressions")`},
			},
			Rollups: []*runtimev1.MetricsViewSpec_RollupTable{
				{
					Table:      "daily_rollup",
					TimeGrain:  runtimev1.TimeGrain_TIME_GRAIN_DAY,
					TimeColumn: "day_ts",
					Dimensions: []string{"publisher"},
					Measures: []*runtimev1.MetricsViewSpec_RollupMeasure{
						{Name: "total_impressions", Expression: `SUM("impressions_sum")`},
					},
				},
			},
		},
	}

	qry := &metricsview.Query{
		Dimensions: []metricsview.Dimension{
			{Name: "publisher"},
		},
		Measures: []metricsview.Measure{
			{Name: "total_impressions"},
		},
	}

	result := e.rewriteQueryForRollup(qry)
	require.NotNil(t, result)
	require.NotNil(t, result.spec)

	// Verify a new dimension entry was added for the time dimension
	var timeDim *runtimev1.MetricsViewSpec_Dimension
	for _, d := range result.spec.Dimensions {
		if d.Name == "timestamp" {
			timeDim = d
			break
		}
	}
	require.NotNil(t, timeDim)
	require.Equal(t, "day_ts", timeDim.Column)
}

func TestRewriteQueryForRollup_Projection(t *testing.T) {
	e := &Executor{
		metricsView: &runtimev1.MetricsViewSpec{
			Table:         "events",
			TimeDimension: "timestamp",
			Dimensions: []*runtimev1.MetricsViewSpec_Dimension{
				{Name: "publisher", Column: "publisher"},
			},
			Measures: []*runtimev1.MetricsViewSpec_Measure{
				{Name: "total_impressions", Expression: `SUM("impressions")`},
			},
			Rollups: []*runtimev1.MetricsViewSpec_RollupTable{
				{
					IsProjection: true,
					TimeGrain:    runtimev1.TimeGrain_TIME_GRAIN_DAY,
					Dimensions:   []string{"publisher"},
					Measures: []*runtimev1.MetricsViewSpec_RollupMeasure{
						{Name: "total_impressions", Expression: `SUM("impressions")`},
					},
				},
			},
		},
	}

	qry := &metricsview.Query{
		Dimensions: []metricsview.Dimension{
			{Name: "publisher"},
			{Compute: &metricsview.DimensionCompute{TimeFloor: &metricsview.DimensionComputeTimeFloor{Dimension: "timestamp", Grain: metricsview.TimeGrainDay}}},
		},
		Measures: []metricsview.Measure{
			{Name: "total_impressions"},
		},
		TimeRange: &metricsview.TimeRange{
			Start: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			End:   time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC),
		},
	}

	result := e.rewriteQueryForRollup(qry)
	require.NotNil(t, result)
	// Projection: no synthetic spec, but timeFilterGrain is set
	require.Nil(t, result.spec)
	require.Equal(t, runtimev1.TimeGrain_TIME_GRAIN_DAY, result.timeFilterGrain)
}

func TestRewriteQueryForRollup_ProjectionIneligible(t *testing.T) {
	e := &Executor{
		metricsView: &runtimev1.MetricsViewSpec{
			Table:         "events",
			TimeDimension: "timestamp",
			Dimensions: []*runtimev1.MetricsViewSpec_Dimension{
				{Name: "publisher", Column: "publisher"},
				{Name: "domain", Column: "domain"},
			},
			Measures: []*runtimev1.MetricsViewSpec_Measure{
				{Name: "total_impressions", Expression: `SUM("impressions")`},
			},
			Rollups: []*runtimev1.MetricsViewSpec_RollupTable{
				{
					IsProjection: true,
					TimeGrain:    runtimev1.TimeGrain_TIME_GRAIN_DAY,
					Dimensions:   []string{"publisher"}, // no "domain"
					Measures: []*runtimev1.MetricsViewSpec_RollupMeasure{
						{Name: "total_impressions", Expression: `SUM("impressions")`},
					},
				},
			},
		},
	}

	qry := &metricsview.Query{
		Dimensions: []metricsview.Dimension{
			{Name: "domain"}, // not in projection rollup
		},
		Measures: []metricsview.Measure{
			{Name: "total_impressions"},
		},
	}

	result := e.rewriteQueryForRollup(qry)
	require.Nil(t, result)
}
