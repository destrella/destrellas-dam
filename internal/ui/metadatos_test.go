package ui

import (
	"testing"
	"time"

	"gioui.org/layout"

	"destrellas-dam/internal/configuracion"
	"destrellas-dam/internal/modelo"
)

func TestNormalizarFechaHoraEditadaAceptaFormatosCompatibles(t *testing.T) {
	t.Parallel()

	fecha, hora, zona, err := normalizarFechaHoraEditada("2026-07-05", "09:15", "+0530")
	if err != nil {
		t.Fatalf("normalizarFechaHoraEditada devolvió error: %v", err)
	}
	if fecha != "2026-07-05" {
		t.Fatalf("fecha inesperada: %q", fecha)
	}
	if hora != "09:15:00" {
		t.Fatalf("hora inesperada: %q", hora)
	}
	if zona != "+05:30" {
		t.Fatalf("zona inesperada: %q", zona)
	}
}

func TestNormalizarFechaHoraEditadaRechazaZonaInvalida(t *testing.T) {
	t.Parallel()

	if _, _, _, err := normalizarFechaHoraEditada("2026-07-05", "09:15", "+14:30"); err == nil {
		t.Fatal("se esperaba error para zona horaria inválida")
	}
	if _, _, _, err := normalizarFechaHoraEditada("2026-07-05", "25:15", "Z"); err == nil {
		t.Fatal("se esperaba error para hora inválida")
	}
}

func TestArchivoTieneFechaYHoraArchivables(t *testing.T) {
	t.Parallel()

	archivoValido := modelo.Archivo{
		Metadatos: modelo.MetadatosArchivo{
			Fecha: "2026-07-05",
			Hora:  "09:15:00",
		},
	}
	if !archivoTieneFechaYHoraArchivables(archivoValido) {
		t.Fatal("el archivo con fecha y hora válidas debería poder archivarse")
	}

	archivoInvalido := modelo.Archivo{
		Metadatos: modelo.MetadatosArchivo{
			Fecha: "2026-07-05",
		},
	}
	if archivoTieneFechaYHoraArchivables(archivoInvalido) {
		t.Fatal("el archivo sin hora no debería poder archivarse")
	}
}

func TestCampoCoincideSugerenciaRequiereSugerenciaActiva(t *testing.T) {
	t.Parallel()

	app := &Aplicacion{}
	if app.campoCoincideSugerencia("2026-07-05", "2026-07-05", false) {
		t.Fatal("no debería marcarse como sugerido si la sugerencia no está activa")
	}
	if !app.campoCoincideSugerencia("2026-07-05", "2026-07-05", true) {
		t.Fatal("debería marcarse como sugerido cuando coincide y la sugerencia está activa")
	}
}

func TestFusionarOpcionesLateralesIncluyeExtrasSinDuplicados(t *testing.T) {
	t.Parallel()

	opciones := fusionarOpcionesLaterales(
		[]string{"Familia", "Viaje", "Cancún"},
		[]string{"Nueva etiqueta", "familia", "  Nuevo lugar  "},
	)

	esperadas := []string{"Nueva etiqueta", "Nuevo lugar", "Familia", "Viaje", "Cancún"}
	if len(opciones) != len(esperadas) {
		t.Fatalf("cantidad inesperada de opciones: %d", len(opciones))
	}
	for indice, esperada := range esperadas {
		if opciones[indice].Clave != esperada || opciones[indice].Etiqueta != esperada {
			t.Fatalf("opción %d inesperada: %+v", indice, opciones[indice])
		}
	}
}

func TestSincronizarEditoresMetadatosNoMarcaSugerenciasSiElArchivoYaTieneValores(t *testing.T) {
	t.Parallel()

	app := &Aplicacion{}
	archivo := modelo.Archivo{
		Nombre: "20260705 091500.jpg",
		Indicadores: modelo.IndicadoresArchivo{
			TieneIA: true,
		},
		Metadatos: modelo.MetadatosArchivo{
			Fecha:       "2026-07-05",
			Hora:        "09:15:00",
			ZonaHoraria: "+08:00",
			Copyright:   "© 2026 Persona",
			Make:        "Red social",
			Modelo:      "modelo_existente.safetensors",
			Software:    "Comfy",
			WhereFroms:  []string{"https://instagram.com/p/demo"},
			Regiones: []modelo.RegionEtiquetada{
				{Nombre: "Persona"},
			},
			Extras: map[string][]string{
				"Parameters": {
					"prompt de prueba\nNegative prompt: nada\nSteps: 20, Model: modelo_existente.safetensors",
				},
			},
		},
	}

	app.sincronizarEditoresMetadatos(archivo)

	if app.formularioMetadatos.FechaSugeridaActiva {
		t.Fatal("la fecha no debería marcarse como sugerida si ya existe en el archivo")
	}
	if app.formularioMetadatos.HoraSugeridaActiva {
		t.Fatal("la hora no debería marcarse como sugerida si ya existe en el archivo")
	}
	if app.formularioMetadatos.CopyrightSugeridoActivo {
		t.Fatal("el copyright no debería marcarse como sugerido si ya existe en el archivo")
	}
	if app.formularioMetadatos.MakeSugeridoActivo {
		t.Fatal("make no debería marcarse como sugerido si ya existe en el archivo")
	}
	if app.formularioMetadatos.ModeloSugeridoActivo {
		t.Fatal("model no debería marcarse como sugerido si ya existe en el archivo")
	}
}

