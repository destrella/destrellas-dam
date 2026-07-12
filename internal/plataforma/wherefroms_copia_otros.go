//go:build !darwin

package plataforma

import "context"

// CopiarWhereFroms fuera de macOS no necesita trabajo adicional.
func CopiarWhereFroms(_ context.Context, _, _ string) error {
	return nil
}
