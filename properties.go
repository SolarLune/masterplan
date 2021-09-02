package main

import (
	"log"
	"os"

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

func (prop *Property) AsJSONString() string {
	str, _ := sjson.Set("", prop.Name, prop.data)
	return str
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

func (prop *Property) AsNumber() float64 {
	if prop.data == nil {
		prop.data = 0.0
	}
	return prop.data.(float64)
}

// func (prop *Property) AsMap() map[interface{}]interface{} {
// 	if prop.data == nil {
// 		prop.data = map[interface{}]interface{}{}
// 	}
// 	return prop.data.(map[interface{}]interface{})
// }

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

func (properties *Properties) Get(name string) *Property {

	if _, exists := properties.Props[name]; !exists {
		properties.Props[name] = NewProperty(name, properties)
		properties.DefinitionOrder = append(properties.DefinitionOrder, name)
	}

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

	// All Properties contained within this object should probably be cleared before parsing...?

	parsed := gjson.Parse(data)

	parsed.ForEach(func(key, value gjson.Result) bool {
		properties.Get(key.String()).SetRaw(value.Value())
		return true
	})

}

func (properties *Properties) Save(filepath string) {

	saveData, _ := sjson.Set("{}", "version", globals.Version.String())

	saveData, _ = sjson.SetRaw(saveData, "properties", properties.Serialize())

	saveData = gjson.Get(saveData, "@pretty").String()

	if file, err := os.Create(filepath); err != nil {
		log.Println(err)
	} else {
		file.Write([]byte(saveData))
		file.Close()
	}

}

func (properties *Properties) Load(filepath string) {

	jsonData, err := os.ReadFile(filepath)
	if err != nil {
		panic(err)
	}

	data := gjson.Get(string(jsonData), "properties").String()

	properties.Deserialize(data)

}
