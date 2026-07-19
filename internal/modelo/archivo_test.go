package modelo

import "testing"

func TestTipoDesdeRutaReconoceExtensionesRawComoImagen(t *testing.T) {
	t.Parallel()

	casos := []string{
		"/tmp/foto.cr2",
		"/tmp/foto.cr3",
		"/tmp/foto.nef",
		"/tmp/foto.arw",
		"/tmp/foto.dng",
		"/tmp/foto.raf",
	}
	for _, ruta := range casos {
		if tipo := TipoDesdeRuta(ruta, false); tipo != TipoImagen {
			t.Fatalf("la ruta %q debería clasificarse como imagen, se obtuvo %q", ruta, tipo)
		}
	}
}

func TestTipoDesdeRutaReconoceWebMComoVideo(t *testing.T) {
	t.Parallel()

	if tipo := TipoDesdeRuta("/tmp/clip-prueba.webm", false); tipo != TipoVideo {
		t.Fatalf("la ruta .webm debería clasificarse como video, se obtuvo %q", tipo)
	}
}

func TestArchivoAdmitePreviewParaFormatosEspeciales(t *testing.T) {
	t.Parallel()

	casos := []struct {
		nombre  string
		archivo Archivo
		espera  bool
	}{
		{
			nombre: "pdf local",
			archivo: Archivo{
				Ruta: "/tmp/documento.pdf",
				Tipo: TipoOtro,
			},
			espera: true,
		},
		{
			nombre: "psd local",
			archivo: Archivo{
				Ruta: "/tmp/diseno.psd",
				Tipo: TipoOtro,
			},
			espera: true,
		},
		{
			nombre: "psd remoto",
			archivo: Archivo{
				Origen:     OrigenYandex,
				Ruta:       "disk:/diseno.psd",
				PreviewURL: "https://preview/yandex",
				Tipo:       TipoOtro,
			},
			espera: true,
		},
		{
			nombre: "txt remoto sin preview",
			archivo: Archivo{
				Origen: OrigenYandex,
				Ruta:   "disk:/nota.txt",
				Tipo:   TipoOtro,
			},
			espera: false,
		},
	}

	for _, caso := range casos {
		if obtenido := caso.archivo.AdmitePreview(); obtenido != caso.espera {
			t.Fatalf("%s: valor inesperado, se obtuvo %v", caso.nombre, obtenido)
		}
	}
}
