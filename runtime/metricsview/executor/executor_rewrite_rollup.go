package executor

import (
	"strings"
	"time"

	runtimev1 "github.com/rilldata/rill/proto/gen/rill/runtime/v1"
	"github.com/rilldata/rill/runtime/metricsview"
	"google.golang.org/protobuf/proto"
)

// rollupRewrite holds the result of rewriting a query for a rollup.
// For separate rollup tables, spec is set to the synthetic MetricsViewSpec.
// For projections, timeFilterGrain is set to wrap the WHERE time expression with date_trunc.
type rollupRewrite struct {
	spec            *runtimev1.MetricsViewSpec // non-nil for separate rollup tables
	timeFilterGrain runtimev1.TimeGrain        // non-zero for projections; used to wrap time WHERE with date_trunc
	timeFilterTZ    string                     // timezone for the date_trunc in WHERE (from rollup config)
}

// rewriteQueryForRollup checks if a rollup table or projection can satisfy the query.
// It returns a rollupRewrite with either a synthetic spec (for separate tables)
// or a time filter grain (for projections). Returns nil if no rollup matches.
func (e *Executor) rewriteQueryForRollup(qry *metricsview.Query) *rollupRewrite {
	if len(e.metricsView.Rollups) == 0 {
		return nil
	}

	// Disqualify: raw rows or spine queries
	if qry.Rows || qry.Spine != nil {
		return nil
	}

	// Disqualify: queries with comparison time ranges (future improvement)
	if qry.ComparisonTimeRange != nil {
		return nil
	}

	// Disqualify: queries with derived or comparison measures
	if hasNonSimpleMeasures(e.metricsView, qry) {
		return nil
	}

	// Extract the time grain from the query (from time floor dimensions)
	queryGrain := extractQueryTimeGrain(qry)

	// Extract dimension names from the WHERE clause
	whereDims := collectWhereDimensions(qry.Where)

	// Find the best eligible rollup (prefer coarsest grain)
	var bestRollup *runtimev1.MetricsViewSpec_RollupTable
	var bestGrainOrder int
	grainOrderMap := map[runtimev1.TimeGrain]int{
		runtimev1.TimeGrain_TIME_GRAIN_MILLISECOND: 0,
		runtimev1.TimeGrain_TIME_GRAIN_SECOND:      1,
		runtimev1.TimeGrain_TIME_GRAIN_MINUTE:      2,
		runtimev1.TimeGrain_TIME_GRAIN_HOUR:        3,
		runtimev1.TimeGrain_TIME_GRAIN_DAY:         4,
		runtimev1.TimeGrain_TIME_GRAIN_WEEK:        5,
		runtimev1.TimeGrain_TIME_GRAIN_MONTH:       6,
		runtimev1.TimeGrain_TIME_GRAIN_QUARTER:     7,
		runtimev1.TimeGrain_TIME_GRAIN_YEAR:        8,
	}

	for _, rollup := range e.metricsView.Rollups {
		if rollup.Table == "" && !rollup.IsProjection {
			continue // not yet resolved and not a projection
		}
		if !rollupEligible(rollup, qry, queryGrain, whereDims) {
			continue
		}
		order := grainOrderMap[rollup.TimeGrain]
		if bestRollup == nil || order > bestGrainOrder {
			bestRollup = rollup
			bestGrainOrder = order
		}
	}

	if bestRollup == nil {
		return nil
	}

	if bestRollup.IsProjection {
		return &rollupRewrite{
			timeFilterGrain: bestRollup.TimeGrain,
			timeFilterTZ:    bestRollup.Timezone,
		}
	}

	return &rollupRewrite{spec: buildSyntheticSpec(e.metricsView, bestRollup)}
}

// rollupEligible checks whether a rollup table can satisfy the given query.
func rollupEligible(rollup *runtimev1.MetricsViewSpec_RollupTable, qry *metricsview.Query, queryGrain runtimev1.TimeGrain, whereDims map[string]bool) bool {
	// 1. Grain derivable: if query has a time grain, it must be derivable from rollup grain
	if queryGrain != runtimev1.TimeGrain_TIME_GRAIN_UNSPECIFIED {
		if !metricsview.GrainDerivableFrom(queryGrain, rollup.TimeGrain) {
			return false
		}
	}

	// 2. For day+ rollup grains, the query timezone must match the rollup's timezone.
	// Sub-day grains are timezone-agnostic (hour boundaries are the same everywhere).
	if rollup.TimeGrain >= runtimev1.TimeGrain_TIME_GRAIN_DAY {
		rollupTZ := normalizeTimezone(rollup.Timezone)
		queryTZ := normalizeTimezone(qry.TimeZone)
		if rollupTZ != queryTZ {
			return false
		}
	}

	// 3. Time range aligned to rollup grain (use rollup timezone for alignment)
	if qry.TimeRange != nil {
		rollupLoc := time.UTC
		if rollup.Timezone != "" {
			if loc, err := time.LoadLocation(rollup.Timezone); err == nil {
				rollupLoc = loc
			}
		}
		if !metricsview.TimeRangeAligned(qry.TimeRange.Start, qry.TimeRange.End, rollup.TimeGrain, rollupLoc, 0) {
			return false
		}
	}

	// 4. All query dimensions present in rollup
	rollupDims := make(map[string]bool, len(rollup.Dimensions))
	for _, d := range rollup.Dimensions {
		rollupDims[strings.ToLower(d)] = true
	}
	for _, d := range qry.Dimensions {
		name := d.Name
		if d.Compute != nil && d.Compute.TimeFloor != nil {
			// Time floor dimensions reference the underlying time dimension; skip for dimension check
			// (the time dimension column exists in the rollup table as the time column)
			continue
		}
		if !rollupDims[strings.ToLower(name)] {
			return false
		}
	}

	// 5. All queried measures present in rollup
	rollupMeasures := make(map[string]bool, len(rollup.Measures))
	for _, m := range rollup.Measures {
		rollupMeasures[strings.ToLower(m.Name)] = true
	}
	for _, m := range qry.Measures {
		if m.Compute != nil {
			// Computed measures (comparison, percent_of_total, etc.) are not supported
			return false
		}
		if !rollupMeasures[strings.ToLower(m.Name)] {
			return false
		}
	}

	// 6. All WHERE dimensions present in rollup
	for dim := range whereDims {
		if !rollupDims[strings.ToLower(dim)] {
			return false
		}
	}

	return true
}

