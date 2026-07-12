//go:build darwin

package plataforma

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

const nombreAtributoWhereFroms = "com.apple.metadata:kMDItemWhereFroms"

// CopiarWhereFroms replica el atributo extendido de procedencia cuando existe.
func CopiarWhereFroms(ctx context.Context, origen, destino string) error {
	salida, err := exec.CommandContext(ctx, "xattr", "-px", nombreAtributoWhereFroms, origen).CombinedOutput()
	if err != nil {
		if _, esErrorSalida := err.(*exec.ExitError); esErrorSalida {
			return nil
		}
		return fmt.Errorf("no se pudo leer el atributo WhereFroms: %w", err)
	}

	hexadecimal := strings.Join(strings.Fields(string(salida)), "")
	if hexadecimal == "" {
		return nil
	}

	escritura, err := exec.CommandContext(ctx, "xattr", "-wx", nombreAtributoWhereFroms, hexadecimal, destino).CombinedOutput()
	if err != nil {
		return fmt.Errorf("no se pudo copiar el atributo WhereFroms: %w: %s", err, strings.TrimSpace(string(escritura)))
	}
	return nil
}
