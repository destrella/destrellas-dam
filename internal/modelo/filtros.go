package modelo

import "encoding/json"

// CriterioOrdenListado describe el campo principal usado para ordenar el listado.
type CriterioOrdenListado string

const (
	CriterioOrdenNombre            CriterioOrdenListado = "nombre"
	CriterioOrdenFechaModificacion CriterioOrdenListado = "fecha_modificacion"
)

// Normalizado evita valores vacios o desconocidos al leer configuraciones antiguas.
func (c CriterioOrdenListado) Normalizado() CriterioOrdenListado {
	switch c {
	case CriterioOrdenFechaModificacion:
		return c
	default:
		return CriterioOrdenNombre
	}
}

// FiltrosListado controla la vista central.
type FiltrosListado struct {
	MostrarOcultos   bool                 `json:"mostrar_ocultos"`
	OcultarCarpetas  bool                 `json:"ocultar_carpetas"`
	SoloMultimedia   bool                 `json:"solo_multimedia"`
	SoloVideos       bool                 `json:"solo_videos"`
	SoloImagenes     bool                 `json:"solo_imagenes"`
	SoloAudio        bool                 `json:"solo_audio"`
	Recursivo        bool                 `json:"recursivo"`
	CriterioOrden    CriterioOrdenListado `json:"criterio_orden"`
	OrdenDescendente bool                 `json:"orden_descendente"`
	VistaGaleria     bool                 `json:"vista_galeria"`
}

// FiltrosPorDefecto devuelve los filtros iniciales pedidos en la especificacion.
func FiltrosPorDefecto() FiltrosListado {
	return FiltrosListado{
		MostrarOcultos:   false,
		OcultarCarpetas:  true,
		SoloMultimedia:   true,
		SoloVideos:       false,
		SoloImagenes:     false,
		SoloAudio:        false,
		Recursivo:        false,
		CriterioOrden:    CriterioOrdenNombre,
		OrdenDescendente: false,
		VistaGaleria:     true,
	}
}

// CriterioOrdenNormalizado devuelve un criterio consistente para usar en UI y consultas.
func (f FiltrosListado) CriterioOrdenNormalizado() CriterioOrdenListado {
	return f.CriterioOrden.Normalizado()
}

// Acepta determina si un archivo debe mostrarse con los filtros activos.
func (f FiltrosListado) Acepta(archivo Archivo) bool {
	if !f.MostrarOcultos && archivo.EsOculto {
		return false
	}

	if archivo.EsDirectorio {
		return !f.OcultarCarpetas
	}

	if f.SoloVideos || f.SoloImagenes || f.SoloAudio {
		aceptado := false
		if f.SoloVideos && archivo.Tipo == TipoVideo {
			aceptado = true
		}
		if f.SoloImagenes && archivo.Tipo == TipoImagen {
			aceptado = true
		}
		if f.SoloAudio && archivo.Tipo == TipoAudio {
			aceptado = true
		}
		return aceptado
	}

	if f.SoloMultimedia {
		return archivo.EsMultimedia()
	}

	return true
}

// UnmarshalJSON aplica valores por defecto para mantener compatibilidad con configuraciones antiguas.
func (f *FiltrosListado) UnmarshalJSON(datos []byte) error {
	type alias FiltrosListado

	auxiliar := alias(FiltrosPorDefecto())
	if err := json.Unmarshal(datos, &auxiliar); err != nil {
		return err
	}

	*f = FiltrosListado(auxiliar)
	return nil
}
