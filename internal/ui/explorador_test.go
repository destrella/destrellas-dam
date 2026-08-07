package ui

import (
	"testing"

	"destrellas-dam/internal/modelo"
)

func TestSeleccionElementoConShiftSeleccionaIntervaloVisible(t *testing.T) {
	t.Parallel()

	app := &Aplicacion{
		elementos: []modelo.Archivo{
			{Ruta: "/tmp/a.jpg"},
			{Ruta: "/tmp/b.jpg"},
			{Ruta: "/tmp/c.jpg"},
			{Ruta: "/tmp/d.jpg"},
		},
		seleccionLote: make(map[string]bool),
	}

	app.seleccionarElementoConPosibleRango("/tmp/a.jpg", false)
	if app.anclaSeleccionLote != "/tmp/a.jpg" {
		t.Fatalf("se esperaba que la ancla inicial fuera /tmp/a.jpg, se obtuvo %q", app.anclaSeleccionLote)
	}

	app.seleccionarElementoConPosibleRango("/tmp/d.jpg", true)

	esperadas := []string{"/tmp/a.jpg", "/tmp/b.jpg", "/tmp/c.jpg", "/tmp/d.jpg"}
	if len(app.seleccionLote) != len(esperadas) {
		t.Fatalf("cantidad inesperada de seleccionados: %d", len(app.seleccionLote))
	}
	for _, ruta := range esperadas {
		if !app.seleccionLote[ruta] {
			t.Fatalf("la ruta %q debería formar parte del rango seleccionado", ruta)
		}
	}
	if app.anclaSeleccionLote != "" {
		t.Fatalf("la ancla debería liberarse tras construir el rango, se obtuvo %q", app.anclaSeleccionLote)
	}
}

func TestSeleccionElementoConShiftMantieneSeleccionLibreTrasRango(t *testing.T) {
	t.Parallel()

	app := &Aplicacion{
		elementos: []modelo.Archivo{
			{Ruta: "/tmp/a.jpg"},
			{Ruta: "/tmp/b.jpg"},
			{Ruta: "/tmp/c.jpg"},
			{Ruta: "/tmp/d.jpg"},
			{Ruta: "/tmp/e.jpg"},
			{Ruta: "/tmp/f.jpg"},
		},
		seleccionLote: make(map[string]bool),
	}

	app.seleccionarElementoConPosibleRango("/tmp/b.jpg", false)
	app.seleccionarElementoConPosibleRango("/tmp/d.jpg", true)
	app.seleccionarElementoConPosibleRango("/tmp/f.jpg", false)

	esperadas := []string{"/tmp/b.jpg", "/tmp/c.jpg", "/tmp/d.jpg", "/tmp/f.jpg"}
	if len(app.seleccionLote) != len(esperadas) {
		t.Fatalf("cantidad inesperada de seleccionados: %d", len(app.seleccionLote))
	}
	for _, ruta := range esperadas {
		if !app.seleccionLote[ruta] {
			t.Fatalf("la ruta %q debería seguir seleccionada", ruta)
		}
	}
	if app.seleccionLote["/tmp/e.jpg"] {
		t.Fatal("la ruta /tmp/e.jpg no debería añadirse automáticamente tras completar el rango anterior")
	}
	if app.anclaSeleccionLote != "" {
		t.Fatalf("no debería quedar ancla activa con varias selecciones independientes, se obtuvo %q", app.anclaSeleccionLote)
	}
}

func TestSeleccionElementoSinShiftNoSeleccionaElIntervalo(t *testing.T) {
	t.Parallel()

	app := &Aplicacion{
		elementos: []modelo.Archivo{
			{Ruta: "/tmp/a.jpg"},
			{Ruta: "/tmp/b.jpg"},
			{Ruta: "/tmp/c.jpg"},
			{Ruta: "/tmp/d.jpg"},
		},
		seleccionLote: make(map[string]bool),
	}

	app.seleccionarElementoConPosibleRango("/tmp/a.jpg", false)
	app.seleccionarElementoConPosibleRango("/tmp/d.jpg", false)

	if len(app.seleccionLote) != 2 {
		t.Fatalf("se esperaban dos archivos seleccionados, se obtuvieron %d", len(app.seleccionLote))
	}
	if !app.seleccionLote["/tmp/a.jpg"] || !app.seleccionLote["/tmp/d.jpg"] {
		t.Fatal("los extremos seleccionados deberían conservarse")
	}
	if app.seleccionLote["/tmp/b.jpg"] || app.seleccionLote["/tmp/c.jpg"] {
		t.Fatal("los archivos intermedios no deberían seleccionarse sin Shift")
	}
}

func TestSeleccionarTodoYDeseleccionarTodoActualizaElEstado(t *testing.T) {
	t.Parallel()

	app := &Aplicacion{
		elementos: []modelo.Archivo{
			{Ruta: "/tmp/a.jpg"},
			{Ruta: "/tmp/b.jpg"},
			{Ruta: "/tmp/c.jpg"},
		},
		seleccionLote: make(map[string]bool),
	}

	app.seleccionarTodosElementosCargados()
	if len(app.seleccionLote) != len(app.elementos) {
		t.Fatalf("se esperaban %d elementos seleccionados, se obtuvieron %d", len(app.elementos), len(app.seleccionLote))
	}
	if app.anclaSeleccionLote != "" {
		t.Fatalf("no debería quedar ancla activa al seleccionar todo, se obtuvo %q", app.anclaSeleccionLote)
	}

	app.deseleccionarTodosElementos()
	if len(app.seleccionLote) != 0 {
		t.Fatalf("la selección debería quedar vacía, se obtuvieron %d elementos", len(app.seleccionLote))
	}
	if app.anclaSeleccionLote != "" {
		t.Fatalf("la ancla debería limpiarse al deseleccionar todo, se obtuvo %q", app.anclaSeleccionLote)
	}
}
