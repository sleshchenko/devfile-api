package genutils

import (
	apiext "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"sigs.k8s.io/controller-tools/pkg/crd"
)

// Visitor is the type of a function that visits one level of Json schema
type Visitor func(schema *apiext.JSONSchemaProps) (newVisitor Visitor, stop bool)

type visitorStruct struct {
	VisitFunc Visitor
}

func (v visitorStruct) Visit(schema *apiext.JSONSchemaProps) crd.SchemaVisitor {
	newVisitor, stop := v.VisitFunc(schema)
	if stop {
		return nil
	}

	if newVisitor == nil {
		return v
	}
	return visitorStruct{newVisitor}
}
