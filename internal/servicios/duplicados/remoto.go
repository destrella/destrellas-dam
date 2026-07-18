package duplicados

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"destrellas-dam/internal/modelo"
	"destrellas-dam/internal/servicios/indexador"
	"destrellas-dam/internal/yandex"
)

const (
	tamanoPaginaDescubrimientoRemoto = 100
	intervaloPeticionesRemotas       = 350 * time.Millisecond
	maximoReintentosPeticionRemota   = 3
	esperaErrorPeticionRemota        = 1200 * time.Millisecond
)

type cursorDescubrimientoRemoto struct {
	Ruta   string `json:"ruta"`
	Offset int    `json:"offset"`
}

type estadoDescubrimientoRemotoPersistido struct {
	Raiz                  string                       `json:"raiz"`
	Pendientes            []cursorDescubrimientoRemoto `json:"pendientes"`
	RutaActual            string                       `json:"ruta_actual"`
	DirectoriosProcesados int64                        `json:"directorios_procesados"`
	ArchivosEncontrados   int64                        `json:"archivos_encontrados"`
	ArchivosAnalizados    int64                        `json:"archivos_analizados"`
	ActualizadoUnix       int64                        `json:"actualizado_unix"`
}

// EstadoDescubrimientoRemoto resume el progreso persistido para la UI.
type EstadoDescubrimientoRemoto struct {
	RutaRaiz              string
	RutaActual            string
	DirectoriosProcesados int64
	ArchivosEncontrados   int64
	ArchivosAnalizados    int64
	Pendiente             bool
}

// GuardarArchivoRemotoDescubierto persiste un archivo remoto para la futura detección de duplicados.
func (s *Servicio) GuardarArchivoRemotoDescubierto(ctx context.Context, archivo modelo.Archivo) error {
	if s == nil || s.repo == nil {
		return errors.New("servicio de duplicados no inicializado")
	}
	if archivo.Origen != modelo.OrigenYandex || archivo.EsDirectorio || strings.TrimSpace(archivo.Ruta) == "" {
		return nil
	}
	return s.repo.GuardarArchivo(ctx, archivo)
}

// CargarEstadoDescubrimientoRemoto recupera el avance persistido del escaneo remoto.
func (s *Servicio) CargarEstadoDescubrimientoRemoto() (EstadoDescubrimientoRemoto, error) {
	estado, err := s.cargarEstadoRemoto()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return EstadoDescubrimientoRemoto{}, nil
		}
		return EstadoDescubrimientoRemoto{}, err
	}
	return resumirEstadoRemoto(estado), nil
}

// IniciarDescubrimientoRemoto recorre Yandex.Disk de forma incremental y persistente.
func (s *Servicio) IniciarDescubrimientoRemoto(ctx context.Context, ruta string, cliente yandex.Cliente) <-chan indexador.EventoProgreso {
	eventos := make(chan indexador.EventoProgreso, 64)
	go s.ejecutarDescubrimientoRemoto(ctx, ruta, cliente, eventos)
	return eventos
}

