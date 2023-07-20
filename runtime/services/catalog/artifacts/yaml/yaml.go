// Package yaml reads and writes artifacts that exactly mirror the internal representation
package yaml

import (
	"context"
	"errors"
	runtimev1 "github.com/rilldata/rill/proto/gen/rill/runtime/v1"
	"github.com/rilldata/rill/runtime/pkg/fileutil"
	"path/filepath"

	"github.com/rilldata/rill/runtime/drivers"
	"github.com/rilldata/rill/runtime/services/catalog/artifacts"
	"gopkg.in/yaml.v3"
)

type artifact struct{}

var ErrNotSupported = errors.New("yaml only supported for sources and dashboards")

func init() {
	artifacts.Register(".yaml", &artifact{})
}

func (r *artifact) DeSerialise(ctx context.Context, filePath, blob string, materializeDefault bool) (*drivers.CatalogEntry, error) {
	dir := filepath.Base(filepath.Dir(filePath))
	switch dir {
	case "sources":
		source := &Source{}
		err := yaml.Unmarshal([]byte(blob), &source)
		if err != nil {
			return nil, err
		}
		return fromSourceArtifact(source, filePath)
	case "dashboards":
		metrics := &MetricsView{}
		err := yaml.Unmarshal([]byte(blob), &metrics)
		if err != nil {
			return nil, err
		}
		return fromMetricsViewArtifact(metrics, filePath)
	case "models":
		modelMeta := &ModelMeta{}
		err := yaml.Unmarshal([]byte(blob), &modelMeta)
		if err != nil {
			return nil, err
		}
		return fromModelMetaArtifact(modelMeta, filePath)
	}

	return nil, ErrNotSupported
}

func fromModelMetaArtifact(meta *ModelMeta, filePath string) (*drivers.CatalogEntry, error) {
	name := fileutil.Stem(filePath)
	columnAccess := make([]*runtimev1.ColumnAccess, len(meta.Access.Columns))
	for i, column := range meta.Access.Columns {
		columnAccess[i] = &runtimev1.ColumnAccess{
			Name:      column.Name,
			Condition: column.Condition,
			Include:   column.Include,
		}
	}
	return &drivers.CatalogEntry{
		Type: drivers.ObjectTypeModelMeta,
		Object: &runtimev1.ModelMeta{
			Access: &runtimev1.ModelAccess{
				Condition: meta.Access.Condition,
				Filter:    meta.Access.Filter,
				Columns:   columnAccess,
			},
		},
		Name: name + "_meta",
		Path: filePath,
	}, nil
}

func (r *artifact) Serialise(ctx context.Context, catalogObject *drivers.CatalogEntry) (string, error) {
	switch catalogObject.Type {
	case drivers.ObjectTypeSource:
		source, err := toSourceArtifact(catalogObject)
		if err != nil {
			return "", err
		}
		out, err := yaml.Marshal(source)
		if err != nil {
			return "", err
		}
		return string(out), nil
	case drivers.ObjectTypeMetricsView:
		metrics, err := toMetricsViewArtifact(catalogObject)
		if err != nil {
			return "", err
		}
		out, err := yaml.Marshal(metrics)
		if err != nil {
			return "", err
		}
		return string(out), nil
	}

	return "", ErrNotSupported
}