func TestArchivoDebeVerificarseConSistemaUnaVezPorRevision(t *testing.T) {
	t.Parallel()

	app := &Aplicacion{
		metadatosPendientes:  make(map[string]bool),
		metadatosVerificados: make(map[string]int64),
	}
	archivo := modelo.Archivo{
		Ruta:       "/tmp/ejemplo.jpg",
		Tipo:       modelo.TipoImagen,
		Modificado: time.Unix(1_700_000_000, 0),
	}

	if !app.archivoDebeVerificarseConSistema(archivo) {
		t.Fatal("el archivo multimedia debería verificarse si aún no se ha contrastado con el sistema")
	}

	app.marcarArchivoVerificadoConSistema(archivo)
	if app.archivoDebeVerificarseConSistema(archivo) {
		t.Fatal("el archivo no debería volver a verificarse si su revisión no cambió")
	}

	archivo.Modificado = archivo.Modificado.Add(time.Second)
	if !app.archivoDebeVerificarseConSistema(archivo) {
		t.Fatal("el archivo debería volver a verificarse cuando cambia su fecha de modificación")
	}
}

func TestPrepararEstadoTrasAccionArchivoVuelveAExploradorDesdeVisor(t *testing.T) {
	t.Parallel()

	app := &Aplicacion{
		vistaActual:        vistaElementoUnico,
		tieneArchivoActivo: true,
		edicionRegiones: estadoEdicionRegiones{
			Ruta:         "/tmp/imagen.jpg",
			RegionesBase: []modelo.RegionEtiquetada{{Nombre: "Persona", X: 0.1, Y: 0.2, Ancho: 0.3, Alto: 0.4}},
		},
		edicionRecorte: estadoEdicionRecorte{
			Ruta:           "/tmp/imagen.jpg",
			TieneSeleccion: true,
			Seleccion:      modelo.RegionEtiquetada{X: 0.1, Y: 0.1, Ancho: 0.8, Alto: 0.8},
		},
		reproductorVideo: estadoReproductorVideo{
			Ruta:          "/tmp/video.mov",
			Reproduciendo: true,
		},
	}

	app.prepararEstadoTrasAccionArchivo()

	if app.vistaActual != vistaPrincipal {
		t.Fatalf("la vista debería regresar a explorador, se obtuvo %q", app.vistaActual)
	}
	if app.tieneArchivoActivo {
		t.Fatal("no debería quedar archivo activo tras la acción")
	}
	if app.edicionRegiones.Ruta != "" {
		t.Fatal("la edición de regiones debería limpiarse")
	}
	if app.edicionRecorte.Ruta != "" {
		t.Fatal("la edición de recorte debería limpiarse")
	}
	if app.reproductorVideo.Ruta != "" || app.reproductorVideo.Reproduciendo {
		t.Fatal("el reproductor de video debería reiniciarse")
	}
}

func TestDescartarArchivoActivoLimpiaDetalleSinCambiarLaVista(t *testing.T) {
	t.Parallel()

	app := &Aplicacion{
		vistaActual:        vistaElementoUnico,
		tieneArchivoActivo: true,
		archivoActivo:      modelo.Archivo{Ruta: "/tmp/archivo.jpg"},
		edicionRegiones: estadoEdicionRegiones{
			Ruta:         "/tmp/archivo.jpg",
			RegionesBase: []modelo.RegionEtiquetada{{Nombre: "Rostro", X: 0.1, Y: 0.1, Ancho: 0.2, Alto: 0.2}},
		},
		edicionRecorte: estadoEdicionRecorte{
			Ruta:           "/tmp/archivo.jpg",
			TieneSeleccion: true,
			Seleccion:      modelo.RegionEtiquetada{X: 0.2, Y: 0.2, Ancho: 0.5, Alto: 0.5},
		},
		reproductorVideo: estadoReproductorVideo{
			Ruta:          "/tmp/video.mov",
			Reproduciendo: true,
		},
	}

	app.descartarArchivoActivo()

	if app.vistaActual != vistaElementoUnico {
		t.Fatalf("la vista actual no debería cambiar, se obtuvo %q", app.vistaActual)
	}
	if app.tieneArchivoActivo {
		t.Fatal("no debería quedar archivo activo tras limpiar el detalle")
	}
	if app.archivoActivo.Ruta != "" {
		t.Fatalf("el archivo activo debería reiniciarse, se obtuvo %q", app.archivoActivo.Ruta)
	}
	if app.edicionRegiones.Ruta != "" {
		t.Fatal("la edición de regiones debería limpiarse")
	}
	if app.edicionRecorte.Ruta != "" {
		t.Fatal("la edición de recorte debería limpiarse")
	}
	if app.reproductorVideo.Ruta != "" || app.reproductorVideo.Reproduciendo {
		t.Fatal("el reproductor de video debería reiniciarse")
	}
}

func TestCalcularObjetivoRestauracionListadoRespetaScrollYPagina(t *testing.T) {
	t.Parallel()

	app := &Aplicacion{
		configuracion: configuracion.Configuracion{
			TamanoPaginaLocal: 80,
		},
		elementos: make([]modelo.Archivo, 120),
	}

	objetivo := app.calcularObjetivoRestauracionListado(layout.Position{First: 140})
	if objetivo != 220 {
		t.Fatalf("objetivo inesperado: %d", objetivo)
	}

	app.elementos = make([]modelo.Archivo, 260)
	objetivo = app.calcularObjetivoRestauracionListado(layout.Position{First: 140})
	if objetivo != 260 {
		t.Fatalf("debería priorizar el número de elementos ya cargados, se obtuvo %d", objetivo)
	}
}
