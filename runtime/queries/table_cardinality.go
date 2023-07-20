package queries

import (
	"context"
	"fmt"
	"github.com/rilldata/rill/runtime/server"
	"github.com/rilldata/rill/runtime/server/auth"
	"io"
	"reflect"

	runtimev1 "github.com/rilldata/rill/proto/gen/rill/runtime/v1"
	"github.com/rilldata/rill/runtime"
	"github.com/rilldata/rill/runtime/drivers"
)

type TableCardinality struct {
	TableName string
	Result    int64
}

var _ runtime.Query = &TableCardinality{}

func (q *TableCardinality) Key() string {
	return fmt.Sprintf("TableCardinality:%s", q.TableName)
}

func (q *TableCardinality) Deps() []string {
	return []string{q.TableName}
}

func (q *TableCardinality) MarshalResult() *runtime.QueryResult {
	return &runtime.QueryResult{
		Value: q.Result,
		Bytes: int64(reflect.TypeOf(q.Result).Size()),
	}
}

func (q *TableCardinality) UnmarshalResult(v any) error {
	res, ok := v.(int64)
	if !ok {
		return fmt.Errorf("TableCardinality: mismatched unmarshal input")
	}
	q.Result = res
	return nil
}

func (q *TableCardinality) Resolve(ctx context.Context, rt *runtime.Runtime, instanceID string, priority int) error {
	return q.resolve(ctx, rt, instanceID, priority, "")
}

func (q *TableCardinality) ResolveRestricted(ctx context.Context, rt *runtime.Runtime, instanceID string, priority int) error {
	if auth.GetClaims(ctx).IsRestrictedRole() {
		// Check if the backing model has access policy

		modelMeta, err := runtime.LookupModelMeta(ctx, rt, instanceID, q.TableName+"_meta")
		if err != nil {
			return err
		}

		evaluatedModel := auth.GetClaims(ctx).Evaluate(modelMeta, "restricted")

		// role should come from the runtime request
		if !evaluatedModel.Access {
			return server.ErrForbidden
		}
		return q.resolve(ctx, rt, instanceID, priority, evaluatedModel.Filter)
	}
	return q.resolve(ctx, rt, instanceID, priority, "")
}

func (q *TableCardinality) resolve(ctx context.Context, rt *runtime.Runtime, instanceID string, priority int, filter string) error {
	if filter != "" {
		filter = " WHERE " + filter
	}
	countSQL := fmt.Sprintf("SELECT count(*) AS count FROM %s %s",
		safeName(q.TableName),
		filter,
	)

	olap, err := rt.OLAP(ctx, instanceID)
	if err != nil {
		return err
	}

	if olap.Dialect() != drivers.DialectDuckDB {
		return fmt.Errorf("not available for dialect '%s'", olap.Dialect())
	}

	rows, err := olap.Execute(ctx, &drivers.Statement{
		Query:            countSQL,
		Priority:         priority,
		ExecutionTimeout: defaultExecutionTimeout,
	})
	if err != nil {
		return err
	}
	defer rows.Close()

	var count int64
	for rows.Next() {
		err = rows.Scan(&count)
		if err != nil {
			return err
		}
	}

	err = rows.Err()
	if err != nil {
		return err
	}

	q.Result = count
	return nil
}

func (q *TableCardinality) Export(ctx context.Context, rt *runtime.Runtime, instanceID string, priority int, format runtimev1.ExportFormat, w io.Writer) error {
	return ErrExportNotSupported
}
