package ui

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/dustin/go-humanize"

	"gioui.org/app"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/paint"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"destrellas-dam/internal/almacen"
	"destrellas-dam/internal/configuracion"
	"destrellas-dam/internal/modelo"
	"destrellas-dam/internal/servicios/archivos"
	"destrellas-dam/internal/servicios/duplicados"
	"destrellas-dam/internal/servicios/indexador"
	"destrellas-dam/internal/servicios/metadatos"
	"destrellas-dam/internal/yandex"
)

type tipoVista string

const (
	vistaPrincipal     tipoVista = "principal"
	vistaElementoUnico tipoVista = "elemento_unico"
	vistaDuplicados    tipoVista = "duplicados"
	vistaUbicaciones   tipoVista = "ubicaciones"
	vistaAsociaciones  tipoVista = "asociaciones"
	vistaConfiguracion tipoVista = "configuracion"
)

type tipoPestanaLateral string

const (
	pestanaDirectorios tipoPestanaLateral = "directorios"
	pestanaPalabras    tipoPestanaLateral = "palabras"
	pestanaLugares     tipoPestanaLateral = "lugares"
	pestanaYandex      tipoPestanaLateral = "yandex"
)

type tipoOrigenListado string

const (
	origenListadoCarpeta            tipoOrigenListado = "carpeta"
	origenListadoCarpetaYandex      tipoOrigenListado = "carpeta_yandex"
	origenListadoEtiqueta           tipoOrigenListado = "etiqueta"
	origenListadoUbicacion          tipoOrigenListado = "ubicacion"
	origenListadoUbicacionSinNombre tipoOrigenListado = "ubicacion_sin_nombre"
	etiquetaUbicacionSinNombre                        = "Ubicación sin nombre"
)

type opcionFiltroLateral struct {
	Clave    string
	Etiqueta string
}

// Dependencias agrupa servicios ya inicializados desde main.
type Dependencias struct {
	RepositorioConfig *configuracion.Repositorio
	Configuracion     configuracion.Configuracion
	Almacen           almacen.Repositorio
	Listador          *indexador.ListadorLocal
	Indexador         *indexador.Servicio
	Metadatos         *metadatos.Servicio
	Archivos          *archivos.Servicio
	Duplicados        *duplicados.Servicio
	Yandex            yandex.Cliente
}

type nodoArbolUI struct {
	Origen      modelo.Origen
	Ruta        string
	Nombre      string
	Expandido   bool
	Cargado     bool
	Cargando    bool
	Hijos       []*nodoArbolUI
	Seleccionar widget.Clickable
	Alternar    widget.Clickable
}

type nodoVisible struct {
	Nodo  *nodoArbolUI
	Nivel int
}

type widgetsElemento struct {
	Fila      widget.Clickable
	Seleccion widget.Bool
}

type widgetsGrupoDuplicado struct {
	BorrarMasAntiguo widget.Clickable
	BorrarMasNuevo   widget.Clickable
	BorrarMarcados   widget.Clickable
	AlternarColapso  widget.Clickable
	Seleccion        map[string]*widget.Bool
	BorrarElemento   map[string]*widget.Clickable
	SeleccionarRuta  map[string]*widget.Clickable
}

type widgetsSelectorDirectorio struct {
	Seleccionar widget.Clickable
	Alternar    widget.Clickable
}

func (a *Aplicacion) cambiarVista(vista tipoVista) {
	if a.vistaActual == vistaElementoUnico && vista != vistaElementoUnico {
		a.detenerReproduccionVideo()
	}
	a.vistaActual = vista
}

type estadoSelectorDirectorio struct {
	Expandido bool
	Alternar  widget.Clickable
}

type estadoPreview struct {
	Imagen      image.Image
	Cargando    bool
	Maximo      int
	Orientacion int
	Rotacion    int
}

type tipoSeleccionLote string

const (
	seleccionLoteVacia  tipoSeleccionLote = "vacia"
	seleccionLoteLocal  tipoSeleccionLote = "local"
	seleccionLoteYandex tipoSeleccionLote = "yandex"
	seleccionLoteMixta  tipoSeleccionLote = "mixta"
)

type tipoDescubrimientoDuplicados string

const (
	descubrimientoDuplicadosNinguno tipoDescubrimientoDuplicados = "ninguno"
	descubrimientoDuplicadosLocal   tipoDescubrimientoDuplicados = "local"
	descubrimientoDuplicadosRemoto  tipoDescubrimientoDuplicados = "remoto"
)

type estadoBusquedaLateral struct {
	Consulta string
	Opciones []opcionFiltroLateral
	Cargando bool
	Version  int
}

type tipoFiltroAsociacionTexto string

const (
	filtroAsociacionTextoTodas      tipoFiltroAsociacionTexto = "todas"
	filtroAsociacionTextoOriginales tipoFiltroAsociacionTexto = "originales"
	filtroAsociacionTextoSugeridas  tipoFiltroAsociacionTexto = "sugeridas"
)

// Aplicacion mantiene el estado inmediato de la UI.
type Aplicacion struct {
	tema   *material.Theme
	paleta Paleta

	repoConfiguracion  *configuracion.Repositorio
	configuracion      configuracion.Configuracion
	almacen            almacen.Repositorio
	listador           *indexador.ListadorLocal
	indexador          *indexador.Servicio
	servicioMetadatos  *metadatos.Servicio
	servicioArchivos   *archivos.Servicio
	servicioDuplicados *duplicados.Servicio
	clienteYandex      yandex.Cliente
	rutaUsuario        string
	rutaLibraryUsuario string

	ventana *app.Window
	ops     op.Ops

	vistaActual    tipoVista
	pestanaLateral tipoPestanaLateral

	filtros             modelo.FiltrosListado
	carpetaSeleccionada string
	origenListado       tipoOrigenListado
	claveListadoActual  string
	versionListado      int
	offsetListado       int
	objetivoListado     int
	sesionListado       *indexador.SesionListado
	elementos           []modelo.Archivo
	cargandoElementos   bool
	hayMasElementos     bool
	seleccionLote       map[string]bool
	anclaSeleccionLote  string

	archivoActivo      modelo.Archivo
	tieneArchivoActivo bool

	raizArbol       *nodoArbolUI
	raizArbolYandex *nodoArbolUI

	carpetaYandexSeleccionada string

	etiquetas                 []opcionFiltroLateral
	ubicacionesNombradas      []opcionFiltroLateral
	ubicacionesGuardadas      []modelo.UbicacionGuardada
	ubicacionSeleccionada     string
	usosUbicacionSeleccionada []modelo.UsoUbicacionGuardada
	cargandoUsosUbicacion     bool
	asociacionesTexto         []modelo.AsociacionTexto
	asociacionTextoActivaID   int64
	filtroAsociacionesTexto   tipoFiltroAsociacionTexto
	busquedaEtiquetas         estadoBusquedaLateral
	busquedaUbicaciones       estadoBusquedaLateral

	previews map[string]*estadoPreview

	metadatosPendientes  map[string]bool
	metadatosVerificados map[string]int64
	reproductorVideo     estadoReproductorVideo
	edicionRegiones      estadoEdicionRegiones
	edicionRecorte       estadoEdicionRecorte

	gruposDuplicados           []modelo.GrupoDuplicados
	tipoCoincidenciaActual     modelo.TipoCoincidencia
	categoriaDuplicados        modelo.CategoriaDuplicados
	ordenDuplicados            modelo.OrdenDuplicados
	rutaPreviewDuplicados      string
	gruposDuplicadosContraidos map[string]bool
	duplicadosInicializados    bool
	progresoDuplicados         indexador.EventoProgreso
	cargandoDuplicados         bool
	limpiandoDuplicados        bool
	cancelarCargaDuplicados    context.CancelFunc
	versionCargaDuplicados     int
	cancelarDescubrimiento     context.CancelFunc
	versionDescubrimiento      int
	tipoDescubrimiento         tipoDescubrimientoDuplicados
	escaneoRemotoPendiente     bool
	progresoMetadatos          indexador.EventoProgreso
	escanandoMetadatos         bool
	cancelarEscaneoMetadatos   context.CancelFunc
	versionEscaneoMetadatos    int

	actualizaciones chan func()
	mensajeEstado   string
	ultimoError     string

	// Navegacion superior.
	botonVistaPrincipal     widget.Clickable
	botonVistaElementoUnico widget.Clickable
	botonVistaDuplicados    widget.Clickable
	botonVistaUbicaciones   widget.Clickable
	botonVistaAsociaciones  widget.Clickable
	botonVistaConfiguracion widget.Clickable

	// Pestañas laterales.
	botonPestanaDirectorios widget.Clickable
	botonPestanaPalabras    widget.Clickable
	botonPestanaLugares     widget.Clickable
	botonPestanaYandex      widget.Clickable

	// Filtros.
	mostrarOcultos           widget.Bool
	ocultarCarpetas          widget.Bool
	soloMultimedia           widget.Bool
	soloVideos               widget.Bool
	soloImagenes             widget.Bool
	soloAudio                widget.Bool
	recursivo                widget.Bool
	botonGaleria             widget.Clickable
	botonLista               widget.Clickable
	botonOrdenAZ             widget.Clickable
	botonOrdenZA             widget.Clickable
	botonOrdenAntiguos       widget.Clickable
	botonOrdenNuevos         widget.Clickable
	botonSeleccionarTodo     widget.Clickable
	botonDeseleccionarTodo   widget.Clickable
	editorFiltroEtiquetas    widget.Editor
	editorFiltroLugares      widget.Editor
	editorFiltroAsociaciones widget.Editor
	selectorActivoLocal      estadoSelectorDirectorio
	selectorActivoRemoto     estadoSelectorDirectorio
	selectorLoteLocal        estadoSelectorDirectorio
	selectorLoteRemoto       estadoSelectorDirectorio

	// Vistas principales.
	listaLateral              widget.List
	listaCentro               widget.List
	listaDetalle              widget.List
	listaDuplicados           widget.List
	listaUbicaciones          widget.List
	listaAsociaciones         widget.List
	listaUsosUbicacion        widget.List
	listaRelacionUbicacion    widget.List
	listaConfiguracion        widget.List
	listaNombreVisor          widget.List
	listaSelectorActivoLocal  widget.List
	listaSelectorActivoRemoto widget.List
	listaSelectorLoteLocal    widget.List
	listaSelectorLoteRemoto   widget.List

	elementoWidgets             map[string]*widgetsElemento
	grupoWidgets                map[string]*widgetsGrupoDuplicado
	widgetsLateral              map[string]*widget.Clickable
	widgetsSelectorActivoLocal  map[string]*widgetsSelectorDirectorio
	widgetsSelectorActivoRemoto map[string]*widgetsSelectorDirectorio
	widgetsSelectorLoteLocal    map[string]*widgetsSelectorDirectorio
	widgetsSelectorLoteRemoto   map[string]*widgetsSelectorDirectorio

	// Acciones generales y de lote.
	editorDestinoMover      widget.Editor
	editorDestinoLote       widget.Editor
	rutaDestinoActivoLocal  string
	rutaDestinoActivoRemoto string
	rutaDestinoLoteLocal    string
	rutaDestinoLoteRemoto   string
	botonMoverActivo        widget.Clickable
	botonArchivarActivo     widget.Clickable
	botonPapeleraActiva     widget.Clickable
	botonDescargarActivo    widget.Clickable
	botonGuardarLocalActivo widget.Clickable
	botonGuardarMetadatos   widget.Clickable
	botonMoverLote          widget.Clickable
	botonArchivarLote       widget.Clickable
	botonPapeleraLote       widget.Clickable
	botonDescargarLote      widget.Clickable

	// Editores de metadatos.
	editorFecha                widget.Editor
	editorHora                 widget.Editor
	editorZonaHoraria          widget.Editor
	editorPalabras             widget.Editor
	editorUbicacion            widget.Editor
	editorFiltroUbicaciones    widget.Editor
	editorRelacionUbicacion    widget.Editor
	editorAsociacionOriginales widget.Editor
	editorAsociacionSugeridas  widget.Editor
	editorComentario           widget.Editor
	editorCopyright            widget.Editor
	editorGPSLatitud           widget.Editor
	editorGPSLongitud          widget.Editor
	editorMake                 widget.Editor
	editorModelo               widget.Editor
	editorSoftware             widget.Editor
	formularioMetadatos        estadoFormularioMetadatos

	// Vista de elemento único.
	botonVisorAnterior             widget.Clickable
	botonVisorSiguiente            widget.Clickable
	botonAgregarRegion             widget.Clickable
	botonLimpiarRegiones           widget.Clickable
	botonGuardarRegiones           widget.Clickable
	botonSeleccionarRecorte        widget.Clickable
	botonRecortar                  widget.Clickable
	reemplazarOriginalRecorte      widget.Bool
	botonConvertir                 widget.Clickable
	botonExtraerFrame              widget.Clickable
	botonOptimizarVideo            widget.Clickable
	editorFormatoImagen            widget.Editor
	sobreescribirVideo             widget.Bool
	controlExtraccionFrame         widget.Float
	formatoExtraccionFrame         string
	formatoExtraccionExpandido     bool
	botonSelectorFormatoExtraccion widget.Clickable
	opcionesFormatoExtraccion      map[string]*widget.Clickable

	// Vista de duplicados.
	editorRutaEscaneoDuplicados widget.Editor
	botonDuplicadosLocales      widget.Clickable
	botonDuplicadosRemotos      widget.Clickable
	botonDuplicadosMixtos       widget.Clickable
	soloDuplicadosMultimedia    widget.Bool
	botonEscanearLocal          widget.Clickable
	botonEscanearRemoto         widget.Clickable
	botonCoincidenciaExacta     widget.Clickable
	botonCoincidenciaImagen     widget.Clickable
	botonCoincidenciaVideo      widget.Clickable
	botonOrdenGrupo             widget.Clickable
	botonOrdenEspacio           widget.Clickable
	botonOrdenAlfabetico        widget.Clickable
	botonRecargarDuplicados     widget.Clickable
	botonLimpiarDuplicados      widget.Clickable

	// Configuracion.
	editorCarpetaInicial              widget.Editor
	editorCarpetaArchivado            widget.Editor
	editorClaveYandex                 widget.Editor
	editorRutaEscaneoMetadatos        widget.Editor
	configMostrarOcultos              widget.Bool
	configOcultarCarpetas             widget.Bool
	configSoloMultimedia              widget.Bool
	configSoloVideos                  widget.Bool
	configSoloImagenes                widget.Bool
	configSoloAudio                   widget.Bool
	configRecursivo                   widget.Bool
	configOrdenPorFecha               widget.Bool
	configOrdenDescendente            widget.Bool
	botonConfigOrdenAZ                widget.Clickable
	botonConfigOrdenZA                widget.Clickable
	botonConfigOrdenAntiguos          widget.Clickable
	botonConfigOrdenNuevos            widget.Clickable
	botonFiltroAsociacionesTodas      widget.Clickable
	botonFiltroAsociacionesOriginales widget.Clickable
	botonFiltroAsociacionesSugeridas  widget.Clickable
	botonNuevaAsociacionTexto         widget.Clickable
	botonGuardarAsociacionTexto       widget.Clickable
	botonEliminarAsociacionTexto      widget.Clickable
	botonEscanearMetadatos            widget.Clickable
	botonPausarEscaneo                widget.Clickable
	botonGuardarRelacionUbicacion     widget.Clickable
	botonQuitarRelacionUbicacion      widget.Clickable
	botonGuardarConfig                widget.Clickable

	botonAlternarVideo           widget.Clickable
	botonReiniciarVideo          widget.Clickable
	botonReproducirVideo         widget.Clickable
	botonLoopVideo               widget.Clickable
	botonAbrirCarpetaContenedora widget.Clickable
	botonPreviewVisor            widget.Clickable
	controlProgresoVideo         widget.Float
	reproducirVideoEnLoop        bool
}

