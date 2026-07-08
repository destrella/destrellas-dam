# DEstrella's DAM

Aplicacion de escritorio en Go con Gio para exploracion, catalogacion y deteccion de duplicados de archivos multimedia, pensada para macOS pero con una arquitectura portable.

## Ejecutar

```bash
go run ./cmd/dam
```

## Herramientas externas integradas

- `exiftool` para lectura y escritura de metadatos.
- `ffmpeg` y `ffprobe` para acciones de video y hashes perceptuales de frames.
- `magick` para conversion y recorte de imagenes.

Si alguna no existe en el sistema, la aplicacion degrada solo la funcionalidad relacionada.

## Arquitectura

- `cmd/dam`: punto de entrada y armado de dependencias.
- `internal/configuracion`: carga y persistencia de configuracion en JSON.
- `internal/modelo`: tipos de dominio, filtros y duplicados.
- `internal/almacen/sqlite`: catalogo persistente con SQLite y pragmas orientados a volumen alto.
- `internal/servicios/indexador`: recorrido incremental, paginacion lazy y descubrimiento en segundo plano.
- `internal/servicios/metadatos`: exiftool, ffmpeg, ImageMagick, dHash y enriquecimiento.
- `internal/servicios/archivos`: mover, archivar, guardar local y papelera de sistema.
- `internal/servicios/duplicados`: consulta y orquestacion de la vista de duplicados.
- `internal/ui`: interfaz Gio con vistas principal, elemento unico, duplicados y configuracion.
- `internal/yandex`: contrato encapsulado para futura integracion remota.

## Decisiones de rendimiento

- Listado local por lotes con `ReadDir(n)` en lugar de cargar directorios completos.
- Escaneo recursivo iterativo con cola acotada, sin recursion profunda.
- Hashes exactos por streaming con buffers reutilizados.
- Persistencia temprana en SQLite para no depender de estructuras gigantes en memoria.
- Carga perezosa de previews y metadatos solo cuando hacen falta en la UI.

## Alcance actual

- Navegacion de arbol local y listado lazy con filtros.
- Vista de galeria y lista.
- Panel de detalle con metadatos editables.
- Descubrimiento local de duplicados exactos y parciales.
- Acciones locales: mover, archivar, papelera, convertir imagen, recortar, extraer frame y optimizar video.
- Integracion remota de Yandex.Disk dejada como contrato listo para completar.

