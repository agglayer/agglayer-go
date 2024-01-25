package client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/0xPolygon/agglayer/rpc/types"
	"github.com/0xPolygon/agglayer/tx"
	"github.com/0xPolygonHermez/zkevm-node/ethtxmanager"
	"github.com/0xPolygonHermez/zkevm-node/jsonrpc/client"
	"github.com/ethereum/go-ethereum/common"
)

// ClientFactoryInterface interface for the client factory
type ClientFactoryInterface interface {
	New(url string) ClientInterface
}

// ClientInterface is the interface that defines the implementation of all the endpoints
type ClientInterface interface {
	SendTx(signedTx tx.SignedTx) (common.Hash, error)
	GetTxStatus(hash common.Hash) (ethtxmanager.MonitoredTxStatus, error)
	WaitTxToBeMined(hash common.Hash, ctx context.Context) error
}

// ClientFactory is the implementation of the data committee client factory
type ClientFactory struct{}

// New returns an implementation of the data committee node client
func (f *ClientFactory) New(url string) ClientInterface {
	return New(url)
}

// Client wraps all the available endpoints of the data abailability committee node server
type Client struct {
	url string
}

// New returns a client ready to be used
func New(url string) *Client {
	return &Client{
		url: url,
	}
}

func (c *Client) SendTx(signedTx tx.SignedTx) (common.Hash, error) {
	response, err := client.JSONRPCCall(c.url, "interop_sendTx", signedTx)
	if err != nil {
		return common.Hash{}, err
	}

	if response.Error != nil {
		return common.Hash{}, fmt.Errorf("%v %v", response.Error.Code, response.Error.Message)
	}

	var result types.ArgHash
	err = json.Unmarshal(response.Result, &result)
	if err != nil {
		return common.Hash{}, err
	}

	return result.Hash(), nil
}

func (c *Client) GetTxStatus(hash common.Hash) (ethtxmanager.MonitoredTxStatus, error) {
	response, err := client.JSONRPCCall(c.url, "interop_getTxStatus", hash)
	if err != nil {
		return ethtxmanager.MonitoredTxStatus(""), err
	}

	if response.Error != nil {
		return ethtxmanager.MonitoredTxStatus(""), fmt.Errorf("%v %v", response.Error.Code, response.Error.Message)
	}

	var result ethtxmanager.MonitoredTxStatus
	err = json.Unmarshal(response.Result, &result)
	if err != nil {
		return ethtxmanager.MonitoredTxStatus(""), err
	}

	return result, nil
}

func (c *Client) WaitTxToBeMined(hash common.Hash, ctx context.Context) error {
	ticker := time.NewTicker(time.Second)
	for {
		select {
		case <-ctx.Done():
			return errors.New("context finished before tx was mined")
		case <-ticker.C:
			response, err := client.JSONRPCCall(c.url, "interop_getTxStatus", hash)
			if err != nil {
				return err
			}

			if response.Error != nil {
				return fmt.Errorf("%v %v", response.Error.Code, response.Error.Message)
			}

			var result ethtxmanager.MonitoredTxStatus
			err = json.Unmarshal(response.Result, &result)
			if err != nil {
				return err
			}
			if result == ethtxmanager.MonitoredTxStatusDone {
				return nil
			}
		}
	}
}
