package indexador

import (
	"context"
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"destrellas-dam/internal/almacen"
	"destrellas-dam/internal/modelo"
	"destrellas-dam/internal/servicios/metadatos"
)

// OpcionesDescubrimiento activa etapas costosas de forma explicita.
type OpcionesDescubrimiento struct {
	CalcularMetadatos       bool
	CalcularHashesExactos   bool
	CalcularHashesParciales bool
	ConcurrenciaAnalisis    int
	SoloMultimedia          bool
	IgnorarArchivosVacios   bool
	RutasExcluidas          []string
}

// EventoProgreso comunica estado al usuario sin bloquear la UI.
type EventoProgreso struct {
	RutaActual            string
	DirectoriosProcesados int64
	ArchivosEncontrados   int64
	ArchivosAnalizados    int64
	Porcentaje            float64
	Finalizado            bool
	Error                 error
}

// Servicio coordina descubrimiento persistente e indexacion en segundo plano.
type Servicio struct {
	repo            almacen.Repositorio
	metadatos       *metadatos.Servicio
	concurrencia    int
	poolBuffersHash sync.Pool
}

// NuevoServicio crea un indexador listo para lotes grandes.
func NuevoServicio(repo almacen.Repositorio, servicioMetadatos *metadatos.Servicio, concurrencia int) *Servicio {
	if concurrencia < 1 {
		concurrencia = 2
	}

	return &Servicio{
		repo:         repo,
		metadatos:    servicioMetadatos,
		concurrencia: concurrencia,
		poolBuffersHash: sync.Pool{
			New: func() any {
				return make([]byte, 512*1024)
			},
		},
	}
}

// Descubrir recorre una raiz de forma iterativa con memoria acotada.
func (s *Servicio) Descubrir(ctx context.Context, raiz string, opciones OpcionesDescubrimiento) <-chan EventoProgreso {
	if opciones.ConcurrenciaAnalisis < 1 {
		opciones.ConcurrenciaAnalisis = s.concurrencia
	}

	eventos := make(chan EventoProgreso, 64)
	go s.ejecutarDescubrimiento(ctx, raiz, opciones, eventos)
	return eventos
}

