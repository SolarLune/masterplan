package main

import (
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
	"github.com/veandco/go-sdl2/sdl"
)

const (
	PropertyTypeCheckbox = "checkbox property"
	PropertyTypeLabel    = "label property"
)

type Property struct {
	Name  string
	Data  interface{}
	InUse bool
}

func NewProperty(name string) *Property {
	return &Property{
		Name: name,
	}
}

func (prop *Property) IsCheckbox() bool {
	_, isOK := prop.Data.(*Checkbox)
	return isOK
}

func (prop *Property) AsCheckbox() *Checkbox {
	if prop.Data == nil {
		prop.Data = NewGUICheckbox(true)
	}
	return prop.Data.(*Checkbox)
}

func (prop *Property) IsLabel() bool {
	_, isOK := prop.Data.(*Label)
	return isOK
}

func (prop *Property) AsLabel() *Label {
	if prop.Data == nil {
		prop.Data = NewLabel("Label", &sdl.FRect{}, true)
	}
	return prop.Data.(*Label)
}

func (prop *Property) Serialize() string {

	data := ""

	if prop.IsLabel() {

		data, _ = sjson.Set(data, "label", prop.AsLabel().TextAsString())

	} else if prop.IsCheckbox() {

		data, _ = sjson.Set(data, "checked", prop.AsCheckbox().Checked)

	}

	return data

}

func (prop *Property) Deserialize(data string) {

	parsed := gjson.Parse(data)

	if parsed.Get("label").Exists() {
		prop.AsLabel().SetText([]rune(parsed.Get("label").String()))
	} else if parsed.Get("checked").Exists() {
		prop.AsCheckbox().Checked = parsed.Get("checked").Bool()
	}

}

// Contains properties for a Card.
type Properties struct {
	Card            *Card
	Data            map[string]*Property
	DefinitionOrder []string
}

func NewProperties(card *Card) *Properties {
	return &Properties{
		Card:            card,
		Data:            map[string]*Property{},
		DefinitionOrder: []string{},
	}
}

func (properties *Properties) Request(name string) *Property {

	if _, exists := properties.Data[name]; !exists {
		properties.Data[name] = NewProperty(name)
	}

	properties.DefinitionOrder = append(properties.DefinitionOrder, name)

	prop := properties.Data[name]
	prop.InUse = true
	return prop

}

func (properties *Properties) Serialize() string {

	data := ""

	for _, name := range properties.DefinitionOrder {

		property := properties.Data[name]

		if !property.InUse {
			continue
		}

		data, _ = sjson.SetRaw(data, name, property.Serialize())
	}

	return data

}

func (properties *Properties) Deserialize(data string) {

	parsed := gjson.Parse(data)

	parsed.ForEach(func(key, value gjson.Result) bool {
		properties.Request(key.String()).Deserialize(value.Raw)
		return true
	})

}
