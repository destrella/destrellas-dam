package metadatos

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"destrellas-dam/internal/modelo"
)

func TestWebMSoportaPreviewYFotogramas(t *testing.T) {
	t.Parallel()

	servicio := NuevoServicio()
	if servicio.rutaFFmpeg == "" || servicio.rutaFFprobe == "" {
		t.Skip("ffmpeg/ffprobe no están disponibles")
	}

	rutaVideo := crearVideoWebMPrueba(t, servicio.rutaFFmpeg)
	ctx := context.Background()

	duracion, err := servicio.duracionVideo(ctx, rutaVideo)
	if err != nil {
		t.Fatalf("no se pudo calcular la duración WebM: %v", err)
	}
	if duracion <= 0 {
		t.Fatalf("la duración WebM debería ser positiva, se obtuvo %v", duracion)
	}

	preview, err := servicio.GenerarPreviewVideo(ctx, rutaVideo, 360, 0)
	if err != nil {
		t.Fatalf("no se pudo generar la previsualización WebM: %v", err)
	}
	if preview == nil {
		t.Fatal("se esperaba una previsualización para el archivo WebM")
	}
	if bounds := preview.Bounds(); bounds.Dx() < 1 || bounds.Dy() < 1 {
		t.Fatalf("la previsualización WebM quedó vacía: %v", bounds)
	}

	fotograma, err := servicio.GenerarFotogramaVideo(ctx, rutaVideo, 900*time.Millisecond, 960, 0)
	if err != nil {
		t.Fatalf("no se pudo generar un fotograma WebM para el visor: %v", err)
	}
	if fotograma == nil {
		t.Fatal("se esperaba un fotograma para el visor de un archivo WebM")
	}

	lote, err := servicio.GenerarLoteFotogramasVideo(ctx, rutaVideo, 400*time.Millisecond, 6, 4, 640, 0)
	if err != nil {
		t.Fatalf("no se pudo generar un lote de fotogramas WebM: %v", err)
	}
	if len(lote) == 0 {
		t.Fatal("se esperaba al menos un fotograma en el lote WebM")
	}
	if lote[0].Imagen == nil {
		t.Fatal("el primer fotograma del lote WebM no debería ser nil")
	}

	archivo, err := servicio.AnalizarArchivo(ctx, modelo.Archivo{
		Ruta: rutaVideo,
		Tipo: modelo.TipoVideo,
	})
	if err != nil {
		t.Fatalf("no se pudo analizar el WebM de prueba: %v", err)
	}
	if archivo.Duracion <= 0 {
		t.Fatalf("el análisis del WebM debería completar una duración positiva, se obtuvo %v", archivo.Duracion)
	}
}

func crearVideoWebMPrueba(t *testing.T, rutaFFmpeg string) string {
	t.Helper()

	directorioTemporal := t.TempDir()
	rutaVideo := filepath.Join(directorioTemporal, "prueba.webm")
	comando := exec.Command(
		rutaFFmpeg,
		"-hide_banner", "-loglevel", "error",
		"-f", "lavfi",
		"-i", "testsrc=size=320x180:rate=12",
		"-t", "2",
		"-an",
		"-c:v", "libvpx-vp9",
		"-pix_fmt", "yuv420p",
		"-y",
		rutaVideo,
	)
	if salida, err := comando.CombinedOutput(); err != nil {
		t.Fatalf("no se pudo generar el video WebM de prueba: %v: %s", err, string(salida))
	}
	if _, err := os.Stat(rutaVideo); err != nil {
		t.Fatalf("no se generó el archivo WebM de prueba: %v", err)
	}
	return rutaVideo
}
