package profiles

import (
	"context"
	"fmt"

	"github.com/openbao/openbao/sdk/v2/framework"
	"github.com/openbao/openbao/sdk/v2/logical"
)

// InputSource is fully dynamic requiring context outside of the profile
// engine to invoke. Each defined field from the schema gets added to the
// framework data and so will be validated and used with GetOk(...) for
// processing.
//
// This source requires the following parameters:
//
// - field_name, the name of the input field to read.
// - missing_ok, whether it is OK to be missing the field.
func WithInputSource(config *InputConfig, request *logical.Request, data *framework.FieldData) func(*ProfileEngine) {
	return func(p *ProfileEngine) {
		p.input = config
		p.request = request
		p.data = data

		p.sourceBuilders["input"] = func(ctx context.Context, engine *ProfileEngine, field map[string]interface{}) Source {
			return &InputSource{
				config:  config,
				request: request,
				data:    data,

				field: field,
			}
		}
	}
}

type InputSource struct {
	config  *InputConfig
	request *logical.Request
	data    *framework.FieldData
	field   map[string]interface{}

	fieldName string
	missingOk bool
}

var _ Source = &InputSource{}

func (s *InputSource) Validate(_ context.Context) ([]string, []string, error) {
	rawFieldName, present := s.field["field_name"]
	if !present {
		return nil, nil, fmt.Errorf("input source is missing required field %q", "field_name")
	}

	fieldName, ok := rawFieldName.(string)
	if !ok {
		return nil, nil, fmt.Errorf("field 'field_name' is of wrong type: expected 'string' go '%T'", rawFieldName)
	}

	if _, present := s.data.Schema[fieldName]; !present {
		return nil, nil, fmt.Errorf("referenced field %q is missing from schema", fieldName)
	}

	s.fieldName = fieldName

	rawMissingOk, present := s.field["missing_ok"]
	if !present {
		return nil, nil, fmt.Errorf("input source is missing required field %q", "missing_ok")
	}

	missingOk, ok := rawMissingOk.(bool)
	if !ok {
		return nil, nil, fmt.Errorf("field 'missing_ok' is of wrong type: expected 'bool' go '%T'", rawFieldName)
	}

	s.missingOk = missingOk

	return nil, nil, nil
}

func (s *InputSource) Evaluate(_ context.Context, eh *EvaluationHistory) (interface{}, error) {
	value, ok := s.data.GetOk(s.fieldName)
	if !ok && !s.missingOk {
		return nil, fmt.Errorf("missing required field %q", s.fieldName)
	}

	return value, nil
}

func (s *InputSource) Close(_ context.Context) error {
	return nil
}