// NuevaAplicacion construye la interfaz y sincroniza sus widgets con la configuracion.
func NuevaAplicacion(dependencias Dependencias) *Aplicacion {
	paleta := nuevaPaleta()
	tema := nuevaTema(paleta)
	rutaUsuario := resolverRutaUsuarioLocal(dependencias.Configuracion.CarpetaInicial)

	appUI := &Aplicacion{
		tema:                        tema,
		paleta:                      paleta,
		repoConfiguracion:           dependencias.RepositorioConfig,
		configuracion:               dependencias.Configuracion,
		almacen:                     dependencias.Almacen,
		listador:                    dependencias.Listador,
		indexador:                   dependencias.Indexador,
		servicioMetadatos:           dependencias.Metadatos,
		servicioArchivos:            dependencias.Archivos,
		servicioDuplicados:          dependencias.Duplicados,
		clienteYandex:               dependencias.Yandex,
		rutaUsuario:                 rutaUsuario,
		rutaLibraryUsuario:          filepath.Join(rutaUsuario, "Library"),
		vistaActual:                 vistaPrincipal,
		pestanaLateral:              pestanaDirectorios,
		filtroAsociacionesTexto:     filtroAsociacionTextoTodas,
		filtros:                     dependencias.Configuracion.FiltrosPorDefecto,
		carpetaSeleccionada:         dependencias.Configuracion.CarpetaInicial,
		origenListado:               origenListadoCarpeta,
		claveListadoActual:          dependencias.Configuracion.CarpetaInicial,
		hayMasElementos:             true,
		seleccionLote:               make(map[string]bool),
		previews:                    make(map[string]*estadoPreview),
		metadatosPendientes:         make(map[string]bool),
		metadatosVerificados:        make(map[string]int64),
		tipoCoincidenciaActual:      modelo.CoincidenciaExacta,
		categoriaDuplicados:         modelo.CategoriaDuplicadosLocales,
		ordenDuplicados:             modelo.OrdenPorTamanoGrupo,
		actualizaciones:             make(chan func(), 512),
		elementoWidgets:             make(map[string]*widgetsElemento),
		gruposDuplicadosContraidos:  make(map[string]bool),
		grupoWidgets:                make(map[string]*widgetsGrupoDuplicado),
		widgetsLateral:              make(map[string]*widget.Clickable),
		widgetsSelectorActivoLocal:  make(map[string]*widgetsSelectorDirectorio),
		widgetsSelectorActivoRemoto: make(map[string]*widgetsSelectorDirectorio),
		widgetsSelectorLoteLocal:    make(map[string]*widgetsSelectorDirectorio),
		widgetsSelectorLoteRemoto:   make(map[string]*widgetsSelectorDirectorio),
	}

	appUI.listaLateral.Axis = layout.Vertical
	appUI.listaCentro.Axis = layout.Vertical
	appUI.listaDetalle.Axis = layout.Vertical
	appUI.listaDuplicados.Axis = layout.Vertical
	appUI.listaUbicaciones.Axis = layout.Vertical
	appUI.listaAsociaciones.Axis = layout.Vertical
	appUI.listaUsosUbicacion.Axis = layout.Vertical
	appUI.listaRelacionUbicacion.Axis = layout.Vertical
	appUI.listaConfiguracion.Axis = layout.Vertical
	appUI.listaNombreVisor.Axis = layout.Horizontal
	appUI.listaSelectorActivoLocal.Axis = layout.Vertical
	appUI.listaSelectorActivoRemoto.Axis = layout.Vertical
	appUI.listaSelectorLoteLocal.Axis = layout.Vertical
	appUI.listaSelectorLoteRemoto.Axis = layout.Vertical

	appUI.mostrarOcultos.Value = appUI.filtros.MostrarOcultos
	appUI.ocultarCarpetas.Value = appUI.filtros.OcultarCarpetas
	appUI.soloMultimedia.Value = appUI.filtros.SoloMultimedia
	appUI.soloVideos.Value = appUI.filtros.SoloVideos
	appUI.soloImagenes.Value = appUI.filtros.SoloImagenes
	appUI.soloAudio.Value = appUI.filtros.SoloAudio
	appUI.recursivo.Value = appUI.filtros.Recursivo

	appUI.editorDestinoMover.SingleLine = true
	appUI.editorDestinoLote.SingleLine = true
	appUI.rutaDestinoActivoLocal = appUI.rutaUsuario
	appUI.rutaDestinoLoteLocal = appUI.rutaUsuario
	appUI.rutaDestinoActivoRemoto = "disk:/"
	appUI.rutaDestinoLoteRemoto = "disk:/"
	appUI.editorFecha.SingleLine = true
	appUI.editorHora.SingleLine = true
	appUI.editorZonaHoraria.SingleLine = true
	appUI.editorUbicacion.SingleLine = true
	appUI.editorCopyright.SingleLine = true
	appUI.editorGPSLatitud.SingleLine = true
	appUI.editorGPSLongitud.SingleLine = true
	appUI.editorMake.SingleLine = true
	appUI.editorModelo.SingleLine = true
	appUI.editorSoftware.SingleLine = true
	appUI.editorFormatoImagen.SingleLine = true
	appUI.editorFormatoImagen.SetText("webp")
	appUI.reemplazarOriginalRecorte.Value = true
	appUI.formatoExtraccionFrame = "webp"
	appUI.editorFiltroEtiquetas.SingleLine = true
	appUI.editorFiltroLugares.SingleLine = true
	appUI.editorFiltroAsociaciones.SingleLine = true
	appUI.editorFiltroUbicaciones.SingleLine = true
	appUI.editorRelacionUbicacion.SingleLine = true
	appUI.editorAsociacionOriginales.SingleLine = true
	appUI.editorAsociacionSugeridas.SingleLine = true
	appUI.formularioMetadatos.listaAtributoExtendido.Axis = layout.Vertical
	appUI.formularioMetadatos.listaUbicacionesSugeridas.Axis = layout.Vertical
	appUI.formularioMetadatos.listaSalidaExiftool.Axis = layout.Vertical

	appUI.editorCarpetaInicial.SingleLine = true
	appUI.editorCarpetaInicial.SetText(appUI.configuracion.CarpetaInicial)
	appUI.editorCarpetaArchivado.SingleLine = true
	appUI.editorCarpetaArchivado.SetText(appUI.configuracion.CarpetaArchivado)
	appUI.editorClaveYandex.SingleLine = true
	appUI.editorClaveYandex.SetText(appUI.configuracion.ClaveAPIYandex)
	appUI.editorRutaEscaneoMetadatos.SingleLine = true
	appUI.editorRutaEscaneoMetadatos.SetText(appUI.rutaUsuario)
	appUI.editorRutaEscaneoDuplicados.SingleLine = true
	appUI.editorRutaEscaneoDuplicados.SetText(appUI.rutaUsuario)

	if appUI.servicioDuplicados != nil {
		if estadoRemoto, err := appUI.servicioDuplicados.CargarEstadoDescubrimientoRemoto(); err == nil && estadoRemoto.Pendiente {
			appUI.progresoDuplicados = indexador.EventoProgreso{
				RutaActual:            estadoRemoto.RutaActual,
				DirectoriosProcesados: estadoRemoto.DirectoriosProcesados,
				ArchivosEncontrados:   estadoRemoto.ArchivosEncontrados,
				ArchivosAnalizados:    estadoRemoto.ArchivosAnalizados,
			}
			appUI.escaneoRemotoPendiente = true
			if strings.TrimSpace(estadoRemoto.RutaRaiz) != "" {
				appUI.editorRutaEscaneoDuplicados.SetText(estadoRemoto.RutaRaiz)
			}
		}
	}

	appUI.configMostrarOcultos.Value = appUI.configuracion.FiltrosPorDefecto.MostrarOcultos
	appUI.configOcultarCarpetas.Value = appUI.configuracion.FiltrosPorDefecto.OcultarCarpetas
	appUI.configSoloMultimedia.Value = appUI.configuracion.FiltrosPorDefecto.SoloMultimedia
	appUI.configSoloVideos.Value = appUI.configuracion.FiltrosPorDefecto.SoloVideos
	appUI.configSoloImagenes.Value = appUI.configuracion.FiltrosPorDefecto.SoloImagenes
	appUI.configSoloAudio.Value = appUI.configuracion.FiltrosPorDefecto.SoloAudio
	appUI.configRecursivo.Value = appUI.configuracion.FiltrosPorDefecto.Recursivo
	appUI.configOrdenPorFecha.Value = appUI.configuracion.FiltrosPorDefecto.CriterioOrdenNormalizado() == modelo.CriterioOrdenFechaModificacion
	appUI.configOrdenDescendente.Value = appUI.configuracion.FiltrosPorDefecto.OrdenDescendente

	appUI.reconstruirArbol()
	appUI.reiniciarListado()
	appUI.recargarColeccionesLaterales()
	appUI.recargarAsociacionesTexto()

	return appUI
}

// Ejecutar inicia el bucle principal de la ventana.
func (a *Aplicacion) Ejecutar(ventana *app.Window) error {
	a.ventana = ventana

	for {
		evento := ventana.Event()
		switch evento := evento.(type) {
		case app.DestroyEvent:
			if a.sesionListado != nil {
				_ = a.sesionListado.Cerrar()
			}
			if a.cancelarCargaDuplicados != nil {
				a.cancelarCargaDuplicados()
			}
			if a.cancelarDescubrimiento != nil {
				a.cancelarDescubrimiento()
			}
			if a.cancelarEscaneoMetadatos != nil {
				a.cancelarEscaneoMetadatos()
			}
			return evento.Err
		case app.FrameEvent:
			a.drenarActualizaciones()
			gtx := app.NewContext(&a.ops, evento)
			a.dibujar(gtx)
			evento.Frame(gtx.Ops)
		}
	}
}

func (a *Aplicacion) drenarActualizaciones() {
	for {
		select {
		case actualizacion := <-a.actualizaciones:
			if actualizacion != nil {
				actualizacion()
			}
		default:
			return
		}
	}
}

func (a *Aplicacion) encolarActualizacion(actualizacion func()) {
	select {
	case a.actualizaciones <- actualizacion:
	default:
		go func() {
			a.actualizaciones <- actualizacion
		}()
	}
	if a.ventana != nil {
		a.ventana.Invalidate()
	}
}

func (a *Aplicacion) establecerEstado(mensaje string, err error) {
	a.mensajeEstado = mensaje
	if err != nil {
		a.ultimoError = err.Error()
	} else {
		a.ultimoError = ""
	}
}

func resolverRutaUsuarioLocal(rutaRespaldo string) string {
	rutaRespaldo = strings.TrimSpace(rutaRespaldo)
	if rutaUsuario, err := os.UserHomeDir(); err == nil {
		rutaUsuario = strings.TrimSpace(rutaUsuario)
		if rutaUsuario != "" {
			return filepath.Clean(rutaUsuario)
		}
	}
	if rutaRespaldo == "" {
		return string(filepath.Separator)
	}
	return filepath.Clean(rutaRespaldo)
}

func (a *Aplicacion) resolverRutaEscaneo(texto string) (string, error) {
	ruta := strings.TrimSpace(texto)
	if ruta == "" {
		ruta = a.rutaUsuario
	}

	rutaAbsoluta, err := filepath.Abs(ruta)
	if err != nil {
		return "", fmt.Errorf("no se pudo resolver la ruta de escaneo: %w", err)
	}
	rutaAbsoluta = filepath.Clean(rutaAbsoluta)

	info, err := os.Stat(rutaAbsoluta)
	if err != nil {
		return "", fmt.Errorf("no se pudo acceder a la ruta de escaneo: %w", err)
	}
	if !info.IsDir() {
		return "", errors.New("la ruta de escaneo debe ser una carpeta")
	}
	if !rutaEsIgualODescendiente(rutaAbsoluta, a.rutaUsuario) {
		return "", fmt.Errorf("la ruta de escaneo debe estar dentro de %s", a.rutaUsuario)
	}
	if rutaEsIgualODescendiente(rutaAbsoluta, a.rutaLibraryUsuario) {
		return "", fmt.Errorf("la ruta de escaneo no puede estar dentro de %s", a.rutaLibraryUsuario)
	}
	return rutaAbsoluta, nil
}

func (a *Aplicacion) rutasExcluidasEscaneo(raiz string) []string {
	if rutaEsIgualODescendiente(a.rutaLibraryUsuario, raiz) {
		return []string{a.rutaLibraryUsuario}
	}
	return nil
}

func rutaEsIgualODescendiente(ruta, base string) bool {
	ruta = filepath.Clean(strings.TrimSpace(ruta))
	base = filepath.Clean(strings.TrimSpace(base))
	if ruta == "" || base == "" {
		return false
	}
	if ruta == base {
		return true
	}
	return strings.HasPrefix(ruta, base+string(filepath.Separator))
}

func (a *Aplicacion) reconstruirArbol() {
	nombre := filepath.Base(a.configuracion.CarpetaInicial)
	if nombre == "" || nombre == "." || nombre == string(filepath.Separator) {
		nombre = a.configuracion.CarpetaInicial
	}
	a.raizArbol = &nodoArbolUI{
		Origen:    modelo.OrigenLocal,
		Ruta:      a.configuracion.CarpetaInicial,
		Nombre:    nombre,
		Expandido: true,
		Cargado:   false,
	}
	a.asegurarHijosNodo(a.raizArbol)
}

func (a *Aplicacion) reconstruirArbolYandex() {
	a.raizArbolYandex = &nodoArbolUI{
		Origen:    modelo.OrigenYandex,
		Ruta:      "disk:/",
		Nombre:    "Yandex.Disk",
		Expandido: true,
		Cargado:   false,
	}
}

func (a *Aplicacion) asegurarArbolYandex() {
	if a.raizArbolYandex == nil {
		a.reconstruirArbolYandex()
	}
	if a.raizArbolYandex != nil && !a.raizArbolYandex.Cargado && !a.raizArbolYandex.Cargando {
		a.asegurarHijosNodo(a.raizArbolYandex)
	}
}

func (a *Aplicacion) asegurarArbolLocal() {
	if a.raizArbol == nil {
		a.reconstruirArbol()
	}
}

func claveSelectorDirectorio(origen modelo.Origen, ruta string) string {
	return string(origen) + "::" + strings.TrimSpace(ruta)
}

func (a *Aplicacion) asegurarWidgetSelectorDirectorio(mapa map[string]*widgetsSelectorDirectorio, origen modelo.Origen, ruta string) *widgetsSelectorDirectorio {
	if mapa == nil {
		return &widgetsSelectorDirectorio{}
	}
	clave := claveSelectorDirectorio(origen, ruta)
	if existente, ok := mapa[clave]; ok && existente != nil {
		return existente
	}
	nuevo := &widgetsSelectorDirectorio{}
	mapa[clave] = nuevo
	return nuevo
}

func (a *Aplicacion) normalizarRutaLocalDestino(ruta string) string {
	ruta = strings.TrimSpace(ruta)
	if ruta == "" {
		return a.rutaUsuario
	}
	return filepath.Clean(ruta)
}

func (a *Aplicacion) establecerRutaDestinoActivoLocal(ruta string) {
	ruta = a.normalizarRutaLocalDestino(ruta)
	a.rutaDestinoActivoLocal = ruta
	a.editorDestinoMover.SetText(ruta)
	a.asegurarArbolLocal()
	if err := a.sincronizarArbolConRuta(ruta); err != nil {
		a.establecerEstado("No se pudo sincronizar el selector de carpeta local", err)
	}
}

func (a *Aplicacion) establecerRutaDestinoLoteLocal(ruta string) {
	ruta = a.normalizarRutaLocalDestino(ruta)
	a.rutaDestinoLoteLocal = ruta
	a.editorDestinoLote.SetText(ruta)
	a.asegurarArbolLocal()
	if err := a.sincronizarArbolConRuta(ruta); err != nil {
		a.establecerEstado("No se pudo sincronizar el selector de carpeta local", err)
	}
}

func (a *Aplicacion) establecerRutaDestinoActivoRemoto(ruta string) {
	ruta = normalizarRutaYandexUI(ruta)
	a.rutaDestinoActivoRemoto = ruta
	a.asegurarArbolYandex()
	if err := a.sincronizarArbolYandexConRuta(ruta); err != nil {
		a.establecerEstado("No se pudo sincronizar el selector remoto de Yandex.Disk", err)
	}
}

func (a *Aplicacion) establecerRutaDestinoLoteRemoto(ruta string) {
	ruta = normalizarRutaYandexUI(ruta)
	a.rutaDestinoLoteRemoto = ruta
	a.asegurarArbolYandex()
	if err := a.sincronizarArbolYandexConRuta(ruta); err != nil {
		a.establecerEstado("No se pudo sincronizar el selector remoto de Yandex.Disk", err)
	}
}

func (a *Aplicacion) aplanarArbol() []nodoVisible {
	return a.aplanarArbolDesdeRaiz(a.raizArbol)
}

func (a *Aplicacion) aplanarArbolYandex() []nodoVisible {
	return a.aplanarArbolDesdeRaiz(a.raizArbolYandex)
}

func (a *Aplicacion) aplanarArbolDesdeRaiz(raiz *nodoArbolUI) []nodoVisible {
	if raiz == nil {
		return nil
	}

	var visibles []nodoVisible
	var recorrer func(nodo *nodoArbolUI, nivel int)
	recorrer = func(nodo *nodoArbolUI, nivel int) {
		visibles = append(visibles, nodoVisible{Nodo: nodo, Nivel: nivel})
		if !nodo.Expandido {
			return
		}
		for _, hijo := range nodo.Hijos {
			recorrer(hijo, nivel+1)
		}
	}
	recorrer(raiz, 0)
	return visibles
}

func (a *Aplicacion) asegurarHijosNodo(nodo *nodoArbolUI) {
	if nodo == nil || nodo.Cargado || nodo.Cargando {
		return
	}

	if nodo.Origen == modelo.OrigenYandex {
		a.asegurarHijosNodoYandex(nodo)
		return
	}

	nodo.Cargando = true
	ruta := nodo.Ruta
	go func() {
		subdirectorios, err := a.listador.ListarSubdirectorios(context.Background(), ruta, a.filtros.MostrarOcultos)
		a.encolarActualizacion(func() {
			nodo.Cargando = false
			if err != nil {
				a.establecerEstado("No se pudieron cargar las subcarpetas", err)
				return
			}

			a.fusionarHijosNodo(nodo, subdirectorios)
			sort.SliceStable(nodo.Hijos, func(i, j int) bool {
				return compararTextoUI(nodo.Hijos[i].Nombre, nodo.Hijos[j].Nombre)
			})
			nodo.Cargado = true
		})
	}()
}

