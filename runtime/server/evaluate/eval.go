package evaluate

import (
	"bytes"
	"github.com/Knetic/govaluate"
	"github.com/Masterminds/sprig/v3"
	"text/template"
)

func Expr(s string) (bool, error) {
	expression, err := govaluate.NewEvaluableExpression(s)
	if err != nil {
		return false, err
	}
	res, err := expression.Evaluate(nil)
	if err != nil {
		return false, err
	}
	return res.(bool), nil
}

func Template(condition string, attrs map[string]string) (string, error) {
	// replace attributes in the condition with role attributes
	funcMap := sprig.TxtFuncMap()
	t, err := template.New("source").Funcs(funcMap).Option("missingkey=error").Parse(condition)
	if err != nil {
		return "", err
	}

	bw := new(bytes.Buffer)
	if err := t.Execute(bw, attrs); err != nil {
		return "", err
	}
	return bw.String(), nil
}

type EvaluatedModel struct {
	Access         bool
	Filter         string
	IncludeColumns []string
	ExcludeColumns []string
}
