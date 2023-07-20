package queries

import (
	"context"
	"fmt"
	"github.com/rilldata/rill/runtime/server"
	"github.com/rilldata/rill/runtime/server/auth"
	"io"

	runtimev1 "github.com/rilldata/rill/proto/gen/rill/runtime/v1"
	"github.com/rilldata/rill/runtime"
	"github.com/rilldata/rill/runtime/drivers"
	"google.golang.org/protobuf/types/known/structpb"
)

type TableHead struct {
	TableName string
	Limit     int
	Result    []*structpb.Struct
}

var _ runtime.Query = &TableHead{}

func (q *TableHead) Key() string {
	return fmt.Sprintf("TableHead:%s:%d", q.TableName, q.Limit)
}

func (q *TableHead) Deps() []string {
	return []string{q.TableName}
}

func (q *TableHead) MarshalResult() *runtime.QueryResult {
	var size int64
	if len(q.Result) > 0 {
		// approx
		size = sizeProtoMessage(q.Result[0]) * int64(len(q.Result))
	}

	return &runtime.QueryResult{
		Value: q.Result,
		Bytes: size,
	}
}

func (q *TableHead) UnmarshalResult(v any) error {
	res, ok := v.([]*structpb.Struct)
	if !ok {
		return fmt.Errorf("TableHead: mismatched unmarshal input")
	}
	q.Result = res
	return nil
}

func (q *TableHead) Resolve(ctx context.Context, rt *runtime.Runtime, instanceID string, priority int) error {
	return q.resolve(ctx, rt, instanceID, priority, "")
}

func (q *TableHead) ResolveRestricted(ctx context.Context, rt *runtime.Runtime, instanceID string, priority int) error {
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

func (q *TableHead) resolve(ctx context.Context, rt *runtime.Runtime, instanceID string, priority int, filter string) error {
	olap, err := rt.OLAP(ctx, instanceID)
	if err != nil {
		return err
	}

	if olap.Dialect() != drivers.DialectDuckDB {
		return fmt.Errorf("not available for dialect '%s'", olap.Dialect())
	}

	if filter != "" {
		filter = "WHERE " + filter
	}

	rows, err := olap.Execute(ctx, &drivers.Statement{
		Query:            fmt.Sprintf("SELECT * FROM %s %s LIMIT %d", safeName(q.TableName), filter, q.Limit),
		Priority:         priority,
		ExecutionTimeout: defaultExecutionTimeout,
	})
	if err != nil {
		return err
	}
	defer rows.Close()

	data, err := rowsToData(rows)
	if err != nil {
		return err
	}

	q.Result = data
	return nil
}

func (q *TableHead) Export(ctx context.Context, rt *runtime.Runtime, instanceID string, priority int, format runtimev1.ExportFormat, w io.Writer) error {
	return ErrExportNotSupported
}
