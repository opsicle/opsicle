package certificate

import (
	"crypto/rsa"
	"crypto/x509"
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
		DefaultValue: "./certs/",
		Usage:        "Defines the path to output the .crt and .key file to",
		Type:         cli.FlagTypeString,
	},
	{
		Name:         "ca-cert-path",
		DefaultValue: "Path to the CA certificate",
		Usage:        "Specifies the path to the CA certificate",
		Type:         cli.FlagTypeString,
	},
	{
		Name:         "ca-key-path",
		DefaultValue: "Path to the CA certificate key",
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
	Short:   "Creates a TLS certificate for server/client use",
	PreRun: func(cmd *cobra.Command, args []string) {
		flags.BindViper(cmd)
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		caCertPath := viper.GetString("ca-cert-path")
		caKeyPath := viper.GetString("ca-key-path")
		commonName := viper.GetString("cn")
		organizations := viper.GetStringSlice("org")
		keyBits := viper.GetInt("bits")

		var caCert *x509.Certificate
		var caKey *rsa.PrivateKey

		if caCertPath == "" && caKeyPath == "" {
			fmt.Printf("Generating a new CA...\n")
			cert, err := tls.GenerateCertificateAuthority(&tls.CertificateAuthorityOptions{
				CommonName:   commonName,
				Organization: organizations,
				KeyBits:      keyBits,
			})
			if err != nil {
				return fmt.Errorf("failed to generate ca: %w", err)
			}
			caCert = cert.X509Certificate
			caKey = cert.Key
		} else {
			fmt.Printf("Loading CA certificate from %s...\n", caCertPath)
			fmt.Printf("Loading CA key from %s...\n", caKeyPath)
			var err error
			caCert, caKey, err = tls.LoadCertificateAuthority(caCertPath, caKeyPath)
			if err != nil {
				return fmt.Errorf("failed to load ca: %w", err)
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
		cert, err := tls.GenerateCertificate(&tls.CertificateOptions{
			CommonName:   commonName,
			Organization: organizations,
			IsClient:     isClient,
			DNSNames:     dnsNames,
			IPs:          ipAddresses,
			KeyBits:      keyBits,
		}, caCert, caKey)
		if err != nil {
			return fmt.Errorf("failed to generate cert: %w", err)
		}

		outputDir := viper.GetString("output-dir")

		certPath, keyPath, err := tls.ExportCertificate(outputDir, cert)
		if err != nil {
			return fmt.Errorf("failed to export ca: %w", err)
		}

		cli.PrintBoxedSuccessMessage(
			fmt.Sprintf(
				"Certificate is at: %s\n"+
					"Key is at: %s\n",
				certPath,
				keyPath,
			),
		)

		return nil
	},
}
