package profiles

import (
	"context"
	"errors"
	"fmt"

	"github.com/openbao/openbao/sdk/v2/helper/template"
)

// TemplateSourceBuilder allows reading inputs from the text/template engine,
// allowing for string interpolation and other operations.
//
// Fields:
//
//   - template (string): template to evaluate
//   - data (map[string]interface{}): additional context for the templating
//     engine. When allowed as sources, this already includes:
//   - requests
//   - responses
//   - input
//     but additional context may be added manually.
func TemplateSourceBuilder(ctx context.Context, engine *ProfileEngine, field map[string]interface{}) Source {
	return &TemplateSource{
		engine: engine,
		field:  field,
	}
}

var _ SourceBuilder = TemplateSourceBuilder

func WithTemplateSource() func(*ProfileEngine) {
	return func(p *ProfileEngine) {
		p.sourceBuilders["template"] = TemplateSourceBuilder
	}
}

type TemplateSource struct {
	engine *ProfileEngine
	field  map[string]interface{}

	data     map[string]interface{}
	template string
}

var _ Source = &TemplateSource{}

func (s *TemplateSource) Validate(_ context.Context) ([]string, []string, error) {
	rawTemplate, present := s.field["template"]
	if !present {
		return nil, nil, errors.New("template source is missing required field 'template'")
	}

	template, ok := rawTemplate.(string)
	if !ok {
		return nil, nil, fmt.Errorf("field 'template' is of wrong type: expected 'string' got '%T'", rawTemplate)
	}

	s.template = template

	rawData, present := s.field["data"]
	if !present {
		rawData = map[string]interface{}{}
	}

	data, ok := rawData.(map[string]interface{})
	if !ok {
		return nil, nil, fmt.Errorf("field 'data' is of wrong type: expected 'map[string]interface{}' got '%T'", rawData)
	}

	s.data = data

	return nil, nil, nil
}

func (s *TemplateSource) Evaluate(_ context.Context, eh *EvaluationHistory) (interface{}, error) {
	templater, err := template.NewTemplate(template.Template(s.template))
	if err != nil {
		return nil, fmt.Errorf("failed to build new templator: %w", err)
	}

	// Inject input data if present as a source.
	if _, ok := s.engine.sourceBuilders["input"]; ok {
		for _, field := range s.engine.input.Fields {
			if _, present := s.engine.data.Raw[field.Name]; !present {
				continue
			}

			s.data["input"] = map[string]interface{}{
				field.Name: s.engine.data.Raw[field.Name],
			}
		}
	}

	// Inject request data if present as a source.
	if _, ok := s.engine.sourceBuilders["request"]; ok {
		s.data["requests"] = eh.Requests
		if s.engine.outerBlockName == "" {
			s.data["requests"] = eh.Requests[""]
		}
	}

	// Inject response data if present as a source.
	if _, ok := s.engine.sourceBuilders["response"]; ok {
		s.data["responses"] = eh.Responses
		if s.engine.outerBlockName == "" {
			s.data["responses"] = eh.Responses[""]
		}
	}

	if _, ok := s.engine.sourceBuilders["input"]; ok {
		s.data["input"] = s.engine.data.Raw
	}

	value, err := templater.Generate(s.data)
	if err != nil {
		return nil, fmt.Errorf("failed to execute templator: %w", err)
	}

	return value, nil
}

func (s *TemplateSource) Close(_ context.Context) error {
	return nil
}
