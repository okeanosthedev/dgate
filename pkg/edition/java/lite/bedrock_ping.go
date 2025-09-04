package lite

import (
	"context"
	"fmt"
	"net"
	"strconv"

	"github.com/xrjr/mcutils/pkg/bedrock"
)

// PingBedrock pings a bedrock server and returns the status.
func PingBedrock(ctx context.Context, addr string) (string, error) {
	// Parse the address to extract hostname and port
	host, portStr, err := net.SplitHostPort(addr)
	if err != nil {
		return "", fmt.Errorf("invalid address format: %w", err)
	}
	
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return "", fmt.Errorf("invalid port: %w", err)
	}

	// mcutils bedrock.Ping does not take a context, so we use timeout in a goroutine
	type result struct {
		pong bedrock.UnconnectedPong
		latency int
		err error
	}
	
	resultChan := make(chan result, 1)
	go func() {
		pong, latency, err := bedrock.Ping(host, port)
		resultChan <- result{pong: pong, latency: latency, err: err}
	}()
	
	select {
	case res := <-resultChan:
		if res.err != nil {
			return "", res.err
		}
		// Format the response as Bedrock server status string
		return fmt.Sprintf("MCPE;%s;%d;%s;%d;%d;%s;%s;%s;%d;%d;",
			res.pong.MOTD,
			res.pong.ProtocolVersion,
			res.pong.MinecraftVersion,
			res.pong.OnlinePlayers,
			res.pong.MaxPlayers,
			res.pong.ServerID,
			res.pong.LevelName,
			res.pong.GameMode,
			res.pong.GameModeNumeric,
			res.pong.IPv4Port,
		), nil
	case <-ctx.Done():
		return "", ctx.Err()
	}
}
