package modelo

import (
	"path/filepath"
	"strings"
	"time"
)

// Origen identifica el entorno donde vive un archivo.
type Origen string

const (
	OrigenLocal  Origen = "local"
	OrigenYandex Origen = "yandex"
)

// TipoArchivo representa una clasificacion ligera para filtros y acciones.
type TipoArchivo string

const (
	TipoDesconocido TipoArchivo = "desconocido"
	TipoDirectorio  TipoArchivo = "directorio"
	TipoImagen      TipoArchivo = "imagen"
	TipoVideo       TipoArchivo = "video"
	TipoAudio       TipoArchivo = "audio"
	TipoOtro        TipoArchivo = "otro"
)

// Coordenadas guarda una ubicacion GPS.
type Coordenadas struct {
	Latitud  float64 `json:"latitud"`
	Longitud float64 `json:"longitud"`
}

// RegionEtiquetada representa una region normalizada dentro de una imagen.
type RegionEtiquetada struct {
	Nombre string  `json:"nombre"`
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Ancho  float64 `json:"ancho"`
	Alto   float64 `json:"alto"`
}

// HashesArchivo agrupa los hashes usados para deteccion de duplicados.
type HashesArchivo struct {
	MD5         string `json:"md5"`
	SHA256      string `json:"sha256"`
	DHashImagen string `json:"dhash_imagen"`
	DHashVideo  string `json:"dhash_video"`
}

// IndicadoresArchivo resume metadatos especiales visibles en la UI.
type IndicadoresArchivo struct {
	TieneGPS       bool `json:"tiene_gps"`
	TieneRegiones  bool `json:"tiene_regiones"`
	TieneWhereFrom bool `json:"tiene_where_froms"`
	TieneIA        bool `json:"tiene_ia"`
	TieneSocial    bool `json:"tiene_social"`
	EsAdulto       bool `json:"es_adulto"`
}

// MetadatosArchivo contiene los campos editables y de apoyo.
type MetadatosArchivo struct {
	PalabrasClave []string            `json:"palabras_clave"`
	Ubicacion     string              `json:"ubicacion"`
	Fecha         string              `json:"fecha"`
	Hora          string              `json:"hora"`
	ZonaHoraria   string              `json:"zona_horaria"`
	WhereFroms    []string            `json:"where_froms"`
	Coordenadas   *Coordenadas        `json:"coordenadas,omitempty"`
	Regiones      []RegionEtiquetada  `json:"regiones"`
	Orientacion   int                 `json:"orientacion"`
	Rotacion      int                 `json:"rotacion"`
	Comentario    string              `json:"comentario"`
	Sujetos       []string            `json:"sujetos"`
	Copyright     string              `json:"copyright"`
	Pais          string              `json:"pais"`
	Estado        string              `json:"estado"`
	Ciudad        string              `json:"ciudad"`
	Make          string              `json:"make"`
	Modelo        string              `json:"modelo"`
	Software      string              `json:"software"`
	Extras        map[string][]string `json:"extras,omitempty"`
}

// Archivo describe tanto archivos locales como remotos.
type Archivo struct {
	ID           int64
	Origen       Origen
	Ruta         string
	RutaPadre    string
	Nombre       string
	PreviewURL   string
	Tamano       int64
	Modificado   time.Time
	Tipo         TipoArchivo
	EsOculto     bool
	EsDirectorio bool
	Ancho        int
	Alto         int
	Duracion     time.Duration
	Indicadores  IndicadoresArchivo
	Metadatos    MetadatosArchivo
	Hashes       HashesArchivo
}

// EsMultimedia informa si el archivo es imagen, video o audio.
func (a Archivo) EsMultimedia() bool {
	switch a.Tipo {
	case TipoImagen, TipoVideo, TipoAudio:
		return true
	default:
		return false
	}
}

// TipoDesdeRuta clasifica un archivo por extension para evitar trabajo costoso temprano.
func TipoDesdeRuta(ruta string, esDirectorio bool) TipoArchivo {
	if esDirectorio {
		return TipoDirectorio
	}

	extension := strings.ToLower(filepath.Ext(ruta))
	switch extension {
	case ".jpg", ".jpeg", ".png", ".gif", ".webp", ".tif", ".tiff", ".bmp", ".heic", ".heif", ".avif",
		".dng", ".raw", ".cr2", ".cr3", ".crw", ".nef", ".nrw", ".arw", ".srf", ".sr2", ".raf", ".rw2",
		".orf", ".pef", ".iiq", ".3fr", ".fff", ".rwl", ".mef", ".mos", ".mrw", ".x3f", ".erf", ".kdc",
		".dcr", ".bay", ".cap", ".eip":
		return TipoImagen
	case ".mp4", ".mov", ".m4v", ".mkv", ".avi", ".webm", ".mpg", ".mpeg", ".mts", ".m2ts", ".3gp":
		return TipoVideo
	case ".mp3", ".wav", ".aac", ".m4a", ".flac", ".ogg", ".opus", ".aiff":
		return TipoAudio
	default:
		if extension == "" {
			return TipoDesconocido
		}
		return TipoOtro
	}
}

// AdmitePreview informa si el archivo puede mostrar miniatura en la UI.
func (a Archivo) AdmitePreview() bool {
	if a.EsDirectorio {
		return false
	}
	switch a.Tipo {
	case TipoImagen, TipoVideo:
		return true
	}
	if a.Origen == OrigenYandex && strings.TrimSpace(a.PreviewURL) != "" {
		return true
	}
	switch strings.ToLower(filepath.Ext(a.Ruta)) {
	case ".pdf", ".psd", ".psb":
		return true
	default:
		return false
	}
}

// NombreVisible devuelve un nombre listo para UI.
func (a Archivo) NombreVisible() string {
	if a.Nombre != "" {
		return a.Nombre
	}
	return filepath.Base(a.Ruta)
}

// EsOcultoPorNombre aplica la regla usual de Unix/macOS.
func EsOcultoPorNombre(nombre string) bool {
	return strings.HasPrefix(nombre, ".")
}

// IndicadoresVacios ayuda a decidir si hay datos enriquecidos reales.
func (i IndicadoresArchivo) IndicadoresVacios() bool {
	return !i.TieneGPS && !i.TieneRegiones && !i.TieneWhereFrom && !i.TieneIA && !i.TieneSocial && !i.EsAdulto
}

// MetadatosVacios informa si el bloque no aporta datos reales.
func (m MetadatosArchivo) MetadatosVacios() bool {
	return len(m.PalabrasClave) == 0 &&
		m.Ubicacion == "" &&
		m.Fecha == "" &&
		m.Hora == "" &&
		m.ZonaHoraria == "" &&
		len(m.WhereFroms) == 0 &&
		m.Coordenadas == nil &&
		len(m.Regiones) == 0 &&
		m.Orientacion == 0 &&
		m.Rotacion == 0 &&
		m.Comentario == "" &&
		len(m.Sujetos) == 0 &&
		m.Copyright == "" &&
		m.Pais == "" &&
		m.Estado == "" &&
		m.Ciudad == "" &&
		m.Make == "" &&
		m.Modelo == "" &&
		m.Software == "" &&
		len(m.Extras) == 0
}