func (a *Aplicacion) asegurarHijosNodoYandex(nodo *nodoArbolUI) {
	if nodo == nil || nodo.Cargado || nodo.Cargando {
		return
	}

	if a.clienteYandex == nil || !a.clienteYandex.Configurado() {
		nodo.Cargado = true
		nodo.Cargando = false
		nodo.Hijos = nil
		return
	}

	nodo.Cargando = true
	ruta := nodo.Ruta
	go func() {
		subdirectorios, err := a.listarDirectoriosYandex(context.Background(), ruta)
		a.encolarActualizacion(func() {
			nodo.Cargando = false
			if err != nil {
				a.establecerEstado("No se pudieron cargar las carpetas remotas de Yandex.Disk", err)
				return
			}

			a.fusionarHijosNodoYandex(nodo, subdirectorios)
			sort.SliceStable(nodo.Hijos, func(i, j int) bool {
				return compararTextoUI(nodo.Hijos[i].Nombre, nodo.Hijos[j].Nombre)
			})
			nodo.Cargado = true
		})
	}()
}

func (a *Aplicacion) seleccionarCarpeta(ruta string) {
	if ruta == "" {
		return
	}
	if err := a.sincronizarArbolConRuta(ruta); err != nil {
		a.establecerEstado("No se pudo sincronizar el árbol con la carpeta seleccionada", err)
	}
	a.carpetaSeleccionada = ruta
	a.origenListado = origenListadoCarpeta
	a.claveListadoActual = ruta
	a.reiniciarListado()
}

func (a *Aplicacion) seleccionarCarpetaYandex(ruta string) {
	ruta = normalizarRutaYandexUI(ruta)
	if ruta == "" {
		return
	}
	a.asegurarArbolYandex()
	if err := a.sincronizarArbolYandexConRuta(ruta); err != nil {
		a.establecerEstado("No se pudo sincronizar el árbol remoto de Yandex.Disk", err)
	}
	a.carpetaYandexSeleccionada = ruta
	a.origenListado = origenListadoCarpetaYandex
	a.claveListadoActual = ruta
	a.reiniciarListado()
}

func (a *Aplicacion) seleccionarEtiqueta(etiqueta string) {
	etiqueta = strings.TrimSpace(etiqueta)
	if etiqueta == "" {
		return
	}
	a.origenListado = origenListadoEtiqueta
	a.claveListadoActual = etiqueta
	a.reiniciarListado()
}

func (a *Aplicacion) seleccionarUbicacion(ubicacion string) {
	ubicacion = strings.TrimSpace(ubicacion)
	if ubicacion == "" {
		return
	}
	a.origenListado = origenListadoUbicacion
	a.claveListadoActual = ubicacion
	a.reiniciarListado()
}

func (a *Aplicacion) seleccionarUbicacionSinNombre() {
	a.origenListado = origenListadoUbicacionSinNombre
	a.claveListadoActual = ""
	a.reiniciarListado()
}

func (a *Aplicacion) seleccionarNodoArbol(nodo *nodoArbolUI) {
	if nodo == nil {
		return
	}
	if !nodo.Cargado {
		a.asegurarHijosNodo(nodo)
	}
	nodo.Expandido = true
	if nodo.Origen == modelo.OrigenYandex {
		a.seleccionarCarpetaYandex(nodo.Ruta)
		return
	}
	a.seleccionarCarpeta(nodo.Ruta)
}

func (a *Aplicacion) cambiarVistaCentral(esGaleria bool) {
	if a.filtros.VistaGaleria == esGaleria {
		return
	}
	a.filtros.VistaGaleria = esGaleria
	a.listaCentro.Position = layout.Position{}
	if esGaleria {
		a.establecerEstado("Vista de galería activada", nil)
	} else {
		a.establecerEstado("Vista de lista activada", nil)
	}
	if a.ventana != nil {
		a.ventana.Invalidate()
	}
}

func (a *Aplicacion) cambiarOrdenListado(criterio modelo.CriterioOrdenListado, descendente bool) {
	criterio = criterio.Normalizado()
	if a.filtros.CriterioOrdenNormalizado() == criterio && a.filtros.OrdenDescendente == descendente {
		return
	}
	a.filtros.CriterioOrden = criterio
	a.filtros.OrdenDescendente = descendente
	a.reiniciarListado()
	a.establecerEstado(descripcionOrdenListado(criterio, descendente), nil)
}

func descripcionOrdenListado(criterio modelo.CriterioOrdenListado, descendente bool) string {
	criterio = criterio.Normalizado()
	if criterio == modelo.CriterioOrdenFechaModificacion {
		if descendente {
			return "Orden por fecha de modificación: más nuevos primero"
		}
		return "Orden por fecha de modificación: más antiguos primero"
	}
	if descendente {
		return "Orden alfabético descendente activado"
	}
	return "Orden alfabético ascendente activado"
}

func (a *Aplicacion) establecerOrdenConfiguracion(criterio modelo.CriterioOrdenListado, descendente bool) {
	a.configOrdenPorFecha.Value = criterio.Normalizado() == modelo.CriterioOrdenFechaModificacion
	a.configOrdenDescendente.Value = descendente
}

func (a *Aplicacion) criterioOrdenConfiguracion() modelo.CriterioOrdenListado {
	if a.configOrdenPorFecha.Value {
		return modelo.CriterioOrdenFechaModificacion
	}
	return modelo.CriterioOrdenNombre
}

func (a *Aplicacion) reiniciarListado() {
	a.reiniciarListadoConPosicion(layout.Position{}, false)
}

func (a *Aplicacion) reiniciarListadoPreservandoPosicion() {
	a.reiniciarListadoConPosicion(a.listaCentro.Position, true)
}

func (a *Aplicacion) reiniciarListadoConPosicion(posicion layout.Position, preservarPosicion bool) {
	if a.sesionListado != nil {
		_ = a.sesionListado.Cerrar()
		a.sesionListado = nil
	}
	a.versionListado++
	a.offsetListado = 0
	a.elementos = nil
	a.seleccionLote = make(map[string]bool)
	a.anclaSeleccionLote = ""
	a.elementoWidgets = make(map[string]*widgetsElemento)
	a.hayMasElementos = true
	a.cargandoElementos = false
	a.objetivoListado = 0
	if preservarPosicion {
		a.listaCentro.Position = posicion
		a.objetivoListado = a.calcularObjetivoRestauracionListado(posicion)
	} else {
		a.listaCentro.Position = layout.Position{}
	}

	switch a.origenListado {
	case origenListadoEtiqueta:
		a.establecerEstado(fmt.Sprintf("Cargando elementos con la etiqueta %q", a.claveListadoActual), nil)
	case origenListadoUbicacion:
		a.establecerEstado(fmt.Sprintf("Cargando elementos en %q", a.claveListadoActual), nil)
	case origenListadoUbicacionSinNombre:
		a.establecerEstado("Cargando elementos con GPS y sin valor Location", nil)
	case origenListadoCarpetaYandex:
		a.establecerEstado("Cargando elementos remotos de Yandex.Disk", nil)
	default:
		sesion, err := a.listador.NuevaSesion(context.Background(), a.carpetaSeleccionada, a.filtros)
		if err != nil {
			a.elementos = nil
			a.hayMasElementos = false
			a.objetivoListado = 0
			a.establecerEstado("No se pudo abrir la carpeta seleccionada", err)
			return
		}
		a.sesionListado = sesion
		a.establecerEstado("Cargando elementos de la carpeta seleccionada", nil)
	}

	a.cargarMasElementos()
}

func (a *Aplicacion) calcularObjetivoRestauracionListado(posicion layout.Position) int {
	objetivo := len(a.elementos)
	pagina := a.configuracion.TamanoPaginaLocal
	if a.origenListado == origenListadoCarpetaYandex {
		pagina = a.configuracion.TamanoPaginaRemota
		if pagina < 20 {
			pagina = 40
		}
	} else if pagina < 32 {
		pagina = 64
	}
	minimoVisible := posicion.First + pagina
	if objetivo < minimoVisible {
		objetivo = minimoVisible
	}
	if objetivo < pagina {
		objetivo = pagina
	}
	return objetivo
}

func (a *Aplicacion) continuarRestauracionListadoSiHaceFalta() {
	if a.objetivoListado <= 0 {
		return
	}
	if len(a.elementos) >= a.objetivoListado || !a.hayMasElementos {
		a.objetivoListado = 0
		return
	}
	a.cargarMasElementos()
}

func (a *Aplicacion) cargarMasElementos() {
	if a.cargandoElementos || !a.hayMasElementos {
		return
	}
	a.cargandoElementos = true

	sesionActual := a.sesionListado
	versionActual := a.versionListado
	origenActual := a.origenListado
	claveActual := a.claveListadoActual
	offsetActual := a.offsetListado
	limite := a.configuracion.TamanoPaginaLocal
	if origenActual == origenListadoCarpetaYandex {
		limite = a.configuracion.TamanoPaginaRemota
		if limite < 20 {
			limite = 40
		}
	} else if limite < 32 {
		limite = 64
	}

	if origenActual == origenListadoCarpeta {
		if sesionActual == nil {
			a.cargandoElementos = false
			a.hayMasElementos = false
			a.objetivoListado = 0
			return
		}

		go func() {
			lote, fin, err := sesionActual.Siguiente(context.Background(), limite)
			a.encolarActualizacion(func() {
				if versionActual != a.versionListado || sesionActual != a.sesionListado {
					return
				}

				a.cargandoElementos = false
				if err != nil {
					a.hayMasElementos = false
					a.objetivoListado = 0
					a.establecerEstado("No se pudo continuar el listado", err)
					return
				}

				for _, elemento := range lote {
					a.elementos = append(a.elementos, elemento)
					if _, existe := a.elementoWidgets[elemento.Ruta]; !existe {
						a.elementoWidgets[elemento.Ruta] = &widgetsElemento{}
					}
				}
				a.hayMasElementos = !fin
				a.continuarRestauracionListadoSiHaceFalta()

				if len(a.elementos) == 0 && fin {
					a.objetivoListado = 0
					a.establecerEstado("La carpeta seleccionada no contiene elementos compatibles con los filtros activos", nil)
					return
				}
				a.establecerEstado(fmt.Sprintf("%d elementos visibles en memoria inmediata", len(a.elementos)), nil)
			})
		}()
		return
	}

	if origenActual == origenListadoCarpetaYandex {
		go func() {
			lote, siguienteOffset, fin, err := a.listarElementosYandex(context.Background(), claveActual, a.filtros, limite, offsetActual)
			a.encolarActualizacion(func() {
				if versionActual != a.versionListado || origenActual != a.origenListado || claveActual != a.claveListadoActual {
					return
				}

				a.cargandoElementos = false
				if err != nil {
					a.hayMasElementos = false
					a.objetivoListado = 0
					a.establecerEstado("No se pudo continuar el listado remoto de Yandex.Disk", err)
					return
				}

				for _, elemento := range lote {
					a.elementos = append(a.elementos, elemento)
					if _, existe := a.elementoWidgets[elemento.Ruta]; !existe {
						a.elementoWidgets[elemento.Ruta] = &widgetsElemento{}
					}
				}
				a.offsetListado = siguienteOffset
				a.hayMasElementos = !fin
				a.continuarRestauracionListadoSiHaceFalta()

				if len(a.elementos) == 0 && fin {
					a.objetivoListado = 0
					a.establecerEstado("La carpeta remota no contiene elementos compatibles con los filtros activos", nil)
					return
				}
				a.establecerEstado(fmt.Sprintf("%d elementos remotos visibles en memoria inmediata", len(a.elementos)), nil)
			})
		}()
		return
	}

	go func() {
		var (
			lote []modelo.Archivo
			err  error
		)
		switch origenActual {
		case origenListadoEtiqueta:
			lote, err = a.almacen.ListarArchivosPorEtiqueta(context.Background(), claveActual, a.filtros, limite, offsetActual)
		case origenListadoUbicacion:
			lote, err = a.almacen.ListarArchivosPorUbicacion(context.Background(), claveActual, a.filtros, limite, offsetActual)
		case origenListadoUbicacionSinNombre:
			lote, err = a.almacen.ListarArchivosSinUbicacionNombrada(context.Background(), a.filtros, limite, offsetActual)
		default:
			err = fmt.Errorf("origen de listado no soportado: %q", origenActual)
		}
		fin := len(lote) < limite
		a.encolarActualizacion(func() {
			if versionActual != a.versionListado || origenActual != a.origenListado || claveActual != a.claveListadoActual {
				return
			}

			a.cargandoElementos = false
			if err != nil {
				a.hayMasElementos = false
				a.objetivoListado = 0
				a.establecerEstado("No se pudo continuar el listado", err)
				return
			}

			for _, elemento := range lote {
				a.elementos = append(a.elementos, elemento)
				if _, existe := a.elementoWidgets[elemento.Ruta]; !existe {
					a.elementoWidgets[elemento.Ruta] = &widgetsElemento{}
				}
			}
			a.offsetListado = offsetActual + len(lote)
			a.hayMasElementos = !fin
			a.continuarRestauracionListadoSiHaceFalta()

			if len(a.elementos) == 0 && fin {
				a.objetivoListado = 0
				a.establecerEstado("No se encontraron elementos compatibles para la fuente seleccionada", nil)
				return
			}
			a.establecerEstado(fmt.Sprintf("%d elementos visibles en memoria inmediata", len(a.elementos)), nil)
		})
	}()
}

func (a *Aplicacion) listarDirectoriosYandex(ctx context.Context, ruta string) ([]yandex.ElementoRemoto, error) {
	if a.clienteYandex == nil || !a.clienteYandex.Configurado() {
		return nil, yandex.ErrNoImplementado
	}

	const tamanoLote = 200
	var (
		offset int
		todos  []yandex.ElementoRemoto
	)
	for {
		lote, err := a.clienteYandex.ListarDirectorios(ctx, ruta, tamanoLote, offset)
		if err != nil {
			return nil, err
		}
		todos = append(todos, lote...)
		if len(lote) < tamanoLote {
			break
		}
		offset += len(lote)
	}
	return todos, nil
}

func (a *Aplicacion) listarElementosYandex(ctx context.Context, ruta string, filtros modelo.FiltrosListado, limite, desplazamiento int) ([]modelo.Archivo, int, bool, error) {
	if a.clienteYandex == nil || !a.clienteYandex.Configurado() {
		return nil, desplazamiento, true, yandex.ErrNoImplementado
	}

	if limite < 1 {
		limite = maximo(20, a.configuracion.TamanoPaginaRemota)
	}
	tamanoPeticion := a.configuracion.TamanoPaginaRemota
	if tamanoPeticion < 20 {
		tamanoPeticion = 40
	}
	if tamanoPeticion < limite {
		tamanoPeticion = limite
	}
	if tamanoPeticion > 200 {
		tamanoPeticion = 200
	}

	offsetActual := desplazamiento
	resultados := make([]modelo.Archivo, 0, limite)
	for len(resultados) < limite {
		loteRemoto, err := a.clienteYandex.ListarElementos(ctx, ruta, tamanoPeticion, offsetActual)
		if err != nil {
			return nil, offsetActual, true, err
		}
		if len(loteRemoto) == 0 {
			a.persistirArchivosRemotosDescubiertos(ctx, resultados)
			return resultados, offsetActual, true, nil
		}
		offsetActual += len(loteRemoto)

		for _, elemento := range loteRemoto {
			archivo := convertirElementoYandexAArchivo(elemento)
			if !filtros.Acepta(archivo) {
				continue
			}
			resultados = append(resultados, archivo)
			if len(resultados) >= limite {
				break
			}
		}

		if len(loteRemoto) < tamanoPeticion {
			a.persistirArchivosRemotosDescubiertos(ctx, resultados)
			return resultados, offsetActual, true, nil
		}
	}

	a.persistirArchivosRemotosDescubiertos(ctx, resultados)
	return resultados, offsetActual, false, nil
}

func (a *Aplicacion) persistirArchivosRemotosDescubiertos(ctx context.Context, archivos []modelo.Archivo) {
	if len(archivos) == 0 {
		return
	}
	for _, archivo := range archivos {
		if !archivoEsRemotoYandex(archivo) || archivo.EsDirectorio || strings.TrimSpace(archivo.Ruta) == "" {
			continue
		}
		if a.servicioDuplicados != nil {
			_ = a.servicioDuplicados.GuardarArchivoRemotoDescubierto(ctx, archivo)
			continue
		}
		if a.almacen != nil {
			_ = a.almacen.GuardarArchivo(ctx, archivo)
		}
	}
}

func (a *Aplicacion) recargarColeccionesLaterales() {
	a.recargarColeccionesLateralesConExtras(nil, nil)
}

