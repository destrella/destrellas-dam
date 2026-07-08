# Proyecto: "DEstrella's DAM".
Lenguaje: Go. GUI: Gio GUI

## Vista principal, interfaz a 3 columnas:
### Columna derecha
Columna estrecha con los siguientes elementos: 
	- Pestaña 1: Árbol de directorios dentro de la carpeta de usuario cada elemento representado por un icono de una carpeta y el nombre de la carpeta. Al pulsar un elemento se alterna su estado entre carpeta abierta y carpeta cerrada, en el estado abierto, se muestran las subcarpetas contenidas. Por defecto se selecciona la carpeta raíz (carpeta de usuario)
	- Pestaña 2: Listado de palabras clave encontradas en los metadatos de archivos multimedia
	- Pestaña 3: Listado de lugares nombrados definidos en el matedato "Location", una entrada adicional "Ubicación sin nombre" para cuando los archivos multimedia tengan coordenadas GPS pero no tengan etiqueta "Location"
	- Pestaña 4: Cuando exista configurada una clave API de Yandex.Disk, en esta pestaña se mostrarán los directorios de ese servicio.

### Columna central
Columna ancha, muestra todos los elementos de la carpeta seleccionada en la columna derecha, se debe usar una paginación lazy tipo "scroll infinito" mas libre cuando se usen elementos locales y más moderada cuando se usen elementos remotos de Yandex.Disk.
Debe contar con los siguientes modificadores: 
- Mostrar ocultos: falso por defecto, cuando se marque como verdadero el contenido mostrado debe incluir los archivos ocultos.
- Solo multimedia: verdadero por defecto, cuando se marque como verdadero el contenido mostrado debe incluir solo archivos de tipo imagen, video o audio.
- Solo videos: falso por defecto, cuando se marque como verdadero el contenido mostrado debe incluir solo archivos de tipo video.
- Solo imágenes: falso por defecto, cuando se marque como verdadero el contenido mostrado debe incluir solo archivos de tipo imagen.
- Solo audio: falso por defecto, cuando se marque como verdadero el contenido mostrado debe incluir solo archivos de tipo audio.
- Recursivo: muestra los elementos dentro de la carpeta seleccionada y dentro de todas las subcarpetas contenidas.
- Ver como galería: muestra los elementos con una pequeña vista previa, su nombre de archivo, peso, y cuando aplique, dimensiones y duración.
- Ver como lista: muestra un listado de los elementos con columnas para nombre de archivo, peso, y cuando aplique, dimensiones y duración.

Si la columna central se muestra como galería, las imágenes que tengan regiones etiquetadas, se deben renderizar como rectángulos con línea verde en la posición que corresponda, teniendo en cuenta la orientación si está definida en los metadatos.
Cada elemento de la columna central debe mostrar un checkbox para seleccionar varios elementos, y un botón para abrir el elemento en una vista de elemento único.
Los elementos multimedia de la columna central deben tener indicadores cuando existan los siguientes metadatos:
	- Pin de ubicación cuando existan coordenadas GPS
	- Icono de persona cuando existan regiones etiquetadas
	- Icono de información cuando exista el atributo extendido "kMDItemWhereFroms"
	- Icono de robot cuando se detecte metadatos de generación por IA
	- Icono de advertencia cuando se detecten metadatos incrustados por redes sociales, ej.: SpecialInstructions: FBMDXXXXXXX…
	- Icono +18 cuando se detecte la etiqueta "+18" en "Subject"

Si en la columna izquierda se ha seleccionado una carpeta la pestaña Yandex.Disk, se debe usar únicamente la información proveída por el endpoint solicitado usando una paginación conservadora para evitar solicitar demasiados elementos.

Al seleccionar varios archivos, mostrar botones de acciones en lote: mover a otra carpeta, archivar, mandar a la papelera de sistema.

### Columna derecha
Columna estrella que muestra la vista previa del archivo pulsado/seleccionado en la columna central, en nombre del archivo, la ruta en que se encuentra, el peso, si existe, el valor del atributo extendido "kMDItemWhereFroms". En caso de ser un archivo tipo video o imagen, se indican sus dimensiones. En caso de ser un archivo tipo de video o audio, se indica su duración. En caso de ser archivo multimedia se muestra un formulario para poder editar los metadatos. Botones de acción que permitan guardar cambios de los metadatos, mover el archivo seleccionado a otra carpeta, archivar, botón para enviar a la papelera del sistema. En el caso de archivos remotos, debe aparecer un botón adicional para guardar localmente el elemento remoto.

