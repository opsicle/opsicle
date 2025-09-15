package cli

import "fmt"

const (
	TimestampHuman       string = "Jan 2 2006 03:04:05 PM"
	TimestampSystem      string = "2006-01-02T15:04:05"
	TimestampCertificate string = "2006-01-02T15:04:05-0700"
)

const (
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