func (a *Aplicacion) recargarColeccionesLateralesConExtras(etiquetasExtra, ubicacionesExtra []string) {
	a.invalidarBusquedasLaterales()
	go func() {
		const limiteColeccionesLaterales = 1_000

		etiquetas, errEtiquetas := a.almacen.ListarEtiquetas(context.Background(), limiteColeccionesLaterales)
		ubicaciones, errUbicaciones := a.almacen.ListarUbicaciones(context.Background(), limiteColeccionesLaterales)
		ubicacionesGuardadas, errUbicacionesGuardadas := a.almacen.ListarUbicacionesGuardadas(context.Background(), 1_000)
		tieneSinNombre, errSinNombre := a.almacen.TieneArchivosConUbicacionSinNombre(context.Background())
		a.encolarActualizacion(func() {
			if errEtiquetas == nil {
				a.etiquetas = fusionarOpcionesLaterales(etiquetas, etiquetasExtra)
			}
			if errUbicaciones == nil {
				a.ubicacionesNombradas = fusionarOpcionesLaterales(ubicaciones, ubicacionesExtra)
			}
			if errUbicacionesGuardadas == nil {
				a.actualizarUbicacionesGuardadasEnMemoria(ubicacionesGuardadas)
			}
			if errSinNombre == nil && tieneSinNombre {
				a.ubicacionesNombradas = append(a.ubicacionesNombradas, opcionFiltroLateral{
					Clave:    etiquetaUbicacionSinNombre,
					Etiqueta: etiquetaUbicacionSinNombre,
				})
			}
			if errEtiquetas != nil {
				a.establecerEstado("No se pudieron cargar las etiquetas", errEtiquetas)
			}
			if errUbicaciones != nil {
				a.establecerEstado("No se pudieron cargar las ubicaciones", errUbicaciones)
			}
			if errUbicacionesGuardadas != nil {
				a.establecerEstado("No se pudieron cargar las ubicaciones guardadas", errUbicacionesGuardadas)
			}
			if errSinNombre != nil {
				a.establecerEstado("No se pudo verificar si existen ubicaciones sin nombre", errSinNombre)
			}
		})
	}()
}

func (a *Aplicacion) invalidarBusquedasLaterales() {
	a.busquedaEtiquetas = estadoBusquedaLateral{}
	a.busquedaUbicaciones = estadoBusquedaLateral{}
}

func (a *Aplicacion) resolverOpcionesLaterales(editorFiltro *widget.Editor, elementos []opcionFiltroLateral, origen tipoOrigenListado) ([]opcionFiltroLateral, bool) {
	if editorFiltro == nil {
		return elementos, false
	}

	consulta := strings.TrimSpace(editorFiltro.Text())
	if consulta == "" {
		a.limpiarBusquedaLateral(origen)
		return elementos, false
	}

	filtradosLocales := filtrarOpcionesLaterales(elementos, consulta)
	a.solicitarBusquedaLateral(origen, consulta)

	estado := a.estadoBusquedaLateral(origen)
	if estado == nil {
		return filtradosLocales, false
	}

	if mismaConsultaBusquedaLateral(estado.Consulta, consulta) {
		opciones := append([]opcionFiltroLateral(nil), estado.Opciones...)
		if origen == origenListadoUbicacion {
			opciones = anexarUbicacionSinNombreCoincidente(opciones, elementos, consulta)
		}
		if len(opciones) > 0 || !estado.Cargando {
			return opciones, estado.Cargando
		}
	}

	return filtradosLocales, true
}

func (a *Aplicacion) solicitarBusquedaLateral(origen tipoOrigenListado, consulta string) {
	estado := a.estadoBusquedaLateral(origen)
	if estado == nil || a.almacen == nil {
		return
	}

	consulta = strings.TrimSpace(consulta)
	if consulta == "" || mismaConsultaBusquedaLateral(estado.Consulta, consulta) {
		return
	}

	estado.Consulta = consulta
	estado.Opciones = nil
	estado.Cargando = true
	estado.Version++
	versionActual := estado.Version

	go func(consultaBusqueda string, versionBusqueda int) {
		const limiteResultadosBusquedaLateral = 200

		var (
			valores []string
			err     error
		)
		switch origen {
		case origenListadoEtiqueta:
			valores, err = a.almacen.BuscarEtiquetas(context.Background(), consultaBusqueda, limiteResultadosBusquedaLateral)
		case origenListadoUbicacion:
			valores, err = a.almacen.BuscarUbicaciones(context.Background(), consultaBusqueda, limiteResultadosBusquedaLateral)
		default:
			return
		}

		opciones := convertirOpcionesLaterales(valores)
		a.encolarActualizacion(func() {
			estadoActual := a.estadoBusquedaLateral(origen)
			if estadoActual == nil || estadoActual.Version != versionBusqueda || !mismaConsultaBusquedaLateral(estadoActual.Consulta, consultaBusqueda) {
				return
			}
			estadoActual.Cargando = false
			if err != nil {
				switch origen {
				case origenListadoEtiqueta:
					a.establecerEstado("No se pudieron buscar las etiquetas", err)
				case origenListadoUbicacion:
					a.establecerEstado("No se pudieron buscar las ubicaciones", err)
				}
				return
			}
			estadoActual.Opciones = opciones
		})
	}(consulta, versionActual)
}

func (a *Aplicacion) limpiarBusquedaLateral(origen tipoOrigenListado) {
	estado := a.estadoBusquedaLateral(origen)
	if estado == nil {
		return
	}
	*estado = estadoBusquedaLateral{}
}

func (a *Aplicacion) estadoBusquedaLateral(origen tipoOrigenListado) *estadoBusquedaLateral {
	switch origen {
	case origenListadoEtiqueta:
		return &a.busquedaEtiquetas
	case origenListadoUbicacion, origenListadoUbicacionSinNombre:
		return &a.busquedaUbicaciones
	default:
		return nil
	}
}

func mismaConsultaBusquedaLateral(actual, esperada string) bool {
	return strings.EqualFold(strings.TrimSpace(actual), strings.TrimSpace(esperada))
}

func anexarUbicacionSinNombreCoincidente(opciones, base []opcionFiltroLateral, consulta string) []opcionFiltroLateral {
	if !coincideTextoBusqueda(etiquetaUbicacionSinNombre, consulta) || contieneOpcionLateral(opciones, etiquetaUbicacionSinNombre) {
		return opciones
	}
	for _, opcion := range base {
		if opcion.Clave == etiquetaUbicacionSinNombre {
			return append(opciones, opcion)
		}
	}
	return opciones
}

func contieneOpcionLateral(opciones []opcionFiltroLateral, clave string) bool {
	for _, opcion := range opciones {
		if strings.EqualFold(strings.TrimSpace(opcion.Clave), strings.TrimSpace(clave)) {
			return true
		}
	}
	return false
}

func (a *Aplicacion) recargarDuplicados() {
	if a.servicioDuplicados == nil {
		return
	}
	if !a.duplicadosInicializados && a.vistaActual != vistaDuplicados {
		return
	}

	if a.cancelarCargaDuplicados != nil {
		a.cancelarCargaDuplicados()
	}

	a.duplicadosInicializados = true
	a.cargandoDuplicados = true
	a.versionCargaDuplicados++
	versionActual := a.versionCargaDuplicados
	a.establecerEstado("Actualizando grupos de duplicados", nil)

	tipo := a.tipoCoincidenciaActual
	categoria := a.categoriaDuplicados
	orden := a.ordenDuplicados
	ctx, cancelar := context.WithCancel(context.Background())
	a.cancelarCargaDuplicados = cancelar

	go func() {
		grupos, err := a.servicioDuplicados.ListarGrupos(ctx, tipo, categoria, orden, 500, 0)
		a.encolarActualizacion(func() {
			if versionActual != a.versionCargaDuplicados {
				return
			}

			a.cancelarCargaDuplicados = nil
			a.cargandoDuplicados = false
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				a.establecerEstado("No se pudieron cargar los grupos de duplicados", err)
				return
			}
			a.gruposDuplicados = grupos
			a.grupoWidgets = make(map[string]*widgetsGrupoDuplicado)
			a.sincronizarPreviewDuplicadosConGrupos(grupos)
			a.establecerEstado(fmt.Sprintf("%d grupos de duplicados cargados", len(grupos)), nil)
		})
	}()
}

func (a *Aplicacion) limpiarRegistrosLocalesAusentesDuplicados() {
	if a.servicioDuplicados == nil {
		a.establecerEstado("No hay un servicio de duplicados disponible para depurar registros locales", nil)
		return
	}
	if a.limpiandoDuplicados {
		return
	}

	a.limpiandoDuplicados = true
	a.establecerEstado("Verificando rutas locales ausentes en el catálogo", nil)

	go func() {
		eliminados, err := a.servicioDuplicados.LimpiarRegistrosLocalesAusentes(context.Background())
		a.encolarActualizacion(func() {
			a.limpiandoDuplicados = false
			if err != nil {
				a.establecerEstado("No se pudieron depurar las rutas locales ausentes", err)
				return
			}

			if eliminados == 0 {
				a.establecerEstado("No se encontraron rutas locales ausentes para depurar", nil)
			} else {
				a.establecerEstado(fmt.Sprintf("Se depuraron %d rutas locales ausentes del catálogo", eliminados), nil)
			}
			a.recargarDuplicados()
		})
	}()
}

func (a *Aplicacion) gruposDuplicadosVisibles() []modelo.GrupoDuplicados {
	if !a.soloDuplicadosMultimedia.Value {
		return a.gruposDuplicados
	}

	filtrados := make([]modelo.GrupoDuplicados, 0, len(a.gruposDuplicados))
	for _, grupo := range a.gruposDuplicados {
		if grupoDuplicadoTieneMultimedia(grupo) {
			filtrados = append(filtrados, grupo)
		}
	}
	return filtrados
}

func grupoDuplicadoTieneMultimedia(grupo modelo.GrupoDuplicados) bool {
	for _, elemento := range grupo.Elementos {
		if elemento.EsMultimedia() {
			return true
		}
	}
	return false
}

func grupoDuplicadoContieneRuta(grupo modelo.GrupoDuplicados, ruta string) bool {
	if strings.TrimSpace(ruta) == "" {
		return false
	}
	for _, elemento := range grupo.Elementos {
		if elemento.Ruta == ruta {
			return true
		}
	}
	return false
}

func (a *Aplicacion) sincronizarPreviewDuplicadosConGrupos(grupos []modelo.GrupoDuplicados) {
	if strings.TrimSpace(a.rutaPreviewDuplicados) == "" {
		return
	}
	for _, grupo := range grupos {
		if grupoDuplicadoContieneRuta(grupo, a.rutaPreviewDuplicados) {
			return
		}
	}
	a.rutaPreviewDuplicados = ""
}

func (a *Aplicacion) seleccionarPreviewDuplicados(archivo modelo.Archivo) {
	if !archivo.EsMultimedia() {
		return
	}
	a.rutaPreviewDuplicados = archivo.Ruta
	a.activarArchivo(archivo)
	if a.ventana != nil {
		a.ventana.Invalidate()
	}
}

func (a *Aplicacion) archivoPreviewDuplicados(grupo modelo.GrupoDuplicados) (modelo.Archivo, bool) {
	if strings.TrimSpace(a.rutaPreviewDuplicados) == "" {
		return modelo.Archivo{}, false
	}
	for _, elemento := range grupo.Elementos {
		if elemento.Ruta != a.rutaPreviewDuplicados {
			continue
		}
		if a.tieneArchivoActivo && a.archivoActivo.Ruta == elemento.Ruta {
			return a.archivoActivo, true
		}
		return elemento, true
	}
	return modelo.Archivo{}, false
}

func claveGrupoDuplicado(grupo modelo.GrupoDuplicados) string {
	return string(grupo.Tipo) + "|" + grupo.Clave
}

func (a *Aplicacion) grupoDuplicadoContraido(grupo modelo.GrupoDuplicados) bool {
	if a.gruposDuplicadosContraidos == nil {
		return false
	}
	return a.gruposDuplicadosContraidos[claveGrupoDuplicado(grupo)]
}

func (a *Aplicacion) alternarColapsoGrupoDuplicado(grupo modelo.GrupoDuplicados) {
	if a.gruposDuplicadosContraidos == nil {
		a.gruposDuplicadosContraidos = make(map[string]bool)
	}
	clave := claveGrupoDuplicado(grupo)
	a.gruposDuplicadosContraidos[clave] = !a.gruposDuplicadosContraidos[clave]
}

func (a *Aplicacion) iniciarEscaneoMetadatos() {
	if a.cancelarEscaneoMetadatos != nil {
		a.cancelarEscaneoMetadatos()
	}
	rutaEscaneo, err := a.resolverRutaEscaneo(a.editorRutaEscaneoMetadatos.Text())
	if err != nil {
		a.establecerEstado("No se pudo iniciar el escaneo de metadatos", err)
		return
	}
	a.editorRutaEscaneoMetadatos.SetText(rutaEscaneo)

	ctx, cancelar := context.WithCancel(context.Background())
	a.cancelarEscaneoMetadatos = cancelar
	a.versionEscaneoMetadatos++
	versionActual := a.versionEscaneoMetadatos
	a.progresoMetadatos = indexador.EventoProgreso{}
	a.escanandoMetadatos = true
	a.establecerEstado("Iniciando escaneo de metadatos locales", nil)

	flujo := a.indexador.Descubrir(ctx, rutaEscaneo, indexador.OpcionesDescubrimiento{
		CalcularMetadatos:    true,
		ConcurrenciaAnalisis: a.configuracion.ConcurrenciaMetadatos,
		SoloMultimedia:       true,
		RutasExcluidas:       a.rutasExcluidasEscaneo(rutaEscaneo),
	})

	go func() {
		for evento := range flujo {
			eventoActual := evento
			a.encolarActualizacion(func() {
				if versionActual != a.versionEscaneoMetadatos {
					return
				}

				a.progresoMetadatos = eventoActual
				if eventoActual.Error != nil {
					a.establecerEstado("Escaneo de metadatos con incidencias", eventoActual.Error)
				}
				if eventoActual.Finalizado {
					a.escanandoMetadatos = false
					a.cancelarEscaneoMetadatos = nil
					if eventoActual.Error != nil && errors.Is(eventoActual.Error, context.Canceled) {
						a.establecerEstado("Escaneo de metadatos cancelado", nil)
						return
					}
					a.establecerEstado(fmt.Sprintf("Escaneo de metadatos finalizado. %d archivos analizados", eventoActual.ArchivosAnalizados), nil)
					a.recargarColeccionesLaterales()
					a.reiniciarListado()
					a.refrescarArchivoActivoDesdeCatalogo()
				}
			})
		}
	}()
}

func (a *Aplicacion) pausarEscaneoMetadatos() {
	if a.cancelarEscaneoMetadatos == nil {
		return
	}
	a.cancelarEscaneoMetadatos()
	a.establecerEstado("Solicitando pausa del escaneo de metadatos", nil)
}

func (a *Aplicacion) escaneoRemotoActivo() bool {
	return a.cancelarDescubrimiento != nil && a.tipoDescubrimiento == descubrimientoDuplicadosRemoto
}

func (a *Aplicacion) etiquetaBotonEscaneoRemoto() string {
	switch {
	case a.escaneoRemotoActivo():
		return "Pausar escaneo remoto"
	case a.escaneoRemotoPendiente:
		return "Reanudar escaneo remoto"
	default:
		return "Escanear remotos"
	}
}

func (a *Aplicacion) refrescarEstadoEscaneoRemotoPendiente() {
	a.escaneoRemotoPendiente = false
	if a.servicioDuplicados == nil {
		return
	}
	estadoRemoto, err := a.servicioDuplicados.CargarEstadoDescubrimientoRemoto()
	if err == nil && estadoRemoto.Pendiente {
		a.escaneoRemotoPendiente = true
	}
}

func (a *Aplicacion) pausarDescubrimientoRemoto() {
	if !a.escaneoRemotoActivo() || a.cancelarDescubrimiento == nil {
		return
	}
	a.cancelarDescubrimiento()
	a.establecerEstado("Solicitando pausa del escaneo remoto", nil)
}

func (a *Aplicacion) resolverRutaEscaneoRemoto(valor string) string {
	valor = strings.TrimSpace(valor)
	if valor == "" {
		if a.escaneoRemotoPendiente && a.servicioDuplicados != nil {
			if estadoRemoto, err := a.servicioDuplicados.CargarEstadoDescubrimientoRemoto(); err == nil && strings.TrimSpace(estadoRemoto.RutaRaiz) != "" {
				return estadoRemoto.RutaRaiz
			}
		}
		if strings.TrimSpace(a.carpetaYandexSeleccionada) != "" {
			return normalizarRutaYandexUI(a.carpetaYandexSeleccionada)
		}
		return "disk:/"
	}

	if strings.HasPrefix(strings.ToLower(valor), "disk:") {
		return normalizarRutaYandexUI(valor)
	}
	if info, err := os.Stat(valor); err == nil && info.IsDir() {
		if strings.TrimSpace(a.carpetaYandexSeleccionada) != "" {
			return normalizarRutaYandexUI(a.carpetaYandexSeleccionada)
		}
		return "disk:/"
	}
	return normalizarRutaYandexUI(valor)
}

