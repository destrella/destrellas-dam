package metadatos

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

// TestStreamingHTTPDeVideoConFFmpeg comprueba que el flujo actual de ffmpeg
// puede consumir un video remoto por HTTP, buscar una posición y leer audio.
// Simula el enlace temporal que entrega Yandex.Disk después de autenticarse.
func TestStreamingHTTPDeVideoConFFmpeg(t *testing.T) {
	servicio := NuevoServicio()
	if servicio.rutaFFmpeg == "" || servicio.rutaFFprobe == "" {
		t.Skip("ffmpeg/ffprobe no están disponibles")
	}

	rutaVideo := crearVideoMP4ConAudioPrueba(t, servicio.rutaFFmpeg)
	datosVideo, err := os.ReadFile(rutaVideo)
	if err != nil {
		t.Fatalf("no se pudo leer el video de prueba: %v", err)
	}

	servidor := nuevoServidorVideoRange(t, datosVideo)
	defer servidor.Close()

	ctx, cancelar := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancelar()
	servicio.EstablecerResolverFuenteVideo(func(_ context.Context, ruta string) (string, error) {
		if ruta != "disk:/media/Videos/prueba.mp4" {
			t.Fatalf("el resolvedor recibió una ruta inesperada: %q", ruta)
		}
		return servidor.URL, nil
	})
	rutaLogica := "disk:/media/Videos/prueba.mp4"

	duracion, err := servicio.duracionVideo(ctx, rutaLogica)
	if err != nil {
		t.Fatalf("ffprobe no pudo analizar el video remoto: %v", err)
	}
	if duracion < 7.5 || duracion > 8.5 {
		t.Fatalf("duración remota inesperada: %.3f segundos", duracion)
	}

	tieneAudio, err := servicio.tienePistaAudioVideo(ctx, rutaLogica)
	if err != nil {
		t.Fatalf("ffprobe no pudo detectar la pista de audio remota: %v", err)
	}
	if !tieneAudio {
		t.Fatal("el video remoto de prueba debería contener una pista de audio")
	}

	fotograma, err := servicio.GenerarFotogramaVideo(ctx, rutaLogica, 5*time.Second, 640, 0)
	if err != nil {
		t.Fatalf("ffmpeg no pudo buscar y extraer un frame remoto: %v", err)
	}
	if fotograma == nil || fotograma.Bounds().Dx() == 0 || fotograma.Bounds().Dy() == 0 {
		t.Fatal("el frame remoto extraído quedó vacío")
	}

	lote, err := servicio.GenerarLoteFotogramasVideo(ctx, rutaLogica, 5*time.Second, 30, 6, 640, 0)
	if err != nil {
		t.Fatalf("ffmpeg no pudo extraer un lote remoto: %v", err)
	}
	if len(lote) < 2 {
		t.Fatalf("se esperaban varios frames remotos, se obtuvieron %d", len(lote))
	}

	// ReproducirAudioVideo envía los samples a un dispositivo oto, que no es
	// estable en CI. Esta orden verifica la misma lectura remota de la pista.
	verificarLecturaAudioRemota(t, ctx, servicio.rutaFFmpeg, servidor.URL)

	servidor.assertoStreamingHTTP(t)
}

