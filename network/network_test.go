package network

import (
	"fmt"
	"net"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_ResolveAddr(t *testing.T) {
	t.Parallel()

	tcpAddrBuilder := func(t *testing.T, address string) *net.TCPAddr {
		tcpAddr, err := net.ResolveTCPAddr("", address)
		require.NoError(t, err)

		return tcpAddr
	}

	cases := []struct {
		name      string
		address   string
		defaultIP string
		errMsg    string
	}{
		{
			name:    "incorrect address",
			address: "Foo Bar",
			errMsg:  "failed to parse addr",
		},
		{
			name:      "only port provided",
			address:   ":8080",
			defaultIP: "127.0.0.1",
		},
		{
			name:      "both address and port provided",
			address:   "255.0.255.0:8080",
			defaultIP: "",
		},
	}

	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			ipAddr, err := ResolveAddr(c.address, c.defaultIP)
			if c.errMsg != "" {
				require.ErrorContains(t, err, c.errMsg)
			} else {
				require.NoError(t, err)
				expectedIPAddr := tcpAddrBuilder(t, fmt.Sprintf("%s%s", c.defaultIP, c.address))
				require.Equal(t, expectedIPAddr, ipAddr)
			}
		})
	}
}
