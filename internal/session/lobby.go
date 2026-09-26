package session

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"strings"
	"time"

	pb "SetSecrets/internal/proto/gamesession"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

func CreateGrpcClient(target string, useTls bool) (pb.GameSessionServiceClient, func(), error) {

	cleanTarget := strings.TrimPrefix(target, "https://")
	cleanTarget = strings.TrimPrefix(cleanTarget, "http://")

	var creds credentials.TransportCredentials
	if useTls {
		creds = credentials.NewTLS(&tls.Config{
			InsecureSkipVerify: true, // Dev-friendly for localhost self-signed certs
		})
	} else {
		creds = insecure.NewCredentials()
	}

	conn, err := grpc.NewClient(cleanTarget, grpc.WithTransportCredentials(creds))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to connect to gRPC server at %s: %w", cleanTarget, err)
	}

	client := pb.NewGameSessionServiceClient(conn)
	cleanup := func() {
		_ = conn.Close()
	}
	return client, cleanup, nil
}

func GetLobby(ctx context.Context, target string, useTls bool, sessionID string) (*pb.SessionLobbyResponse, error) {
	client, cleanup, err := CreateGrpcClient(target, useTls)
	if err != nil {
		return nil, err
	}
	defer cleanup()
	req := &pb.GetLobbyRequest{
		SessionId: sessionID,
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	return client.GetLobby(ctx, req)
}

func KickPlayer(ctx context.Context, target string, useTls bool, sessionID string, slotNumber int32, reason string) (*pb.KickPlayerResponse, error) {
	client, cleanup, err := CreateGrpcClient(target, useTls)
	if err != nil {
		return nil, err
	}
	defer cleanup()
	req := &pb.KickPlayerRequest{
		SessionId:  sessionID,
		SlotNumber: slotNumber,
	}
	if reason != "" {
		req.Reason = &reason
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	return client.KickPlayer(ctx, req)
}

// LaunchSession transitions the session from Created to Started.
func LaunchSession(ctx context.Context, target string, useTls bool, sessionID string) (*pb.LaunchSessionResponse, error) {
	client, cleanup, err := CreateGrpcClient(target, useTls)
	if err != nil {
		return nil, err
	}
	defer cleanup()

	req := &pb.LaunchSessionRequest{
		SessionId: sessionID,
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	return client.LaunchSession(ctx, req)
}

func WatchLobby(ctx context.Context, target string, useTls bool, sessionID string, onUpdate func(*pb.SessionLobbyResponse)) error {
	client, cleanup, err := CreateGrpcClient(target, useTls)
	if err != nil {
		return err
	}
	defer cleanup()
	req := &pb.WatchLobbyRequest{
		SessionId: sessionID,
	}
	stream, err := client.WatchLobby(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to start watching lobby: %w", err)
	}
	for {
		resp, err := stream.Recv()
		if err == io.EOF {
			// Stream ended gracefully by the server (e.g. session started)
			return nil
		}
		if err != nil {
			return err
		}
		onUpdate(resp)

	}
}