func (s *Servicio) ejecutarDescubrimientoRemoto(ctx context.Context, ruta string, cliente yandex.Cliente, eventos chan<- indexador.EventoProgreso) {
	defer close(eventos)

	emitir := func(evento indexador.EventoProgreso) {
		select {
		case <-ctx.Done():
		case eventos <- evento:
		}
	}

	if s == nil || s.repo == nil {
		emitir(indexador.EventoProgreso{
			Finalizado: true,
			Error:      errors.New("servicio de duplicados no inicializado"),
			Porcentaje: 100,
		})
		return
	}
	if cliente == nil || !cliente.Configurado() {
		emitir(indexador.EventoProgreso{
			Finalizado: true,
			Error:      yandex.ErrNoImplementado,
			Porcentaje: 100,
		})
		return
	}

	estado, err := s.resolverEstadoInicialRemoto(ruta)
	if err != nil {
		emitir(indexador.EventoProgreso{
			Finalizado: true,
			Error:      err,
			Porcentaje: 100,
		})
		return
	}

	emitir(eventoDesdeEstadoRemoto(estado, false, nil))
	_ = s.guardarEstadoRemoto(estado)

	reloj := time.NewTicker(intervaloPeticionesRemotas)
	defer reloj.Stop()

	for len(estado.Pendientes) > 0 {
		if err := ctx.Err(); err != nil {
			_ = s.guardarEstadoRemoto(estado)
			emitir(eventoDesdeEstadoRemoto(estado, true, err))
			return
		}

		indiceActual := len(estado.Pendientes) - 1
		cursor := estado.Pendientes[indiceActual]
		estado.RutaActual = cursor.Ruta
		_ = s.guardarEstadoRemoto(estado)

		if err := esperarTurnoRemoto(ctx, reloj); err != nil {
			_ = s.guardarEstadoRemoto(estado)
			emitir(eventoDesdeEstadoRemoto(estado, true, err))
			return
		}

		lote, err := s.listarPaginaRemotaConReintentos(ctx, cliente, cursor.Ruta, cursor.Offset)
		if err != nil {
			emitir(eventoDesdeEstadoRemoto(estado, false, err))
			estado.Pendientes = estado.Pendientes[:indiceActual]
			estado.DirectoriosProcesados++
			_ = s.guardarEstadoRemoto(estado)
			continue
		}

		if len(lote) == 0 {
			estado.Pendientes = estado.Pendientes[:indiceActual]
			estado.DirectoriosProcesados++
			_ = s.guardarEstadoRemoto(estado)
			emitir(eventoDesdeEstadoRemoto(estado, false, nil))
			continue
		}

		cursor.Offset += len(lote)
		if len(lote) < tamanoPaginaDescubrimientoRemoto {
			estado.Pendientes = estado.Pendientes[:indiceActual]
			estado.DirectoriosProcesados++
		} else {
			estado.Pendientes[indiceActual] = cursor
		}

		for _, elemento := range lote {
			archivo := convertirElementoRemotoADuplicado(elemento)
			if archivo.EsDirectorio {
				estado.Pendientes = append(estado.Pendientes, cursorDescubrimientoRemoto{
					Ruta:   archivo.Ruta,
					Offset: 0,
				})
				continue
			}

			estado.ArchivosEncontrados++
			if err := s.GuardarArchivoRemotoDescubierto(ctx, archivo); err != nil {
				emitir(eventoDesdeEstadoRemoto(estado, false, err))
				continue
			}
			estado.ArchivosAnalizados++
		}

		if err := s.guardarEstadoRemoto(estado); err != nil {
			emitir(eventoDesdeEstadoRemoto(estado, false, err))
		}
		emitir(eventoDesdeEstadoRemoto(estado, false, nil))
	}

	_ = s.limpiarEstadoRemoto()
	emitir(indexador.EventoProgreso{
		RutaActual:            estado.RutaActual,
		DirectoriosProcesados: estado.DirectoriosProcesados,
		ArchivosEncontrados:   estado.ArchivosEncontrados,
		ArchivosAnalizados:    estado.ArchivosAnalizados,
		Porcentaje:            100,
		Finalizado:            true,
	})
}

func (s *Servicio) resolverEstadoInicialRemoto(ruta string) (estadoDescubrimientoRemotoPersistido, error) {
	ruta = normalizarRutaRemotaDuplicados(ruta)
	guardado, err := s.cargarEstadoRemoto()
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return estadoDescubrimientoRemotoPersistido{}, err
	}

	if err == nil && len(guardado.Pendientes) > 0 && strings.EqualFold(strings.TrimSpace(guardado.Raiz), strings.TrimSpace(ruta)) {
		return guardado, nil
	}

	return estadoDescubrimientoRemotoPersistido{
		Raiz: ruta,
		Pendientes: []cursorDescubrimientoRemoto{
			{Ruta: ruta, Offset: 0},
		},
		RutaActual:      ruta,
		ActualizadoUnix: time.Now().Unix(),
	}, nil
}

