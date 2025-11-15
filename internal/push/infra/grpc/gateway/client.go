package gateway

import (
	"sync"

	gatewaypb "github.com/wsx864321/kim/idl/gateway"
	"github.com/wsx864321/kim/pkg/krpc"
	"github.com/wsx864321/kim/pkg/log"
)

// ClientManager Gateway 客户端管理器
type ClientManager struct {
	clients sync.Map // map[string]gatewaypb.GatewayServiceClient
}

// NewClientManager 创建 Gateway 客户端管理器
func NewClientManager() *ClientManager {
	return &ClientManager{
		clients: sync.Map{},
	}
}

// GetClient 获取或创建 Gateway 客户端
func (m *ClientManager) GetClient(gatewayID string) (gatewaypb.GatewayServiceClient, error) {
	// 如果已经存在，直接返回
	if client, ok := m.clients.Load(gatewayID); ok {
		return client.(gatewaypb.GatewayServiceClient), nil
	}

	// 创建新的 Gateway 客户端
	cli, err := krpc.NewKClient(krpc.WithClientServiceName("kim-gateway"))
	if err != nil {
		log.Error(nil, "create gateway client failed",
			log.String("gateway_id", gatewayID),
			log.String("error", err.Error()),
		)
		return nil, err
	}

	gatewayClient := gatewaypb.NewGatewayServiceClient(cli.Conn())
	m.clients.Store(gatewayID, gatewayClient)

	return gatewayClient, nil
}
