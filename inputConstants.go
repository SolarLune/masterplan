package main

const (
	STATE_NONE = iota
	STATE_ACUTE
	STATE_GRAVE
)

var acuteMap = map[string]string{
	"a": "á", "e": "é", "i": "í", "o": "ó", "u": "ú",
	"A": "Á", "E": "É", "I": "Í", "O": "Ó", "U": "Ú",
}
var graveMap = map[string]string{
	"a": "à", "e": "è", "i": "ì", "o": "ò", "u": "ù",
	"A": "À", "E": "È", "I": "Ì", "O": "Ò", "U": "Ù",
}
