package gsemanager

import (
	"context"
	"fmt"
	uuid "github.com/satori/go.uuid"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"os"
	"strconv"
	"supertuxkart/grpcsdk"
	"supertuxkart/logger"
	"sync"
)

var (
	gseManagerIns *gsemanager
	once          sync.Once
)

const (
	localhost = "127.0.0.1"
	agentPort = 5758
)

type gsemanager struct {
	pid               string
	gameServerSession *grpcsdk.GameServerSession
	terminationTime   int64
	rpcClient         grpcsdk.GseGrpcSdkServiceClient
}

func GetGseManager() *gsemanager {
	once.Do(func() {
		gseManagerIns = &gsemanager{
			pid: strconv.Itoa(os.Getpid()),
		}

		url := fmt.Sprintf("%s:%d", localhost, agentPort)

		conn, err := grpc.DialContext(context.Background(), url, grpc.WithInsecure())
		if err != nil {
			logger.Fatal("dail to gse fail", zap.String("url", url), zap.Error(err))
		}

		gseManagerIns.rpcClient = grpcsdk.NewGseGrpcSdkServiceClient(conn)
	})

	return gseManagerIns
}

func (g *gsemanager) SetGameServerSession(gameserversession *grpcsdk.GameServerSession) {
	g.gameServerSession = gameserversession
}

func (g *gsemanager) SetTerminationTime(terminationTime int64) {
	g.terminationTime = terminationTime
}

func (g *gsemanager) getContext() context.Context {
	requestId := uuid.NewV4().String()
	ctx := metadata.AppendToOutgoingContext(context.Background(), "pid", g.pid)
	return metadata.AppendToOutgoingContext(ctx, "requestId", requestId)
}

// 1. ProcessReady
func (g *gsemanager) ProcessReady(logPath []string, clientPort int32, grpcPort int32) error {
	logger.Info("start to processready", zap.Any("logPath", logPath), zap.Int32("clientPort", clientPort),
		zap.Int32("grpcPort", grpcPort))
	req := &grpcsdk.ProcessReadyRequest{
		LogPathsToUpload: logPath,
		ClientPort:       clientPort,
		GrpcPort:         grpcPort,
	}

	_, err := g.rpcClient.ProcessReady(g.getContext(), req)
	if err != nil {
		logger.Info("ProcessReady fail", zap.Error(err))
		return err
	}

	logger.Info("ProcessReady success")
	return nil
}

// 2. ActivateGameServerSession
func (g *gsemanager) ActivateGameServerSession(gameServerSessionId string, maxPlayers int32) error {
	logger.Info("start to ActivateGameServerSession", zap.String("gameServerSessionId", gameServerSessionId),
		zap.Int32("maxPlayers", maxPlayers))
	req := &grpcsdk.ActivateGameServerSessionRequest{
		GameServerSessionId: gameServerSessionId,
		MaxPlayers:          maxPlayers,
	}

	_, err := g.rpcClient.ActivateGameServerSession(g.getContext(), req)
	if err != nil {
		logger.Error("ActivateGameServerSession fail", zap.Error(err))
		return err
	}

	logger.Info("ActivateGameServerSession success")
	return nil
}

// 3. AcceptPlayerSession
func (g *gsemanager) AcceptPlayerSession(playerSessionId string) (*grpcsdk.AuxProxyResponse, error) {
	logger.Info("start to AcceptPlayerSession", zap.String("playerSessionId", playerSessionId))
	req := &grpcsdk.AcceptPlayerSessionRequest{
		GameServerSessionId: g.gameServerSession.GameServerSessionId,
		PlayerSessionId:     playerSessionId,
	}

	return g.rpcClient.AcceptPlayerSession(g.getContext(), req)
}

// 4. RemovePlayerSession
func (g *gsemanager) RemovePlayerSession(playerSessionId string) (*grpcsdk.AuxProxyResponse, error) {
	logger.Info("start to RemovePlayerSession", zap.String("playerSessionId", playerSessionId))
	req := &grpcsdk.RemovePlayerSessionRequest{
		GameServerSessionId: g.gameServerSession.GameServerSessionId,
		PlayerSessionId:     playerSessionId,
	}

	return g.rpcClient.RemovePlayerSession(g.getContext(), req)
}

// 5. TerminateGameServerSession
func (g *gsemanager) TerminateGameServerSession() (*grpcsdk.AuxProxyResponse, error) {
	logger.Info("start to TerminateGameServerSession")
	req := &grpcsdk.TerminateGameServerSessionRequest{
		GameServerSessionId: g.gameServerSession.GameServerSessionId,
	}

	return g.rpcClient.TerminateGameServerSession(g.getContext(), req)
}

// 6. ProcessEnding
func (g *gsemanager) ProcessEnding() (*grpcsdk.AuxProxyResponse, error) {
	logger.Info("start to ProcessEnding")
	req := &grpcsdk.ProcessEndingRequest{}

	return g.rpcClient.ProcessEnding(g.getContext(), req)
}

// 7. DescribePlayerSessions
func (g *gsemanager) DescribePlayerSessions(gameServerSessionId, playerId, playerSessionId, playerSessionStatusFilter, nextToken string,
	limit int32) (*grpcsdk.DescribePlayerSessionsResponse, error) {
	logger.Info("start to DescribePlayerSessions", zap.String("gameServerSessionId", gameServerSessionId),
		zap.String("playerId", playerId), zap.String("playerSessionId", playerSessionId),
		zap.String("playerSessionStatusFilter", playerSessionStatusFilter), zap.String("nextToken", nextToken),
		zap.Int32("limit", limit))

	req := &grpcsdk.DescribePlayerSessionsRequest{
		GameServerSessionId:       gameServerSessionId,
		PlayerId:                  playerId,
		PlayerSessionId:           playerSessionId,
		PlayerSessionStatusFilter: playerSessionStatusFilter,
		NextToken:                 nextToken,
		Limit:                     limit,
	}

	return g.rpcClient.DescribePlayerSessions(g.getContext(), req)
}

// 8. UpdatePlayerSessionCreationPolicy
func (g *gsemanager) UpdatePlayerSessionCreationPolicy(newPolicy string) (*grpcsdk.AuxProxyResponse, error) {
	logger.Info("start to UpdatePlayerSessionCreationPolicy", zap.String("newPolicy", newPolicy))
	req := &grpcsdk.UpdatePlayerSessionCreationPolicyRequest{
		GameServerSessionId:            g.gameServerSession.GameServerSessionId,
		NewPlayerSessionCreationPolicy: newPolicy,
	}

	return g.rpcClient.UpdatePlayerSessionCreationPolicy(g.getContext(), req)
}

// 9.ReportCustomData
func (g *gsemanager) ReportCustomData(currentCustomCount, maxCustomCount int32) (*grpcsdk.AuxProxyResponse, error) {
	logger.Info("start to UpdatePlayerSessionCreationPolicy", zap.Int32("currentCustomCount", currentCustomCount),
		zap.Int32("maxCustomCount", maxCustomCount))
	req := &grpcsdk.ReportCustomDataRequest{
		CurrentCustomCount: currentCustomCount,
		MaxCustomCount:     maxCustomCount,
	}

	return g.rpcClient.ReportCustomData(g.getContext(), req)
}