func (s *Servicio) listarPaginaRemotaConReintentos(ctx context.Context, cliente yandex.Cliente, ruta string, offset int) ([]yandex.ElementoRemoto, error) {
	var ultimoError error
	for intento := 0; intento < maximoReintentosPeticionRemota; intento++ {
		lote, err := cliente.ListarElementos(ctx, ruta, tamanoPaginaDescubrimientoRemoto, offset)
		if err == nil {
			return lote, nil
		}
		ultimoError = err

		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		if intento+1 >= maximoReintentosPeticionRemota {
			break
		}

		temporizador := time.NewTimer(esperaErrorPeticionRemota)
		select {
		case <-ctx.Done():
			temporizador.Stop()
			return nil, ctx.Err()
		case <-temporizador.C:
		}
	}
	return nil, ultimoError
}

func esperarTurnoRemoto(ctx context.Context, reloj *time.Ticker) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-reloj.C:
		return nil
	}
}

func eventoDesdeEstadoRemoto(estado estadoDescubrimientoRemotoPersistido, finalizado bool, err error) indexador.EventoProgreso {
	porcentaje := porcentajeEstadoRemoto(estado, finalizado && err == nil)
	return indexador.EventoProgreso{
		RutaActual:            estado.RutaActual,
		DirectoriosProcesados: estado.DirectoriosProcesados,
		ArchivosEncontrados:   estado.ArchivosEncontrados,
		ArchivosAnalizados:    estado.ArchivosAnalizados,
		Porcentaje:            porcentaje,
		Finalizado:            finalizado,
		Error:                 err,
	}
}

func porcentajeEstadoRemoto(estado estadoDescubrimientoRemotoPersistido, finalizado bool) float64 {
	if finalizado {
		return 100
	}
	totalDirectorios := estado.DirectoriosProcesados + int64(len(estado.Pendientes))
	if totalDirectorios <= 0 {
		return 0
	}
	porcentaje := (100 * float64(estado.DirectoriosProcesados)) / float64(totalDirectorios)
	if porcentaje >= 100 {
		return 99
	}
	return porcentaje
}

func resumirEstadoRemoto(estado estadoDescubrimientoRemotoPersistido) EstadoDescubrimientoRemoto {
	return EstadoDescubrimientoRemoto{
		RutaRaiz:              estado.Raiz,
		RutaActual:            estado.RutaActual,
		DirectoriosProcesados: estado.DirectoriosProcesados,
		ArchivosEncontrados:   estado.ArchivosEncontrados,
		ArchivosAnalizados:    estado.ArchivosAnalizados,
		Pendiente:             len(estado.Pendientes) > 0,
	}
}

func (s *Servicio) guardarEstadoRemoto(estado estadoDescubrimientoRemotoPersistido) error {
	if strings.TrimSpace(s.rutaEstadoRemoto) == "" {
		return nil
	}
	estado.ActualizadoUnix = time.Now().Unix()

	if err := os.MkdirAll(filepath.Dir(s.rutaEstadoRemoto), 0o755); err != nil {
		return fmt.Errorf("no se pudo preparar la persistencia del escaneo remoto: %w", err)
	}

	datos, err := json.MarshalIndent(estado, "", "\t")
	if err != nil {
		return fmt.Errorf("no se pudo serializar el estado del escaneo remoto: %w", err)
	}

	rutaTemporal := s.rutaEstadoRemoto + ".tmp"
	if err := os.WriteFile(rutaTemporal, datos, 0o644); err != nil {
		return fmt.Errorf("no se pudo escribir el estado temporal del escaneo remoto: %w", err)
	}
	if err := os.Rename(rutaTemporal, s.rutaEstadoRemoto); err != nil {
		return fmt.Errorf("no se pudo confirmar el estado del escaneo remoto: %w", err)
	}
	return nil
}

