package ui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"destrellas-dam/internal/modelo"
)

func TestSugerenciasAsociacionesTextoOmiteValoresYaExistentes(t *testing.T) {
	t.Parallel()

	asociaciones := []modelo.AsociacionTexto{
		{
			ID:         1,
			Originales: []string{"cadenaTexto1"},
			Sugeridas:  []string{"Texto A", "Texto A2"},
		},
	}

	sugeridas := sugerenciasAsociacionesTexto("archivo-cadenaTexto1-final.jpg", []string{"Texto A"}, asociaciones)
	if len(sugeridas) != 1 || sugeridas[0] != "Texto A2" {
		t.Fatalf("sugerencias inesperadas: %+v", sugeridas)
	}
}

func TestSincronizarEditoresMetadatosSugierePalabrasFaltantesAunqueYaExistanOtras(t *testing.T) {
	t.Parallel()

	app := &Aplicacion{
		asociacionesTexto: []modelo.AsociacionTexto{
			{
				ID:         1,
				Originales: []string{"vacaciones"},
				Sugeridas:  []string{"Playa", "Familia"},
			},
		},
	}
	archivo := modelo.Archivo{
		Nombre: "2026-vacaciones-en-merida.jpg",
		Metadatos: modelo.MetadatosArchivo{
			PalabrasClave: []string{"Recuerdo", "Playa"},
			Sujetos:       []string{"Recuerdo", "Playa"},
		},
	}

	app.sincronizarEditoresMetadatos(archivo)

	if len(app.formularioMetadatos.PalabrasSugeridas) != 1 || app.formularioMetadatos.PalabrasSugeridas[0] != "Familia" {
		t.Fatalf("las palabras sugeridas deberían contener únicamente la faltante: %+v", app.formularioMetadatos.PalabrasSugeridas)
	}
	if !strings.Contains(app.editorPalabras.Text(), "Familia") {
		t.Fatalf("la palabra sugerida debería agregarse al editor: %q", app.editorPalabras.Text())
	}
}

func TestFiltrarAsociacionesTextoBuscaEnElCampoSeleccionado(t *testing.T) {
	t.Parallel()

	asociaciones := []modelo.AsociacionTexto{
		{
			ID:         1,
			Originales: []string{"viaje", "vacaciones"},
			Sugeridas:  []string{"Playa"},
		},
		{
			ID:         2,
			Originales: []string{"cumple"},
			Sugeridas:  []string{"Familia", "Fiesta"},
		},
	}

	filtradas := filtrarAsociacionesTexto(asociaciones, "fiesta", filtroAsociacionTextoSugeridas)
	if len(filtradas) != 1 || filtradas[0].ID != 2 {
		t.Fatalf("el filtro por sugeridas no devolvió el grupo esperado: %+v", filtradas)
	}

	filtradas = filtrarAsociacionesTexto(asociaciones, "vacaciones", filtroAsociacionTextoOriginales)
	if len(filtradas) != 1 || filtradas[0].ID != 1 {
		t.Fatalf("el filtro por originales no devolvió el grupo esperado: %+v", filtradas)
	}
}

func TestLeerAsociacionesTextoCSVLeeJSONYEliminaDuplicados(t *testing.T) {
	t.Parallel()

	ruta := filepath.Join(t.TempDir(), "asociaciones.csv")
	contenido := "\uFEFFID,\"Usuarios / alias (JSON)\",\"Nombres sugeridos (JSON)\"\n" +
		"1,\"[\"\"Alias\"\",\"\"alias\"\"]\",\"[\"\"Nombre\"\",\"\"Nombre\"\"]\"\n"
	if err := os.WriteFile(ruta, []byte(contenido), 0600); err != nil {
		t.Fatalf("no se pudo crear el CSV de prueba: %v", err)
	}

	asociaciones, err := leerAsociacionesTextoCSV(ruta)
	if err != nil {
		t.Fatalf("no se pudo leer el CSV de prueba: %v", err)
	}
	if len(asociaciones) != 1 {
		t.Fatalf("se esperaba una asociación, se obtuvieron %d", len(asociaciones))
	}
	if len(asociaciones[0].Originales) != 1 || asociaciones[0].Originales[0] != "Alias" {
		t.Fatalf("originales inesperadas: %+v", asociaciones[0].Originales)
	}
	if len(asociaciones[0].Sugeridas) != 1 || asociaciones[0].Sugeridas[0] != "Nombre" {
		t.Fatalf("sugeridas inesperadas: %+v", asociaciones[0].Sugeridas)
	}
}
