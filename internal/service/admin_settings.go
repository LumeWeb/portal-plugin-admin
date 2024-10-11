package service

import (
	"github.com/invopop/jsonschema"
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

	err := ctx.Config().FieldProcessor(ctx.Config().Config(), "", func(field *reflect.StructField, value reflect.Value, prefix string) error {
		if field == nil {
			return nil
		}

		fieldName := field.Name
		fullFieldName := prefix
		if prefix != "" {
			fullFieldName = prefix + "." + fieldName
		} else {
			fullFieldName = fieldName
		}

		fieldSchema := findSchemaForField(originalSchema, fullFieldName)
		if fieldSchema == nil {
			fieldSchema = &jsonschema.Schema{Type: getJSONSchemaType(field.Type)}
		}

		// Only use $ref for struct fields
		if field.Type.Kind() == reflect.Struct {
			structName := field.Type.Name()
			if _, ok := originalSchema.Definitions[structName]; ok {
				fieldSchema = &jsonschema.Schema{
					Ref: "#/$defs/" + structName,
				}
			}
		}

		// Build nested structure
		parts := strings.Split(fullFieldName, ".")
		currentProperties := newSchema.Properties
		for i, part := range parts {
			if i == len(parts)-1 {
				// This is the last part, set the actual field
				currentProperties.Set(part, fieldSchema)
			} else {
				// This is an intermediate part, ensure the nested structure exists
				var nextSchema *jsonschema.Schema
				if existingSchema, exists := currentProperties.Get(part); exists {
					nextSchema = existingSchema
				} else {
					nextSchema = &jsonschema.Schema{
						Type:       "object",
						Properties: orderedmap.New[string, *jsonschema.Schema](),
					}
					currentProperties.Set(part, nextSchema)
				}
				currentProperties = nextSchema.Properties
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return newSchema, nil
}

func findSchemaForField(schema *jsonschema.Schema, fieldName string) *jsonschema.Schema {
	parts := strings.Split(fieldName, ".")
	currentSchema := schema

	for _, part := range parts {
		if currentSchema.Properties != nil {
			if prop, ok := currentSchema.Properties.Get(part); ok {
				currentSchema = prop
				continue
			}
		}

		// Check in definitions
		if def, ok := schema.Definitions[part]; ok {
			currentSchema = def
			continue
		}

		// If we can't find the field, return nil
		return nil
	}

	return currentSchema
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
