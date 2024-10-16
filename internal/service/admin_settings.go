package service

import (
	"github.com/invopop/jsonschema"
	"github.com/stoewer/go-strcase"
	orderedmap "github.com/wk8/go-ordered-map/v2"
	"go.lumeweb.com/portal/core"
	"gopkg.in/yaml.v3"
	"reflect"
	"strings"
)

const ADMIN_SETTINGS_SERVICE = "admin_settings"

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
			adminSettingsService.ctx = ctx

			schema := &jsonschema.Schema{
				Version:    "https://json-schema.org/draft/2020-12/schema",
				ID:         "https://go.lumeweb.com/portal/config/config",
				Type:       "object",
				Properties: orderedmap.New[string, *jsonschema.Schema](),
			}

			builder := &schemaBuilder{schema: schema}
			err := ctx.Config().FieldProcessor(ctx.Config().Config(), "", builder.buildSchema)
			if err != nil {
				return err
			}

			configSchema = schema
			return nil
		}),
	)

	return adminSettingsService, opts, nil
}

func (a *AdminSettingsService) ListSettings() *jsonschema.Schema {
	return configSchema
}

type schemaBuilder struct {
	schema *jsonschema.Schema
}

func (sb *schemaBuilder) buildSchema(_ *reflect.StructField, field reflect.StructField, value reflect.Value, prefix string) error {
	fieldName := getFieldName(field)
	fullPath := buildFullPath(prefix, fieldName)

	if fullPath == "" {
		return nil
	}

	fieldSchema := sb.getFieldSchema(field, value)
	if fieldSchema != nil {
		sb.setSchemaProperty(fullPath, fieldSchema)
	}

	return nil
}

func getFieldName(field reflect.StructField) string {
	if configTag := field.Tag.Get("config"); configTag != "" {
		return configTag
	}
	return strcase.SnakeCase(field.Name)
}

func buildFullPath(prefix, fieldName string) string {
	if prefix == "" {
		return fieldName
	}
	return prefix
}

func (sb *schemaBuilder) setSchemaProperty(path string, schema *jsonschema.Schema) {
	parts := strings.Split(path, ".")
	current := sb.schema

	for i, part := range parts {
		if current.Properties == nil {
			current.Properties = orderedmap.New[string, *jsonschema.Schema]()
		}
		if i == len(parts)-1 {
			// This is the last part, set the schema
			current.Properties.Set(part, schema)
		} else {
			// This is an intermediate part, ensure the nested structure exists
			next, exists := current.Properties.Get(part)
			if !exists {
				next = &jsonschema.Schema{
					Type:       "object",
					Properties: orderedmap.New[string, *jsonschema.Schema](),
				}
				current.Properties.Set(part, next)
			}
			current = next
		}
	}
}

func (sb *schemaBuilder) getFieldSchema(field reflect.StructField, v reflect.Value) *jsonschema.Schema {
	schema := &jsonschema.Schema{}

	switch v.Kind() {
	case reflect.Bool:
		schema.Type = "boolean"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		schema.Type = "integer"
	case reflect.Float32, reflect.Float64:
		schema.Type = "number"
	case reflect.String:
		schema.Type = "string"
	case reflect.Slice, reflect.Array:
		schema.Type = "array"
		if v.Len() > 0 {
			schema.Items = sb.getFieldSchema(reflect.StructField{}, v.Index(0))
		}
	case reflect.Map:
		schema.Type = "object"
		schema.AdditionalProperties = jsonschema.TrueSchema
	case reflect.Struct:
		// Check if the struct implements MarshalYAML
		if marshaler, ok := v.Interface().(yaml.Marshaler); ok {
			yamlData, err := marshaler.MarshalYAML()
			if err == nil {
				return sb.handleYAMLMarshaled(yamlData)
			}
		}

		schema.Type = "object"
		schema.Properties = orderedmap.New[string, *jsonschema.Schema]()
		for i := 0; i < v.NumField(); i++ {
			field := v.Type().Field(i)
			fieldValue := v.Field(i)
			fieldName := getFieldName(field)
			fieldSchema := sb.getFieldSchema(field, fieldValue)
			if fieldSchema != nil {
				schema.Properties.Set(fieldName, fieldSchema)
			}
		}
	case reflect.Interface:
		if !v.IsNil() {
			return sb.getFieldSchema(field, v.Elem())
		}

	case reflect.Ptr:
		if !v.IsNil() {
			return sb.getFieldSchema(field, v.Elem())
		}
	}

	return schema
}

func (sb *schemaBuilder) handleYAMLMarshaled(data interface{}) *jsonschema.Schema {
	schema := &jsonschema.Schema{}

	switch v := data.(type) {
	case map[string]interface{}:
		schema.Type = "object"
		schema.Properties = orderedmap.New[string, *jsonschema.Schema]()
		for key, val := range v {
			schema.Properties.Set(key, sb.getFieldSchema(reflect.StructField{}, reflect.ValueOf(val)))
		}
	case []interface{}:
		schema.Type = "array"
		if len(v) > 0 {
			schema.Items = sb.getFieldSchema(reflect.StructField{}, reflect.ValueOf(v[0]))
		}
	default:
		// If it's not a map or slice, treat it as a simple value
		return sb.getFieldSchema(reflect.StructField{}, reflect.ValueOf(v))
	}

	return schema
}
