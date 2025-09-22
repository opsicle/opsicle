package tests

import (
	intController "opsicle/internal/controller"
	"opsicle/internal/controller/models"
	pkgController "opsicle/pkg/controller"
	"testing"
)

func validateSimilarty(name string, i, j error, t *testing.T) {
	if i.Error() != j.Error() {
		t.Errorf("expected %s to be consistent across pkg/controller and internal/controller", name)
	}
}

func TestControllerErrors(t *testing.T) {
	validateSimilarty("ErrorAuthRequired", pkgController.ErrorAuthRequired, intController.ErrorAuthRequired, t)
	validateSimilarty("ErrorEmailExists", pkgController.ErrorEmailExists, intController.ErrorEmailExists, t)
	validateSimilarty("ErrorGeneric", pkgController.ErrorGeneric, intController.ErrorGeneric, t)
	validateSimilarty("ErrorNotFound", pkgController.ErrorNotFound, intController.ErrorNotFound, t)
	validateSimilarty("ErrorMfaRequired", pkgController.ErrorMfaRequired, intController.ErrorMfaRequired, t)
	validateSimilarty("ErrorOrgRequiresOneAdmin", pkgController.ErrorOrgRequiresOneAdmin, intController.ErrorOrgRequiresOneAdmin, t)
}

func TestModelTypes(t *testing.T) {
	if pkgController.MfaTypeTotp != models.MfaTypeTotp {
		t.Errorf("expected MfaTypeTotp to be consistent across pkg/controller and internal/controller/models")
	}
}
