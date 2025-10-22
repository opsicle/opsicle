package cli

import (
	"errors"
	"fmt"
	"opsicle/internal/types"
	"opsicle/pkg/controller"

	"github.com/spf13/cobra"
)

func RequireAuth(controllerUrl string, methodId string, runOnUnauth ...*cobra.Command) (string, error) {
	sessionToken, _, err := controller.GetSessionToken()
	if err != nil {
		if len(runOnUnauth) == 0 {
			return "", ErrorNotAuthenticated
		}
		remediate := runOnUnauth[0]
		if err := remediate.Execute(); err != nil {
			return "", errors.Join(ErrorAuthError, err)
		}
	}

tryAuth:
	client, err := controller.NewClient(controller.NewClientOpts{
		ControllerUrl: controllerUrl,
		BearerAuth: &controller.NewClientBearerAuthOpts{
			Token: sessionToken,
		},
		Id: methodId,
	})
	if err != nil {
		if errors.Is(err, types.ErrorInvalidInput) {
			return "", errors.Join(ErrorClientUnavailable, err)
		} else if errors.Is(err, types.ErrorHealthcheckFailed) {
			return "", errors.Join(ErrorControllerUnavailable, err)
		} else if len(runOnUnauth) > 0 {
			remediate := runOnUnauth[0]
			if err := remediate.Execute(); err != nil {
				return "", errors.Join(ErrorAuthError, err)
			}
			goto tryAuth
		}
		return "", err
	}

	output, err := client.ValidateSessionV1()
	if err != nil || output.Data.IsExpired {
		if err := controller.DeleteSessionToken(); err != nil {
			fmt.Printf("⚠️ We failed to remove the session token for you, please do it yourself\n")
		}
		fmt.Println("⚠️  Please login again using `opsicle login`")
		return "", fmt.Errorf("re-authentication needed")
	}

	return sessionToken, nil
}
