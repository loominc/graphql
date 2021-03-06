package graphql

import (
	"context"

	"github.com/graphql-go/graphql/gqlerrors"
	"github.com/graphql-go/graphql/language/parser"
	"github.com/graphql-go/graphql/language/source"
	opentracing "github.com/opentracing/opentracing-go"
)

type Params struct {
	// The GraphQL type system to use when validating and executing a query.
	Schema Schema

	// A GraphQL language formatted string representing the requested operation.
	RequestString string

	// The value provided as the first argument to resolver functions on the top
	// level type (e.g. the query object type).
	RootObject map[string]interface{}

	// A mapping of variable name to runtime value to use for all variables
	// defined in the requestString.
	VariableValues map[string]interface{}

	// The name of the operation to use if requestString contains multiple
	// possible operations. Can be omitted if requestString contains only
	// one operation.
	OperationName string

	// Context may be provided to pass application-specific per-request
	// information to resolve functions.
	Context context.Context

	// PanicHandler will be called if any of the resolvers or mutations panic
	PanicHandler func(ctx context.Context, err interface{})

	// Executor allows to control the behavior of how to perform resolving function that
	// can be run concurrently. If not given, they will be executed serially.
	Executor Executor
}

func Do(p Params) *Result {
	source := source.NewSource(&source.Source{
		Body: []byte(p.RequestString),
		Name: "GraphQL request",
	})
	var span opentracing.Span
	if p.Context != nil {
		span, _ = opentracing.StartSpanFromContext(p.Context, "GraphQL Parsing")
	}
	AST, err := parser.Parse(parser.ParseParams{Source: source})
	if span != nil {
		span.Finish()
	}
	if err != nil {
		return &Result{
			Errors: gqlerrors.FormatErrors(err),
		}
	}

	if p.Context != nil {
		span, _ = opentracing.StartSpanFromContext(p.Context, "GraphQL Validation")
	}
	validationResult := ValidateDocument(&p.Schema, AST, nil)
	if span != nil {
		span.Finish()
	}

	if !validationResult.IsValid {
		return &Result{
			Errors: validationResult.Errors,
		}
	}

	return Execute(ExecuteParams{
		Schema:        p.Schema,
		Root:          p.RootObject,
		AST:           AST,
		OperationName: p.OperationName,
		Args:          p.VariableValues,
		Context:       p.Context,
		PanicHandler:  p.PanicHandler,
		Executor:      p.Executor,
	})
}