func (a *Aplicacion) iniciarDescubrimientoLocal() {
	if a.cancelarDescubrimiento != nil {
		a.cancelarDescubrimiento()
	}
	rutaEscaneo, err := a.resolverRutaEscaneo(a.editorRutaEscaneoDuplicados.Text())
	if err != nil {
		a.establecerEstado("No se pudo iniciar el descubrimiento local", err)
		return
	}
	a.editorRutaEscaneoDuplicados.SetText(rutaEscaneo)

	ctx, cancelar := context.WithCancel(context.Background())
	a.cancelarDescubrimiento = cancelar
	a.versionDescubrimiento++
	versionActual := a.versionDescubrimiento
	a.tipoDescubrimiento = descubrimientoDuplicadosLocal
	a.refrescarEstadoEscaneoRemotoPendiente()
	a.progresoDuplicados = indexador.EventoProgreso{}
	a.establecerEstado("Iniciando descubrimiento local para hashes y duplicados", nil)

	flujo := a.servicioDuplicados.IniciarDescubrimientoLocal(ctx, rutaEscaneo, a.rutasExcluidasEscaneo(rutaEscaneo))
	go func() {
		for evento := range flujo {
			eventoActual := evento
			a.encolarActualizacion(func() {
				if versionActual != a.versionDescubrimiento || a.tipoDescubrimiento != descubrimientoDuplicadosLocal {
					return
				}
				if eventoActual.Error != nil {
					a.establecerEstado("Descubrimiento local con incidencias", eventoActual.Error)
				}
				a.progresoDuplicados = eventoActual
				if eventoActual.Finalizado {
					a.cancelarDescubrimiento = nil
					a.tipoDescubrimiento = descubrimientoDuplicadosNinguno
					a.refrescarEstadoEscaneoRemotoPendiente()
					if eventoActual.Error != nil && errors.Is(eventoActual.Error, context.Canceled) {
						a.establecerEstado("Descubrimiento local cancelado", nil)
						return
					}
					a.establecerEstado(fmt.Sprintf("Descubrimiento finalizado. %d archivos analizados", eventoActual.ArchivosAnalizados), nil)
					a.recargarColeccionesLaterales()
					a.recargarDuplicados()
				}
			})
		}
	}()
}

func (a *Aplicacion) iniciarDescubrimientoRemoto() {
	if a.servicioDuplicados == nil {
		a.establecerEstado("No hay un servicio de duplicados disponible para el escaneo remoto", nil)
		return
	}
	if a.clienteYandex == nil || !a.clienteYandex.Configurado() {
		a.establecerEstado("Configura una clave API de Yandex.Disk para escanear remotos", yandex.ErrNoImplementado)
		return
	}
	if a.escaneoRemotoActivo() {
		a.pausarDescubrimientoRemoto()
		return
	}
	if a.cancelarDescubrimiento != nil {
		a.cancelarDescubrimiento()
	}

	rutaEscaneo := a.resolverRutaEscaneoRemoto(a.editorRutaEscaneoDuplicados.Text())
	a.editorRutaEscaneoDuplicados.SetText(rutaEscaneo)

	ctx, cancelar := context.WithCancel(context.Background())
	a.cancelarDescubrimiento = cancelar
	a.versionDescubrimiento++
	versionActual := a.versionDescubrimiento
	a.tipoDescubrimiento = descubrimientoDuplicadosRemoto

	if estadoRemoto, err := a.servicioDuplicados.CargarEstadoDescubrimientoRemoto(); err == nil &&
		estadoRemoto.Pendiente &&
		strings.EqualFold(strings.TrimSpace(estadoRemoto.RutaRaiz), strings.TrimSpace(rutaEscaneo)) {
		a.progresoDuplicados = indexador.EventoProgreso{
			RutaActual:            estadoRemoto.RutaActual,
			DirectoriosProcesados: estadoRemoto.DirectoriosProcesados,
			ArchivosEncontrados:   estadoRemoto.ArchivosEncontrados,
			ArchivosAnalizados:    estadoRemoto.ArchivosAnalizados,
		}
		a.escaneoRemotoPendiente = true
		a.establecerEstado("Reanudando descubrimiento remoto para hashes y duplicados", nil)
	} else {
		a.progresoDuplicados = indexador.EventoProgreso{}
		a.escaneoRemotoPendiente = false
		a.establecerEstado("Iniciando descubrimiento remoto para hashes y duplicados", nil)
	}

	flujo := a.servicioDuplicados.IniciarDescubrimientoRemoto(ctx, rutaEscaneo, a.clienteYandex)
	go func() {
		for evento := range flujo {
			eventoActual := evento
			a.encolarActualizacion(func() {
				if versionActual != a.versionDescubrimiento || a.tipoDescubrimiento != descubrimientoDuplicadosRemoto {
					return
				}
				a.progresoDuplicados = eventoActual
				if eventoActual.Error != nil && !errors.Is(eventoActual.Error, context.Canceled) {
					a.establecerEstado("Descubrimiento remoto con incidencias", eventoActual.Error)
				}
				if !eventoActual.Finalizado {
					return
				}

				a.cancelarDescubrimiento = nil
				a.tipoDescubrimiento = descubrimientoDuplicadosNinguno
				if eventoActual.Error != nil && errors.Is(eventoActual.Error, context.Canceled) {
					a.escaneoRemotoPendiente = true
					a.establecerEstado("Escaneo remoto en pausa. Puedes reanudarlo más tarde.", nil)
					return
				}

				a.escaneoRemotoPendiente = false
				if eventoActual.Error != nil {
					a.establecerEstado("Descubrimiento remoto finalizado con incidencias", eventoActual.Error)
				} else {
					a.establecerEstado(fmt.Sprintf("Descubrimiento remoto finalizado. %d archivos analizados", eventoActual.ArchivosAnalizados), nil)
				}
				a.recargarDuplicados()
			})
		}
	}()
}

func (a *Aplicacion) refrescarArchivoActivoDesdeCatalogo() {
	if !a.tieneArchivoActivo || a.archivoActivo.Ruta == "" || !archivoEsLocal(a.archivoActivo) {
		return
	}

	archivo, err := a.almacen.ObtenerArchivoPorRuta(context.Background(), a.archivoActivo.Ruta)
	if err != nil {
		return
	}

	a.archivoActivo = archivo
	a.sincronizarEditoresMetadatos(archivo)
	a.reemplazarArchivoEnMemoria(archivo)
	a.sincronizarEdicionRegiones(archivo)
	a.sincronizarEdicionRecorte(archivo)
	a.solicitarSalidaExiftool(archivo, true)
}

func (a *Aplicacion) activarArchivo(archivo modelo.Archivo) {
	a.archivoActivo = archivo
	a.tieneArchivoActivo = true
	if archivoEsLocal(archivo) {
		a.sincronizarEdicionRegiones(archivo)
		a.sincronizarEdicionRecorte(archivo)
		a.sincronizarEditoresMetadatos(archivo)
		a.solicitarSalidaExiftool(archivo, false)
		a.establecerRutaDestinoActivoLocal(filepath.Dir(archivo.Ruta))
		a.sincronizarReproductorVideo(archivo)
		a.solicitarEnriquecimientoExplorador(archivo)

		if archivo.Tipo == modelo.TipoImagen || archivo.Tipo == modelo.TipoVideo {
			a.solicitarPreview(archivo, 2_048)
		}
		return
	}
	a.descartarEdicionRegiones()
	a.descartarEdicionRecorte()
	a.limpiarReproductorVideo()
	a.establecerRutaDestinoActivoLocal(a.rutaUsuario)
	a.establecerRutaDestinoActivoRemoto(rutaPadreYandex(archivo.Ruta))
	if archivo.Tipo == modelo.TipoImagen || archivo.Tipo == modelo.TipoVideo {
		a.solicitarPreview(archivo, 2_048)
	}
}

func (a *Aplicacion) archivoNecesitaEnriquecimiento(archivo modelo.Archivo) bool {
	if !archivoEsLocal(archivo) || !archivo.EsMultimedia() || archivo.EsDirectorio {
		return false
	}
	return archivo.Metadatos.MetadatosVacios() ||
		archivo.Ancho == 0 ||
		archivo.Alto == 0 ||
		(archivo.Tipo == modelo.TipoVideo && archivo.Duracion == 0)
}

func (a *Aplicacion) revisionArchivoSistema(archivo modelo.Archivo) int64 {
	if archivo.Modificado.IsZero() {
		return -1
	}
	return archivo.Modificado.UTC().UnixNano()
}

func (a *Aplicacion) archivoDebeVerificarseConSistema(archivo modelo.Archivo) bool {
	if !archivoEsLocal(archivo) || !archivo.EsMultimedia() || archivo.EsDirectorio || strings.TrimSpace(archivo.Ruta) == "" {
		return false
	}
	if a.metadatosPendientes[archivo.Ruta] {
		return false
	}

	revisionActual := a.revisionArchivoSistema(archivo)
	revisionVerificada, existe := a.metadatosVerificados[archivo.Ruta]
	return !existe || revisionVerificada != revisionActual
}

func (a *Aplicacion) marcarArchivoVerificadoConSistema(archivo modelo.Archivo) {
	if a.metadatosVerificados == nil {
		a.metadatosVerificados = make(map[string]int64)
	}
	a.metadatosVerificados[archivo.Ruta] = a.revisionArchivoSistema(archivo)
}

func (a *Aplicacion) esElementoActivo(archivo modelo.Archivo) bool {
	return a.tieneArchivoActivo && a.archivoActivo.Ruta == archivo.Ruta
}

func (a *Aplicacion) manejarActivacionElemento(archivo modelo.Archivo, abrirVista bool) {
	a.activarArchivo(archivo)
	if archivo.EsDirectorio {
		if archivoEsRemotoYandex(archivo) {
			a.seleccionarCarpetaYandex(archivo.Ruta)
			return
		}
		a.seleccionarCarpeta(archivo.Ruta)
		return
	}
	if abrirVista {
		a.cambiarVista(vistaElementoUnico)
	}
	if a.ventana != nil {
		a.ventana.Invalidate()
	}
}

func (a *Aplicacion) indiceArchivoActivoEnListado() int {
	if !a.tieneArchivoActivo || a.archivoActivo.Ruta == "" {
		return -1
	}
	for indice := range a.elementos {
		if a.elementos[indice].Ruta == a.archivoActivo.Ruta {
			return indice
		}
	}
	return -1
}

func (a *Aplicacion) puedeNavegarVisor(desplazamiento int) bool {
	indice := a.indiceArchivoActivoEnListado()
	if indice < 0 || desplazamiento == 0 {
		return false
	}
	destino := indice + desplazamiento
	return destino >= 0 && destino < len(a.elementos)
}

func (a *Aplicacion) navegarVisor(desplazamiento int) {
	if !a.puedeNavegarVisor(desplazamiento) {
		return
	}

	destino := a.indiceArchivoActivoEnListado() + desplazamiento
	a.activarArchivo(a.elementos[destino])
	a.cambiarVista(vistaElementoUnico)
	if a.ventana != nil {
		a.ventana.Invalidate()
	}
}

func (a *Aplicacion) enriquecerArchivoActiva(archivo modelo.Archivo) {
	if !archivoEsLocal(archivo) {
		return
	}
	go func() {
		enriquecido, err := a.servicioMetadatos.AnalizarArchivo(context.Background(), archivo)
		a.encolarActualizacion(func() {
			if err != nil {
				a.establecerEstado("No se pudieron cargar todos los metadatos del archivo activo", err)
				return
			}
			if err := a.almacen.GuardarArchivo(context.Background(), enriquecido); err != nil {
				a.establecerEstado("No se pudo persistir el enriquecimiento del archivo activo", err)
				return
			}
			a.reemplazarArchivoEnMemoria(enriquecido)
			if a.tieneArchivoActivo && a.archivoActivo.Ruta == enriquecido.Ruta {
				a.archivoActivo = enriquecido
				a.sincronizarEdicionRegiones(enriquecido)
				a.sincronizarEditoresMetadatos(enriquecido)
			}
			a.recargarColeccionesLaterales()
		})
	}()
}

func (a *Aplicacion) solicitarEnriquecimientoExplorador(archivo modelo.Archivo) {
	if !archivoEsLocal(archivo) || archivo.Ruta == "" || !a.archivoDebeVerificarseConSistema(archivo) {
		return
	}
	if a.metadatosPendientes[archivo.Ruta] {
		return
	}
	a.metadatosPendientes[archivo.Ruta] = true

	go func() {
		enriquecido, err := a.servicioMetadatos.AnalizarArchivo(context.Background(), archivo)
		a.encolarActualizacion(func() {
			delete(a.metadatosPendientes, archivo.Ruta)
			if err != nil {
				a.establecerEstado("No se pudieron precargar todos los metadatos del explorador", err)
				return
			}
			if err := a.almacen.GuardarArchivo(context.Background(), enriquecido); err != nil {
				a.establecerEstado("No se pudo persistir el enriquecimiento precargado del explorador", err)
				return
			}
			a.marcarArchivoVerificadoConSistema(enriquecido)
			a.reemplazarArchivoEnMemoria(enriquecido)
			if a.tieneArchivoActivo && a.archivoActivo.Ruta == enriquecido.Ruta {
				a.archivoActivo = enriquecido
				a.sincronizarEdicionRegiones(enriquecido)
				a.sincronizarEdicionRecorte(enriquecido)
				a.sincronizarEditoresMetadatos(enriquecido)
			}
			if len(enriquecido.Metadatos.PalabrasClave) > 0 || len(enriquecido.Metadatos.Sujetos) > 0 || enriquecido.Metadatos.Ubicacion != "" || enriquecido.Metadatos.Coordenadas != nil {
				a.recargarColeccionesLaterales()
			}
		})
	}()
}

func (a *Aplicacion) reemplazarArchivoEnMemoria(archivo modelo.Archivo) {
	for indice := range a.elementos {
		if a.elementos[indice].Ruta == archivo.Ruta {
			a.elementos[indice] = archivo
			return
		}
	}
}

func (a *Aplicacion) solicitarPreview(archivo modelo.Archivo, tamanoMaximo int) {
	if archivo.Ruta == "" {
		return
	}
	if archivoEsLocal(archivo) && (archivo.Tipo == modelo.TipoImagen || archivo.Tipo == modelo.TipoVideo) {
		a.solicitarEnriquecimientoExplorador(archivo)
	}

	orientacionObjetivo := orientacionPreviewArchivo(archivo)
	rotacionObjetivo := rotacionPreviewArchivo(archivo)
	preview, existe := a.previews[archivo.Ruta]
	if existe && preview != nil {
		if preview.Cargando {
			return
		}
		if preview.Imagen != nil &&
			preview.Maximo >= tamanoMaximo &&
			preview.Orientacion == orientacionObjetivo &&
			preview.Rotacion == rotacionObjetivo {
			return
		}
	}

	var imagenActual image.Image
	maximoActual := 0
	if preview != nil && preview.Orientacion == orientacionObjetivo && preview.Rotacion == rotacionObjetivo {
		imagenActual = preview.Imagen
		maximoActual = preview.Maximo
	}
	if maximoActual > tamanoMaximo {
		return
	}
	a.previews[archivo.Ruta] = &estadoPreview{
		Imagen:      imagenActual,
		Cargando:    true,
		Maximo:      maximo(maximoActual, tamanoMaximo),
		Orientacion: orientacionObjetivo,
		Rotacion:    rotacionObjetivo,
	}

	go func() {
		imagen, err := a.decodificarPreview(archivo, tamanoMaximo)
		a.encolarActualizacion(func() {
			if err != nil {
				if imagenActual != nil {
					a.previews[archivo.Ruta] = &estadoPreview{
						Imagen:      imagenActual,
						Cargando:    false,
						Maximo:      maximoActual,
						Orientacion: orientacionObjetivo,
						Rotacion:    rotacionObjetivo,
					}
					return
				}
				delete(a.previews, archivo.Ruta)
				return
			}
			a.previews[archivo.Ruta] = &estadoPreview{
				Imagen:      imagen,
				Cargando:    false,
				Maximo:      tamanoMaximo,
				Orientacion: orientacionObjetivo,
				Rotacion:    rotacionObjetivo,
			}
		})
	}()
}

func orientacionPreviewArchivo(archivo modelo.Archivo) int {
	if archivo.Tipo != modelo.TipoImagen {
		return 1
	}
	return modelo.NormalizarOrientacionVisual(archivo.Metadatos.Orientacion)
}

func rotacionPreviewArchivo(archivo modelo.Archivo) int {
	if archivo.Tipo != modelo.TipoVideo {
		return 0
	}
	return modelo.NormalizarRotacionCuartos(archivo.Metadatos.Rotacion)
}

func (a *Aplicacion) validarCarpetaDestinoLocal(ruta string) (string, error) {
	ruta = a.normalizarRutaLocalDestino(ruta)
	info, err := os.Stat(ruta)
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		return "", fmt.Errorf("%s no es una carpeta", ruta)
	}
	return ruta, nil
}

