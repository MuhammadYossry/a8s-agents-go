package core

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	types "github.com/MuhammadYossry/a8s-agents-go/types"
)

type SchemaParser struct {
	inputSchema  types.SchemaConfig
	outputSchema types.SchemaConfig
	defs         map[string]types.SchemaConfig // Store resolved $defs for reference
}

func NewSchemaParser(action types.Action) *SchemaParser {
	// Extract all $defs from input and output schemas
	defs := make(map[string]types.SchemaConfig)
	if action.InputSchema.Defs != nil {
		for name, def := range action.InputSchema.Defs {
			defs[name] = def
		}
	}
	if action.OutputSchema.Defs != nil {
		for name, def := range action.OutputSchema.Defs {
			defs[name] = def
		}
	}

	return &SchemaParser{
		inputSchema:  action.InputSchema,
		outputSchema: action.OutputSchema,
		defs:         defs,
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

func (p *SchemaParser) validateSchema(data map[string]interface{}, schema types.SchemaConfig) error {
	// Handle $ref if present
	if schema.Ref != "" {
		resolvedSchema, err := p.resolveRef(schema.Ref)
		if err != nil {
			return err
		}
		schema = resolvedSchema
	}

	// Validate type
	if schema.Type != "" && !isValidType(schema.Type) {
		return fmt.Errorf("invalid schema type: %s", schema.Type)
	}

	// Validate required fields
	for _, field := range schema.Required {
		if _, exists := data[field]; !exists {
			return fmt.Errorf("missing required field: %s", field)
		}
	}

	// Validate additional properties if specified
	if schema.AdditionalProperties != nil {
		if err := p.validateAdditionalProperties(data, schema); err != nil {
			return err
		}
	}

	// Validate properties
	return p.validateProperties(data, schema.Properties)
}

func (p *SchemaParser) validateProperties(data map[string]interface{}, properties map[string]types.Property) error {
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

func (p *SchemaParser) validateProperty(fieldName string, prop types.Property, value interface{}) error {
	if value == nil {
		return nil
	}

	// Handle $ref if present
	if prop.Ref != "" {
		resolvedSchema, err := p.resolveRef(prop.Ref)
		if err != nil {
			return err
		}
		return p.validateSchema(value.(map[string]interface{}), resolvedSchema)
	}

	// Handle anyOf, allOf, oneOf
	if len(prop.AnyOf) > 0 {
		return p.validateAnyOf(fieldName, prop.AnyOf, value)
	}
	if len(prop.AllOf) > 0 {
		return p.validateAllOf(fieldName, prop.AllOf, value)
	}
	if len(prop.OneOf) > 0 {
		return p.validateOneOf(fieldName, prop.OneOf, value)
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

func (p *SchemaParser) validateObject(fieldName string, prop types.Property, value interface{}) error {
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

func (p *SchemaParser) validateString(fieldName string, prop types.Property, value interface{}) error {
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("field %s must be a string", fieldName)
	}

	// Validate enum
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

	// Validate const if specified
	if prop.Const != "" && str != prop.Const {
		return fmt.Errorf("field %s must be exactly %s", fieldName, prop.Const)
	}

	// Validate format
	if prop.Format != "" {
		if err := p.validateFormat(fieldName, prop.Format, str); err != nil {
			return err
		}
	}

	// Validate pattern
	if prop.Pattern != "" {
		matched, err := regexp.MatchString(prop.Pattern, str)
		if err != nil {
			return fmt.Errorf("invalid pattern in schema for field %s: %w", fieldName, err)
		}
		if !matched {
			return fmt.Errorf("field %s does not match required pattern %s", fieldName, prop.Pattern)
		}
	}

	return nil
}

func (p *SchemaParser) validateNumber(fieldName string, prop types.Property, value interface{}) error {
	var num float64
	switch v := value.(type) {
	case float64:
		num = v
	case int:
		num = float64(v)
	default:
		return fmt.Errorf("field %s must be a number", fieldName)
	}

	if prop.Minimum != nil && num < *prop.Minimum {
		return fmt.Errorf("field %s must be greater than or equal to %v", fieldName, *prop.Minimum)
	}
	if prop.Maximum != nil && num > *prop.Maximum {
		return fmt.Errorf("field %s must be less than or equal to %v", fieldName, *prop.Maximum)
	}

	return nil
}

func (p *SchemaParser) validateArray(fieldName string, prop types.Property, value interface{}) error {
	arr, ok := value.([]interface{})
	if !ok {
		return fmt.Errorf("field %s must be an array", fieldName)
	}

	if prop.MinimumItems > 0 && len(arr) < prop.MinimumItems {
		return fmt.Errorf("array %s must have at least %d items", fieldName, prop.MinimumItems)
	}
	if prop.MaximumItems > 0 && len(arr) > prop.MaximumItems {
		return fmt.Errorf("array %s must have at most %d items", fieldName, prop.MaximumItems)
	}

	if prop.Items != nil {
		for i, item := range arr {
			if err := p.validateProperty(fmt.Sprintf("%s[%d]", fieldName, i), *prop.Items, item); err != nil {
				return err
			}
		}
	}

	return nil
}

func (p *SchemaParser) validateBoolean(fieldName string, _ types.Property, value interface{}) error {
	if _, ok := value.(bool); !ok {
		return fmt.Errorf("field %s must be a boolean", fieldName)
	}
	return nil
}
func (p *SchemaParser) validateFormat(fieldName, format, value string) error {
	switch format {
	case "date-time":
		_, err := time.Parse(time.RFC3339, value)
		if err != nil {
			return fmt.Errorf("field %s must be a valid RFC3339 date-time", fieldName)
		}
	case "email":
		if !isValidEmail(value) {
			return fmt.Errorf("field %s must be a valid email address", fieldName)
		}
	case "uri":
		if !isValidURI(value) {
			return fmt.Errorf("field %s must be a valid URI", fieldName)
		}
	}
	return nil
}

func (p *SchemaParser) resolveRef(ref string) (types.SchemaConfig, error) {
	// Remove the #/$defs/ prefix
	defName := strings.TrimPrefix(ref, "#/$defs/")
	schema, exists := p.defs[defName]
	if !exists {
		return types.SchemaConfig{}, fmt.Errorf("undefined $ref: %s", ref)
	}
	return schema, nil
}

func (p *SchemaParser) validateAnyOf(fieldName string, schemas []types.Property, value interface{}) error {
	var errors []string
	for _, schema := range schemas {
		if err := p.validateProperty(fieldName, schema, value); err == nil {
			return nil
		} else {
			errors = append(errors, err.Error())
		}
	}
	return fmt.Errorf("field %s failed to match any schema: %s", fieldName, strings.Join(errors, "; "))
}

func (p *SchemaParser) validateAllOf(fieldName string, schemas []types.Property, value interface{}) error {
	for _, schema := range schemas {
		if err := p.validateProperty(fieldName, schema, value); err != nil {
			return err
		}
	}
	return nil
}

func (p *SchemaParser) validateOneOf(fieldName string, schemas []types.Property, value interface{}) error {
	validCount := 0
	var lastError error
	for _, schema := range schemas {
		if err := p.validateProperty(fieldName, schema, value); err == nil {
			validCount++
		} else {
			lastError = err
		}
	}
	if validCount == 1 {
		return nil
	}
	if validCount == 0 {
		return fmt.Errorf("field %s did not match any schema: %v", fieldName, lastError)
	}
	return fmt.Errorf("field %s matched multiple schemas when exactly one was required", fieldName)
}

func (p *SchemaParser) validateAdditionalProperties(data map[string]interface{}, schema types.SchemaConfig) error {
	for fieldName, value := range data {
		// Skip fields that are defined in properties
		if _, exists := schema.Properties[fieldName]; exists {
			continue
		}

		if err := p.validateProperty(fieldName, *schema.AdditionalProperties, value); err != nil {
			return fmt.Errorf("additional property validation failed for field %s: %w", fieldName, err)
		}
	}
	return nil
}

// Helper functions

func isValidType(t string) bool {
	validTypes := map[string]bool{
		"string":  true,
		"number":  true,
		"integer": true,
		"boolean": true,
		"array":   true,
		"object":  true,
		"null":    true,
	}
	return validTypes[t]
}

func isValidEmail(email string) bool {
	// Basic email validation - could be made more sophisticated
	pattern := `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`
	matched, err := regexp.MatchString(pattern, email)
	return err == nil && matched
}

func isValidURI(uri string) bool {
	// Basic URI validation - could be made more sophisticated
	pattern := `^[a-zA-Z][a-zA-Z0-9+.-]*://`
	matched, err := regexp.MatchString(pattern, uri)
	return err == nil && matched
}
