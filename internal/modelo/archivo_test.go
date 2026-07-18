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
