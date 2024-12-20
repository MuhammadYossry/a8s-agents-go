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
	if err := p.validateSchema(data, p.inputSchema); err != nil {
		return fmt.Errorf("validating request: %w", err)
	}
	return nil
}

func (p *SchemaParser) ValidateResponse(response map[string]interface{}) error {
	if err := p.validateSchema(response, p.outputSchema); err != nil {
		return fmt.Errorf("validating response: %w", err)
	}
	return nil
}

func (p *SchemaParser) validateSchema(data map[string]interface{}, schema SchemaConfig) error {
	// Validate required fields
	for _, field := range schema.Required {
		if _, exists := data[field]; !exists {
			return fmt.Errorf("missing required field: %s", field)
		}
	}

	// Validate properties
	return p.validateProperties(data, schema.Properties)
}

func (p *SchemaParser) validateProperties(data map[string]interface{}, properties map[string]Property) error {
	for fieldName, prop := range properties {
		value, exists := data[fieldName]
		if !exists {
			continue
		}

		if err := p.validateProperty(fieldName, prop, value); err != nil {
			return err
		}
	}
	return nil
}

func (p *SchemaParser) validateProperty(fieldName string, prop Property, value interface{}) error {
	if value == nil {
		return nil // Skip validation for nil values
	}

	switch prop.Type {
	case "object":
		return p.validateObject(fieldName, prop, value)
	case "array":
		return p.validateArray(fieldName, prop, value)
	case "string":
		return p.validateString(fieldName, prop, value)
	case "number", "integer":
		return p.validateNumber(fieldName, prop, value)
	case "boolean":
		return p.validateBoolean(fieldName, prop, value)
	}

	return nil
}

func (p *SchemaParser) validateObject(fieldName string, prop Property, value interface{}) error {
	obj, ok := value.(map[string]interface{})
	if !ok {
		return fmt.Errorf("field %s must be an object", fieldName)
	}

	if prop.Properties != nil {
		if err := p.validateProperties(obj, prop.Properties); err != nil {
			return fmt.Errorf("in object %s: %w", fieldName, err)
		}
	}

	// Validate required fields within the object
	for _, required := range prop.Required {
		if _, exists := obj[required]; !exists {
			return fmt.Errorf("missing required field %s in object %s", required, fieldName)
		}
	}

	return nil
}

func (p *SchemaParser) validateArray(fieldName string, prop Property, value interface{}) error {
	arr, ok := value.([]interface{})
	if !ok {
		return fmt.Errorf("field %s must be an array", fieldName)
	}

	// Validate array length constraints
	if prop.MinimumItems > 0 && len(arr) < prop.MinimumItems {
		return fmt.Errorf("array %s must have at least %d items", fieldName, prop.MinimumItems)
	}
	if prop.MaximumItems > 0 && len(arr) > prop.MaximumItems {
		return fmt.Errorf("array %s must have at most %d items", fieldName, prop.MaximumItems)
	}

	// Validate array items if items schema is provided
	if prop.Items != nil {
		for i, item := range arr {
			if err := p.validateProperty(fmt.Sprintf("%s[%d]", fieldName, i), *prop.Items, item); err != nil {
				return err
			}
		}
	}

	return nil
}

func (p *SchemaParser) validateString(fieldName string, prop Property, value interface{}) error {
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("field %s must be a string", fieldName)
	}

	// Validate enum if specified
	if len(prop.Enum) > 0 {
		valid := false
		for _, enumVal := range prop.Enum {
			if str == enumVal {
				valid = true
				break
			}
		}
		if !valid {
			return fmt.Errorf("field %s must be one of %v", fieldName, prop.Enum)
		}
	}

	// Add pattern validation if needed
	if prop.Pattern != "" {
		// You could add regex pattern validation here
	}

	return nil
}

func (p *SchemaParser) validateNumber(fieldName string, _ Property, value interface{}) error {
	// Handle both float64 and int
	switch value.(type) {
	case float64:
		return nil // Add any float-specific validations here
	case int:
		return nil // Add any int-specific validations here
	default:
		return fmt.Errorf("field %s must be a number", fieldName)
	}
}

func (p *SchemaParser) validateBoolean(fieldName string, _ Property, value interface{}) error {
	if _, ok := value.(bool); !ok {
		return fmt.Errorf("field %s must be a boolean", fieldName)
	}
	return nil
}
