package certificate

import (
	"bytes"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"opsicle/internal/cli"
	"opsicle/internal/tls"
	"strings"

	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var flags cli.Flags = cli.Flags{
	{
		Name:         "cert-path",
		Short:        'c',
		DefaultValue: "",
		Usage:        "Defines the path to a .crt file to check",
		Type:         cli.FlagTypeString,
	},
	{
		Name:         "key-path",
		Short:        'k',
		DefaultValue: "",
		Usage:        "Defines the path to a .key file to check",
		Type:         cli.FlagTypeString,
	},
}

func init() {
	flags.AddToCommand(Command)
}

var Command = &cobra.Command{
	Use:     "certificate",
	Aliases: []string{"cert"},
	Short:   "Checks a TLS certificate that's meant for server/client use",
	PreRun: func(cmd *cobra.Command, args []string) {
		flags.BindViper(cmd)
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		certPath := viper.GetString("cert-path")
		if certPath == "" {
			return fmt.Errorf("invalid cert path")
		}
		keyPath := viper.GetString("key-path")
		if keyPath == "" {
			return fmt.Errorf("invalid key path")
		}
		cert, err := tls.LoadCertificate(certPath, keyPath)
		if err != nil {
			return fmt.Errorf("load cert: %w", err)
		}

		switch viper.GetString("output") {
		case "json":
			o, _ := json.MarshalIndent(cert.X509Certificate, "", "  ")
			fmt.Println(string(o))
		case "text":
			fallthrough
		default:
			var tableData bytes.Buffer
			table := tablewriter.NewTable(&tableData)
			table.Header([]string{"property", "value"})
			table.Append([]string{"serial number", cert.X509Certificate.SerialNumber.String()})
			table.Append([]string{"dnsname(s)", fmt.Sprintf(`["%s"]`, strings.Join(cert.X509Certificate.DNSNames, `", "`))})
			ips := []string{}
			for _, ip := range cert.X509Certificate.IPAddresses {
				ips = append(ips, ip.String())
			}
			sprintfFormat := `[%s]`
			if len(ips) > 0 {
				sprintfFormat = `["%s"]`
			}
			table.Append([]string{"ip(s)", fmt.Sprintf(sprintfFormat, strings.Join(ips, `", "`))})
			table.Append([]string{"valid", fmt.Sprintf("%v", cert.X509Certificate.BasicConstraintsValid)})
			table.Append([]string{"not before", cert.X509Certificate.NotBefore.Format("2006-01-02T15:04:05-0700")})
			table.Append([]string{"not after", cert.X509Certificate.NotAfter.Format("2006-01-02T15:04:05-0700")})
			certSha1 := sha1.Sum(cert.X509Certificate.Raw)
			table.Append([]string{"sha1 fingerprint", strings.ToUpper(hex.EncodeToString(certSha1[:]))})
			certSha256 := sha256.Sum256(cert.X509Certificate.Raw)
			table.Append([]string{"sha256 fingerprint", strings.ToUpper(hex.EncodeToString(certSha256[:]))})

			table.Append([]string{"-"})

			table.Append([]string{"issuer cn", string(cert.X509Certificate.Issuer.CommonName)})
			sprintfFormat = `[%s]`
			if len(cert.X509Certificate.Issuer.Organization) > 0 {
				sprintfFormat = `["%s"]`
			}
			table.Append([]string{"issuer org(s)", fmt.Sprintf(sprintfFormat, strings.Join(cert.X509Certificate.Issuer.Organization, `", "`))})
			sprintfFormat = `[%s]`
			if len(cert.X509Certificate.Issuer.OrganizationalUnit) > 0 {
				sprintfFormat = `["%s"]`
			}
			table.Append([]string{"subject ou(s)", fmt.Sprintf(sprintfFormat, strings.Join(cert.X509Certificate.Issuer.OrganizationalUnit, `", "`))})
			issuerNames := []string{}
			for _, sn := range cert.X509Certificate.Issuer.Names {
				issuerNames = append(issuerNames, sn.Value.(string))
			}
			sprintfFormat = `[%s]`
			if len(issuerNames) > 0 {
				sprintfFormat = `["%s"]`
			}
			table.Append([]string{"subject name(s)", fmt.Sprintf(sprintfFormat, strings.Join(issuerNames, `", "`))})
			issuerExtraNames := []string{}
			for _, sn := range cert.X509Certificate.Issuer.ExtraNames {
				issuerExtraNames = append(issuerExtraNames, sn.Value.(string))
			}
			sprintfFormat = `[%s]`
			if len(issuerNames) > 0 {
				sprintfFormat = `["%s"]`
			}
			table.Append([]string{"subject extra name(s)", fmt.Sprintf(sprintfFormat, strings.Join(issuerExtraNames, `", "`))})

			table.Append([]string{"-"})

			table.Append([]string{"subject cn", string(cert.X509Certificate.Subject.CommonName)})
			sprintfFormat = `[%s]`
			if len(cert.X509Certificate.Subject.Organization) > 0 {
				sprintfFormat = `["%s"]`
			}
			table.Append([]string{"subject org(s)", fmt.Sprintf(sprintfFormat, strings.Join(cert.X509Certificate.Subject.Organization, `", "`))})
			sprintfFormat = `[%s]`
			if len(cert.X509Certificate.Subject.OrganizationalUnit) > 0 {
				sprintfFormat = `["%s"]`
			}
			table.Append([]string{"subject ou(s)", fmt.Sprintf(sprintfFormat, strings.Join(cert.X509Certificate.Subject.OrganizationalUnit, `", "`))})
			subjectNames := []string{}
			for _, sn := range cert.X509Certificate.Subject.Names {
				subjectNames = append(subjectNames, sn.Value.(string))
			}
			sprintfFormat = `[%s]`
			if len(subjectNames) > 0 {
				sprintfFormat = `["%s"]`
			}
			table.Append([]string{"subject name(s)", fmt.Sprintf(sprintfFormat, strings.Join(subjectNames, `", "`))})
			subjectExtraNames := []string{}
			for _, sn := range cert.X509Certificate.Subject.ExtraNames {
				subjectExtraNames = append(subjectExtraNames, sn.Value.(string))
			}
			sprintfFormat = `[%s]`
			if len(subjectExtraNames) > 0 {
				sprintfFormat = `["%s"]`
			}
			table.Append([]string{"subject extra name(s)", fmt.Sprintf(sprintfFormat, strings.Join(subjectExtraNames, `", "`))})

			table.Append([]string{"-"})

			publicKey := cert.X509Certificate.PublicKey.(*rsa.PublicKey)
			table.Append([]string{"key length", fmt.Sprintf("%v", publicKey.N.BitLen())})
			keyUsages := tls.GetUsage(cert.X509Certificate)
			table.Append([]string{"key usages", strings.Join(keyUsages, "\n")})
			publicDer, err := x509.MarshalPKIXPublicKey(publicKey)
			if err != nil {
				publicDer = nil
			}
			publicPem := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: publicDer})
			table.Append([]string{"public key", string(publicPem)})
			table.Render()
			fmt.Println(tableData.String())
		}
		return nil
	},
}
