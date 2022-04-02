package sqlgenerator

import "github.com/iancoleman/strcase"

func ToCamelCase(s string) string {
	if s == "id" {
		return "ID"
	}
	return strcase.ToCamel(s)
}
