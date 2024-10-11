package service

import (
	"fmt"
	"github.com/invopop/jsonschema"
	"github.com/stoewer/go-strcase"
	orderedmap "github.com/wk8/go-ordered-map/v2"
	"go.lumeweb.com/portal/core"
	"reflect"
	"strings"
)

const ADMIN_SETTINGS_SERVICE = "admin_settings"

var _ core.Service = (*AdminSettingsService)(nil)
var configSchema *jsonschema.Schema

type AdminSettingsService struct {
	ctx core.Context
}

func (a *AdminSettingsService) ID() string {
	return ADMIN_SETTINGS_SERVICE
}

func NewAdminSettingsService() (core.Service, []core.ContextBuilderOption, error) {
	adminSettingsService := &AdminSettingsService{}

	opts := core.ContextOptions(
		core.ContextWithStartupFunc(func(ctx core.Context) error {
			var err error
			adminSettingsService.ctx = ctx
			r := &jsonschema.Reflector{}
			r.RequiredFromJSONSchemaTags = true
			r.KeyNamer = strcase.SnakeCase
			configSchema = r.ReflectFromType(reflect.TypeOf(ctx.Config().Config()))
			configSchema, err = buildConfigSchema(ctx, configSchema)
			if err != nil {
				return err
			}
			return nil
		}),
	)

	return adminSettingsService, opts, nil
}

func (a *AdminSettingsService) ListSettings() *jsonschema.Schema {
	return configSchema
}

func buildConfigSchema(ctx core.Context, originalSchema *jsonschema.Schema) (*jsonschema.Schema, error) {
	newSchema := &jsonschema.Schema{
		Type:        "object",
		Properties:  orderedmap.New[string, *jsonschema.Schema](),
		Definitions: originalSchema.Definitions,
	}

	err := ctx.Config().FieldProcessor(ctx.Config().Config(), "", func(_ *reflect.StructField, field reflect.StructField, value reflect.Value, prefix string) error {
		if field.Type == nil {
			return nil
		}

		fieldSchema, err := buildFieldSchema(field.Type, originalSchema.Definitions)
		if err != nil {
			return fmt.Errorf("error building field schema for %s: %v", field.Name, err)
		}

		// Build nested structure
		parts := strings.Split(prefix, ".")
		currentSchema := newSchema
		for i, part := range parts {
			snakeCasePart := strcase.SnakeCase(part)
			if snakeCasePart == "" {
				return fmt.Errorf("empty property name after conversion to snake case: original=%s", part)
			}

			if i == len(parts)-1 {
				// This is the last part, set the actual field
				if currentSchema.Properties == nil {
					currentSchema.Properties = orderedmap.New[string, *jsonschema.Schema]()
				}
				currentSchema.Properties.Set(snakeCasePart, fieldSchema)
			} else {
				// This is an intermediate part, ensure the nested structure exists
				var nextSchema *jsonschema.Schema
				if existingSchema, exists := currentSchema.Properties.Get(snakeCasePart); exists {
					nextSchema = existingSchema
				} else {
					nextSchema = &jsonschema.Schema{
						Type:       "object",
						Properties: orderedmap.New[string, *jsonschema.Schema](),
					}
					currentSchema.Properties.Set(snakeCasePart, nextSchema)
				}
				currentSchema = nextSchema
			}
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("error processing fields: %v", err)
	}

	removeEmptyProperties(newSchema)

	return newSchema, nil
}

func buildFieldSchema(t reflect.Type, definitions map[string]*jsonschema.Schema) (*jsonschema.Schema, error) {
	// If it's a pointer, get the underlying type
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	// Check if this type has a definition
	if _, ok := definitions[t.Name()]; ok {
		// If it has a definition, just use the $ref
		return &jsonschema.Schema{
			Ref: "#/$defs/" + t.Name(),
		}, nil
	}

	// If it doesn't have a definition, create a new schema
	schema := &jsonschema.Schema{
		Type: getJSONSchemaType(t),
	}

	// Handle specific types
	switch t.Kind() {
	case reflect.Struct:
		schema.Properties = orderedmap.New[string, *jsonschema.Schema]()
		for i := 0; i < t.NumField(); i++ {
			field := t.Field(i)
			if field.PkgPath != "" && !field.Anonymous { // unexported
				continue
			}
			fieldSchema, err := buildFieldSchema(field.Type, definitions)
			if err != nil {
				return nil, err
			}
			schema.Properties.Set(strcase.SnakeCase(field.Name), fieldSchema)
		}
	case reflect.Slice, reflect.Array:
		itemSchema, err := buildFieldSchema(t.Elem(), definitions)
		if err != nil {
			return nil, err
		}
		schema.Items = itemSchema
	case reflect.Map:
		valueSchema, err := buildFieldSchema(t.Elem(), definitions)
		if err != nil {
			return nil, err
		}
		schema.AdditionalProperties = valueSchema
	}

	return schema, nil
}

func removeEmptyProperties(schema *jsonschema.Schema) {
	if schema == nil || schema.Properties == nil {
		return
	}

	for pair := schema.Properties.Oldest(); pair != nil; pair = pair.Next() {
		if pair.Value.Ref != "" {
			// If it's a reference, remove any properties
			pair.Value.Properties = nil
		} else {
			removeEmptyProperties(pair.Value)
		}

		if pair.Value.Properties != nil && pair.Value.Properties.Len() == 0 {
			pair.Value.Properties = nil
		}
	}

	// Remove empty definitions
	for key, def := range schema.Definitions {
		removeEmptyProperties(def)
		if def.Properties != nil && def.Properties.Len() == 0 {
			def.Properties = nil
		}
		if isEmptySchema(def) {
			delete(schema.Definitions, key)
		}
	}
}

func isEmptySchema(schema *jsonschema.Schema) bool {
	return schema.Properties == nil &&
		schema.Items == nil &&
		schema.AdditionalProperties == nil &&
		schema.Ref == ""
}

func getJSONSchemaType(t reflect.Type) string {
	switch t.Kind() {
	case reflect.Bool:
		return "boolean"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return "integer"
	case reflect.Float32, reflect.Float64:
		return "number"
	case reflect.String:
		return "string"
	case reflect.Slice, reflect.Array:
		return "array"
	case reflect.Map, reflect.Struct:
		return "object"
	default:
		return "string" // Default to string for unknown types
	}
}
