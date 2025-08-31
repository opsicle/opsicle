package cli

import "fmt"

const (
	AsciiBlue   string = "33"
	AsciiRed    string = "9"
	AsciiYellow string = "11"
	AsciiGreen  string = "2"
	AsciiGray   string = "239"

	FlagTypeBool        FlagType = "bool"
	FlagTypeDuration    FlagType = "duration"
	FlagTypeFloat       FlagType = "float"
	FlagTypeInteger     FlagType = "integer"
	FlagTypeString      FlagType = "string"
	FlagTypeStringSlice FlagType = "stringslice"
)

const (
	Logo = "" +
		`   ___   ___  ___  ___  ___  _     ___ ` + "\n" +
		`  / _ \ | _ \/ __||_ _|/ __|| |   | __|` + "\n" +
		` | (_) ||  _/\__ \ | || (__ | |__ | _| ` + "\n" +
		`  \___/ |_|  |___/|___|\___||____||___|` + "\n"
)

func PrintLogo() {
	fmt.Println(Logo)
}
