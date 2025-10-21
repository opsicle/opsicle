package coordinator

import (
	"opsicle/internal/common"
	"opsicle/internal/coordinator/pb"

	"google.golang.org/grpc"
)

type coordinatorService struct {
	pb.UnimplementedCoordinatorServiceServer
	IsShuttingDown bool
	ServiceLog     chan<- common.ServiceLog
}

func (svc *coordinatorService) GetNextJob(req *pb.GetNextJobRequest, stream grpc.ServerStreamingServer[pb.GetNextJobResponse]) error {
	svc.ServiceLog <- common.ServiceLogf(common.LogLevelDebug, "GetNextJob triggered")
	return svc.UnimplementedCoordinatorServiceServer.GetNextJob(req, stream)
}

func (svc *coordinatorService) UpdateExecution(req *pb.UpdateExecutionRequest, stream grpc.ServerStreamingServer[pb.UpdateExecutionResponse]) error {
	svc.ServiceLog <- common.ServiceLogf(common.LogLevelDebug, "UpdateExecution triggered")
	return svc.UnimplementedCoordinatorServiceServer.UpdateExecution(req, stream)
}
