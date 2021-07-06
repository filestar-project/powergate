package fchost

import (
	"fmt"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p/config"
	"github.com/multiformats/go-multiaddr"
)

var (
	networkBootstrappers = map[string][]string{
		"mainnet": {
			"/dns4/seed1.filestar.net/tcp/51024/p2p/12D3KooWNFBi3Ysg8cH8r6xHVzSLMFjKx2Ee14PzYts8qxymKPca",
			"/dns4/seed2.filestar.net/tcp/51024/p2p/12D3KooWF15XCfM8XNLANFfZZQaj89xtgrroTZ1GAXpfEgd8mAYp",
			"/dns4/seed3.filestar.net/tcp/51024/p2p/12D3KooWRWpnA6KMpoqXgXB1Ssc77B6TcEWJcNFtMqSYgS2UumT9",
			"/dns4/seed4.filestar.net/tcp/51024/p2p/12D3KooWDfiABDDLFsEUNrNpSvsWELRoiXxEenyXSsjxsSNUyFTd",
			"/dns4/seed5.filestar.net/tcp/51024/p2p/12D3KooW9tfsWnw7vZUzX6Bie12YcdyVuj7enskHKUTkotZkfnox",
		},
		"calibrationnet": {
			"/dns4/bootstrap-0.calibration.fildev.network/tcp/1347/p2p/12D3KooWRLZAseMo9h7fRD6ojn6YYDXHsBSavX5YmjBZ9ngtAEec",
			"/dns4/bootstrap-1.calibration.fildev.network/tcp/1347/p2p/12D3KooWJFtDXgZEQMEkjJPSrbfdvh2xfjVKrXeNFG1t8ioJXAzv",
			"/dns4/bootstrap-2.calibration.fildev.network/tcp/1347/p2p/12D3KooWP1uB9Lo7yCA3S17TD4Y5wStP5Nk7Vqh53m8GsFjkyujD",
			"/dns4/bootstrap-3.calibration.fildev.network/tcp/1347/p2p/12D3KooWLrPM4WPK1YRGPCUwndWcDX8GCYgms3DiuofUmxwvhMCn",
		},
	}
)

func getBootstrapPeers(network string) ([]peer.AddrInfo, error) {
	addrs, ok := networkBootstrappers[network]
	if !ok {
		return nil, fmt.Errorf("network doesn't have any configured bootstrappers")
	}

	maddrs := make([]multiaddr.Multiaddr, len(addrs))
	for i, addr := range addrs {
		var err error
		maddrs[i], err = multiaddr.NewMultiaddr(addr)
		if err != nil {
			return nil, fmt.Errorf("converting multiaddrs: %s", err)
		}
	}
	peers, err := peer.AddrInfosFromP2pAddrs(maddrs...)
	if err != nil {
		return nil, fmt.Errorf("multiaddr conversion: %s", err)
	}
	return peers, nil
}

func getDefaultOpts() []config.Option {
	return []config.Option{libp2p.Defaults}
}
