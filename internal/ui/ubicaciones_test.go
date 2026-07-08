package ui

import (
	"testing"

	"destrellas-dam/internal/modelo"
)

func TestFiltrarUbicacionesGuardadasRespetaConsultaYExclusion(t *testing.T) {
	t.Parallel()

	ubicaciones := []modelo.UbicacionGuardada{
		{Nombre: "The Palace Company"},
		{Nombre: "Palacio Nacional"},
		{Nombre: "Palace Studio"},
	}

	filtradas := filtrarUbicacionesGuardadas(ubicaciones, "palace", "Palace Studio", 10)
	if len(filtradas) != 1 {
		t.Fatalf("se esperaba una única coincidencia filtrada, se obtuvieron %d", len(filtradas))
	}
	if filtradas[0].Nombre != "The Palace Company" {
		t.Fatalf("coincidencia inesperada: %q", filtradas[0].Nombre)
	}
}

func TestAplicarUbicacionGuardadaAlFormularioCopiaCoordenadasYDireccion(t *testing.T) {
	t.Parallel()

	app := &Aplicacion{
		tieneArchivoActivo: true,
		archivoActivo:      modelo.Archivo{},
	}
	ubicacion := modelo.UbicacionGuardada{
		Nombre: "The Palace Company",
		Coordenadas: &modelo.Coordenadas{
			Latitud:  20.62212012,
			Longitud: -87.07333123,
		},
		Ciudad: "Playa del Carmen",
		Estado: "Quintana Roo",
		Pais:   "México",
	}

	app.aplicarUbicacionGuardadaAlFormulario(ubicacion)

	if app.editorUbicacion.Text() != "The Palace Company" {
		t.Fatalf("nombre de ubicación inesperado: %q", app.editorUbicacion.Text())
	}
	if app.editorGPSLatitud.Text() != "20.62212012" {
		t.Fatalf("latitud inesperada: %q", app.editorGPSLatitud.Text())
	}
	if app.editorGPSLongitud.Text() != "-87.07333123" {
		t.Fatalf("longitud inesperada: %q", app.editorGPSLongitud.Text())
	}
	if app.archivoActivo.Metadatos.Ciudad != "Playa del Carmen" {
		t.Fatalf("ciudad inesperada: %q", app.archivoActivo.Metadatos.Ciudad)
	}
	if app.archivoActivo.Metadatos.Estado != "Quintana Roo" {
		t.Fatalf("estado inesperado: %q", app.archivoActivo.Metadatos.Estado)
	}
	if app.archivoActivo.Metadatos.Pais != "México" {
		t.Fatalf("país inesperado: %q", app.archivoActivo.Metadatos.Pais)
	}
	if app.archivoActivo.Metadatos.Coordenadas == nil {
		t.Fatal("se esperaban coordenadas copiadas al archivo activo")
	}
}
