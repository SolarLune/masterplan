package main

import (
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
	"github.com/veandco/go-sdl2/sdl"
)

const (
	PropertyTypeCheckbox = "checkbox property"
	PropertyTypeLabel    = "label property"
	PropertyTypeString   = "string property"
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
		prop.Data = NewLabel("", &sdl.FRect{}, true, AlignLeft)
	}
	return prop.Data.(*Label)
}

func (prop *Property) IsString() bool {
	_, isOK := prop.Data.(string)
	return isOK
}

func (prop *Property) AsString() string {
	if prop.Data == nil {
		prop.Data = ""
	}
	return prop.Data.(string)
}

func (prop *Property) SetValue(value interface{}) {
	prop.Data = value
}

func (prop *Property) Serialize() string {

	data := "{}"

	if prop.IsLabel() {
		data, _ = sjson.Set(data, "label", prop.AsLabel().TextAsString())
	} else if prop.IsCheckbox() {
		data, _ = sjson.Set(data, "checked", prop.AsCheckbox().Checked)
	} else if prop.IsString() {
		data, _ = sjson.Set(data, "string", prop.AsString())
	}

	return data

}

func (prop *Property) Deserialize(data string) {

	parsed := gjson.Parse(data)

	if parsed.Get("label").Exists() {
		prop.AsLabel().SetText([]rune(parsed.Get("label").String()))
	} else if parsed.Get("checked").Exists() {
		prop.AsCheckbox().Checked = parsed.Get("checked").Bool()
	} else if parsed.Get("string").Exists() {
		prop.SetValue(parsed.Get("string").String())
	}

}

// Contains properties for a Card.
type Properties struct {
	Card            *Card
	Props           map[string]*Property
	DefinitionOrder []string
}

func NewProperties(card *Card) *Properties {
	return &Properties{
		Card:            card,
		Props:           map[string]*Property{},
		DefinitionOrder: []string{},
	}
}

func (properties *Properties) Request(name string) *Property {

	if _, exists := properties.Props[name]; !exists {
		properties.Props[name] = NewProperty(name)
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
			data, _ = sjson.SetRaw(data, name, properties.Props[name].Serialize())
		}
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
