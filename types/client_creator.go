package types

import "github.com/0xPolygonHermez/zkevm-node/jsonrpc/client"

var _ IZkEVMClientClientCreator = (*ZkEVMClientCreator)(nil)

type ZkEVMClientCreator struct{}

func (zc *ZkEVMClientCreator) NewClient(rpc string) IZkEVMClient {
	return client.NewClient(rpc)
}
