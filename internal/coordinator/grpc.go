package coordinator

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"opsicle/internal/common"
	"opsicle/internal/coordinator/pb"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/keepalive"
)

type GrpcServer struct {
	Addr     string
	Done     chan struct{}
	Errors   chan error
	Instance *grpc.Server
	Cert     tls.Certificate
	CaPem    []byte

	ServiceLogs chan<- common.ServiceLog
}

func (g *GrpcServer) Listen() {
	grpcAddr := g.Addr
	grpcListener, err := net.Listen("tcp", grpcAddr)
	if err != nil {
		panic(fmt.Errorf("failed to start grpc listener: %w", err))
	}
	g.Done = make(chan struct{})
	caPool := x509.NewCertPool()
	if ok := caPool.AppendCertsFromPEM(g.CaPem); !ok {
		panic(fmt.Errorf("failed to create ca pool"))
	}
	serveTls := credentials.NewTLS(&tls.Config{
		Certificates: []tls.Certificate{g.Cert}, // your x509 cert
		MinVersion:   tls.VersionTLS13,
		ClientCAs:    caPool,
		RootCAs:      caPool,
		ClientAuth:   tls.RequestClientCert,
		VerifyPeerCertificate: func(rawCerts [][]byte, _ [][]*x509.Certificate) error {
			fmt.Println("verifyPeerCertificate")
			cert, _ := x509.ParseCertificate(rawCerts[0])
			subject := cert.Subject
			clientName := subject.CommonName
			orgId := subject.Organization[0]
			orgToken := subject.Organization[1]
			fmt.Printf("connection from client[%s] via org[%s] and token[%s]\n", clientName, orgId, orgToken)
			for _, uri := range cert.URIs {
				if uri.Scheme == "spiffe" && uri.String() == "spiffe://opsicle/ns/default/sa/worker" {
					return nil
				}
			}
			return nil
		},
	})
	g.Instance = grpc.NewServer(
		grpc.Creds(serveTls),
		grpc.MaxConcurrentStreams(3),
		grpc.KeepaliveParams(keepalive.ServerParameters{
			Time:    2 * time.Second,
			Timeout: 500 * time.Millisecond,
		}),
	)
	coordinatorServiceInstance := &coordinatorService{}
	pb.RegisterCoordinatorServiceServer(g.Instance, coordinatorServiceInstance)
	done := g.Done
	go func() {
		defer close(done)
		g.ServiceLogs <- common.ServiceLogf(common.LogLevelDebug, "starting grpc listener...")
		if serveErr := g.Instance.Serve(grpcListener); serveErr != nil {
			g.Errors <- fmt.Errorf("failed to start grpc server: %w", serveErr)
		}
	}()
}