func TestResolverFuenteVideoReutilizaCacheLocalDeSesion(t *testing.T) {
	t.Parallel()

	servicio := NuevoServicio()
	var descargas int
	servicio.EstablecerResolverFuenteVideo(func(_ context.Context, ruta string) (string, error) {
		return "https://ejemplo.invalid/" + ruta, nil
	})
	servicio.EstablecerDescargadorFuenteVideo(t.TempDir(), func(_ context.Context, ruta string) (io.ReadCloser, error) {
		descargas++
		return io.NopCloser(strings.NewReader("video " + ruta)), nil
	})

	primera, err := servicio.resolverFuente(context.Background(), "disk:/media/prueba.mp4")
	if err != nil {
		t.Fatalf("no se pudo preparar la primera fuente: %v", err)
	}
	segunda, err := servicio.resolverFuente(context.Background(), "disk:/media/prueba.mp4")
	if err != nil {
		t.Fatalf("no se pudo preparar la segunda fuente: %v", err)
	}
	if descargas != 1 {
		t.Fatalf("se esperaba una sola descarga para la misma fuente, se obtuvieron %d", descargas)
	}
	if primera != segunda {
		t.Fatalf("la segunda solicitud debería reutilizar la caché, se obtuvieron %q y %q", primera, segunda)
	}
	contenido, err := os.ReadFile(primera)
	if err != nil {
		t.Fatalf("no se pudo leer el archivo en caché: %v", err)
	}
	if string(contenido) != "video disk:/media/prueba.mp4" {
		t.Fatalf("contenido inesperado de la caché: %q", contenido)
	}
}

func TestResolverFuenteVideoOmiteCacheParaFuenteRemotaGrande(t *testing.T) {
	t.Parallel()

	servicio := NuevoServicio()
	var descargas int
	servicio.EstablecerResolverFuenteVideo(func(_ context.Context, ruta string) (string, error) {
		return "https://ejemplo.invalid/" + ruta, nil
	})
	servicio.EstablecerDescargadorFuenteVideo(t.TempDir(), func(_ context.Context, _ string) (io.ReadCloser, error) {
		descargas++
		return io.NopCloser(strings.NewReader("no debería descargarse")), nil
	})

	const ruta = "disk:/media/grande.mp4"
	servicio.RegistrarTamanoFuenteVideo(ruta, limiteCacheFuenteVideo+1)
	fuente, err := servicio.resolverFuente(context.Background(), ruta)
	if err != nil {
		t.Fatalf("no se pudo resolver la fuente grande: %v", err)
	}
	if fuente != "https://ejemplo.invalid/"+ruta {
		t.Fatalf("debería conservar el streaming directo, se obtuvo %q", fuente)
	}
	if descargas != 0 {
		t.Fatalf("una fuente mayor al límite no debe iniciar la caché, se obtuvieron %d descargas", descargas)
	}
}

func TestTienePistaAudioVideoDetectaPistaLocal(t *testing.T) {
	servicio := NuevoServicio()
	if servicio.rutaFFmpeg == "" || servicio.rutaFFprobe == "" {
		t.Skip("ffmpeg/ffprobe no están disponibles")
	}

	rutaVideo := crearVideoMP4ConAudioPrueba(t, servicio.rutaFFmpeg)
	tieneAudio, err := servicio.TienePistaAudioVideo(context.Background(), rutaVideo)
	if err != nil {
		t.Fatalf("no se pudo detectar el audio del video local: %v", err)
	}
	if !tieneAudio {
		t.Fatal("el video local de prueba debería informar una pista de audio")
	}
}

type servidorVideoRange struct {
	*httptest.Server
	mu               sync.Mutex
	solicitudes      int
	solicitudesRange int
	bytesEntregados  int64
	maximoRespuesta  int64
	tamano           int64
}

func nuevoServidorVideoRange(t *testing.T, datos []byte) *servidorVideoRange {
	t.Helper()

	estado := &servidorVideoRange{tamano: int64(len(datos))}
	estado.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		estado.mu.Lock()
		estado.solicitudes++
		if r.Header.Get("Range") != "" {
			estado.solicitudesRange++
		}
		estado.mu.Unlock()

		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.Error(w, "método no permitido", http.StatusMethodNotAllowed)
			return
		}
		if r.Method == http.MethodHead {
			w.Header().Set("Content-Length", strconv.Itoa(len(datos)))
			w.Header().Set("Accept-Ranges", "bytes")
			return
		}

		peticion := r.Header.Get("Range")
		inicio, fin, parcial := rangoHTTP(peticion, int64(len(datos)))
		if parcial {
			w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", inicio, fin, len(datos)))
			w.Header().Set("Content-Length", strconv.FormatInt(fin-inicio+1, 10))
			w.Header().Set("Accept-Ranges", "bytes")
			w.WriteHeader(http.StatusPartialContent)
		} else {
			inicio = 0
			fin = int64(len(datos)) - 1
			w.Header().Set("Content-Length", strconv.Itoa(len(datos)))
			w.Header().Set("Accept-Ranges", "bytes")
		}

		cantidad := fin - inicio + 1
		estado.mu.Lock()
		estado.bytesEntregados += cantidad
		if cantidad > estado.maximoRespuesta {
			estado.maximoRespuesta = cantidad
		}
		estado.mu.Unlock()
		_, _ = w.Write(datos[inicio : fin+1])
	}))
	return estado
}

