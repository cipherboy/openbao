package profiles

import (
	"context"
	"errors"
	"fmt"

	"github.com/openbao/openbao/sdk/v2/helper/cel"
)

// CELSourceBuilder allows reading inputs from the CEL engine,
// allowing for more advanced operations than the template engine.
// This includes reading environment variables and files.
//
// Fields:
//
//   - expression (string): expression to evaluate
//   - variables([]map[string]string): additional variables to evaluate;
//     each entry in the list is a map string->string which has two keys:
//   - name, the name of the variable to inject into the CEL context,
//   - expression, the expression for this variable to equal.
//     When allowed as sources, the CEL context already includes:
//   - requests
//   - responses
//   - input
func CELSourceBuilder(ctx context.Context, engine *ProfileEngine, field map[string]interface{}) Source {
	return &CELSource{
		engine: engine,
		field:  field,
	}
}

var _ SourceBuilder = CELSourceBuilder

func WithCELSource() func(*ProfileEngine) {
	return func(p *ProfileEngine) {
		p.sourceBuilders["cel"] = CELSourceBuilder
	}
}

type CELSource struct {
	engine *ProfileEngine
	field  map[string]interface{}

	program cel.Program
}

var _ Source = &CELSource{}

func (s *CELSource) getConfig() *cel.EvalConfig {
	return &cel.EvalConfig{
		WithExtLib: true,
	}
}

func (s *CELSource) Validate(_ context.Context) ([]string, []string, error) {
	rawExpr, present := s.field["expression"]
	if !present {
		return nil, nil, errors.New("cel source is missing required field 'expression'")
	}

	expr, ok := rawExpr.(string)
	if !ok {
		return nil, nil, fmt.Errorf("field 'expression' is of wrong type: expected 'string' got '%T'", rawExpr)
	}

	rawVariables, present := s.field["variables"]
	if !present {
		rawVariables = []interface{}{}
	}

	listVariables, ok := rawVariables.([]interface{})
	if !ok {
		return nil, nil, fmt.Errorf("field 'variables' is of wrong outer type: expected '[]interface{}' got '%T'", listVariables)
	}

	var variables []cel.Variable
	for index, rawVariableMap := range listVariables {
		variableMap, ok := rawVariableMap.(map[string]interface{})
		if !ok {
			return nil, nil, fmt.Errorf("field 'variables[%d]' is of wrong inner type: expected 'map[string]interface{}' got '%T'", index, listVariables)
		}

		rawName, present := variableMap["name"]
		if !present {
			return nil, nil, fmt.Errorf("field 'variables[%d].name' is missing", index)
		}
		delete(variableMap, "name")

		name, ok := rawName.(string)
		if !ok {
			return nil, nil, fmt.Errorf("field 'variables[%d].name' is of wrong type: expected 'string' got '%T'", index, rawName)
		}

		rawExpression, present := variableMap["expression"]
		if !present {
			return nil, nil, fmt.Errorf("field 'variables[%d].expression' is missing", index)
		}
		delete(variableMap, "expression")

		expression, ok := rawExpression.(string)
		if !ok {
			return nil, nil, fmt.Errorf("field 'variables[%d].expression' is of wrong type: expected 'string' got '%T'", index, rawExpression)
		}

		if len(variableMap) > 0 {
			return nil, nil, fmt.Errorf("field 'variables[%d].name' has extraneous elements besides 'name' and 'expression'", index)
		}

		variables = append(variables, cel.Variable{
			Name:       name,
			Expression: expression,
		})
	}

	program := cel.Program{
		Variables:  variables,
		Expression: expr,
	}

	if err := program.Validate(s.getConfig()); err != nil {
		return nil, nil, fmt.Errorf("CEL source failed to validate program: %w", err)
	}

	s.program = program

	return nil, nil, nil
}

func (s *CELSource) Evaluate(ctx context.Context, eh *EvaluationHistory) (interface{}, error) {
	var data map[string]interface{}

	result, err := s.program.Evaluate(ctx, s.getConfig(), data)
	if err != nil {
		return nil, fmt.Errorf("CEL source failed to evaluate: %w", err)
	}

	return result.Value(), nil
}

func (s *CELSource) Close(_ context.Context) error {
	return nil
}
