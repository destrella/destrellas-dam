package modelo

// AsociacionTexto agrupa una o más cadenas originales y las palabras clave
// sugeridas que deben proponerse cuando alguna de esas cadenas aparece en el
// nombre del archivo.
type AsociacionTexto struct {
	ID         int64
	Originales []string
	Sugeridas  []string
}
