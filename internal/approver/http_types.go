package approver

type requestLogger func(string, string)

type httpResponse struct {
	Data    any    `json:"data"`
	Message string `json:"message"`
	Success bool   `json:"success"`
}
