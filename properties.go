package main

import (
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

const (
	PropertyTypeCheckbox = "checkbox property"
	PropertyTypeLabel    = "label property"
	PropertyTypeString   = "string property"
)

type Property struct {
	Properties *Properties
	Name       string
	data       interface{}
	InUse      bool
}

func NewProperty(name string, properties *Properties) *Property {
	return &Property{
		Properties: properties,
		Name:       name,
	}
}

func (prop *Property) IsString() bool {
	_, isOK := prop.data.(string)
	return isOK
}

func (prop *Property) AsString() string {
	if prop.data == nil {
		prop.data = ""
	}
	return prop.data.(string)
}

func (prop *Property) IsBool() bool {
	_, isOK := prop.data.(bool)
	return isOK
}

func (prop *Property) AsBool() bool {
	if prop.data == nil {
		prop.data = false
	}
	return prop.data.(bool)
}

func (prop *Property) AsSlice() []interface{} {
	if prop.data == nil {
		prop.data = []interface{}{}
	}
	return prop.data.([]interface{})
}

func (prop *Property) AsMap() map[interface{}]interface{} {
	if prop.data == nil {
		prop.data = map[interface{}]interface{}{}
	}
	return prop.data.(map[interface{}]interface{})
}

func (prop *Property) Set(value interface{}) {
	if prop.data != value {
		if prop.Properties.OnChange != nil {
			prop.Properties.OnChange(prop)
		}
		prop.data = value
	}
}

// SetRaw sets the property, but without triggering OnChange
func (prop *Property) SetRaw(value interface{}) {
	if prop.data != value {
		prop.data = value
	}
}

// Contains serializable properties for a Card.
type Properties struct {
	Card            *Card
	Props           map[string]*Property
	DefinitionOrder []string
	OnChange        func(property *Property)
}

func NewProperties(card *Card) *Properties {
	return &Properties{
		Card:            card,
		Props:           map[string]*Property{},
		DefinitionOrder: []string{},
	}
}

func (properties *Properties) Get(name string) *Property {

	if _, exists := properties.Props[name]; !exists {
		properties.Props[name] = NewProperty(name, properties)
	}

	properties.DefinitionOrder = append(properties.DefinitionOrder, name)

	prop := properties.Props[name]
	prop.InUse = true
	return prop

}

func (properties *Properties) Serialize() string {

	data := "{}"

	for _, name := range properties.DefinitionOrder {
		if properties.Props[name].InUse {
			data, _ = sjson.Set(data, name, properties.Props[name].data)
		}
	}

	return data

}

func (properties *Properties) Deserialize(data string) {

	parsed := gjson.Parse(data)

	parsed.ForEach(func(key, value gjson.Result) bool {
		properties.Get(key.String()).SetRaw(value.Value())
		return true
	})

}