func (s *Servicio) ejecutarDescubrimiento(ctx context.Context, raiz string, opciones OpcionesDescubrimiento, eventos chan<- EventoProgreso) {
	defer close(eventos)

	archivosPendientes := make(chan modelo.Archivo, opciones.ConcurrenciaAnalisis*8)
	var wg sync.WaitGroup
	var directoriosProcesados atomic.Int64
	var archivosEncontrados atomic.Int64
	var archivosAnalizados atomic.Int64

	emitir := func(evento EventoProgreso) {
		select {
		case <-ctx.Done():
		case eventos <- evento:
		}
	}

	worker := func() {
		defer wg.Done()
		for archivo := range archivosPendientes {
			select {
			case <-ctx.Done():
				return
			default:
			}

			enriquecido := archivo
			if opciones.CalcularMetadatos {
				if resultado, err := s.metadatos.AnalizarArchivo(ctx, enriquecido); err == nil {
					enriquecido = resultado
				} else {
					emitir(EventoProgreso{
						RutaActual:            enriquecido.Ruta,
						DirectoriosProcesados: directoriosProcesados.Load(),
						ArchivosEncontrados:   archivosEncontrados.Load(),
						ArchivosAnalizados:    archivosAnalizados.Load(),
						Error:                 err,
					})
				}
			}

			if opciones.CalcularHashesExactos {
				if hashes, err := s.calcularHashesExactos(enriquecido.Ruta); err == nil {
					enriquecido.Hashes.MD5 = hashes.MD5
					enriquecido.Hashes.SHA256 = hashes.SHA256
				} else {
					emitir(EventoProgreso{
						RutaActual:            enriquecido.Ruta,
						DirectoriosProcesados: directoriosProcesados.Load(),
						ArchivosEncontrados:   archivosEncontrados.Load(),
						ArchivosAnalizados:    archivosAnalizados.Load(),
						Error:                 err,
					})
				}
			}

			if opciones.CalcularHashesParciales {
				switch enriquecido.Tipo {
				case modelo.TipoImagen:
					if hash, err := s.metadatos.CalcularDHashImagen(ctx, enriquecido.Ruta); err == nil {
						enriquecido.Hashes.DHashImagen = hash
					}
				case modelo.TipoVideo:
					if hash, err := s.metadatos.CalcularDHashVideo(ctx, enriquecido.Ruta); err == nil {
						enriquecido.Hashes.DHashVideo = hash
					}
				}
			}

			if err := s.repo.GuardarArchivo(ctx, enriquecido); err != nil {
				emitir(EventoProgreso{
					RutaActual:            enriquecido.Ruta,
					DirectoriosProcesados: directoriosProcesados.Load(),
					ArchivosEncontrados:   archivosEncontrados.Load(),
					ArchivosAnalizados:    archivosAnalizados.Load(),
					Error:                 err,
				})
			}

			archivosAnalizados.Add(1)
		}
	}

	for i := 0; i < opciones.ConcurrenciaAnalisis; i++ {
		wg.Add(1)
		go worker()
	}

	pendientes := []string{raiz}
	ultimoEvento := time.Now().Add(-time.Second)
	for len(pendientes) > 0 {
		select {
		case <-ctx.Done():
			close(archivosPendientes)
			wg.Wait()
			emitir(EventoProgreso{Finalizado: true, Error: ctx.Err(), Porcentaje: 100})
			return
		default:
		}

		actual := pendientes[len(pendientes)-1]
		pendientes = pendientes[:len(pendientes)-1]
		if rutaEstaExcluida(actual, opciones.RutasExcluidas) {
			continue
		}

		directorio, err := os.Open(actual)
		if err != nil {
			emitir(EventoProgreso{
				RutaActual:            actual,
				DirectoriosProcesados: directoriosProcesados.Load(),
				ArchivosEncontrados:   archivosEncontrados.Load(),
				ArchivosAnalizados:    archivosAnalizados.Load(),
				Error:                 fmt.Errorf("no se pudo abrir el directorio %q: %w", actual, err),
			})
			continue
		}

		for {
			entradas, err := directorio.ReadDir(256)
			if err != nil && err != io.EOF {
				emitir(EventoProgreso{
					RutaActual:            actual,
					DirectoriosProcesados: directoriosProcesados.Load(),
					ArchivosEncontrados:   archivosEncontrados.Load(),
					ArchivosAnalizados:    archivosAnalizados.Load(),
					Error:                 fmt.Errorf("no se pudo leer el directorio %q: %w", actual, err),
				})
				break
			}

			for _, entrada := range entradas {
				if debeOmitirDirectorioOculto(entrada.Name(), entrada.IsDir(), false) {
					continue
				}
				archivo, err := construirArchivoDesdeEntrada(actual, entrada)
				if err != nil {
					continue
				}
				if rutaEstaExcluida(archivo.Ruta, opciones.RutasExcluidas) {
					continue
				}
				if opciones.SoloMultimedia && !archivo.EsDirectorio && !archivo.EsMultimedia() {
					continue
				}
				if debeIgnorarArchivoVacio(archivo, opciones) {
					if err := s.repo.EliminarArchivo(ctx, archivo.Ruta); err != nil {
						emitir(EventoProgreso{
							RutaActual:            archivo.Ruta,
							DirectoriosProcesados: directoriosProcesados.Load(),
							ArchivosEncontrados:   archivosEncontrados.Load(),
							ArchivosAnalizados:    archivosAnalizados.Load(),
							Error:                 err,
						})
					}
					continue
				}
				if err := s.repo.GuardarArchivo(ctx, archivo); err != nil {
					emitir(EventoProgreso{
						RutaActual:            archivo.Ruta,
						DirectoriosProcesados: directoriosProcesados.Load(),
						ArchivosEncontrados:   archivosEncontrados.Load(),
						ArchivosAnalizados:    archivosAnalizados.Load(),
						Error:                 err,
					})
				}
				archivosEncontrados.Add(1)

				if archivo.EsDirectorio {
					pendientes = append(pendientes, archivo.Ruta)
				} else if opciones.CalcularMetadatos || opciones.CalcularHashesExactos || opciones.CalcularHashesParciales {
					select {
					case <-ctx.Done():
						break
					case archivosPendientes <- archivo:
					}
				}
			}

			if err == io.EOF {
				break
			}
		}
		directorio.Close()

		procesados := directoriosProcesados.Add(1)
		if time.Since(ultimoEvento) > 350*time.Millisecond {
			totalEstimado := procesados + int64(len(pendientes))
			porcentaje := 0.0
			if totalEstimado > 0 {
				porcentaje = (float64(procesados) / float64(totalEstimado)) * 100
			}
			emitir(EventoProgreso{
				RutaActual:            actual,
				DirectoriosProcesados: procesados,
				ArchivosEncontrados:   archivosEncontrados.Load(),
				ArchivosAnalizados:    archivosAnalizados.Load(),
				Porcentaje:            porcentaje,
			})
			ultimoEvento = time.Now()
		}
	}

	close(archivosPendientes)
	wg.Wait()
	emitir(EventoProgreso{
		RutaActual:            filepath.Clean(raiz),
		DirectoriosProcesados: directoriosProcesados.Load(),
		ArchivosEncontrados:   archivosEncontrados.Load(),
		ArchivosAnalizados:    archivosAnalizados.Load(),
		Porcentaje:            100,
		Finalizado:            true,
	})
}

func rutaEstaExcluida(ruta string, rutasExcluidas []string) bool {
	ruta = filepath.Clean(strings.TrimSpace(ruta))
	if ruta == "." || ruta == "" {
		return false
	}
	for _, rutaExcluida := range rutasExcluidas {
		rutaExcluida = filepath.Clean(strings.TrimSpace(rutaExcluida))
		if rutaExcluida == "." || rutaExcluida == "" {
			continue
		}
		if ruta == rutaExcluida {
			return true
		}
		if strings.HasPrefix(ruta, rutaExcluida+string(os.PathSeparator)) {
			return true
		}
	}
	return false
}

func debeIgnorarArchivoVacio(archivo modelo.Archivo, opciones OpcionesDescubrimiento) bool {
	if !opciones.IgnorarArchivosVacios {
		return false
	}
	if archivo.EsDirectorio {
		return false
	}
	return archivo.Tamano <= 0
}

func (s *Servicio) calcularHashesExactos(ruta string) (modelo.HashesArchivo, error) {
	archivo, err := os.Open(ruta)
	if err != nil {
		return modelo.HashesArchivo{}, fmt.Errorf("no se pudo abrir el archivo para hash: %w", err)
	}
	defer archivo.Close()

	hashMD5 := md5.New()
	hashSHA256 := sha256.New()
	escritor := io.MultiWriter(hashMD5, hashSHA256)

	buffer := s.poolBuffersHash.Get().([]byte)
	defer s.poolBuffersHash.Put(buffer)

	if _, err := io.CopyBuffer(escritor, archivo, buffer); err != nil {
		return modelo.HashesArchivo{}, fmt.Errorf("no se pudo calcular los hashes exactos: %w", err)
	}

	return modelo.HashesArchivo{
		MD5:    hex.EncodeToString(hashMD5.Sum(nil)),
		SHA256: hex.EncodeToString(hashSHA256.Sum(nil)),
	}, nil
}
