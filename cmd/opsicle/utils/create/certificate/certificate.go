package certificate

import (
	"fmt"
	"net"
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
		Name:         "ca-cert-path",
		DefaultValue: "",
		Usage:        "Specifies the path to the CA certificate",
		Type:         cli.FlagTypeString,
	},
	{
		Name:         "ca-key-path",
		DefaultValue: "",
		Usage:        "Specifies the path to the CA certificate key",
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
		Name:         "dns-name",
		DefaultValue: []string{"localhost", "default.service"},
		Usage:        "Defines the DNSNames",
		Type:         cli.FlagTypeStringSlice,
	},
	{
		Name:         "ip",
		DefaultValue: []string{"127.0.0.1"},
		Usage:        "Defines the IP addresses",
		Type:         cli.FlagTypeStringSlice,
	},
	{
		Name:         "usage",
		DefaultValue: "client",
		Usage:        "Defines whether this certificate is for a `client` or `server` usage",
		Type:         cli.FlagTypeString,
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
	Use:     "certificate",
	Aliases: []string{"cert"},
	Short:   "Creates a TLS certificate and key for server/client use and places it in the directory provided at --output-dir",
	PreRun: func(cmd *cobra.Command, args []string) {
		flags.BindViper(cmd)
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		name := "default"
		if len(args) > 0 {
			name = args[0]
		}
		caCertPath := viper.GetString("ca-cert-path")
		caKeyPath := viper.GetString("ca-key-path")
		commonName := viper.GetString("cn")
		organizations := viper.GetStringSlice("org")
		keyBits := viper.GetInt("bits")

		var caCert *tls.Certificate
		var caKey *tls.Key

		isCaProvided := caCertPath != "" && caKeyPath != ""

		if !isCaProvided {
			fmt.Printf("⏳ CA not provided, generating a new CA certificate/key pair...\n")
			var err error
			caCert, caKey, err = tls.GenerateCertificateAuthority(&tls.CertificateOptions{
				CommonName:   commonName,
				Organization: organizations,
				KeyBits:      keyBits,
			})
			if err != nil {
				return fmt.Errorf("failed to generate ca: %w", err)
			}
		} else {
			fmt.Printf("⏳ Loading CA certificate from %s...\n", caCertPath)
			fmt.Printf("⏳ Loading CA key from %s...\n", caKeyPath)
			var err error
			caCert, err = tls.LoadCertificate(caCertPath, caKeyPath)
			if err != nil {
				return fmt.Errorf("failed to load ca cert: %w", err)
			}
			caKey, err = tls.LoadKey(caKeyPath)
			if err != nil {
				return fmt.Errorf("failed to load ca key: %w", err)
			}
		}

		isClient := true
		if viper.GetString("usage") == "server" {
			isClient = false
		}
		dnsNames := viper.GetStringSlice("dns-name")
		ips := viper.GetStringSlice("ip")
		ipAddresses := []net.IP{}
		for _, ip := range ips {
			ipAddresses = append(ipAddresses, net.ParseIP(ip))
		}

		cert, key, err := tls.GenerateCertificate(&tls.CertificateOptions{
			CommonName:   commonName,
			Organization: organizations,
			IsClient:     isClient,
			DNSNames:     dnsNames,
			IPs:          ipAddresses,
			KeyBits:      keyBits,
		}, caCert.X509Certificate, caKey.RsaKey)
		if err != nil {
			return fmt.Errorf("failed to generate cert: %w", err)
		}

		outputDir := viper.GetString("output-dir")

		certPath, err := cert.Export(outputDir, name)
		if err != nil {
			return fmt.Errorf("failed to export cert: %w", err)
		}
		keyPath, err := key.Export(outputDir, name)
		if err != nil {
			return fmt.Errorf("failed to export key: %w", err)
		}
		outputMessage := fmt.Sprintf(
			"📄 Leaf certificate is at: %s\n"+
				"🔑 Leaf key is at: %s",
			certPath,
			keyPath,
		)
		if !isCaProvided {
			caCertPath, err := caCert.Export(outputDir, name+"-ca")
			if err != nil {
				return fmt.Errorf("failed to export cert: %w", err)
			}
			caKeyPath, err := caKey.Export(outputDir, name+"-ca")
			if err != nil {
				return fmt.Errorf("failed to export key: %w", err)
			}
			outputMessage = fmt.Sprintf(
				"📄 CA certificate is at: %s\n"+
					"🔑 CA key is at: %s\n%s",
				caCertPath,
				caKeyPath,
				outputMessage,
			)
		}

		cli.PrintBoxedSuccessMessage(outputMessage)

		return nil
	},
}
