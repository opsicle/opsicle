package certificate_authority

import (
	"fmt"
	"opsicle/internal/cli"
	"opsicle/internal/tls"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var flags cli.Flags = cli.Flags{
	{
		Name:         "output-dir",
		Short:        'O',
		DefaultValue: "./data/.app/certs/",
		Usage:        "Defines the path to output the .crt and .key file to",
		Type:         cli.FlagTypeString,
	},
	{
		Name:         "cn",
		DefaultValue: "Default CN",
		Usage:        "Defines the CommonName",
		Type:         cli.FlagTypeString,
	},
	{
		Name:         "org",
		DefaultValue: []string{"Default Org", "Default Org Alias"},
		Usage:        "Defines the Organization(s) field, define this multiple times to add more Organizations",
		Type:         cli.FlagTypeStringSlice,
	},
	{
		Name:         "bits",
		DefaultValue: 4096,
		Usage:        "Defines the number of bits to use for the CA key",
		Type:         cli.FlagTypeInteger,
	},
}

func init() {
	flags.AddToCommand(Command)
}

var Command = &cobra.Command{
	Use:     "certificate-authority",
	Aliases: []string{"cert-authority", "ca"},
	Short:   "Creates a TLS Certificate Authority for signing server/client certificates",
	PreRun: func(cmd *cobra.Command, args []string) {
		flags.BindViper(cmd)
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		name := "default"
		if len(args) > 0 {
			name = args[0]
		}
		commonName := viper.GetString("cn")
		organizations := viper.GetStringSlice("org")
		keyBits := viper.GetInt("bits")

		fmt.Printf("⏳ Generating a new CA certificate/key pair...\n")
		cert, key, err := tls.GenerateCertificateAuthority(&tls.CertificateOptions{
			CommonName:   commonName,
			Organization: organizations,
			KeyBits:      keyBits,
		})
		if err != nil {
			return fmt.Errorf("failed to generate ca: %w", err)
		}

		outputDir := viper.GetString("output-dir")

		certPath, err := cert.Export(outputDir, name+"-ca")
		if err != nil {
			return fmt.Errorf("failed to export ca cert: %w", err)
		}
		keyPath, err := key.Export(outputDir, name+"-ca")
		if err != nil {
			return fmt.Errorf("failed to export ca key: %w", err)
		}

		cli.PrintBoxedSuccessMessage(
			fmt.Sprintf(
				"📄 CA certificate is at: %s\n"+
					"🔑 CA key is at: %s\n",
				certPath,
				keyPath,
			),
		)

		return nil
	},
}
