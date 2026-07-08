package main

import (
	"log"

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

	servicioMetadatos := metadatos.NuevoServicio()
	servicioArchivos := archivos.NuevoServicio(cfg.CarpetaArchivado)
	listador := indexador.NuevoListadorLocal(almacenSQLite)
	servicioIndexador := indexador.NuevoServicio(almacenSQLite, servicioMetadatos, cfg.ConcurrenciaIndexado)
	servicioDuplicados := duplicados.NuevoServicio(almacenSQLite, servicioIndexador)
	clienteYandex := yandex.NuevoClienteNulo(cfg.ClaveAPIYandex)

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
