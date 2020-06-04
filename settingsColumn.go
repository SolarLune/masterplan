package main

// SettingsColumn is a struct specifically for spacing options in the Settings menu.
type SettingsColumn struct {
	Data         map[string]PersistentGUIElement
	OrderOfEntry []string
}

func (project *Project) AddSettingsColumn() *SettingsColumn {
	column := &SettingsColumn{map[string]PersistentGUIElement{}, []string{}}
	project.SettingsColumns = append(project.SettingsColumns, column)
	return column
}

func (sc *SettingsColumn) Add(text string, data PersistentGUIElement) {
	sc.Data[text] = data
	sc.OrderOfEntry = append(sc.OrderOfEntry, text)
}
