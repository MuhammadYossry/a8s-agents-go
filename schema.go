// schema.go
package main

import (
	"fmt"
)

type SchemaParser struct {
	inputSchema  SchemaConfig
	outputSchema SchemaConfig
}

func NewSchemaParser(action Action) *SchemaParser {
	return &SchemaParser{
		inputSchema:  action.InputSchema,
		outputSchema: action.OutputSchema,
	}
}

func (p *SchemaParser) ValidateAndPrepareRequest(data map[string]interface{}) error {
	for _, field := range p.inputSchema.Required {
		if _, exists := data[field]; !exists {
			return fmt.Errorf("missing required field: %s", field)
		}
	}
	return p.validateProperties(data)
}

func (p *SchemaParser) ValidateResponse(response map[string]interface{}) error {
	if p.outputSchema.Fields != nil {
		for _, field := range p.outputSchema.Fields {
			if _, exists := response[field]; !exists {
				return fmt.Errorf("response missing required field: %s", field)
			}
		}
	}
	return nil
}

func (p *SchemaParser) validateProperties(data map[string]interface{}) error {
	for fieldName, prop := range p.inputSchema.Properties {
		value, exists := data[fieldName]
		if !exists {
			if contains(p.inputSchema.Required, fieldName) {
				return fmt.Errorf("missing required field: %s", fieldName)
			}
			continue
		}

		if err := p.validateProperty(fieldName, prop, value); err != nil {
			return err
		}
	}
	return nil
}

func (p *SchemaParser) validateProperty(fieldName string, prop Property, value interface{}) error {
	switch prop.Type {
	case "string":
		str, ok := value.(string)
		if !ok {
			return fmt.Errorf("field %s must be a string", fieldName)
		}
		if len(prop.Enum) > 0 && !contains(prop.Enum, str) {
			return fmt.Errorf("field %s must be one of %v", fieldName, prop.Enum)
		}
	case "boolean":
		if _, ok := value.(bool); !ok {
			return fmt.Errorf("field %s must be a boolean", fieldName)
		}
	}
	return nil
}

func contains(slice []string, str string) bool {
	for _, s := range slice {
		if s == str {
			return true
		}
	}
	return false
}
