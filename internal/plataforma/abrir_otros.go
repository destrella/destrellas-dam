//go:build !darwin

package plataforma

import (
	"context"
	"errors"
	"os/exec"
)

// AbrirEnSistema intenta usar el abridor habitual del entorno fuera de macOS.
func AbrirEnSistema(ctx context.Context, ruta string) error {
	comando, err := exec.LookPath("xdg-open")
	if err != nil {
		return errors.New("no existe un comando de apertura predeterminado disponible")
	}
	return exec.CommandContext(ctx, comando, ruta).Run()
}
