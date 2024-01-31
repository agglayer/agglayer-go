package types

import "github.com/0xPolygonHermez/zkevm-node/jsonrpc/client"

var _ IZkEVMClientCache = (*ZkEVMClientCache)(nil)

type ZkEVMClientCache struct {
	clients map[string]IZkEVMClient
}

func New() *ZkEVMClientCache {
	return &ZkEVMClientCache{clients: map[string]IZkEVMClient{}}
}

func (zc *ZkEVMClientCache) GetClient(rpc string) IZkEVMClient {
	c, ok := zc.clients[rpc]
	if !ok {
		c = client.NewClient(rpc)
		zc.clients[rpc] = c
	}

	return c
}