func (a *Aplicacion) moverArchivoActivo() {
	if !a.tieneArchivoActivo || !archivoEsLocal(a.archivoActivo) {
		return
	}

	destinoDir, err := a.validarCarpetaDestinoLocal(a.rutaDestinoActivoLocal)
	if err != nil {
		a.establecerEstado("Selecciona una carpeta local válida para mover el archivo activo", err)
		return
	}

	archivo := a.archivoActivo
	destino := filepath.Join(destinoDir, filepath.Base(archivo.Ruta))
	go func() {
		if err := a.servicioArchivos.Mover(context.Background(), archivo.Ruta, destino); err != nil {
			a.encolarActualizacion(func() {
				a.establecerEstado("No se pudo mover el archivo activo", err)
			})
			return
		}

		archivoAnterior := archivo.Ruta
		archivo.Ruta = destino
		archivo.RutaPadre = filepath.Dir(destino)
		archivo.Nombre = filepath.Base(destino)

		errEliminar := a.almacen.EliminarArchivo(context.Background(), archivoAnterior)
		errGuardar := a.almacen.GuardarArchivo(context.Background(), archivo)
		a.encolarActualizacion(func() {
			if errEliminar != nil {
				a.establecerEstado("El archivo se movió, pero no se pudo limpiar la ruta anterior del catálogo", errEliminar)
			} else if errGuardar != nil {
				a.establecerEstado("El archivo se movió, pero no se pudo persistir su nueva ruta", errGuardar)
			} else {
				a.establecerEstado("Archivo movido correctamente", nil)
			}
			a.refrescarExploradorTrasAccionArchivo()
		})
	}()
}

func (a *Aplicacion) archivarArchivoActivo() {
	if !a.tieneArchivoActivo || !archivoEsLocal(a.archivoActivo) {
		return
	}

	archivo := a.archivoActivo
	if !archivoTieneFechaYHoraArchivables(archivo) {
		a.establecerEstado("El archivo necesita fecha y hora guardadas para archivarse", nil)
		return
	}

	go func() {
		destino, err := archivarArchivoConFecha(context.Background(), a.servicioArchivos, archivo)
		if err != nil {
			a.encolarActualizacion(func() {
				a.establecerEstado("No se pudo archivar el archivo activo", err)
			})
			return
		}

		errEliminar := a.almacen.EliminarArchivo(context.Background(), archivo.Ruta)
		archivo.Ruta = destino
		archivo.RutaPadre = filepath.Dir(destino)
		archivo.Nombre = filepath.Base(destino)
		errGuardar := a.almacen.GuardarArchivo(context.Background(), archivo)
		a.encolarActualizacion(func() {
			if errEliminar != nil {
				a.establecerEstado("El archivo se archivó, pero quedó un registro antiguo en el catálogo", errEliminar)
			} else if errGuardar != nil {
				a.establecerEstado("El archivo se archivó, pero no se pudo persistir su nueva ruta", errGuardar)
			} else {
				a.establecerEstado("Archivo archivado correctamente", nil)
			}
			a.refrescarExploradorTrasAccionArchivo()
		})
	}()
}

func (a *Aplicacion) enviarArchivoActivoAPapelera() {
	if !a.tieneArchivoActivo || !archivoEsLocal(a.archivoActivo) {
		return
	}

	ruta := a.archivoActivo.Ruta
	go func() {
		errAccion := a.servicioArchivos.EnviarAPapelera(context.Background(), ruta)
		errCatalogo := a.almacen.EliminarArchivo(context.Background(), ruta)
		a.encolarActualizacion(func() {
			if errAccion != nil {
				a.establecerEstado("No se pudo enviar el archivo activo a la papelera", errAccion)
				return
			}
			if errCatalogo != nil {
				a.establecerEstado("El archivo fue enviado a la papelera, pero no se limpió el catálogo", errCatalogo)
			} else {
				a.establecerEstado("Archivo enviado a la papelera", nil)
			}
			a.refrescarExploradorTrasAccionArchivo()
		})
	}()
}

func (a *Aplicacion) guardarArchivoRemotoActivo() {
	if !a.tieneArchivoActivo || !archivoEsRemotoYandex(a.archivoActivo) || a.archivoActivo.EsDirectorio {
		return
	}
	if a.clienteYandex == nil || !a.clienteYandex.Configurado() {
		a.establecerEstado("No hay una conexión de Yandex.Disk disponible para descargar el archivo remoto", yandex.ErrNoImplementado)
		return
	}

	destinoDir, err := a.validarCarpetaDestinoLocal(a.rutaDestinoActivoLocal)
	if err != nil {
		a.establecerEstado("Selecciona una carpeta local válida para descargar el archivo remoto", err)
		return
	}

	archivo := a.archivoActivo
	go func() {
		destinoGuardado, err := a.descargarArchivoRemotoEnCarpeta(context.Background(), archivo, destinoDir)
		if err != nil {
			a.encolarActualizacion(func() {
				a.establecerEstado("No se pudo descargar el archivo remoto de Yandex.Disk", err)
			})
			return
		}

		archivoLocal, errAnalisis := a.analizarArchivoLocalEnRuta(context.Background(), destinoGuardado)
		if errAnalisis == nil {
			if errGuardar := a.almacen.GuardarArchivo(context.Background(), archivoLocal); errGuardar != nil {
				errAnalisis = errGuardar
			}
		}

		a.encolarActualizacion(func() {
			if errAnalisis != nil {
				a.establecerEstado("Archivo remoto descargado con incidencias al integrarlo en el catálogo local", errAnalisis)
			} else {
				a.establecerEstado("Archivo remoto descargado localmente", nil)
			}
			a.reiniciarListadoPreservandoPosicion()
			a.recargarColeccionesLaterales()
		})
	}()
}

func (a *Aplicacion) moverArchivoRemotoActivo() {
	if !a.tieneArchivoActivo || !archivoEsRemotoYandex(a.archivoActivo) {
		return
	}
	if a.clienteYandex == nil || !a.clienteYandex.Configurado() {
		a.establecerEstado("No hay una conexión de Yandex.Disk disponible para mover el elemento remoto", yandex.ErrNoImplementado)
		return
	}

	destinoBase := normalizarRutaYandexUI(a.rutaDestinoActivoRemoto)
	if destinoBase == "" {
		destinoBase = "disk:/"
	}

	archivo := a.archivoActivo
	destino := normalizarRutaYandexUI(destinoBase + "/" + archivo.NombreVisible())
	if destino == normalizarRutaYandexUI(archivo.Ruta) {
		a.establecerEstado("Selecciona una carpeta remota distinta para mover el elemento", nil)
		return
	}

	go func() {
		err := a.clienteYandex.Mover(context.Background(), archivo.Ruta, destino)
		a.encolarActualizacion(func() {
			if err != nil {
				a.establecerEstado("No se pudo mover el elemento remoto", err)
				return
			}
			a.establecerEstado("Elemento remoto movido correctamente", nil)
			a.refrescarExploradorTrasAccionArchivo()
		})
	}()
}

func (a *Aplicacion) enviarArchivoRemotoActivoAPapelera() {
	if !a.tieneArchivoActivo || !archivoEsRemotoYandex(a.archivoActivo) {
		return
	}
	if a.clienteYandex == nil || !a.clienteYandex.Configurado() {
		a.establecerEstado("No hay una conexión de Yandex.Disk disponible para mover el elemento remoto a la papelera", yandex.ErrNoImplementado)
		return
	}

	ruta := a.archivoActivo.Ruta
	go func() {
		err := a.clienteYandex.EnviarAPapelera(context.Background(), ruta)
		a.encolarActualizacion(func() {
			if err != nil {
				a.establecerEstado("No se pudo mover el elemento remoto a la papelera", err)
				return
			}
			a.establecerEstado("Elemento remoto enviado a la papelera", nil)
			a.refrescarExploradorTrasAccionArchivo()
		})
	}()
}

func (a *Aplicacion) descargarArchivoRemotoEnCarpeta(ctx context.Context, archivo modelo.Archivo, destinoDir string) (string, error) {
	if a.clienteYandex == nil || !a.clienteYandex.Configurado() {
		return "", yandex.ErrNoImplementado
	}

	lector, err := a.clienteYandex.Descargar(ctx, archivo.Ruta)
	if err != nil {
		return "", err
	}
	defer lector.Close()

	destino := filepath.Join(destinoDir, archivo.NombreVisible())
	return a.servicioArchivos.GuardarContenidoLocalDisponible(destino, lector)
}

func (a *Aplicacion) refrescarExploradorTrasAccionArchivo() {
	a.prepararEstadoTrasAccionArchivo()
	a.reiniciarListadoPreservandoPosicion()
	a.recargarDuplicados()
}

func (a *Aplicacion) prepararEstadoTrasAccionArchivo() {
	if a.vistaActual == vistaElementoUnico {
		a.cambiarVista(vistaPrincipal)
	}
	a.tieneArchivoActivo = false
	a.descartarEdicionRegiones()
	a.descartarEdicionRecorte()
	a.limpiarReproductorVideo()
}

// La ancla de selección sólo vive mientras exista un único elemento seleccionado.
// Eso permite construir rangos con primer/último elemento sin romper la selección libre.
func (a *Aplicacion) actualizarAnclaSeleccionLote() {
	if len(a.seleccionLote) != 1 {
		a.anclaSeleccionLote = ""
		return
	}

	for _, archivo := range a.elementos {
		if a.seleccionLote[archivo.Ruta] {
			a.anclaSeleccionLote = archivo.Ruta
			return
		}
	}

	for ruta := range a.seleccionLote {
		a.anclaSeleccionLote = ruta
		return
	}
	a.anclaSeleccionLote = ""
}

func (a *Aplicacion) indiceElementoPorRuta(ruta string) int {
	for indice := range a.elementos {
		if a.elementos[indice].Ruta == ruta {
			return indice
		}
	}
	return -1
}

func (a *Aplicacion) seleccionarRangoEntreRutas(inicioRuta, finRuta string) bool {
	inicio := a.indiceElementoPorRuta(inicioRuta)
	fin := a.indiceElementoPorRuta(finRuta)
	if inicio < 0 || fin < 0 {
		return false
	}
	if inicio > fin {
		inicio, fin = fin, inicio
	}
	if a.seleccionLote == nil {
		a.seleccionLote = make(map[string]bool)
	}
	for indice := inicio; indice <= fin; indice++ {
		ruta := strings.TrimSpace(a.elementos[indice].Ruta)
		if ruta == "" {
			continue
		}
		a.seleccionLote[ruta] = true
	}
	return true
}

func (a *Aplicacion) seleccionarElementoConPosibleRango(ruta string) {
	ruta = strings.TrimSpace(ruta)
	if ruta == "" {
		return
	}
	if a.seleccionLote == nil {
		a.seleccionLote = make(map[string]bool)
	}
	if a.anclaSeleccionLote != "" && a.anclaSeleccionLote != ruta {
		if a.seleccionarRangoEntreRutas(a.anclaSeleccionLote, ruta) {
			a.actualizarAnclaSeleccionLote()
			return
		}
	}
	a.seleccionLote[ruta] = true
	a.actualizarAnclaSeleccionLote()
}

func (a *Aplicacion) deseleccionarElemento(ruta string) {
	if a.seleccionLote == nil {
		return
	}
	delete(a.seleccionLote, strings.TrimSpace(ruta))
	a.actualizarAnclaSeleccionLote()
}

func (a *Aplicacion) actualizarSeleccionElemento(ruta string, estabaSeleccionado, ahoraSeleccionado bool) bool {
	if estabaSeleccionado == ahoraSeleccionado {
		return false
	}
	if ahoraSeleccionado {
		a.seleccionarElementoConPosibleRango(ruta)
	} else {
		a.deseleccionarElemento(ruta)
	}
	return true
}

func (a *Aplicacion) seleccionarTodosElementosCargados() {
	if a.seleccionLote == nil {
		a.seleccionLote = make(map[string]bool)
	}
	for _, archivo := range a.elementos {
		ruta := strings.TrimSpace(archivo.Ruta)
		if ruta == "" {
			continue
		}
		a.seleccionLote[ruta] = true
	}
	a.actualizarAnclaSeleccionLote()
}

func (a *Aplicacion) deseleccionarTodosElementos() {
	a.seleccionLote = make(map[string]bool)
	a.anclaSeleccionLote = ""
}

func (a *Aplicacion) seleccionLoteAdmiteAccionesLocales() bool {
	return a.tipoSeleccionLote() == seleccionLoteLocal
}

func (a *Aplicacion) seleccionLoteAdmiteAccionesYandex() bool {
	return a.tipoSeleccionLote() == seleccionLoteYandex
}

func (a *Aplicacion) tipoSeleccionLote() tipoSeleccionLote {
	rutas := a.rutasSeleccionadas()
	if len(rutas) == 0 {
		return seleccionLoteVacia
	}

	tieneLocales := false
	tieneRemotos := false
	for _, ruta := range rutas {
		archivo, ok := a.archivoPorRuta(ruta)
		if !ok {
			continue
		}
		if archivoEsRemotoYandex(archivo) {
			tieneRemotos = true
		} else {
			tieneLocales = true
		}
		if tieneLocales && tieneRemotos {
			return seleccionLoteMixta
		}
	}

	switch {
	case tieneLocales && !tieneRemotos:
		return seleccionLoteLocal
	case tieneRemotos && !tieneLocales:
		return seleccionLoteYandex
	default:
		return seleccionLoteVacia
	}
}

func (a *Aplicacion) rutasSeleccionadas() []string {
	rutas := make([]string, 0, len(a.seleccionLote))
	for ruta, activa := range a.seleccionLote {
		if activa {
			rutas = append(rutas, ruta)
		}
	}
	return rutas
}

func (a *Aplicacion) archivoPorRuta(ruta string) (modelo.Archivo, bool) {
	for _, archivo := range a.elementos {
		if archivo.Ruta == ruta {
			return archivo, true
		}
	}
	return modelo.Archivo{}, false
}

func (a *Aplicacion) moverSeleccionLote() {
	destinoDir, err := a.validarCarpetaDestinoLocal(a.rutaDestinoLoteLocal)
	if err != nil {
		a.establecerEstado("Selecciona una carpeta local válida para mover la selección", err)
		return
	}
	a.procesarSeleccionLote(func(archivo modelo.Archivo) error {
		destino := filepath.Join(destinoDir, filepath.Base(archivo.Ruta))
		if err := a.servicioArchivos.Mover(context.Background(), archivo.Ruta, destino); err != nil {
			return err
		}
		if err := a.almacen.EliminarArchivo(context.Background(), archivo.Ruta); err != nil {
			return err
		}
		archivo.Ruta = destino
		archivo.RutaPadre = filepath.Dir(destino)
		archivo.Nombre = filepath.Base(destino)
		return a.almacen.GuardarArchivo(context.Background(), archivo)
	})
}

func (a *Aplicacion) archivarSeleccionLote() {
	a.procesarSeleccionLote(func(archivo modelo.Archivo) error {
		destino, err := archivarArchivoConFecha(context.Background(), a.servicioArchivos, archivo)
		if err != nil {
			return err
		}
		if err := a.almacen.EliminarArchivo(context.Background(), archivo.Ruta); err != nil {
			return err
		}
		archivo.Ruta = destino
		archivo.RutaPadre = filepath.Dir(destino)
		archivo.Nombre = filepath.Base(destino)
		return a.almacen.GuardarArchivo(context.Background(), archivo)
	})
}

func archivarArchivoConFecha(ctx context.Context, servicio *archivos.Servicio, archivo modelo.Archivo) (string, error) {
	if servicio == nil {
		return "", errors.New("servicio de archivos no inicializado")
	}
	if !archivoTieneFechaYHoraArchivables(archivo) {
		return "", errors.New("el archivo necesita fecha y hora guardadas para archivarse")
	}
	return servicio.ArchivarConFecha(ctx, archivo.Ruta, archivo.Metadatos.Fecha, archivo.Metadatos.Hora)
}

func (a *Aplicacion) enviarSeleccionLoteAPapelera() {
	a.procesarSeleccionLote(func(archivo modelo.Archivo) error {
		if err := a.servicioArchivos.EnviarAPapelera(context.Background(), archivo.Ruta); err != nil {
			return err
		}
		return a.almacen.EliminarArchivo(context.Background(), archivo.Ruta)
	})
}

func (a *Aplicacion) descargarSeleccionLoteRemota() {
	destinoDir, err := a.validarCarpetaDestinoLocal(a.rutaDestinoLoteLocal)
	if err != nil {
		a.establecerEstado("Selecciona una carpeta local válida para descargar la selección remota", err)
		return
	}

	a.procesarSeleccionLoteYandex(func(archivo modelo.Archivo) error {
		if archivo.EsDirectorio {
			return fmt.Errorf("la descarga de carpetas remotas aún no está disponible: %s", archivo.NombreVisible())
		}
		rutaGuardada, err := a.descargarArchivoRemotoEnCarpeta(context.Background(), archivo, destinoDir)
		if err != nil {
			return err
		}
		archivoLocal, err := a.analizarArchivoLocalEnRuta(context.Background(), rutaGuardada)
		if err != nil {
			return err
		}
		return a.almacen.GuardarArchivo(context.Background(), archivoLocal)
	}, true)
}

