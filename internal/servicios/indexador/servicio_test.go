package indexador

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	almacensqlite "destrellas-dam/internal/almacen/sqlite"
	"destrellas-dam/internal/modelo"
)

func TestDescubrirIgnoraArchivosVaciosYEliminaRegistrosPrevios(t *testing.T) {
	t.Parallel()

	directorio := t.TempDir()
	rutaVacia := filepath.Join(directorio, "vacia.jpg")
	rutaLlena := filepath.Join(directorio, "llena.jpg")

	if err := os.WriteFile(rutaVacia, nil, 0o644); err != nil {
		t.Fatalf("no se pudo crear el archivo vacío: %v", err)
	}
	if err := os.WriteFile(rutaLlena, []byte("contenido"), 0o644); err != nil {
		t.Fatalf("no se pudo crear el archivo con contenido: %v", err)
	}

	repo, err := almacensqlite.Nuevo(filepath.Join(t.TempDir(), "catalogo.sqlite"))
	if err != nil {
		t.Fatalf("no se pudo crear el almacén sqlite: %v", err)
	}
	defer repo.Cerrar()

	if err := repo.GuardarArchivo(context.Background(), modelo.Archivo{
		Origen:     modelo.OrigenLocal,
		Ruta:       rutaVacia,
		RutaPadre:  directorio,
		Nombre:     filepath.Base(rutaVacia),
		Tamano:     128,
		Modificado: time.Unix(100, 0),
		Tipo:       modelo.TipoImagen,
	}); err != nil {
		t.Fatalf("no se pudo preparar el registro previo del archivo vacío: %v", err)
	}

	servicio := NuevoServicio(repo, nil, 1)
	for range servicio.Descubrir(context.Background(), directorio, OpcionesDescubrimiento{
		SoloMultimedia:        true,
		IgnorarArchivosVacios: true,
	}) {
	}

	if _, err := repo.ObtenerArchivoPorRuta(context.Background(), rutaVacia); err == nil {
		t.Fatal("el archivo vacío no debería seguir en la base tras el descubrimiento")
	}

	archivo, err := repo.ObtenerArchivoPorRuta(context.Background(), rutaLlena)
	if err != nil {
		t.Fatalf("el archivo con contenido debería haberse guardado: %v", err)
	}
	if archivo.Tamano <= 0 {
		t.Fatalf("se esperaba un tamaño mayor a cero para el archivo con contenido, se obtuvo %d", archivo.Tamano)
	}
}
