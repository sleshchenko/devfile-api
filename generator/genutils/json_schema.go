package genutils

import (
	"strings"

	"github.com/iancoleman/strcase"
	apiext "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"sigs.k8s.io/controller-tools/pkg/crd"
	"sigs.k8s.io/controller-tools/pkg/markers"
)

// EditJSONSchema allows editing a Json Schema by applying the visitor function to each level of the schema.
func EditJSONSchema(jsonSchema *apiext.JSONSchemaProps, visitor Visitor) {
	crd.EditSchema(jsonSchema, visitorStruct{visitor})
}

// AddUnionOneOfConstraints adds oneOf constraints in the given Json schema for all the unions, based on the authorized values of the union members
func AddUnionOneOfConstraints(jsonSchema *apiext.JSONSchemaProps, unionDiscriminators []markers.FieldInfo, removeDiscriminators bool, fieldsToSkip ...string) {
	mainVisitor := func(schema *apiext.JSONSchemaProps) (newVisitor Visitor, stop bool) {
		if schema == nil {
			return
		}
		if schema.Type != "object" {
			return
		}

		if len(schema.Properties) == 0 {
			return
		}

	discriminatorLoop:
		for _, discriminator := range unionDiscriminators {
			discriminatorPropertyName := strcase.ToLowerCamel(discriminator.Name)

			discriminatorProp, found := schema.Properties[discriminatorPropertyName]
			if !found || len(discriminatorProp.Enum) == 0 {
				continue discriminatorLoop
			}

			oneOf := []apiext.JSONSchemaProps{}
			newEnumValues := []apiext.JSON{}
		enumValueLoop:
			for _, enumValue := range discriminatorProp.Enum {
				fieldName := strings.Trim(string(enumValue.Raw), `"`)
				unionMemberProperty := strcase.ToLowerCamel(fieldName)
				if _, foundUnionMember := schema.Properties[unionMemberProperty]; !foundUnionMember {
					// The discriminator enum contains values that do not correspond to any enum field.
					// So so just skip the oneOf creation here since this union definition seems invalid.
					continue discriminatorLoop
				}
				for _, toSkip := range fieldsToSkip {
					if toSkip == fieldName {
						continue enumValueLoop
					}
				}
				newEnumValues = append(newEnumValues, enumValue)
				oneOf = append(oneOf, apiext.JSONSchemaProps{
					Required: []string{unionMemberProperty},
				})
			}
			schema.OneOf = oneOf
			discriminatorProp.Enum = newEnumValues

			if removeDiscriminators {
				delete(schema.Properties, discriminatorPropertyName)
			} else {
				schema.Properties[discriminatorPropertyName] = discriminatorProp
			}
			for _, toSkip := range fieldsToSkip {
				delete(schema.Properties, strcase.ToLowerCamel(toSkip))
			}
		}
		return
	}

	EditJSONSchema(jsonSchema, mainVisitor)
}