func (a *Aplicacion) moverSeleccionLoteRemota() {
	destinoBase := normalizarRutaYandexUI(a.rutaDestinoLoteRemoto)
	if destinoBase == "" {
		destinoBase = "disk:/"
	}

	a.procesarSeleccionLoteYandex(func(archivo modelo.Archivo) error {
		destino := normalizarRutaYandexUI(destinoBase + "/" + archivo.NombreVisible())
		if destino == normalizarRutaYandexUI(archivo.Ruta) {
			return fmt.Errorf("el elemento %s ya está en la carpeta remota seleccionada", archivo.NombreVisible())
		}
		return a.clienteYandex.Mover(context.Background(), archivo.Ruta, destino)
	}, false)
}

func (a *Aplicacion) enviarSeleccionLoteRemotaAPapelera() {
	a.procesarSeleccionLoteYandex(func(archivo modelo.Archivo) error {
		return a.clienteYandex.EnviarAPapelera(context.Background(), archivo.Ruta)
	}, false)
}

func (a *Aplicacion) procesarSeleccionLote(accion func(archivo modelo.Archivo) error) {
	rutas := a.rutasSeleccionadas()
	if len(rutas) == 0 {
		a.establecerEstado("No hay elementos seleccionados para la acción en lote", nil)
		return
	}
	if !a.seleccionLoteAdmiteAccionesLocales() {
		a.establecerEstado("Las acciones por lote sólo están disponibles para archivos locales en el Explorador actual", nil)
		return
	}

	go func() {
		var primerError error
		procesados := 0
		for _, ruta := range rutas {
			archivo, ok := a.archivoPorRuta(ruta)
			if !ok {
				continue
			}
			if err := accion(archivo); err != nil && primerError == nil {
				primerError = err
			} else if err == nil {
				procesados++
			}
		}

		a.encolarActualizacion(func() {
			a.seleccionLote = make(map[string]bool)
			a.anclaSeleccionLote = ""
			if primerError != nil {
				a.establecerEstado(fmt.Sprintf("Acción por lotes con errores. %d elementos procesados", procesados), primerError)
			} else {
				a.establecerEstado(fmt.Sprintf("Acción por lotes completada para %d elementos", procesados), nil)
			}
			a.reiniciarListadoPreservandoPosicion()
			a.recargarDuplicados()
		})
	}()
}

func (a *Aplicacion) procesarSeleccionLoteYandex(accion func(archivo modelo.Archivo) error, refrescarColecciones bool) {
	rutas := a.rutasSeleccionadas()
	if len(rutas) == 0 {
		a.establecerEstado("No hay elementos remotos seleccionados para la acción en lote", nil)
		return
	}
	if !a.seleccionLoteAdmiteAccionesYandex() {
		a.establecerEstado("Las acciones en lote remotas requieren que toda la selección pertenezca a Yandex.Disk", nil)
		return
	}
	if a.clienteYandex == nil || !a.clienteYandex.Configurado() {
		a.establecerEstado("No hay una conexión de Yandex.Disk disponible para la acción en lote", yandex.ErrNoImplementado)
		return
	}

	go func() {
		var primerError error
		procesados := 0
		for _, ruta := range rutas {
			archivo, ok := a.archivoPorRuta(ruta)
			if !ok {
				continue
			}
			if err := accion(archivo); err != nil && primerError == nil {
				primerError = err
			} else if err == nil {
				procesados++
			}
		}

		a.encolarActualizacion(func() {
			a.seleccionLote = make(map[string]bool)
			a.anclaSeleccionLote = ""
			if primerError != nil {
				a.establecerEstado(fmt.Sprintf("Acción remota por lotes con errores. %d elementos procesados", procesados), primerError)
			} else {
				a.establecerEstado(fmt.Sprintf("Acción remota completada para %d elementos", procesados), nil)
			}
			a.reiniciarListadoPreservandoPosicion()
			if refrescarColecciones {
				a.recargarColeccionesLaterales()
			}
			a.recargarDuplicados()
		})
	}()
}

func (a *Aplicacion) recortarImagenActiva() {
	if !a.tieneArchivoActivo || !archivoEsLocal(a.archivoActivo) || a.archivoActivo.Tipo != modelo.TipoImagen {
		return
	}
	archivo := a.archivoActivo
	a.sincronizarEdicionRecorte(archivo)
	if a.edicionRecorte.Guardando {
		return
	}
	if a.hayCambiosPendientesRegiones() || a.edicionRegiones.RegionPendiente != nil {
		a.establecerEstado("Guarda o limpia primero los cambios pendientes de regiones antes de recortar la imagen", nil)
		return
	}
	if archivo.Ancho == 0 || archivo.Alto == 0 {
		a.establecerEstado("No se conocen las dimensiones de la imagen para proponer un recorte", nil)
		return
	}

	rect, ok := a.rectanguloRecorteActivoPixeles(archivo)
	if !ok {
		if sugerencia, sugerida := a.sugerenciaRecorte(archivo); sugerida {
			rect, ok = rectanguloRecortePixelesParaRegion(archivo, sugerencia)
			if ok {
				a.edicionRecorte.Seleccion = sugerencia
				a.edicionRecorte.TieneSeleccion = true
				a.edicionRecorte.Sugerida = true
			}
		}
	}
	if !ok {
		anchoOrientado, altoOrientado := dimensionesOrientadasArchivo(archivo)
		lado := anchoOrientado
		if altoOrientado < lado {
			lado = altoOrientado
		}
		lado = int(float64(lado) * 0.9)
		inicioX := maximo(0, (anchoOrientado-lado)/2)
		inicioY := maximo(0, (altoOrientado-lado)/2)
		rect = image.Rect(inicioX, inicioY, inicioX+lado, inicioY+lado)
	}
	salida := rutaDerivada(archivo.Ruta, "recorte", filepath.Ext(archivo.Ruta))
	reemplazarOriginal := a.reemplazarOriginalRecorte.Value
	a.edicionRecorte.Guardando = true

	go func() {
		rutaFinal, err := a.servicioMetadatos.RecortarImagen(context.Background(), archivo, rect, salida, reemplazarOriginal)
		archivoActualizado, errAnalisis := modelo.Archivo{}, error(nil)
		errCatalogo := error(nil)
		if err == nil {
			archivoActualizado, errAnalisis = a.analizarArchivoLocalEnRuta(context.Background(), rutaFinal)
			if errAnalisis == nil {
				errCatalogo = a.almacen.GuardarArchivo(context.Background(), archivoActualizado)
			}
		}
		a.encolarActualizacion(func() {
			a.edicionRecorte.Guardando = false
			if err != nil {
				a.establecerEstado("No se pudo recortar la imagen", err)
				return
			}
			delete(a.previews, archivo.Ruta)
			delete(a.previews, rutaFinal)
			delete(a.metadatosPendientes, archivo.Ruta)
			delete(a.metadatosPendientes, rutaFinal)
			delete(a.metadatosVerificados, archivo.Ruta)
			delete(a.metadatosVerificados, rutaFinal)
			a.descartarEdicionRecorte()
			a.descartarEdicionRegiones()
			if archivoActualizado.Ruta != "" {
				if errAnalisis == nil {
					a.marcarArchivoVerificadoConSistema(archivoActualizado)
				}
				a.activarArchivo(archivoActualizado)
			}
			mensaje := "Imagen recortada creada"
			if reemplazarOriginal {
				mensaje = "Imagen recortada reemplazando el archivo original"
			} else {
				mensaje = "Imagen recortada creada y abierta en el visor"
			}
			if errAnalisis != nil {
				a.establecerEstado(mensaje+" con incidencias al refrescar el catálogo", errAnalisis)
			} else if errCatalogo != nil {
				a.establecerEstado(mensaje+" con incidencias al guardar en el catálogo", errCatalogo)
			} else {
				a.establecerEstado(mensaje, nil)
			}
			a.reiniciarListado()
			a.recargarColeccionesLaterales()
			a.recargarDuplicados()
		})
	}()
}

func (a *Aplicacion) convertirImagenActiva() {
	if !a.tieneArchivoActivo || !archivoEsLocal(a.archivoActivo) || a.archivoActivo.Tipo != modelo.TipoImagen {
		return
	}
	formato := strings.TrimSpace(a.editorFormatoImagen.Text())
	if formato == "" {
		formato = "webp"
	}
	salida := rutaDerivada(a.archivoActivo.Ruta, "convertida", "."+strings.TrimPrefix(formato, "."))

	go func() {
		err := a.servicioMetadatos.ConvertirImagen(context.Background(), a.archivoActivo.Ruta, formato, salida)
		a.encolarActualizacion(func() {
			if err != nil {
				a.establecerEstado("No se pudo convertir la imagen", err)
				return
			}
			a.establecerEstado("Imagen convertida correctamente", nil)
			a.reiniciarListado()
		})
	}()
}

func (a *Aplicacion) extraerFrameActivo() {
	if !a.tieneArchivoActivo || !archivoEsLocal(a.archivoActivo) || a.archivoActivo.Tipo != modelo.TipoVideo || a.servicioMetadatos == nil {
		return
	}

	a.sincronizarReproductorVideo(a.archivoActivo)
	archivo := a.archivoActivo
	formato := a.formatoExtraccionFrameNormalizado()
	instante := a.instanteExtraerFrameActivo()
	rotacion := modelo.NormalizarRotacionCuartos(archivo.Metadatos.Rotacion)

	go func() {
		resultado, err := a.servicioMetadatos.ExtraerFrameEnInstante(context.Background(), archivo.Ruta, instante, formato, rotacion)
		a.encolarActualizacion(func() {
			if err != nil {
				if strings.TrimSpace(resultado.Ruta) != "" {
					if _, errStat := os.Stat(resultado.Ruta); errStat == nil {
						a.establecerEstado("Frame extraído con incidencias al copiar metadatos", err)
						a.reiniciarListado()
						return
					}
				}
				a.establecerEstado("No se pudo extraer el frame del video", err)
				return
			}
			a.establecerEstado(fmt.Sprintf("Frame %d extraído correctamente", resultado.Numero), nil)
			a.reiniciarListado()
		})
	}()
}

func (a *Aplicacion) optimizarVideoActivo() {
	if !a.tieneArchivoActivo || !archivoEsLocal(a.archivoActivo) || a.archivoActivo.Tipo != modelo.TipoVideo {
		return
	}
	salida := rutaDerivada(a.archivoActivo.Ruta, "web", ".mp4")
	sobreescribir := a.sobreescribirVideo.Value

	go func() {
		err := a.servicioMetadatos.OptimizarVideoWeb(context.Background(), a.archivoActivo.Ruta, salida, sobreescribir)
		a.encolarActualizacion(func() {
			if err != nil {
				a.establecerEstado("No se pudo optimizar el video", err)
				return
			}
			a.establecerEstado("Video optimizado para web", nil)
			a.reiniciarListado()
		})
	}()
}

func (a *Aplicacion) reproducirArchivoActivo() {
	if !a.tieneArchivoActivo || !archivoEsLocal(a.archivoActivo) || a.archivoActivo.Ruta == "" {
		return
	}

	ruta := a.archivoActivo.Ruta
	go func() {
		err := a.servicioArchivos.AbrirEnSistema(context.Background(), ruta)
		a.encolarActualizacion(func() {
			if err != nil {
				a.establecerEstado("No se pudo abrir el archivo en el sistema", err)
				return
			}
			a.establecerEstado("Archivo abierto en el sistema", nil)
		})
	}()
}

func (a *Aplicacion) abrirCarpetaContenedoraArchivoActivo() {
	if !a.tieneArchivoActivo || !archivoEsLocal(a.archivoActivo) || a.archivoActivo.Ruta == "" {
		return
	}

	ruta := filepath.Dir(a.archivoActivo.Ruta)
	go func() {
		err := a.servicioArchivos.AbrirEnSistema(context.Background(), ruta)
		a.encolarActualizacion(func() {
			if err != nil {
				a.establecerEstado("No se pudo abrir la carpeta contenedora", err)
				return
			}
			a.establecerEstado("Carpeta contenedora abierta en el sistema", nil)
		})
	}()
}

func (a *Aplicacion) guardarConfiguracion() {
	nueva := a.configuracion
	nueva.CarpetaInicial = strings.TrimSpace(a.editorCarpetaInicial.Text())
	nueva.CarpetaArchivado = strings.TrimSpace(a.editorCarpetaArchivado.Text())
	nueva.ClaveAPIYandex = strings.TrimSpace(a.editorClaveYandex.Text())
	nueva.FiltrosPorDefecto = modelo.FiltrosListado{
		MostrarOcultos:   a.configMostrarOcultos.Value,
		OcultarCarpetas:  a.configOcultarCarpetas.Value,
		SoloMultimedia:   a.configSoloMultimedia.Value,
		SoloVideos:       a.configSoloVideos.Value,
		SoloImagenes:     a.configSoloImagenes.Value,
		SoloAudio:        a.configSoloAudio.Value,
		Recursivo:        a.configRecursivo.Value,
		CriterioOrden:    a.criterioOrdenConfiguracion(),
		OrdenDescendente: a.configOrdenDescendente.Value,
		VistaGaleria:     true,
	}

	go func() {
		err := a.repoConfiguracion.Guardar(nueva)
		a.encolarActualizacion(func() {
			if err != nil {
				a.establecerEstado("No se pudo guardar la configuración", err)
				return
			}
			cfgNormalizada, err := configuracion.NormalizarConfiguracion(nueva)
			if err != nil {
				a.establecerEstado("La configuración se guardó, pero no se pudo normalizar en memoria", err)
				return
			}
			a.configuracion = cfgNormalizada
			a.filtros = cfgNormalizada.FiltrosPorDefecto
			a.mostrarOcultos.Value = a.filtros.MostrarOcultos
			a.ocultarCarpetas.Value = a.filtros.OcultarCarpetas
			a.soloMultimedia.Value = a.filtros.SoloMultimedia
			a.soloVideos.Value = a.filtros.SoloVideos
			a.soloImagenes.Value = a.filtros.SoloImagenes
			a.soloAudio.Value = a.filtros.SoloAudio
			a.recursivo.Value = a.filtros.Recursivo
			a.configMostrarOcultos.Value = a.filtros.MostrarOcultos
			a.configOcultarCarpetas.Value = a.filtros.OcultarCarpetas
			a.configSoloMultimedia.Value = a.filtros.SoloMultimedia
			a.configSoloVideos.Value = a.filtros.SoloVideos
			a.configSoloImagenes.Value = a.filtros.SoloImagenes
			a.configSoloAudio.Value = a.filtros.SoloAudio
			a.configRecursivo.Value = a.filtros.Recursivo
			a.configOrdenPorFecha.Value = a.filtros.CriterioOrdenNormalizado() == modelo.CriterioOrdenFechaModificacion
			a.configOrdenDescendente.Value = a.filtros.OrdenDescendente
			a.carpetaSeleccionada = cfgNormalizada.CarpetaInicial
			a.origenListado = origenListadoCarpeta
			a.claveListadoActual = cfgNormalizada.CarpetaInicial
			a.servicioArchivos.ActualizarCarpetaArchivado(cfgNormalizada.CarpetaArchivado)
			a.clienteYandex = yandex.NuevoCliente(cfgNormalizada.ClaveAPIYandex)
			a.raizArbolYandex = nil
			a.carpetaYandexSeleccionada = ""
			a.reconstruirArbol()
			a.reiniciarListado()
			a.establecerEstado("Configuración guardada correctamente", nil)
		})
	}()
}

func (a *Aplicacion) alternarFiltrosDesdeUI() {
	nuevosFiltros := modelo.FiltrosListado{
		MostrarOcultos:   a.mostrarOcultos.Value,
		OcultarCarpetas:  a.ocultarCarpetas.Value,
		SoloMultimedia:   a.soloMultimedia.Value,
		SoloVideos:       a.soloVideos.Value,
		SoloImagenes:     a.soloImagenes.Value,
		SoloAudio:        a.soloAudio.Value,
		Recursivo:        a.recursivo.Value,
		CriterioOrden:    a.filtros.CriterioOrdenNormalizado(),
		OrdenDescendente: a.filtros.OrdenDescendente,
		VistaGaleria:     a.filtros.VistaGaleria,
	}
	if nuevosFiltros != a.filtros {
		a.filtros = nuevosFiltros
		a.reiniciarListado()
	}
}