func (s *Servicio) cargarEstadoRemoto() (estadoDescubrimientoRemotoPersistido, error) {
	if strings.TrimSpace(s.rutaEstadoRemoto) == "" {
		return estadoDescubrimientoRemotoPersistido{}, os.ErrNotExist
	}
	datos, err := os.ReadFile(s.rutaEstadoRemoto)
	if err != nil {
		return estadoDescubrimientoRemotoPersistido{}, err
	}

	var estado estadoDescubrimientoRemotoPersistido
	if err := json.Unmarshal(datos, &estado); err != nil {
		return estadoDescubrimientoRemotoPersistido{}, fmt.Errorf("no se pudo interpretar el estado del escaneo remoto: %w", err)
	}
	estado.Raiz = normalizarRutaRemotaDuplicados(estado.Raiz)
	estado.RutaActual = normalizarRutaRemotaDuplicados(estado.RutaActual)
	for indice := range estado.Pendientes {
		estado.Pendientes[indice].Ruta = normalizarRutaRemotaDuplicados(estado.Pendientes[indice].Ruta)
		if estado.Pendientes[indice].Offset < 0 {
			estado.Pendientes[indice].Offset = 0
		}
	}
	return estado, nil
}

func (s *Servicio) limpiarEstadoRemoto() error {
	if strings.TrimSpace(s.rutaEstadoRemoto) == "" {
		return nil
	}
	if err := os.Remove(s.rutaEstadoRemoto); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("no se pudo limpiar el estado del escaneo remoto: %w", err)
	}
	return nil
}

func convertirElementoRemotoADuplicado(elemento yandex.ElementoRemoto) modelo.Archivo {
	ruta := normalizarRutaRemotaDuplicados(elemento.Ruta)
	nombre := strings.TrimSpace(elemento.Nombre)
	if nombre == "" {
		partes := strings.Split(strings.TrimPrefix(ruta, "disk:/"), "/")
		if len(partes) > 0 {
			nombre = partes[len(partes)-1]
		}
		if nombre == "" {
			nombre = "Yandex.Disk"
		}
	}

	return modelo.Archivo{
		Origen:       modelo.OrigenYandex,
		Ruta:         ruta,
		RutaPadre:    rutaPadreRemotoDuplicados(ruta),
		Nombre:       nombre,
		PreviewURL:   strings.TrimSpace(elemento.PreviewURL),
		Tamano:       elemento.Tamano,
		Modificado:   elemento.Modificado,
		Tipo:         elemento.Tipo,
		EsOculto:     modelo.EsOcultoPorNombre(nombre),
		EsDirectorio: elemento.EsDirectorio,
		Hashes: modelo.HashesArchivo{
			MD5:    strings.TrimSpace(elemento.HashMD5),
			SHA256: strings.TrimSpace(elemento.HashSHA256),
		},
	}
}

func rutaPadreRemotoDuplicados(ruta string) string {
	ruta = normalizarRutaRemotaDuplicados(ruta)
	if ruta == "disk:/" {
		return "disk:/"
	}
	partes := strings.Split(strings.TrimPrefix(ruta, "disk:/"), "/")
	if len(partes) <= 1 {
		return "disk:/"
	}
	return normalizarRutaRemotaDuplicados("disk:/" + strings.Join(partes[:len(partes)-1], "/"))
}

func normalizarRutaRemotaDuplicados(ruta string) string {
	ruta = strings.TrimSpace(ruta)
	if ruta == "" || ruta == "/" || strings.EqualFold(ruta, "disk:") || strings.EqualFold(ruta, "disk:/") {
		return "disk:/"
	}
	if strings.HasPrefix(strings.ToLower(ruta), "disk:/") {
		return "disk:/" + strings.TrimPrefix(strings.TrimPrefix(ruta[6:], "/"), "/")
	}
	if strings.HasPrefix(ruta, "/") {
		return "disk:" + ruta
	}
	return "disk:/" + strings.TrimPrefix(ruta, "/")
}
