package main

import (
	"strconv"

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
	OnChange   func()
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

func (prop *Property) AsJSON() gjson.Result {
	if prop.data == nil {
		prop.data = "{}"
	}
	return gjson.Parse(prop.data.(string))
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

func (prop *Property) IsNumber() bool {
	_, isOK := prop.data.(float64)
	return isOK
}

func (prop *Property) AsFloat() float64 {
	if prop.data == nil {
		prop.data = 0.0
	}
	return prop.data.(float64)
}

func (prop *Property) AsMap() map[string]interface{} {
	if prop.data == nil {
		prop.data = map[string]interface{}{}
	}
	return prop.data.(map[string]interface{})
}

func (prop *Property) Set(value interface{}) {

	if prop.data != value {

		prop.data = value

		if prop.OnChange != nil {
			prop.OnChange()
		}

		if prop.Properties.OnChange != nil {
			prop.Properties.OnChange(prop)
		}

	}

}

func (prop *Property) AsArrayOfInts() []int64 {

	if !prop.IsString() {
		prop.data = "{}"
	}

	out := []int64{}

	for _, value := range prop.AsJSON().Array() {
		out = append(out, value.Int())
	}

	return out

}

func (prop *Property) SetInts(values ...int64) {

	jsonStr := "["

	for _, v := range values {
		jsonStr += strconv.Itoa(int(v))
	}

	jsonStr += "]"

	prop.Set(jsonStr)
}

// func (prop *Property) SetAfterParse(value interface{}) {

// 	current := prop.AsJSON()

// 	parsed, err := sjson.Set(current.String(), "#0", value)

// 	prop.Set(parsed)

// 	fmt.Println(parsed, err)

// }

// SetRaw sets the property, but without triggering OnChange
func (prop *Property) SetRaw(value interface{}) {
	prop.data = value
}

// Contains serializable properties for a Card.
type Properties struct {
	Props           map[string]*Property
	DefinitionOrder []string
	OnChange        func(property *Property)
}

func NewProperties() *Properties {
	return &Properties{
		Props:           map[string]*Property{},
		DefinitionOrder: []string{},
	}
}

func (properties *Properties) Has(name string) bool {
	if _, exists := properties.Props[name]; exists {
		return true
	}
	return false
}

func (properties *Properties) Get(name string) *Property {

	if _, exists := properties.Props[name]; !exists {
		properties.Props[name] = NewProperty(name, properties)
		properties.DefinitionOrder = append(properties.DefinitionOrder, name)
	}

	prop := properties.Props[name]
	prop.InUse = true
	return prop

}

func (properties *Properties) Remove(propertyName string) {

	delete(properties.Props, propertyName)

	for i, p := range properties.DefinitionOrder {
		if p == propertyName {
			properties.DefinitionOrder = append(properties.DefinitionOrder[:i], properties.DefinitionOrder[i+1:]...)
			break
		}
	}

}

func (properties *Properties) Serialize() string {

	data := "{}"

	for _, name := range properties.DefinitionOrder {
		prop := properties.Props[name]
		if prop.InUse {
			data, _ = sjson.Set(data, name, prop.data)
		}
	}

	return data

}

func (properties *Properties) Deserialize(data string) {

	// All Properties contained within this object should probably be cleared before parsing...?

	parsed := gjson.Parse(data)

	parsed.ForEach(func(key, value gjson.Result) bool {
		properties.Get(key.String()).SetRaw(value.Value())
		return true
	})

}