func rangoHTTP(valor string, tamano int64) (int64, int64, bool) {
	if !strings.HasPrefix(valor, "bytes=") || tamano <= 0 {
		return 0, tamano - 1, false
	}
	partes := strings.SplitN(strings.TrimPrefix(valor, "bytes="), "-", 2)
	if len(partes) != 2 {
		return 0, tamano - 1, false
	}
	inicio, err := strconv.ParseInt(partes[0], 10, 64)
	if err != nil || inicio < 0 || inicio >= tamano {
		return 0, tamano - 1, false
	}
	fin := tamano - 1
	if partes[1] != "" {
		fin, err = strconv.ParseInt(partes[1], 10, 64)
		if err != nil || fin < inicio {
			return 0, tamano - 1, false
		}
		if fin >= tamano {
			fin = tamano - 1
		}
	}
	return inicio, fin, true
}

func (s *servidorVideoRange) assertoStreamingHTTP(t *testing.T) {
	t.Helper()
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.solicitudesRange == 0 {
		t.Fatalf("ffmpeg no solicitó rangos HTTP; solicitudes=%d", s.solicitudes)
	}
	t.Logf("HTTP remoto: solicitudes=%d, con Range=%d, bytes transferidos=%d, mayor respuesta=%d, tamaño=%d",
		s.solicitudes, s.solicitudesRange, s.bytesEntregados, s.maximoRespuesta, s.tamano)
}

func crearVideoMP4ConAudioPrueba(t *testing.T, rutaFFmpeg string) string {
	t.Helper()

	rutaVideo := filepath.Join(t.TempDir(), "prueba-streaming.mp4")
	comando := exec.Command(
		rutaFFmpeg,
		"-hide_banner", "-loglevel", "error",
		"-f", "lavfi", "-i", "testsrc2=size=640x360:rate=30",
		"-f", "lavfi", "-i", "sine=frequency=880:sample_rate=48000",
		"-t", "8",
		"-c:v", "libx264", "-preset", "ultrafast", "-b:v", "8M", "-g", "30", "-pix_fmt", "yuv420p",
		"-c:a", "aac", "-b:a", "128k",
		"-movflags", "+faststart", "-y", rutaVideo,
	)
	if salida, err := comando.CombinedOutput(); err != nil {
		t.Fatalf("no se pudo generar el MP4 de prueba: %v: %s", err, bytes.TrimSpace(salida))
	}
	return rutaVideo
}

func verificarLecturaAudioRemota(t *testing.T, ctx context.Context, rutaFFmpeg, ruta string) {
	t.Helper()
	comando := exec.CommandContext(ctx, rutaFFmpeg,
		"-hide_banner", "-loglevel", "error", "-nostdin",
		"-ss", "5", "-i", ruta,
		"-map", "0:a:0", "-t", "0.5",
		"-vn", "-ac", "2", "-ar", "48000", "-acodec", "pcm_s16le",
		"-f", "s16le", "pipe:1",
	)
	salida, err := comando.Output()
	if err != nil {
		t.Fatalf("ffmpeg no pudo leer el audio remoto: %v", err)
	}
	if len(salida) == 0 {
		t.Fatal("ffmpeg no devolvió samples de audio remoto")
	}
}
