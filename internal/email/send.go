package email

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"mime/multipart"
	"mime/quotedprintable"
	"net/smtp"
	"net/textproto"
	"opsicle/internal/common"
	"strings"
)

type SendSmtpOpts struct {
	To     []User
	Cc     []User
	Bcc    []User
	Sender User

	Smtp        SmtpConfig
	Message     Message
	ServiceLogs chan<- common.ServiceLog
}

type Message struct {
	Body   []byte
	Title  string
	Images map[string]MessageAttachment
}

type MessageAttachment struct {
	Data []byte
	Type string
}

type User struct {
	Address string
	Name    string
}

type SmtpConfig struct {
	Hostname string
	Port     int
	Username string
	Password string
}

func (o SendSmtpOpts) Validate() error {
	errs := []error{}

	if o.To == nil {
		errs = append(errs, fmt.Errorf("missing receivers"))
	} else {
		for receiverIndex, receiver := range o.To {
			if receiver.Address == "" {
				errs = append(errs, fmt.Errorf("missing receiver address for receiver[%v]", receiverIndex))
			}
		}
	}
	if o.Sender.Address == "" {
		errs = append(errs, fmt.Errorf("missing sender address"))
	}
	if o.Message.Title == "" {
		errs = append(errs, fmt.Errorf("missing message title"))
	}
	if o.Message.Body == nil || string(o.Message.Body) == "" {
		errs = append(errs, fmt.Errorf("missing message body"))
	}
	if o.Smtp.Hostname == "" {
		errs = append(errs, fmt.Errorf("missing smtp hostname"))
	}
	if o.Smtp.Port == 0 {
		errs = append(errs, fmt.Errorf("missing smtp port"))
	}
	if o.Smtp.Username == "" {
		errs = append(errs, fmt.Errorf("missing smtp username"))
	}
	if o.Smtp.Password == "" {
		errs = append(errs, fmt.Errorf("missing smtp password"))
	}

	if len(errs) > 0 {
		errs = append([]error{fmt.Errorf("SendSmtpOpts validation failed")}, errs...)
		return errors.Join(errs...)
	}
	return nil
}

func SendSmtp(opts SendSmtpOpts) error {
	var serviceLogs chan<- common.ServiceLog = nil
	if opts.ServiceLogs == nil {
		noopServiceLogs := make(chan common.ServiceLog, 32)
		serviceLogs = noopServiceLogs
		go func() {
			if _, ok := <-noopServiceLogs; !ok {
				return
			}
		}()
	} else {
		serviceLogs = opts.ServiceLogs
	}

	if err := opts.Validate(); err != nil {
		return fmt.Errorf("failed to validate input to Send: %s", err)
	}

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	// Headers
	headers := make(map[string]string)
	from := opts.Sender.Address
	if opts.Sender.Name != "" {
		from = fmt.Sprintf("%s <%s>", opts.Sender.Name, from)
	}
	to := []string{}
	toAddresses := []string{}
	for _, receiver := range opts.To {
		toAddresses = append(toAddresses, receiver.Address)
		receiverInstance := receiver.Address
		if receiver.Name != "" {
			receiverInstance = fmt.Sprintf("%s <%s>", receiver.Name, receiverInstance)
		}
		to = append(to, receiverInstance)
	}
	cc := []string{}
	ccAddresses := []string{}
	for _, receiver := range opts.Cc {
		ccAddresses = append(ccAddresses, receiver.Address)
		receiverInstance := receiver.Address
		if receiver.Name != "" {
			receiverInstance = fmt.Sprintf("%s <%s>", receiver.Name, receiverInstance)
		}
		cc = append(cc, receiverInstance)
	}
	bcc := []string{}
	bccAddresses := []string{}
	for _, receiver := range opts.Bcc {
		bccAddresses = append(bccAddresses, receiver.Address)
		receiverInstance := receiver.Address
		if receiver.Name != "" {
			receiverInstance = fmt.Sprintf("%s <%s>", receiver.Name, receiverInstance)
		}
		bcc = append(bcc, receiverInstance)
	}
	headers["From"] = from
	headers["To"] = strings.Join(to, ",")
	if len(cc) > 0 {
		headers["Cc"] = strings.Join(cc, ",")
	}
	headers["Subject"] = opts.Message.Title
	headers["MIME-Version"] = "1.0"
	headers["Content-Type"] = "multipart/related; boundary=" + writer.Boundary()

	for k, v := range headers {
		fmt.Fprintf(&buf, "%s: %s\r\n", k, v)
	}
	fmt.Fprint(&buf, "\r\n")
	serviceLogs <- common.ServiceLogf(common.LogLevelDebug, "added headers")

	// HTML Part
	htmlPart, _ := writer.CreatePart(map[string][]string{
		"Content-Type":              {"text/html; charset=UTF-8"},
		"Content-Transfer-Encoding": {"quoted-printable"},
	})
	qp := quotedprintable.NewWriter(htmlPart)
	qp.Write(opts.Message.Body)
	qp.Close()
	serviceLogs <- common.ServiceLogf(common.LogLevelDebug, "added message body of size[%v bytes]", len(opts.Message.Body))

	// Image Part
	for imageFilename, imageContent := range opts.Message.Images {
		serviceLogs <- common.ServiceLogf(common.LogLevelDebug, "creating cid-style attachment for file[%s] of type[%s] and size[%v bytes]", imageFilename, imageContent.Type, len(imageContent.Data))
		imageHeader := make(textproto.MIMEHeader)
		imageHeader.Set("Content-Type", imageContent.Type)
		imageHeader.Set("Content-Transfer-Encoding", "base64")
		imageHeader.Set("Content-ID", fmt.Sprintf("<%s>", imageFilename))
		imageHeader.Set("Content-Disposition", fmt.Sprintf("inline; filename=\"%s\"", imageFilename))
		imagePart, _ := writer.CreatePart(imageHeader)
		encoded := make([]byte, base64.StdEncoding.EncodedLen(len(imageContent.Data)))
		base64.StdEncoding.Encode(encoded, imageContent.Data)
		for i := 0; i < len(encoded); i += 76 {
			end := i + 76
			if end > len(encoded) {
				end = len(encoded)
			}
			imagePart.Write(encoded[i:end])
			imagePart.Write([]byte("\r\n"))
		}
	}
	writer.Close()
	serviceLogs <- common.ServiceLogf(common.LogLevelDebug, "message composed successfully")

	// Compose full message with MIME headers
	smtpAddr := fmt.Sprintf("%s:%v", opts.Smtp.Hostname, opts.Smtp.Port)
	auth := smtp.PlainAuth("", opts.Smtp.Username, opts.Smtp.Password, opts.Smtp.Hostname)
	allReceipients := append([]string{}, toAddresses...)
	allReceipients = append(allReceipients, ccAddresses...)
	allReceipients = append(allReceipients, bccAddresses...)
	if err := smtp.SendMail(
		smtpAddr,
		auth,
		opts.Sender.Address,
		allReceipients,
		buf.Bytes(),
	); err != nil {
		return fmt.Errorf("failed to send email: %s", err)
	}

	serviceLogs <- common.ServiceLogf(common.LogLevelDebug, "email sent successfully to people['%s'] from address[%s]", strings.Join(allReceipients, "', '"), opts.Sender.Address)

	return nil
}
