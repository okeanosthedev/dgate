package lite

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/go-logr/logr"
	"github.com/jellydator/ttlcache/v3"
	"github.com/sandertv/gophertunnel/minecraft"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"go.minekube.com/gate/pkg/edition/java/lite/config"
	"go.minekube.com/gate/pkg/util/errs"
	"golang.org/x/sync/errgroup"
)

var bedrockPingCache = ttlcache.New[string, string]()

func init() {
	go bedrockPingCache.Start()
}

// StartBedrock starts the Bedrock edition lite mode proxy.
func StartBedrock(ctx context.Context, cfg *config.Config) error {
	log := logr.FromContextOrDiscard(ctx).WithName("bedrock-lite")

	log.Info("starting bedrock lite mode proxy", "bind", cfg.Bedrock.Bind)

	// Set up a default status provider for unconnected pings
	serverStatus := fmt.Sprintf("MCPE;Gate Proxy;%d;%s;0;0;%d;Gate Bedrock;Survival;1;%d;",
		protocol.CurrentProtocol,
		protocol.CurrentVersion,
		time.Now().UnixNano(), // Unique ID
		19132,                 // Default port
	)
	listenConfig := minecraft.ListenConfig{
		StatusProvider: minecraft.NewStatusProvider(serverStatus),
	}

	listener, err := listenConfig.Listen("raknet", cfg.Bedrock.Bind)
	if err != nil {
		return fmt.Errorf("error listening on bedrock address: %w", err)
	}
	defer listener.Close()

	go func() {
		<-ctx.Done()
		_ = listener.Close()
	}()

	for {
		conn, err := listener.Accept()
		if err != nil {
			if errs.IsConnClosedErr(err) {
				return nil
			}
			log.Error(err, "error accepting new bedrock connection")
			continue
		}
		go handleBedrockConn(logr.NewContext(ctx, log), listener, conn.(*minecraft.Conn), cfg.Routes)
	}
}

func handleBedrockConn(ctx context.Context, listener *minecraft.Listener, conn *minecraft.Conn, routes []config.Route) {
	log := logr.FromContextOrDiscard(ctx)
	defer conn.Close()

	log.Info("new bedrock connection", "remoteAddr", conn.RemoteAddr())

	hostname := conn.ClientData().ServerAddress

	// Find route
	host, route := FindRoute(hostname, routes...)
	if route == nil || (route.Protocol != "udp" && route.Protocol != "any") {
		log.Info("no route found for bedrock connection", "hostname", hostname)
		return
	}
	log = log.WithValues("route", host)

	// Ping backend
	backendAddr := route.Backend.Random()
	var pingResponse string
	if item := bedrockPingCache.Get(backendAddr); item != nil {
		pingResponse = item.Value()
		log.Info("using cached ping response")
	} else {
		pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		resp, err := PingBedrock(pingCtx, backendAddr)
		if err != nil {
			if route.Fallback != nil {
				// Implement fallback for Bedrock
				fallbackMotd := route.Fallback.MOTD.T().Content // Assuming MOTD is a simple text component
				_ = listener.Disconnect(conn, fallbackMotd)
				log.Info("backend is down, sending fallback message", "fallbackMotd", fallbackMotd)
			} else {
				log.Info("backend is down and no fallback is configured")
			}
			return
		}
		pingResponse = resp
		if route.CachePingTTL > 0 {
			bedrockPingCache.Set(backendAddr, pingResponse, time.Duration(route.CachePingTTL))
		}
	}

	// Dial backend
	dialer := minecraft.Dialer{
		ClientData: conn.ClientData(),
	}

	// Modify virtual host
	if route.ModifyVirtualHost {
		dialer.ClientData.ServerAddress = route.Host.Random()
		log.Info("modified bedrock virtual host", "originalServerAddress", conn.ClientData().ServerAddress, "modifiedServerAddress", dialer.ClientData.ServerAddress)
	}

	// Use ModifyBedrockIP for UDP connections (explicit or automatic for "any" protocol)
	useBedrockIPForwarding := route.ModifyBedrockIP || (route.Protocol == "any")
	if useBedrockIPForwarding {
		// Modify ClientData to include the real IP
		clientIP := conn.RemoteAddr().(*net.UDPAddr).IP.String()
		dialer.ClientData.ServerAddress = fmt.Sprintf("%s;%s", dialer.ClientData.ServerAddress, clientIP)
		log.Info("modified bedrock client data for IP forwarding", "originalServerAddress", conn.ClientData().ServerAddress, "modifiedServerAddress", dialer.ClientData.ServerAddress)
	}
	dialCtx, cancel2 := context.WithTimeout(ctx, 10*time.Second)
	defer cancel2()
	backend, err := dialer.DialContext(dialCtx, "raknet", backendAddr)
	if err != nil {
		log.Error(err, "error dialing backend", "backendAddr", backendAddr)
		return
	}
	defer backend.Close()

	// Spawn
	if err := conn.DoSpawn(); err != nil {
		log.Error(err, "error spawning client")
		return
	}
	if err := backend.DoSpawn(); err != nil {
		log.Error(err, "error spawning backend")
		return
	}

	log.Info("forwarding bedrock connection", "backendAddr", backendAddr)

	var g errgroup.Group
	g.Go(func() error {
		return conn.StartGame(backend.GameData())
	})
	g.Go(func() error {
		return backend.StartGame(conn.GameData())
	})

	go func() {
		<-ctx.Done()
		_ = conn.Close()
		_ = backend.Close()
	}()

	if err := g.Wait(); err != nil && !errs.IsConnClosedErr(err) {
		log.V(1).Info("connection closed", "error", err)
	}
}