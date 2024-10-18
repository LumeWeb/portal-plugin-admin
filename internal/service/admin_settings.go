package service

import (
	"github.com/samber/lo"
	"github.com/stoewer/go-strcase"
	orderedmap "github.com/wk8/go-ordered-map/v2"
	"go.lumeweb.com/portal-plugin-admin/internal/api/messages"
	"go.lumeweb.com/portal-plugin-admin/internal/schema"
	"go.lumeweb.com/portal/core"
	"gopkg.in/yaml.v3"
	"reflect"
	"strconv"
	"strings"
)

const ADMIN_SETTINGS_SERVICE = "admin_settings"

var configSchema *schema.Schema

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

			_schema := &schema.Schema{
				Version:    "https://json-schema.org/draft/2020-12/schema",
				ID:         "https://go.lumeweb.com/portal/config/config",
				Type:       "object",
				Properties: orderedmap.New[string, *schema.Schema](),
			}

			builder := &schemaBuilder{schema: _schema, ctx: ctx}
			err := ctx.Config().FieldProcessor(ctx.Config().Config(), "", builder.buildSchema)
			if err != nil {
				return err
			}

			configSchema = _schema
			return nil
		}),
	)

	return adminSettingsService, opts, nil
}

func (a *AdminSettingsService) GetSchema() *schema.Schema {
	return configSchema
}

func (a *AdminSettingsService) GetSettings() []*messages.SettingsItem {
	return lo.MapToSlice(a.ctx.Config().All(), func(k string, v any) *messages.SettingsItem {
		return &messages.SettingsItem{
			Key:      k,
			Value:    v,
			Editable: a.ctx.Config().IsEditable(k),
		}
	})
}

func (a *AdminSettingsService) GetSetting(key string) *messages.SettingsItem {
	exists := a.ctx.Config().Exists(key)
	if !exists {
		return nil
	}
	return &messages.SettingsItem{
		Key:   key,
		Value: a.ctx.Config().Get(key),
	}
}

func (a *AdminSettingsService) UpdateSetting(setting *messages.SettingsItem) error {
	return a.ctx.Config().Update(setting.Key, setting.Value)
}

type schemaBuilder struct {
	schema *schema.Schema
	ctx    core.Context
}

func (sb *schemaBuilder) buildSchema(_ *reflect.StructField, field reflect.StructField, value reflect.Value, prefix string) error {
	fieldName := getFieldName(field)
	fullPath := buildFullPath(prefix, fieldName)

	if fullPath == "" {
		return nil
	}

	fieldSchema := sb.getFieldSchema(field.Type, value, fullPath)
	if fieldSchema != nil {
		sb.setSchemaProperty(fullPath, fieldSchema)
	}

	return nil
}

func (sb *schemaBuilder) getFieldSchema(field reflect.Type, v reflect.Value, path string) *schema.Schema {
	_schema := &schema.Schema{}

	checkReadyOnly := false

	switch v.Kind() {
	case reflect.Bool:
		_schema.Type = "boolean"
		checkReadyOnly = true

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		_schema.Type = "integer"
		checkReadyOnly = true
	case reflect.Float32, reflect.Float64:
		_schema.Type = "number"
		checkReadyOnly = true
	case reflect.String:
		_schema.Type = "string"
		checkReadyOnly = true
	case reflect.Slice, reflect.Array:
		_schema.Type = "array"
		if v.Len() > 0 {
			fullPath := buildFullPath(path, strconv.Itoa(0))
			_schema.Items = sb.getFieldSchema(v.Index(0).Type(), v.Index(0), fullPath)
		}
	case reflect.Map:
		_schema.Type = "object"
		_schema.AdditionalProperties = &schema.Schema{}
		if v.Len() > 0 {
			for _, key := range v.MapKeys() {
				fullPath := buildFullPath(path, key.String())
				_schema.AdditionalProperties = sb.getFieldSchema(v.MapIndex(key).Type(), v.MapIndex(key), fullPath)
				break
			}
		}
	case reflect.Struct, reflect.Interface:
		// Check if the struct implements MarshalYAML
		if marshaler, ok := v.Interface().(yaml.Marshaler); ok {
			yamlData, err := marshaler.MarshalYAML()
			if err == nil {
				return sb.handleYAMLMarshaled(yamlData)
			}
		}

		return nil
	case reflect.Ptr:
		if !v.IsNil() {
			return sb.getFieldSchema(field, v.Elem(), path)
		}
	}

	if checkReadyOnly {
		if !sb.ctx.Config().IsEditable(path) {
			_schema.ReadOnly = true
		}
	}

	return _schema
}

func (sb *schemaBuilder) handleYAMLMarshaled(data interface{}) *schema.Schema {
	_schema := &schema.Schema{}

	switch v := data.(type) {
	case map[string]interface{}:
		_schema.Type = "object"
		_schema.Properties = orderedmap.New[string, *schema.Schema]()
		for key, val := range v {
			_schema.Properties.Set(key, sb.getFieldSchema(reflect.TypeOf(val), reflect.ValueOf(val), key))
		}
	case []interface{}:
		_schema.Type = "array"
		if len(v) > 0 {
			_schema.Items = sb.getFieldSchema(reflect.TypeOf(v[0]), reflect.ValueOf(v[0]), "")
		}
	default:
		// If it's not a map or slice, treat it as a simple value
		return sb.getFieldSchema(reflect.TypeOf(v), reflect.ValueOf(v), "")
	}

	return _schema
}

func (sb *schemaBuilder) setSchemaProperty(path string, _schema *schema.Schema) {
	parts := strings.Split(path, ".")
	current := sb.schema

	for i, part := range parts {
		if current.Properties == nil {
			current.Properties = orderedmap.New[string, *schema.Schema]()
		}
		if i == len(parts)-1 {
			// This is the last part, set the schema
			current.Properties.Set(part, _schema)
		} else {
			// This is an intermediate part, ensure the nested structure exists
			next, exists := current.Properties.Get(part)
			if !exists {
				next = &schema.Schema{
					Type:       "object",
					Properties: orderedmap.New[string, *schema.Schema](),
				}
				current.Properties.Set(part, next)
			}
			current = next
		}
	}
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
	return strings.Join([]string{prefix, fieldName}, ".")
}
