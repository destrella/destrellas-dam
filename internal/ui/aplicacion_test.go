package ui

import (
	"testing"

	"gioui.org/layout"
	"gioui.org/widget"

	"destrellas-dam/internal/modelo"
)

func TestDrenarActualizacionesLimitaElTrabajoPorFrame(t *testing.T) {
	t.Parallel()

	app := &Aplicacion{
		actualizaciones: make(chan func(), maxActualizacionesPorFrame+12),
	}

	procesadas := 0
	for indice := 0; indice < maxActualizacionesPorFrame+12; indice++ {
		app.actualizaciones <- func() {
			procesadas++
		}
	}

	app.drenarActualizaciones()

	if procesadas != maxActualizacionesPorFrame {
		t.Fatalf("se esperaban %d actualizaciones procesadas en un frame, se obtuvieron %d", maxActualizacionesPorFrame, procesadas)
	}
	if restantes := len(app.actualizaciones); restantes != 12 {
		t.Fatalf("deberían quedar 12 actualizaciones pendientes, se obtuvieron %d", restantes)
	}
}

func TestEstablecerListadoFiltradoDescartaDetalleAlCambiarEtiqueta(t *testing.T) {
	t.Parallel()

	app := &Aplicacion{
		origenListado:      origenListadoEtiqueta,
		claveListadoActual: "Una etiqueta",
		tieneArchivoActivo: true,
		archivoActivo:      modelo.Archivo{Ruta: "/tmp/activo.jpg"},
	}

	app.establecerListadoFiltrado(origenListadoEtiqueta, "Otra etiqueta")

	if app.tieneArchivoActivo {
		t.Fatal("el detalle debería limpiarse al cambiar de etiqueta")
	}
	if app.archivoActivo.Ruta != "" {
		t.Fatalf("el archivo activo debería quedar vacío, se obtuvo %q", app.archivoActivo.Ruta)
	}
}

func TestEstablecerListadoFiltradoConservaDetalleSiSeRepiteElMismoLugar(t *testing.T) {
	t.Parallel()

	app := &Aplicacion{
		origenListado:      origenListadoUbicacion,
		claveListadoActual: "Mérida",
		tieneArchivoActivo: true,
		archivoActivo:      modelo.Archivo{Ruta: "/tmp/activo.jpg"},
	}

	app.establecerListadoFiltrado(origenListadoUbicacion, " mérida ")

	if !app.tieneArchivoActivo || app.archivoActivo.Ruta != "/tmp/activo.jpg" {
		t.Fatal("repetir el mismo lugar no debería limpiar el detalle")
	}
}

func TestRestaurarAnclaScrollListadoSigueLaRutaTrasEliminarElementoAnterior(t *testing.T) {
	t.Parallel()

	app := &Aplicacion{
		filtros: modelo.FiltrosListado{VistaGaleria: false},
		elementos: []modelo.Archivo{
			{Ruta: "uno"},
			{Ruta: "dos"},
			{Ruta: "tres"},
			{Ruta: "cuatro"},
			{Ruta: "cinco"},
		},
		listaCentro: widget.List{},
	}
	app.listaCentro.Position = layout.Position{First: 3, Offset: 9}
	app.prepararAnclaScrollListado(app.listaCentro.Position)

	app.elementos = []modelo.Archivo{
		{Ruta: "dos"},
		{Ruta: "tres"},
		{Ruta: "cuatro"},
		{Ruta: "cinco"},
	}
	if !app.restaurarAnclaScrollListado() {
		t.Fatal("se esperaba restaurar el ancla por ruta")
	}
	if app.listaCentro.Position.First != 2 {
		t.Fatalf("posición restaurada inesperada: %d", app.listaCentro.Position.First)
	}
	if app.listaCentro.Position.Offset != 9 {
		t.Fatalf("offset restaurado inesperado: %d", app.listaCentro.Position.Offset)
	}
}

func TestRetirarArchivoInactivoDelListadoPorEtiqueta(t *testing.T) {
	t.Parallel()

	app := &Aplicacion{
		origenListado:      origenListadoEtiqueta,
		claveListadoActual: "Familia",
		elementos: []modelo.Archivo{
			{Ruta: "/tmp/activo.jpg", Metadatos: modelo.MetadatosArchivo{Sujetos: []string{"Familia"}}},
			{Ruta: "/tmp/otro.jpg", Metadatos: modelo.MetadatosArchivo{Sujetos: []string{"Familia"}}},
		},
		listaCentro: widget.List{},
	}

	archivoActualizado := modelo.Archivo{
		Ruta:      "/tmp/activo.jpg",
		Metadatos: modelo.MetadatosArchivo{Sujetos: []string{"Viaje"}},
	}
	if !app.retirarArchivoInactivoDelListado(archivoActualizado) {
		t.Fatal("se esperaba retirar el archivo que dejó de coincidir con la etiqueta")
	}
	if len(app.elementos) != 1 || app.elementos[0].Ruta != "/tmp/otro.jpg" {
		t.Fatalf("el listado no quedó depurado: %+v", app.elementos)
	}
}

func TestRestaurarAnclaScrollGaleriaConservaLaFilaDelArchivo(t *testing.T) {
	t.Parallel()

	app := &Aplicacion{
		filtros:               modelo.FiltrosListado{VistaGaleria: true},
		columnasGaleriaActual: 3,
		elementos: []modelo.Archivo{
			{Ruta: "uno"},
			{Ruta: "dos"},
			{Ruta: "tres"},
			{Ruta: "cuatro"},
			{Ruta: "cinco"},
			{Ruta: "seis"},
			{Ruta: "siete"},
		},
		listaCentro: widget.List{},
	}
	app.listaCentro.Position = layout.Position{First: 1, Offset: 4}
	app.prepararAnclaScrollListado(app.listaCentro.Position)

	app.elementos = []modelo.Archivo{
		{Ruta: "dos"},
		{Ruta: "tres"},
		{Ruta: "cuatro"},
		{Ruta: "cinco"},
		{Ruta: "seis"},
		{Ruta: "siete"},
	}
	if !app.restaurarAnclaScrollListado() {
		t.Fatal("se esperaba restaurar el ancla de la galería")
	}
	if app.listaCentro.Position.First != 0 {
		t.Fatalf("fila restaurada inesperada: %d", app.listaCentro.Position.First)
	}
}
