package sqlite

const (
	pragmasSQLite = `
PRAGMA journal_mode = WAL;
PRAGMA synchronous = NORMAL;
PRAGMA foreign_keys = ON;
PRAGMA temp_store = MEMORY;
PRAGMA busy_timeout = 5000;
PRAGMA cache_size = -65536;
PRAGMA mmap_size = 268435456;
`

	esquemaSQLite = `
CREATE TABLE IF NOT EXISTS archivos (
	ruta TEXT PRIMARY KEY,
	origen TEXT NOT NULL,
	ruta_padre TEXT NOT NULL,
	nombre TEXT NOT NULL,
	tamano INTEGER NOT NULL,
	modificado_unix INTEGER NOT NULL,
	tipo TEXT NOT NULL,
	es_oculto INTEGER NOT NULL,
	es_directorio INTEGER NOT NULL,
	ancho INTEGER NOT NULL DEFAULT 0,
	alto INTEGER NOT NULL DEFAULT 0,
	duracion_ms INTEGER NOT NULL DEFAULT 0,
	metadatos_json TEXT NOT NULL DEFAULT '{}',
	hash_md5 TEXT NOT NULL DEFAULT '',
	hash_sha256 TEXT NOT NULL DEFAULT '',
	hash_dhash_imagen TEXT NOT NULL DEFAULT '',
	hash_dhash_video TEXT NOT NULL DEFAULT '',
	tiene_gps INTEGER NOT NULL DEFAULT 0,
	tiene_regiones INTEGER NOT NULL DEFAULT 0,
	tiene_where_froms INTEGER NOT NULL DEFAULT 0,
	tiene_ia INTEGER NOT NULL DEFAULT 0,
	tiene_social INTEGER NOT NULL DEFAULT 0,
	es_adulto INTEGER NOT NULL DEFAULT 0,
	ubicacion TEXT NOT NULL DEFAULT '',
	where_froms TEXT NOT NULL DEFAULT '',
	ultima_revision_unix INTEGER NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_archivos_ruta_padre ON archivos (ruta_padre);
CREATE INDEX IF NOT EXISTS idx_archivos_tipo ON archivos (tipo);
CREATE INDEX IF NOT EXISTS idx_archivos_sha256 ON archivos (hash_sha256);
CREATE INDEX IF NOT EXISTS idx_archivos_md5 ON archivos (hash_md5);
CREATE INDEX IF NOT EXISTS idx_archivos_dhash_imagen ON archivos (hash_dhash_imagen);
CREATE INDEX IF NOT EXISTS idx_archivos_dhash_video ON archivos (hash_dhash_video);
CREATE INDEX IF NOT EXISTS idx_archivos_ubicacion ON archivos (ubicacion);
CREATE INDEX IF NOT EXISTS idx_archivos_origen ON archivos (origen);
CREATE INDEX IF NOT EXISTS idx_archivos_nombre ON archivos (nombre COLLATE NOCASE);

CREATE TABLE IF NOT EXISTS palabras_clave (
	ruta TEXT NOT NULL,
	palabra TEXT NOT NULL,
	PRIMARY KEY (ruta, palabra),
	FOREIGN KEY (ruta) REFERENCES archivos (ruta) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_palabras_clave_palabra ON palabras_clave (palabra);

CREATE TABLE IF NOT EXISTS etiquetas (
	ruta TEXT NOT NULL,
	etiqueta TEXT NOT NULL,
	PRIMARY KEY (ruta, etiqueta),
	FOREIGN KEY (ruta) REFERENCES archivos (ruta) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_etiquetas_etiqueta ON etiquetas (etiqueta);

CREATE TABLE IF NOT EXISTS ubicaciones_relaciones (
	origen TEXT NOT NULL,
	origen_normalizado TEXT NOT NULL PRIMARY KEY,
	destino TEXT NOT NULL,
	destino_normalizado TEXT NOT NULL,
	actualizado_unix INTEGER NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_ubicaciones_relaciones_destino ON ubicaciones_relaciones (destino_normalizado);
`
)
