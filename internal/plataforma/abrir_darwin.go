//go:build darwin

package plataforma

import (
	"context"
	"os/exec"
)

// AbrirEnSistema delega en LaunchServices para abrir un archivo con la app predeterminada.
func AbrirEnSistema(ctx context.Context, ruta string) error {
	return exec.CommandContext(ctx, "open", ruta).Run()
}