// hasNonSimpleMeasures returns true if the query references derived or comparison measures.
func hasNonSimpleMeasures(mv *runtimev1.MetricsViewSpec, qry *metricsview.Query) bool {
	for _, qm := range qry.Measures {
		if qm.Compute != nil {
			return true
		}
		// Look up measure type in spec
		for _, specM := range mv.Measures {
			if strings.EqualFold(specM.Name, qm.Name) {
				if specM.Type != runtimev1.MetricsViewSpec_MEASURE_TYPE_SIMPLE && specM.Type != runtimev1.MetricsViewSpec_MEASURE_TYPE_UNSPECIFIED {
					return true
				}
				break
			}
		}
	}
	return false
}

// extractQueryTimeGrain finds the time grain from the query's dimensions.
// It returns the grain from the first time floor dimension found, or UNSPECIFIED.
func extractQueryTimeGrain(qry *metricsview.Query) runtimev1.TimeGrain {
	for _, d := range qry.Dimensions {
		if d.Compute != nil && d.Compute.TimeFloor != nil {
			return d.Compute.TimeFloor.Grain.ToProto()
		}
	}
	return runtimev1.TimeGrain_TIME_GRAIN_UNSPECIFIED
}

// collectWhereDimensions recursively collects dimension names referenced in a WHERE expression.
func collectWhereDimensions(expr *metricsview.Expression) map[string]bool {
	dims := make(map[string]bool)
	collectWhereDimensionsRec(expr, dims)
	return dims
}

func collectWhereDimensionsRec(expr *metricsview.Expression, dims map[string]bool) {
	if expr == nil {
		return
	}
	if expr.Name != "" {
		dims[expr.Name] = true
	}
	if expr.Condition != nil {
		for _, sub := range expr.Condition.Expressions {
			collectWhereDimensionsRec(sub, dims)
		}
	}
	if expr.Subquery != nil {
		if expr.Subquery.Dimension.Name != "" {
			dims[expr.Subquery.Dimension.Name] = true
		}
		collectWhereDimensionsRec(expr.Subquery.Where, dims)
		collectWhereDimensionsRec(expr.Subquery.Having, dims)
	}
}

// buildSyntheticSpec creates a MetricsViewSpec that points to the rollup table
// with rewritten measure expressions using rollup functions.
func buildSyntheticSpec(original *runtimev1.MetricsViewSpec, rollup *runtimev1.MetricsViewSpec_RollupTable) *runtimev1.MetricsViewSpec {
	synth := proto.Clone(original).(*runtimev1.MetricsViewSpec)

	// Point to rollup table
	synth.Table = rollup.Table
	synth.Model = ""
	if rollup.Connector != "" {
		synth.Connector = rollup.Connector
	}
	if rollup.Database != "" {
		synth.Database = rollup.Database
	}
	if rollup.DatabaseSchema != "" {
		synth.DatabaseSchema = rollup.DatabaseSchema
	}

	// Clear rollups to prevent recursion
	synth.Rollups = nil

	// Build a map from measure name -> rollup measure config
	rollupMeasureMap := make(map[string]*runtimev1.MetricsViewSpec_RollupMeasure, len(rollup.Measures))
	for _, rm := range rollup.Measures {
		rollupMeasureMap[strings.ToLower(rm.Name)] = rm
	}

	// Rewrite measure expressions
	for _, m := range synth.Measures {
		rm, ok := rollupMeasureMap[strings.ToLower(m.Name)]
		if !ok {
			continue
		}
		m.Expression = rm.Expression
	}

	// Override the time dimension column if the rollup specifies a different one
	if rollup.TimeColumn != "" && synth.TimeDimension != "" {
		found := false
		for _, d := range synth.Dimensions {
			if d.Name == synth.TimeDimension {
				d.Column = rollup.TimeColumn
				d.Expression = ""
				found = true
				break
			}
		}
		if !found {
			// Time dimension not explicitly in dimensions list; add explicit entry so override takes effect.
			synth.Dimensions = append(synth.Dimensions, &runtimev1.MetricsViewSpec_Dimension{
				Name:   synth.TimeDimension,
				Column: rollup.TimeColumn,
			})
		}
	}

	return synth
}

// normalizeTimezone returns a canonical timezone string for comparison.
// Empty, "UTC", and "Etc/UTC" are all treated as equivalent.
func normalizeTimezone(tz string) string {
	if tz == "" || strings.EqualFold(tz, "UTC") || strings.EqualFold(tz, "Etc/UTC") {
		return "UTC"
	}
	return tz
}
