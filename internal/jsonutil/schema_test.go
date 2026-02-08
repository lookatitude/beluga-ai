package jsonutil

import (
	"reflect"
	"testing"
)

func TestGenerateSchemaBasicTypes(t *testing.T) {
	tests := []struct {
		name     string
		value    any
		wantType string
	}{
		{"string", "", "string"},
		{"int", 0, "integer"},
		{"float64", 0.0, "number"},
		{"bool", false, "boolean"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schema := GenerateSchema(tt.value)
			if schema["type"] != tt.wantType {
				t.Errorf("expected type %q, got %q", tt.wantType, schema["type"])
			}
		})
	}
}

func TestGenerateSchemaStruct(t *testing.T) {
	type Input struct {
		Name  string `json:"name" description:"The name" required:"true"`
		Age   int    `json:"age" minimum:"0" maximum:"200"`
		Email string `json:"email" default:"none"`
	}

	schema := GenerateSchema(Input{})

	if schema["type"] != "object" {
		t.Fatalf("expected type object, got %v", schema["type"])
	}

	props, ok := schema["properties"].(map[string]any)
	if !ok {
		t.Fatal("expected properties map")
	}

	// Check name property
	nameProp, ok := props["name"].(map[string]any)
	if !ok {
		t.Fatal("expected name property")
	}
	if nameProp["type"] != "string" {
		t.Errorf("expected name type string, got %v", nameProp["type"])
	}
	if nameProp["description"] != "The name" {
		t.Errorf("expected description 'The name', got %v", nameProp["description"])
	}

	// Check age property
	ageProp, ok := props["age"].(map[string]any)
	if !ok {
		t.Fatal("expected age property")
	}
	if ageProp["type"] != "integer" {
		t.Errorf("expected age type integer, got %v", ageProp["type"])
	}
	if ageProp["minimum"] != "0" {
		t.Errorf("expected minimum 0, got %v", ageProp["minimum"])
	}
	if ageProp["maximum"] != "200" {
		t.Errorf("expected maximum 200, got %v", ageProp["maximum"])
	}

	// Check email default
	emailProp, ok := props["email"].(map[string]any)
	if !ok {
		t.Fatal("expected email property")
	}
	if emailProp["default"] != "none" {
		t.Errorf("expected default 'none', got %v", emailProp["default"])
	}

	// Check required
	req, ok := schema["required"].([]string)
	if !ok {
		t.Fatal("expected required slice")
	}
	if len(req) != 1 || req[0] != "name" {
		t.Errorf("expected required [name], got %v", req)
	}
}

func TestGenerateSchemaEnum(t *testing.T) {
	type Config struct {
		Mode string `json:"mode" enum:"fast,slow,balanced"`
	}

	schema := GenerateSchema(Config{})
	props := schema["properties"].(map[string]any)
	modeProp := props["mode"].(map[string]any)

	enumVals, ok := modeProp["enum"].([]any)
	if !ok {
		t.Fatal("expected enum slice")
	}
	expected := []string{"fast", "slow", "balanced"}
	if len(enumVals) != len(expected) {
		t.Fatalf("expected %d enum values, got %d", len(expected), len(enumVals))
	}
	for i, v := range enumVals {
		if v != expected[i] {
			t.Errorf("enum[%d]: expected %q, got %q", i, expected[i], v)
		}
	}
}

func TestGenerateSchemaNestedStruct(t *testing.T) {
	type Address struct {
		City string `json:"city"`
	}
	type Person struct {
		Name    string  `json:"name"`
		Address Address `json:"address"`
	}

	schema := GenerateSchema(Person{})
	props := schema["properties"].(map[string]any)

	addrProp, ok := props["address"].(map[string]any)
	if !ok {
		t.Fatal("expected address property")
	}
	if addrProp["type"] != "object" {
		t.Errorf("expected address type object, got %v", addrProp["type"])
	}

	addrProps, ok := addrProp["properties"].(map[string]any)
	if !ok {
		t.Fatal("expected address properties")
	}
	cityProp, ok := addrProps["city"].(map[string]any)
	if !ok {
		t.Fatal("expected city property")
	}
	if cityProp["type"] != "string" {
		t.Errorf("expected city type string, got %v", cityProp["type"])
	}
}

func TestGenerateSchemaSlice(t *testing.T) {
	type List struct {
		Items []string `json:"items"`
	}

	schema := GenerateSchema(List{})
	props := schema["properties"].(map[string]any)
	itemsProp := props["items"].(map[string]any)

	if itemsProp["type"] != "array" {
		t.Errorf("expected type array, got %v", itemsProp["type"])
	}
	items := itemsProp["items"].(map[string]any)
	if items["type"] != "string" {
		t.Errorf("expected items type string, got %v", items["type"])
	}
}

func TestGenerateSchemaMap(t *testing.T) {
	type Config struct {
		Metadata map[string]int `json:"metadata"`
	}

	schema := GenerateSchema(Config{})
	props := schema["properties"].(map[string]any)
	metaProp := props["metadata"].(map[string]any)

	if metaProp["type"] != "object" {
		t.Errorf("expected type object, got %v", metaProp["type"])
	}
	addlProps := metaProp["additionalProperties"].(map[string]any)
	if addlProps["type"] != "integer" {
		t.Errorf("expected additionalProperties type integer, got %v", addlProps["type"])
	}
}

func TestGenerateSchemaPointer(t *testing.T) {
	type Opt struct {
		Value *string `json:"value"`
	}

	schema := GenerateSchema(Opt{})
	props := schema["properties"].(map[string]any)
	valProp := props["value"].(map[string]any)

	if valProp["type"] != "string" {
		t.Errorf("expected type string for pointer, got %v", valProp["type"])
	}
}

func TestGenerateSchemaSkipDash(t *testing.T) {
	type Hidden struct {
		Visible string `json:"visible"`
		Hidden  string `json:"-"`
	}

	schema := GenerateSchema(Hidden{})
	props := schema["properties"].(map[string]any)

	if _, ok := props["Hidden"]; ok {
		t.Error("expected Hidden field to be skipped")
	}
	if _, ok := props["visible"]; !ok {
		t.Error("expected visible field to be present")
	}
}

func TestGenerateSchemaUnexportedFields(t *testing.T) {
	type WithUnexported struct {
		Public  string `json:"public"`
		private string //nolint:unused
	}
	_ = WithUnexported{private: ""}

	schema := GenerateSchema(WithUnexported{})
	props := schema["properties"].(map[string]any)

	if len(props) != 1 {
		t.Errorf("expected 1 property, got %d", len(props))
	}
}

func TestGenerateSchemaSliceOfStructs(t *testing.T) {
	type Item struct {
		ID int `json:"id"`
	}
	type Container struct {
		Items []Item `json:"items"`
	}

	schema := GenerateSchema(Container{})
	props := schema["properties"].(map[string]any)
	itemsProp := props["items"].(map[string]any)

	if itemsProp["type"] != "array" {
		t.Fatalf("expected array, got %v", itemsProp["type"])
	}

	itemSchema := itemsProp["items"].(map[string]any)
	if itemSchema["type"] != "object" {
		t.Errorf("expected item type object, got %v", itemSchema["type"])
	}
}

func TestGenerateSchemaNilValue(t *testing.T) {
	schema := GenerateSchema(nil)
	if !reflect.DeepEqual(schema, map[string]any{"type": "object"}) {
		t.Errorf("expected {type: object} for nil, got %v", schema)
	}
}
