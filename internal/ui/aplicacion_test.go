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
