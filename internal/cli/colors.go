package cli

type AnsiColor string

// ref https://hexdocs.pm/color_palette/ansi_color_codes.html
const (
	AnsiBlack     AnsiColor = "0"
	AnsiWhite     AnsiColor = "7"
	AnsiGray      AnsiColor = "239"
	AnsiDarkGray  AnsiColor = "235"
	AnsiLightGray AnsiColor = "247"

	AnsiRed     AnsiColor = "9"
	AnsiOrange  AnsiColor = "166"
	AnsiYellow  AnsiColor = "3"
	AnsiGreen   AnsiColor = "2"
	AnsiBlue    AnsiColor = "33"
	AnsiPurple  AnsiColor = "54"
	AnsiIndigo  AnsiColor = AnsiPurple
	AnsiPink    AnsiColor = "13"
	AnsiMagenta AnsiColor = AnsiPink
)
