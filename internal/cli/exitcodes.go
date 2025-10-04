package cli

var ExitCode = 0

var (
	ExitCodeError              = 0b1
	ExitCodeDbError            = 0b10
	ExitCodeInputError         = 0b100
	ExitCodeServiceUnavailable = 0b1000
)
