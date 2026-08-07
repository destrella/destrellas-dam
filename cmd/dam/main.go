package main

import (
	"context"
	"io"
	"log"
	"path/filepath"
	"strings"

	"gioui.org/app"
	"gioui.org/unit"

	"destrellas-dam/internal/almacen/sqlite"
	"destrellas-dam/internal/configuracion"
	"destrellas-dam/internal/servicios/archivos"
	"destrellas-dam/internal/servicios/duplicados"
	"destrellas-dam/internal/servicios/indexador"
	"destrellas-dam/internal/servicios/metadatos"
	"destrellas-dam/internal/ui"
	"destrellas-dam/internal/yandex"
)

func main() {
	rutas, err := configuracion.ResolverRutas()
	if err != nil {
		log.Fatal(err)
	}

	repoConfiguracion := configuracion.NuevoRepositorio(rutas.ArchivoConfig)
	cfg, err := repoConfiguracion.Cargar()
	if err != nil {
		log.Fatal(err)
	}

	almacenSQLite, err := sqlite.Nuevo(cfg.RutaBaseDatos)
	if err != nil {
		log.Fatal(err)
	}
	defer almacenSQLite.Cerrar()

	clienteYandex := yandex.NuevoCliente(cfg.ClaveAPIYandex)
	servicioMetadatos := metadatos.NuevoServicio()
	servicioMetadatos.EstablecerResolverFuenteVideo(func(ctx context.Context, ruta string) (string, error) {
		if strings.HasPrefix(strings.TrimSpace(ruta), "disk:/") {
			return clienteYandex.URLDescarga(ctx, ruta)
		}
		return ruta, nil
	})
	servicioMetadatos.EstablecerDescargadorFuenteVideo(filepath.Join(rutas.DirectorioCache, "videos"), func(ctx context.Context, ruta string) (io.ReadCloser, error) {
		if !strings.HasPrefix(strings.TrimSpace(ruta), "disk:/") {
			return nil, yandex.ErrNoImplementado
		}
		return clienteYandex.Descargar(ctx, ruta)
	})
	servicioArchivos := archivos.NuevoServicio(cfg.CarpetaArchivado)
	listador := indexador.NuevoListadorLocal(almacenSQLite)
	servicioIndexador := indexador.NuevoServicio(almacenSQLite, servicioMetadatos, cfg.ConcurrenciaIndexado)
	servicioDuplicados := duplicados.NuevoServicio(almacenSQLite, servicioIndexador)

	go func() {
		ventana := new(app.Window)
		ventana.Option(
			app.Title("DEstrella's DAM"),
			app.Size(unit.Dp(1600), unit.Dp(960)),
		)

		aplicacion := ui.NuevaAplicacion(ui.Dependencias{
			RepositorioConfig: repoConfiguracion,
			Configuracion:     cfg,
			Almacen:           almacenSQLite,
			Listador:          listador,
			Indexador:         servicioIndexador,
			Metadatos:         servicioMetadatos,
			Archivos:          servicioArchivos,
			Duplicados:        servicioDuplicados,
			Yandex:            clienteYandex,
		})
		if err := aplicacion.Ejecutar(ventana); err != nil {
			log.Fatal(err)
		}
	}()

	app.Main()
}
