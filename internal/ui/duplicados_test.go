package ui

import (
	"testing"
	"time"

	"destrellas-dam/internal/modelo"
)

func TestGruposDuplicadosVisiblesFiltraSoloMultimedia(t *testing.T) {
	t.Parallel()

	app := &Aplicacion{
		gruposDuplicados: []modelo.GrupoDuplicados{
			{
				Clave: "texto",
				Elementos: []modelo.Archivo{
					{Ruta: "/tmp/nota.txt", Tipo: modelo.TipoOtro},
					{Ruta: "/tmp/copia-nota.txt", Tipo: modelo.TipoOtro},
				},
			},
			{
				Clave: "imagen",
				Elementos: []modelo.Archivo{
					{Ruta: "/tmp/foto.webp", Tipo: modelo.TipoImagen},
					{Ruta: "/tmp/copia-foto.webp", Tipo: modelo.TipoImagen},
				},
			},
		},
	}
	app.soloDuplicadosMultimedia.Value = true

	grupos := app.gruposDuplicadosVisibles()
	if len(grupos) != 1 {
		t.Fatalf("se esperaba un único grupo multimedia visible, se obtuvieron %d", len(grupos))
	}
	if grupos[0].Clave != "imagen" {
		t.Fatalf("grupo inesperado tras filtrar: %q", grupos[0].Clave)
	}
}

func TestSincronizarPreviewDuplicadosConGruposDescartaRutaAusente(t *testing.T) {
	t.Parallel()

	app := &Aplicacion{
		rutaPreviewDuplicados: "/tmp/inexistente.jpg",
	}

	app.sincronizarPreviewDuplicadosConGrupos([]modelo.GrupoDuplicados{
		{
			Clave: "otro",
			Elementos: []modelo.Archivo{
				{Ruta: "/tmp/otra.jpg", Tipo: modelo.TipoImagen},
			},
		},
	})

	if app.rutaPreviewDuplicados != "" {
		t.Fatalf("la ruta de preview debería limpiarse cuando ya no existe en los grupos, se obtuvo %q", app.rutaPreviewDuplicados)
	}
}

func TestArchivoPreviewDuplicadosPrefiereArchivoActivoEnriquecido(t *testing.T) {
	t.Parallel()

	grupo := modelo.GrupoDuplicados{
		Clave: "imagen",
		Elementos: []modelo.Archivo{
			{
				Ruta: "/tmp/foto.jpg",
				Tipo: modelo.TipoImagen,
			},
		},
	}
	app := &Aplicacion{
		rutaPreviewDuplicados: "/tmp/foto.jpg",
		tieneArchivoActivo:    true,
		archivoActivo: modelo.Archivo{
			Ruta:  "/tmp/foto.jpg",
			Tipo:  modelo.TipoImagen,
			Ancho: 1920,
			Alto:  1080,
		},
	}

	archivo, ok := app.archivoPreviewDuplicados(grupo)
	if !ok {
		t.Fatal("se esperaba recuperar el archivo seleccionado para preview")
	}
	if archivo.Ancho != 1920 || archivo.Alto != 1080 {
		t.Fatalf("se esperaba usar el archivo activo enriquecido, se obtuvo %dx%d", archivo.Ancho, archivo.Alto)
	}
}

func TestAlternarColapsoGrupoDuplicadoConservaEstadoPorClave(t *testing.T) {
	t.Parallel()

	grupo := modelo.GrupoDuplicados{
		Clave: "grupo-prueba",
		Tipo:  modelo.CoincidenciaExacta,
	}
	app := &Aplicacion{
		gruposDuplicadosContraidos: make(map[string]bool),
	}

	if app.grupoDuplicadoContraido(grupo) {
		t.Fatal("el grupo no debería iniciar colapsado")
	}

	app.alternarColapsoGrupoDuplicado(grupo)
	if !app.grupoDuplicadoContraido(grupo) {
		t.Fatal("el grupo debería quedar colapsado tras alternarlo")
	}

	app.alternarColapsoGrupoDuplicado(grupo)
	if app.grupoDuplicadoContraido(grupo) {
		t.Fatal("el grupo debería volver a expandirse al alternarlo de nuevo")
	}
}

func TestAnchoPreviewGrupoDuplicadoUsaUnCuartoDelAncho(t *testing.T) {
	t.Parallel()

	if ancho := anchoPreviewGrupoDuplicado(1200); ancho != 300 {
		t.Fatalf("ancho de preview inesperado: %d", ancho)
	}
}

func TestResumenElementoDuplicadoIncluyeDimensionesEnDHashImagen(t *testing.T) {
	t.Parallel()

	grupo := modelo.GrupoDuplicados{Tipo: modelo.CoincidenciaParcialImagen}
	instante := time.Date(2026, time.July, 11, 13, 14, 15, 0, time.Local)
	elemento := modelo.Archivo{
		Origen:     modelo.OrigenLocal,
		Ancho:      1536,
		Alto:       1024,
		Tamano:     4096,
		Modificado: instante,
	}

	resumen := resumenElementoDuplicado(grupo, elemento)
	esperado := "local | 1536x1024 px | 4.1 kB | 2026-07-11 13:14:15"
	if resumen != esperado {
		t.Fatalf("resumen inesperado:\nesperado: %q\nobtenido: %q", esperado, resumen)
	}
}

func TestResumenElementoDuplicadoIncluyeDuracionEnDHashVideo(t *testing.T) {
	t.Parallel()

	grupo := modelo.GrupoDuplicados{Tipo: modelo.CoincidenciaParcialVideo}
	instante := time.Date(2026, time.July, 11, 13, 14, 15, 0, time.Local)
	elemento := modelo.Archivo{
		Origen:     modelo.OrigenLocal,
		Duracion:   95 * time.Second,
		Tamano:     4096,
		Modificado: instante,
	}

	resumen := resumenElementoDuplicado(grupo, elemento)
	esperado := "local | 01:35 | 4.1 kB | 2026-07-11 13:14:15"
	if resumen != esperado {
		t.Fatalf("resumen inesperado:\nesperado: %q\nobtenido: %q", esperado, resumen)
	}
}