Todos los pesos deben ser expresados en unidades fácilmente legibles (KB, MB, GB, etc.).
Todas las acciones de mover, enviar a la papelera deben realizarse en el ambiente en el que existen los archivos: local o remoto.

## Vista de elemento único. Interfaz a dos columnas:
- Columna izquierda, ancha: muestra la vista del archivo multimedia, con sus iconos indicadores.
- Columna derecha, angosta: misma columna que la vista principal, con la información del archivo en visualización.
Si el archivo es una imagen agregar a la columna derecha:
	- un botón de acción adicional para agregar nuevas regiones en la imagen, al pulsarlo, se habilita la posibilidad de crear una región en la imagen mostrada en el panel izquierdo haciendo un clic en un punto de la imagen y arrastrando para definir un área.
	- un botón de recortar, al pulsarlo se crea un rectángulo redimensionable sobre la imagen para poder definir el área de recorte, en la esquina superior izquierda, mostrar la dimensión en pixeles del área; si la imagen contiene un color sólido o degradado de fondo, detectarlo para centrar el rectángulo de recorte en el área de interés.
	- un botón para convertir la imagen a otro formato.
Si el archivo es un video agregar a la columna derecha:
	- un botón para extraer un frame del video como imagen, al pulsarlo debe poder permitir un keyframe, un frame o un timestamp para extraer el cuadro, mostrando una vista previa del cuadro seleccionado. Debe ofrecer un selector de formato para guardar la imagen generada: jpg, png, webp, avif.
	- un botón para optimizar el video para web, con la opción de sobreescribir el archivo o crear uno nuevo.

## Vista de duplicados
Los archivos listados por los endpoints de Yandex.Disk contienen los hashes MD5 y SHA-256 que se pueden usar para buscar duplicados. Para los archivos locales, se deben determinar sus hashes conforme se vayan descubriendo al navegar. 

Interfaz a tres columnas:

### Columna izquierda
De ancho estrecho, muestra la cantidad de archivos duplicados encontrados; y los siguientes botones:
- para mostrar archivos duplicados locales en la columna central, con indicador de cantidad
- para mostrar archivos duplicados remotos en la columna central, con indicador de cantidad
- para mostrar archivos duplicados mixtos en la columna central, con indicador de cantidad
- para iniciar un proceso de descubrimiento de elementos locales y generación de sus hashes
	- al iniciarse el proceso debe mostrar una barra de progreso, porcentaje de progreso y elementos encontrados hasta el momento, debe ser un proceso asíncrono que permita cambiar de vista sin que se detenga
- para iniciar un proceso de descubrimiento de elementos remotos y guardar sus hashes
	- al iniciarse el proceso debe mostrar una barra de progreso, porcentaje de progreso y elementos encontrados hasta el momento, debe ser un proceso asíncrono que permita cambiar de vista sin que se detenga

### Columna central
Columna ancha que muestre los grupos de archivos duplicados de las siguientes categorías:
- Coincidencia exacta: coincidencia de hashes para archivos locales y remotos
- Coincidencia parcial para archivos de imagen locales: por dhash
- Coincidencia parcial para archivos de video locales: por dhash de los frames al 2%, 25%, 50%, 75% y 98% de la duración del video.
Botones de acción:
Para cada grupo de duplicados mostrar:
- Botón para borrar el más antiguo en el grupo
- Botón para borrar el más nuevo en el grupo.

Si un grupo de duplicados contiene únicamente dos elementos: botón para eliminar en cada elemento.
Si un grupo de duplicados contiene más de dos elementos: checkbox para seleccionar elemento, agregar un botón al grupo para borrar los seleccionados.

La vista debe contar con la posibilidad de ordenar por tamaño de grupo (mayor número de duplicados) descendente, por tamaño de espacio recuperable (tamaño de archivos más grandes) descendente, alfabéticamente,

Filtrado para ver grupos de coincidencias exactas y grupos de coincidencias parciales

## Vista de configuración:
La vista de configuración debe mostrar opciones para determinar:
- la carpeta inicial, por defecto, la carpeta de usuario
- la carpeta de archivado usada al pulsar el botón archivar
- el filtro por defecto para la columna central:
	- todos los archivos
	- solo archivos multimedia
	- solo videos
	- solo imágenes
	- ver archivos ocultos
	- ver elementos recursivamente
- campo para ingresar una clave API para integrar el servicio Yandex.Disk

## Herramientas a integrar
Integrar el uso de exiftool, ffmpeg, imagick en el caso de que sea muy complejo replicar la funcionalidad requerida en Go