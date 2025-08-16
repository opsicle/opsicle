package tests

import (
	intController "opsicle/internal/controller"
	"opsicle/internal/controller/models"
	pkgController "opsicle/pkg/controller"
	"testing"
)

func TestControllerErrors(t *testing.T) {
	if pkgController.ErrorAuthRequired.Error() != intController.ErrorAuthRequired.Error() {
		t.Errorf("expected ErrorAuthRequired to be consistent across pkg/controller and internal/controller")
	}
	if pkgController.ErrorGeneric.Error() != intController.ErrorGeneric.Error() {
		t.Errorf("expected ErrorGeneric to be consistent across pkg/controller and internal/controller")
	}
	if pkgController.ErrorMfaRequired.Error() != intController.ErrorMfaRequired.Error() {
		t.Errorf("expected ErrorMfaRequired to be consistent across pkg/controller and internal/controller")
	}
}
func TestModelTypes(t *testing.T) {
	if pkgController.MfaTypeTotp != models.MfaTypeTotp {
		t.Errorf("expected MfaTypeTotp to be consistent across pkg/controller and internal/controller/models")
	}
}
