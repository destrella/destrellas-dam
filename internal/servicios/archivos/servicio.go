package archivos

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"destrellas-dam/internal/plataforma"
)

// Servicio centraliza operaciones sobre archivos locales.
type Servicio struct {
	carpetaArchivado string
	poolBuffers      sync.Pool
}

// NuevoServicio crea el servicio con un pool de buffers reutilizable.
func NuevoServicio(carpetaArchivado string) *Servicio {
	return &Servicio{
		carpetaArchivado: carpetaArchivado,
		poolBuffers: sync.Pool{
			New: func() any {
				return make([]byte, 256*1024)
			},
		},
	}
}

// ActualizarCarpetaArchivado permite reaccionar a cambios de configuracion sin reconstruir el servicio.
func (s *Servicio) ActualizarCarpetaArchivado(carpeta string) {
	s.carpetaArchivado = carpeta
}

// Mover traslada un archivo y hace copia+eliminacion si cambia de volumen.
func (s *Servicio) Mover(_ context.Context, origen, destino string) error {
	if err := os.MkdirAll(filepath.Dir(destino), 0o755); err != nil {
		return fmt.Errorf("no se pudo preparar el destino: %w", err)
	}

	if err := os.Rename(origen, destino); err == nil {
		return nil
	} else if !errors.Is(err, syscall.EXDEV) {
		return fmt.Errorf("no se pudo mover el archivo: %w", err)
	}

	if err := s.copiarArchivo(origen, destino); err != nil {
		return err
	}
	return os.Remove(origen)
}

// Archivar mueve el archivo a la carpeta configurada respetando colisiones de nombre.
func (s *Servicio) Archivar(ctx context.Context, ruta string) (string, error) {
	if s.carpetaArchivado == "" {
		return "", errors.New("no hay carpeta de archivado configurada")
	}

	if err := os.MkdirAll(s.carpetaArchivado, 0o755); err != nil {
		return "", fmt.Errorf("no se pudo crear la carpeta de archivado: %w", err)
	}

	destino, err := rutaDisponible(filepath.Join(s.carpetaArchivado, filepath.Base(ruta)))
	if err != nil {
		return "", err
	}
	if err := s.Mover(ctx, ruta, destino); err != nil {
		return "", err
	}
	return destino, nil
}

// ArchivarConFecha mueve el archivo a una jerarquia año/mes/día basada en sus metadatos.
func (s *Servicio) ArchivarConFecha(ctx context.Context, ruta, fecha, hora string) (string, error) {
	if s.carpetaArchivado == "" {
		return "", errors.New("no hay carpeta de archivado configurada")
	}

	fecha = strings.TrimSpace(fecha)
	hora = strings.TrimSpace(hora)
	if fecha == "" || hora == "" {
		return "", errors.New("se requieren metadatos de fecha y hora para archivar")
	}

	instanteFecha, err := time.Parse("2006-01-02", fecha)
	if err != nil {
		return "", fmt.Errorf("la fecha de archivado no es válida: %w", err)
	}
	if _, err := normalizarHoraArchivado(hora); err != nil {
		return "", fmt.Errorf("la hora de archivado no es válida: %w", err)
	}

	destinoBase := filepath.Join(
		s.carpetaArchivado,
		instanteFecha.Format("2006"),
		instanteFecha.Format("01"),
		instanteFecha.Format("02"),
	)
	if err := os.MkdirAll(destinoBase, 0o755); err != nil {
		return "", fmt.Errorf("no se pudo crear la jerarquía de archivado: %w", err)
	}

	destino, err := rutaDisponible(filepath.Join(destinoBase, filepath.Base(ruta)))
	if err != nil {
		return "", err
	}
	if err := s.Mover(ctx, ruta, destino); err != nil {
		return "", err
	}
	return destino, nil
}

// EnviarAPapelera delega en la plataforma principal.
func (s *Servicio) EnviarAPapelera(ctx context.Context, ruta string) error {
	return plataforma.EnviarAPapelera(ctx, ruta)
}

// AbrirEnSistema solicita al sistema abrir el archivo con su aplicacion predeterminada.
func (s *Servicio) AbrirEnSistema(ctx context.Context, ruta string) error {
	return plataforma.AbrirEnSistema(ctx, ruta)
}

// GuardarContenidoLocal guarda un flujo remoto en una ruta local.
func (s *Servicio) GuardarContenidoLocal(destino string, lector io.Reader) error {
	_, err := s.GuardarContenidoLocalDisponible(destino, lector)
	return err
}

// GuardarContenidoLocalDisponible guarda un flujo remoto respetando colisiones de nombre.
func (s *Servicio) GuardarContenidoLocalDisponible(destino string, lector io.Reader) (string, error) {
	rutaFinal, err := rutaDisponible(destino)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(rutaFinal), 0o755); err != nil {
		return "", fmt.Errorf("no se pudo preparar el directorio local: %w", err)
	}

	archivo, err := os.Create(rutaFinal)
	if err != nil {
		return "", fmt.Errorf("no se pudo crear el archivo local: %w", err)
	}
	defer archivo.Close()

	buffer := s.poolBuffers.Get().([]byte)
	defer s.poolBuffers.Put(buffer)

	if _, err := io.CopyBuffer(archivo, lector, buffer); err != nil {
		return "", fmt.Errorf("no se pudo guardar el contenido remoto: %w", err)
	}
	return rutaFinal, nil
}

func (s *Servicio) copiarArchivo(origen, destino string) error {
	entrada, err := os.Open(origen)
	if err != nil {
		return fmt.Errorf("no se pudo abrir el archivo origen: %w", err)
	}
	defer entrada.Close()

	info, err := entrada.Stat()
	if err != nil {
		return fmt.Errorf("no se pudo leer el estado del archivo origen: %w", err)
	}

	salida, err := os.Create(destino)
	if err != nil {
		return fmt.Errorf("no se pudo crear el archivo destino: %w", err)
	}
	defer salida.Close()

	buffer := s.poolBuffers.Get().([]byte)
	defer s.poolBuffers.Put(buffer)

	if _, err := io.CopyBuffer(salida, entrada, buffer); err != nil {
		return fmt.Errorf("no se pudo copiar el archivo: %w", err)
	}
	if err := salida.Chmod(info.Mode()); err != nil {
		return fmt.Errorf("no se pudo aplicar el modo del archivo: %w", err)
	}
	return os.Chtimes(destino, info.ModTime(), info.ModTime())
}

func rutaDisponible(destino string) (string, error) {
	if _, err := os.Stat(destino); errors.Is(err, os.ErrNotExist) {
		return destino, nil
	}

	extension := filepath.Ext(destino)
	base := strings.TrimSuffix(destino, extension)
	for indice := 1; indice < 10_000; indice++ {
		candidato := fmt.Sprintf("%s-%d%s", base, indice, extension)
		if _, err := os.Stat(candidato); errors.Is(err, os.ErrNotExist) {
			return candidato, nil
		}
	}
	return "", fmt.Errorf("no se pudo encontrar una ruta libre para %q", destino)
}

func normalizarHoraArchivado(hora string) (string, error) {
	hora = strings.TrimSpace(hora)
	disenos := []string{"15:04:05", "15:04"}
	for _, diseno := range disenos {
		instante, err := time.Parse(diseno, hora)
		if err == nil {
			return instante.Format("15:04:05"), nil
		}
	}
	return "", fmt.Errorf("hora inválida")
}
