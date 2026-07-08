package modelo

// TipoCoincidencia identifica el algoritmo que produjo el grupo.
type TipoCoincidencia string

const (
	CoincidenciaExacta        TipoCoincidencia = "exacta"
	CoincidenciaParcialImagen TipoCoincidencia = "parcial_imagen"
	CoincidenciaParcialVideo  TipoCoincidencia = "parcial_video"
)

// CategoriaDuplicados permite filtrar por origen.
type CategoriaDuplicados string

const (
	CategoriaDuplicadosTodos   CategoriaDuplicados = "todos"
	CategoriaDuplicadosLocales CategoriaDuplicados = "locales"
	CategoriaDuplicadosRemotos CategoriaDuplicados = "remotos"
	CategoriaDuplicadosMixtos  CategoriaDuplicados = "mixtos"
)

// OrdenDuplicados describe las opciones de orden visibles en la UI.
type OrdenDuplicados string

const (
	OrdenPorTamanoGrupo       OrdenDuplicados = "tamano_grupo"
	OrdenPorEspacioRecuperado OrdenDuplicados = "espacio_recuperado"
	OrdenAlfabetico           OrdenDuplicados = "alfabetico"
)

// GrupoDuplicados reune archivos equivalentes bajo algun criterio.
type GrupoDuplicados struct {
	Clave              string
	Tipo               TipoCoincidencia
	Elementos          []Archivo
	TamanoRecuperable  int64
	CategoriaSugerida  CategoriaDuplicados
	CantidadElementos  int
	NombreRepresentivo string
}

// EstadisticasDuplicados resume la vista lateral.
type EstadisticasDuplicados struct {
	TotalGrupos int
	Locales     int
	Remotos     int
	Mixtos      int
}

// CalcularTamanoRecuperable estima el espacio que se recuperaria conservando un archivo.
func CalcularTamanoRecuperable(elementos []Archivo) int64 {
	if len(elementos) < 2 {
		return 0
	}

	var total int64
	var mayor int64
	for _, elemento := range elementos {
		total += elemento.Tamano
		if elemento.Tamano > mayor {
			mayor = elemento.Tamano
		}
	}

	if total <= mayor {
		return 0
	}
	return total - mayor
}
