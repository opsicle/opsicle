package cli

import (
	"errors"
	"fmt"
	"opsicle/pkg/controller"
)

func RequireAuth(controllerUrl string, methodId string) (string, error) {
	sessionToken, _, err := controller.GetSessionToken()
	if err != nil {
		fmt.Println("⚠️ You must be logged-in to run this command")
		return "", fmt.Errorf("not authenticated")
	}

	client, err := controller.NewClient(controller.NewClientOpts{
		ControllerUrl: controllerUrl,
		BearerAuth: &controller.NewClientBearerAuthOpts{
			Token: sessionToken,
		},
		Id: methodId,
	})
	if err != nil {
		if errors.Is(err, controller.ErrorConnectionRefused) {
			return "", fmt.Errorf("unexpected error: %w", err)
		}
		return "", fmt.Errorf("unexpected error: %w", err)
	}

	output, err := client.ValidateSessionV1()
	if err != nil || output.Data.IsExpired {
		if err := controller.DeleteSessionToken(); err != nil {
			fmt.Printf("⚠️ We failed to remove the session token for you, please do it yourself\n")
		}
		fmt.Println("⚠️ Please login again using `opsicle login`")
		return "", fmt.Errorf("re-authentication needed")
	}

	return sessionToken, nil
}
