package initializer

import (
	"context"

	iplugin "go-smilo/src/blockchain/smilobft/internal/plugin"
	"go-smilo/src/blockchain/smilobft/plugin/gen/proto_common"

	"github.com/hashicorp/go-plugin"
	"google.golang.org/grpc"
)

const ConnectorName = "init"

type PluginConnector struct {
	plugin.Plugin
}

func (p *PluginConnector) GRPCServer(b *plugin.GRPCBroker, s *grpc.Server) error {
	return iplugin.ErrNotSupported
}

func (p *PluginConnector) GRPCClient(ctx context.Context, b *plugin.GRPCBroker, cc *grpc.ClientConn) (interface{}, error) {
	return &PluginGateway{
		client: proto_common.NewPluginInitializerClient(cc),
	}, nil
}
