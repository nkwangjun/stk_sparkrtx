package api

import (
	"context"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"net"
	"strconv"
	"strings"
	"supertuxkart/grpcsdk"
	"supertuxkart/gsemanager"
	"supertuxkart/logger"
	"sync"
)

var (
	rpcServerIns *rpcService
	once         sync.Once
)

type rpcService struct {
	healthStatus bool
	grpcPort     int
}

//func GetRpcService() grpcsdk.GameServerGrpcSdkServiceServer {
func GetRpcService() *rpcService {
	once.Do(func() {
		rpcServerIns = new(rpcService)
		rpcServerIns.healthStatus = true
	})

	return rpcServerIns
}

func (s *rpcService) StartGrpcServer() {
	listen, err := net.Listen("tcp", "127.0.0.1:")
	if err != nil {
		logger.Fatal("grpc fail to listen", zap.Error(err))
	}

	addr := listen.Addr().String()
	portStr := strings.Split(addr, ":")[1]
	s.grpcPort, err = strconv.Atoi(portStr)
	if err != nil {
		logger.Fatal("grpc fail to get port", zap.Error(err))
	}

	logger.Info("grpc listen port is", zap.Int("port", s.grpcPort))

	grpcServer := grpc.NewServer()
	grpcsdk.RegisterGameServerGrpcSdkServiceServer(grpcServer, s)
	logger.Info("start grpc server success")
	go grpcServer.Serve(listen)
}

func (s *rpcService) GetGrpcPort() int {
	return s.grpcPort
}

func (s *rpcService) SetHealthStatus(healthStatus bool) {
	s.healthStatus = healthStatus
}

func (s *rpcService) OnHealthCheck(ctx context.Context, req *grpcsdk.HealthCheckRequest) (*grpcsdk.HealthCheckResponse, error) {
	resp := &grpcsdk.HealthCheckResponse{
		HealthStatus: s.healthStatus,
	}

	return resp, nil
}

func (s *rpcService) OnStartGameServerSession(ctx context.Context, req *grpcsdk.StartGameServerSessionRequest) (*grpcsdk.ProcessResponse, error) {
	gseManager := gsemanager.GetGseManager()
	gseManager.SetGameServerSession(req.GameServerSession)
	gseManager.ActivateGameServerSession(req.GameServerSession.GameServerSessionId, req.GameServerSession.MaxPlayers)

	resp := new(grpcsdk.ProcessResponse)

	return resp, nil
}

func (s *rpcService) OnProcessTerminate(ctx context.Context, req *grpcsdk.ProcessTerminateRequest) (*grpcsdk.ProcessResponse, error) {
	gseManager := gsemanager.GetGseManager()
	gseManager.SetTerminationTime(req.TerminationTime)
	//结束游戏会话
	gseManager.TerminateGameServerSession()

	// 进程退出
	gseManager.ProcessEnding()

	resp := new(grpcsdk.ProcessResponse)
	return resp, nil
}