func (a *Aplicacion) decodificarPreview(archivo modelo.Archivo, maximo int) (image.Image, error) {
	if archivoEsRemotoYandex(archivo) {
		if a.clienteYandex == nil || !a.clienteYandex.Configurado() {
			return nil, yandex.ErrNoImplementado
		}
		var (
			lector io.ReadCloser
			err    error
		)
		if strings.TrimSpace(archivo.PreviewURL) != "" {
			lector, err = a.clienteYandex.DescargarPreviewURL(context.Background(), archivo.PreviewURL)
		} else {
			lector, err = a.clienteYandex.DescargarPreview(context.Background(), archivo.Ruta, tamanoPreviewYandex(maximo))
		}
		if err != nil {
			return nil, err
		}
		defer lector.Close()

		contenido, err := io.ReadAll(lector)
		if err != nil {
			return nil, fmt.Errorf("no se pudo leer la vista previa remota: %w", err)
		}
		imagenPreview, _, err := image.Decode(bytes.NewReader(contenido))
		if err != nil {
			return nil, fmt.Errorf("no se pudo decodificar la vista previa remota: %w", err)
		}
		return imagenPreview, nil
	}

	switch archivo.Tipo {
	case modelo.TipoImagen:
		if a.servicioMetadatos == nil {
			return nil, errors.New("servicio de metadatos no inicializado")
		}
		return a.servicioMetadatos.GenerarPreviewImagen(context.Background(), archivo.Ruta, maximo, archivo.Metadatos.Orientacion)
	case modelo.TipoVideo:
		if a.servicioMetadatos == nil {
			return nil, errors.New("servicio de metadatos no inicializado")
		}
		return a.servicioMetadatos.GenerarPreviewVideo(context.Background(), archivo.Ruta, maximo, archivo.Metadatos.Rotacion)
	default:
		if a.servicioMetadatos == nil {
			return nil, errors.New("servicio de metadatos no inicializado")
		}
		return a.servicioMetadatos.GenerarPreviewImagen(context.Background(), archivo.Ruta, maximo, archivo.Metadatos.Orientacion)
	}
}

func tamanoPreviewYandex(maximo int) string {
	switch {
	case maximo <= 360:
		return "M"
	case maximo <= 540:
		return "L"
	case maximo <= 900:
		return "XL"
	case maximo <= 1_120:
		return "XXL"
	default:
		return "XXXL"
	}
}

func (a *Aplicacion) analizarArchivoLocalEnRuta(ctx context.Context, ruta string) (modelo.Archivo, error) {
	info, err := os.Stat(ruta)
	if err != nil {
		return modelo.Archivo{}, fmt.Errorf("no se pudo leer el archivo recortado: %w", err)
	}

	archivo := modelo.Archivo{
		Origen:       modelo.OrigenLocal,
		Ruta:         ruta,
		RutaPadre:    filepath.Dir(ruta),
		Nombre:       filepath.Base(ruta),
		Tamano:       info.Size(),
		Modificado:   info.ModTime(),
		Tipo:         modelo.TipoDesdeRuta(ruta, false),
		EsOculto:     modelo.EsOcultoPorNombre(filepath.Base(ruta)),
		EsDirectorio: false,
	}
	if a.servicioMetadatos == nil {
		return archivo, nil
	}

	enriquecido, err := a.servicioMetadatos.AnalizarArchivo(ctx, archivo)
	if err != nil {
		return archivo, err
	}
	return enriquecido, nil
}

func rutaDerivada(origen, sufijo, extension string) string {
	base := strings.TrimSuffix(origen, filepath.Ext(origen))
	return base + "-" + sufijo + extension
}

func partirListaCSV(texto string) []string {
	partes := strings.Split(texto, ",")
	vistos := make(map[string]struct{}, len(partes))
	var salida []string
	for _, parte := range partes {
		parte = strings.TrimSpace(parte)
		if parte == "" {
			continue
		}
		clave := strings.ToLower(parte)
		if _, existe := vistos[clave]; existe {
			continue
		}
		vistos[clave] = struct{}{}
		salida = append(salida, parte)
	}
	return salida
}

func formatearTamano(tamano int64) string {
	if tamano <= 0 {
		return "0 B"
	}
	return humanize.Bytes(uint64(tamano))
}

func formatearDuracion(duracion time.Duration) string {
	if duracion <= 0 {
		return "-"
	}
	totalSegundos := int(duracion.Round(time.Second) / time.Second)
	horas := totalSegundos / 3600
	minutos := (totalSegundos % 3600) / 60
	segundos := totalSegundos % 60
	if horas > 0 {
		return fmt.Sprintf("%02d:%02d:%02d", horas, minutos, segundos)
	}
	return fmt.Sprintf("%02d:%02d", minutos, segundos)
}

func maximo(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func minimo(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func tieneEtiquetaAdulta(etiquetas []string) bool {
	for _, etiqueta := range etiquetas {
		if strings.TrimSpace(strings.ToLower(etiqueta)) == "+18" {
			return true
		}
	}
	return false
}

func (a *Aplicacion) previewOp(ruta string) (paint.ImageOp, bool) {
	preview, existe := a.previews[ruta]
	if !existe || preview == nil || preview.Imagen == nil {
		return paint.ImageOp{}, false
	}
	return paint.NewImageOp(preview.Imagen), true
}

func compararTextoUI(izquierda, derecha string) bool {
	izquierda = strings.ToLower(strings.TrimSpace(izquierda))
	derecha = strings.ToLower(strings.TrimSpace(derecha))
	if izquierda == derecha {
		return strings.TrimSpace(izquierda) < strings.TrimSpace(derecha)
	}
	return izquierda < derecha
}

func normalizarRutaYandexUI(ruta string) string {
	ruta = strings.TrimSpace(ruta)
	if ruta == "" || ruta == "/" || ruta == "disk:" || ruta == "disk:/" {
		return "disk:/"
	}
	if strings.HasPrefix(ruta, "disk:/") {
		return "disk:/" + strings.TrimPrefix(strings.TrimPrefix(ruta, "disk:/"), "/")
	}
	if strings.HasPrefix(ruta, "/") {
		return "disk:" + ruta
	}
	return "disk:/" + strings.TrimPrefix(ruta, "/")
}

func convertirElementoYandexAArchivo(elemento yandex.ElementoRemoto) modelo.Archivo {
	nombre := strings.TrimSpace(elemento.Nombre)
	if nombre == "" {
		rutaNormalizada := normalizarRutaYandexUI(elemento.Ruta)
		partes := strings.Split(strings.TrimPrefix(rutaNormalizada, "disk:/"), "/")
		if len(partes) > 0 {
			nombre = partes[len(partes)-1]
		}
		if nombre == "" {
			nombre = "Yandex.Disk"
		}
	}
	ruta := normalizarRutaYandexUI(elemento.Ruta)
	return modelo.Archivo{
		Origen:       modelo.OrigenYandex,
		Ruta:         ruta,
		RutaPadre:    rutaPadreYandex(ruta),
		Nombre:       nombre,
		PreviewURL:   strings.TrimSpace(elemento.PreviewURL),
		Tamano:       elemento.Tamano,
		Modificado:   elemento.Modificado,
		Tipo:         elemento.Tipo,
		EsOculto:     modelo.EsOcultoPorNombre(nombre),
		EsDirectorio: elemento.EsDirectorio,
		Hashes: modelo.HashesArchivo{
			MD5:    strings.TrimSpace(elemento.HashMD5),
			SHA256: strings.TrimSpace(elemento.HashSHA256),
		},
	}
}

func rutaPadreYandex(ruta string) string {
	ruta = normalizarRutaYandexUI(ruta)
	if ruta == "disk:/" {
		return "disk:/"
	}
	relativa := strings.TrimPrefix(ruta, "disk:/")
	relativa = strings.TrimPrefix(relativa, "/")
	if relativa == "" || !strings.Contains(relativa, "/") {
		return "disk:/"
	}
	partes := strings.Split(relativa, "/")
	return normalizarRutaYandexUI("disk:/" + strings.Join(partes[:len(partes)-1], "/"))
}

func archivoEsRemotoYandex(archivo modelo.Archivo) bool {
	return archivo.Origen == modelo.OrigenYandex
}

func archivoEsLocal(archivo modelo.Archivo) bool {
	return !archivoEsRemotoYandex(archivo)
}

func convertirOpcionesLaterales(valores []string) []opcionFiltroLateral {
	opciones := make([]opcionFiltroLateral, 0, len(valores))
	for _, valor := range valores {
		valor = strings.TrimSpace(valor)
		if valor == "" {
			continue
		}
		opciones = append(opciones, opcionFiltroLateral{
			Clave:    valor,
			Etiqueta: valor,
		})
	}
	return opciones
}

func fusionarOpcionesLaterales(valores, extras []string) []opcionFiltroLateral {
	vistosEnBase := make(map[string]struct{}, len(valores))
	for _, valor := range valores {
		valor = strings.TrimSpace(valor)
		if valor == "" {
			continue
		}
		clave := strings.ToLower(valor)
		vistosEnBase[clave] = struct{}{}
	}

	opciones := make([]opcionFiltroLateral, 0, len(valores)+len(extras))
	vistos := make(map[string]struct{}, len(valores)+len(extras))

	for _, valor := range extras {
		valor = strings.TrimSpace(valor)
		if valor == "" {
			continue
		}
		clave := strings.ToLower(valor)
		if _, existe := vistosEnBase[clave]; existe {
			continue
		}
		if _, existe := vistos[clave]; existe {
			continue
		}
		vistos[clave] = struct{}{}
		opciones = append(opciones, opcionFiltroLateral{
			Clave:    valor,
			Etiqueta: valor,
		})
	}

	for _, valor := range valores {
		valor = strings.TrimSpace(valor)
		if valor == "" {
			continue
		}
		clave := strings.ToLower(valor)
		if _, existe := vistos[clave]; existe {
			continue
		}
		vistos[clave] = struct{}{}
		opciones = append(opciones, opcionFiltroLateral{
			Clave:    valor,
			Etiqueta: valor,
		})
	}
	return opciones
}

func (a *Aplicacion) tituloListadoActual() string {
	switch a.origenListado {
	case origenListadoEtiqueta:
		return "Tag: " + a.claveListadoActual
	case origenListadoUbicacion:
		return "Lugar: " + a.claveListadoActual
	case origenListadoUbicacionSinNombre:
		return "Lugar: " + etiquetaUbicacionSinNombre
	case origenListadoCarpetaYandex:
		if a.carpetaYandexSeleccionada == "" || a.carpetaYandexSeleccionada == "disk:/" {
			return "Yandex.Disk"
		}
		return "Yandex.Disk: " + strings.TrimPrefix(a.carpetaYandexSeleccionada, "disk:/")
	default:
		return a.carpetaSeleccionada
	}
}

func (a *Aplicacion) esFiltroLateralActivo(origen tipoOrigenListado, clave string) bool {
	if a.origenListado != origen {
		return false
	}
	if origen == origenListadoUbicacionSinNombre {
		return true
	}
	return strings.EqualFold(strings.TrimSpace(a.claveListadoActual), strings.TrimSpace(clave))
}

func (a *Aplicacion) asegurarWidgetLateral(clave string) *widget.Clickable {
	if clic, existe := a.widgetsLateral[clave]; existe {
		return clic
	}
	clic := &widget.Clickable{}
	a.widgetsLateral[clave] = clic
	return clic
}

func (a *Aplicacion) fusionarHijosNodo(nodo *nodoArbolUI, subdirectorios []indexador.NodoDirectorio) {
	existentes := make(map[string]*nodoArbolUI, len(nodo.Hijos))
	for _, hijo := range nodo.Hijos {
		existentes[hijo.Ruta] = hijo
	}

	nuevos := make([]*nodoArbolUI, 0, len(subdirectorios))
	for _, subdirectorio := range subdirectorios {
		if previo, existe := existentes[subdirectorio.Ruta]; existe {
			previo.Nombre = subdirectorio.Nombre
			nuevos = append(nuevos, previo)
			continue
		}
		nuevos = append(nuevos, &nodoArbolUI{
			Origen:  modelo.OrigenLocal,
			Ruta:    subdirectorio.Ruta,
			Nombre:  subdirectorio.Nombre,
			Cargado: false,
		})
	}

	nodo.Hijos = nuevos
}

func (a *Aplicacion) fusionarHijosNodoYandex(nodo *nodoArbolUI, subdirectorios []yandex.ElementoRemoto) {
	existentes := make(map[string]*nodoArbolUI, len(nodo.Hijos))
	for _, hijo := range nodo.Hijos {
		existentes[hijo.Ruta] = hijo
	}

	nuevos := make([]*nodoArbolUI, 0, len(subdirectorios))
	for _, subdirectorio := range subdirectorios {
		if !subdirectorio.EsDirectorio {
			continue
		}
		if previo, existe := existentes[subdirectorio.Ruta]; existe {
			previo.Nombre = subdirectorio.Nombre
			previo.Origen = modelo.OrigenYandex
			nuevos = append(nuevos, previo)
			continue
		}
		nuevos = append(nuevos, &nodoArbolUI{
			Origen:  modelo.OrigenYandex,
			Ruta:    subdirectorio.Ruta,
			Nombre:  subdirectorio.Nombre,
			Cargado: false,
		})
	}

	nodo.Hijos = nuevos
}

func (a *Aplicacion) cargarHijosNodoSincrono(nodo *nodoArbolUI) error {
	if nodo == nil {
		return nil
	}
	subdirectorios, err := a.listador.ListarSubdirectorios(context.Background(), nodo.Ruta, a.filtros.MostrarOcultos)
	if err != nil {
		return err
	}
	a.fusionarHijosNodo(nodo, subdirectorios)
	sort.SliceStable(nodo.Hijos, func(i, j int) bool {
		return compararTextoUI(nodo.Hijos[i].Nombre, nodo.Hijos[j].Nombre)
	})
	nodo.Cargado = true
	nodo.Cargando = false
	return nil
}

func (a *Aplicacion) sincronizarArbolConRuta(ruta string) error {
	if a.raizArbol == nil || ruta == "" {
		return nil
	}

	ruta = filepath.Clean(ruta)
	raiz := filepath.Clean(a.raizArbol.Ruta)

	relativa, err := filepath.Rel(raiz, ruta)
	if err != nil {
		return err
	}
	if strings.HasPrefix(relativa, "..") {
		return nil
	}

	actual := a.raizArbol
	actual.Expandido = true
	if err := a.cargarHijosNodoSincrono(actual); err != nil {
		return err
	}

	if relativa == "." {
		return nil
	}

	partes := strings.Split(relativa, string(filepath.Separator))
	for _, parte := range partes {
		if parte == "" || parte == "." {
			continue
		}

		if !actual.Cargado {
			if err := a.cargarHijosNodoSincrono(actual); err != nil {
				return err
			}
		}

		var siguiente *nodoArbolUI
		for _, hijo := range actual.Hijos {
			if strings.EqualFold(hijo.Nombre, parte) {
				siguiente = hijo
				break
			}
		}
		if siguiente == nil {
			rutaHija := filepath.Join(actual.Ruta, parte)
			siguiente = &nodoArbolUI{
				Origen:    modelo.OrigenLocal,
				Ruta:      rutaHija,
				Nombre:    parte,
				Expandido: false,
				Cargado:   false,
			}
			actual.Hijos = append(actual.Hijos, siguiente)
			sort.SliceStable(actual.Hijos, func(i, j int) bool {
				return compararTextoUI(actual.Hijos[i].Nombre, actual.Hijos[j].Nombre)
			})
		}

		siguiente.Expandido = true
		actual = siguiente
	}

	if actual != nil {
		actual.Expandido = true
		a.asegurarHijosNodo(actual)
	}

	return nil
}

func (a *Aplicacion) sincronizarArbolYandexConRuta(ruta string) error {
	if a.raizArbolYandex == nil || ruta == "" {
		return nil
	}

	ruta = normalizarRutaYandexUI(ruta)
	actual := a.raizArbolYandex
	actual.Expandido = true
	if err := a.cargarHijosNodoYandexSincrono(actual); err != nil {
		return err
	}

	if ruta == actual.Ruta {
		return nil
	}

	relativa := strings.TrimPrefix(ruta, "disk:/")
	relativa = strings.TrimPrefix(relativa, "/")
	partes := strings.Split(relativa, "/")
	for _, parte := range partes {
		parte = strings.TrimSpace(parte)
		if parte == "" || parte == "." {
			continue
		}

		if !actual.Cargado {
			if err := a.cargarHijosNodoYandexSincrono(actual); err != nil {
				return err
			}
		}

		var siguiente *nodoArbolUI
		for _, hijo := range actual.Hijos {
			if strings.EqualFold(strings.TrimSpace(hijo.Nombre), parte) {
				siguiente = hijo
				break
			}
		}
		if siguiente == nil {
			rutaHija := normalizarRutaYandexUI(actual.Ruta + "/" + parte)
			siguiente = &nodoArbolUI{
				Origen:    modelo.OrigenYandex,
				Ruta:      rutaHija,
				Nombre:    parte,
				Expandido: false,
				Cargado:   false,
			}
			actual.Hijos = append(actual.Hijos, siguiente)
			sort.SliceStable(actual.Hijos, func(i, j int) bool {
				return compararTextoUI(actual.Hijos[i].Nombre, actual.Hijos[j].Nombre)
			})
		}

		siguiente.Expandido = true
		actual = siguiente
	}

	if actual != nil {
		actual.Expandido = true
		a.asegurarHijosNodo(actual)
	}

	return nil
}

func (a *Aplicacion) cargarHijosNodoYandexSincrono(nodo *nodoArbolUI) error {
	if nodo == nil {
		return nil
	}
	subdirectorios, err := a.listarDirectoriosYandex(context.Background(), nodo.Ruta)
	if err != nil {
		return err
	}
	a.fusionarHijosNodoYandex(nodo, subdirectorios)
	sort.SliceStable(nodo.Hijos, func(i, j int) bool {
		return compararTextoUI(nodo.Hijos[i].Nombre, nodo.Hijos[j].Nombre)
	})
	nodo.Cargado = true
	nodo.Cargando = false
	return nil
}
