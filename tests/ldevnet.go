package tests

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
	"github.com/stretchr/testify/require"
	"github.com/textileio/powergate/v2/lotus"
	"github.com/textileio/powergate/v2/util"
)

// TestingTWithCleanup is an augmented require.TestingT with a Cleanup function.
type TestingTWithCleanup interface {
	require.TestingT
	Cleanup(func())
}

// LaunchDevnetDocker launches the devnet docker image.
func LaunchDevnetDocker(t TestingTWithCleanup, numMiners, speed int, ipfsMaddr string, mountVolumes bool) *dockertest.Resource {
	pool, err := dockertest.NewPool("")
	require.NoError(t, err)
	envs := []string{
		devnetEnv("NUMMINERS", strconv.Itoa(numMiners)),
		devnetEnv("SPEED", strconv.Itoa(speed)),
		devnetEnv("IPFSADDR", ipfsMaddr),
		devnetEnv("BIGSECTORS", false),
	}
	var mounts []string
	if mountVolumes {
		mounts = append(mounts, "/tmp/powergate:/tmp/powergate")
	}

	repository := "textile/lotus-devnet"
	tag := "v1.3.2"
	lotusDevnet, err := pool.RunWithOptions(&dockertest.RunOptions{Repository: repository, Tag: tag, Env: envs, Mounts: mounts})
	require.NoError(t, err)
	err = lotusDevnet.Expire(180)
	require.NoError(t, err)
	time.Sleep(time.Second * time.Duration(2+numMiners))
	t.Cleanup(func() {
		err := pool.Purge(lotusDevnet)
		require.NoError(t, err)
	})
	debug := false
	if debug {
		go func() {
			opts := docker.LogsOptions{
				Context: context.Background(),

				Stderr:      true,
				Stdout:      true,
				Follow:      true,
				Timestamps:  true,
				RawTerminal: true,

				Container: lotusDevnet.Container.ID,

				OutputStream: os.Stdout,
			}

			err := pool.Client.Logs(opts)
			require.NoError(t, err)
		}()
	}
	return lotusDevnet
}

// CreateLocalDevnetWithIPFS creates a local devnet connected to an IPFS node.
func CreateLocalDevnetWithIPFS(t TestingTWithCleanup, numMiners, speed int, ipfsMaddr string, mountVolumes bool) (lotus.ClientBuilder, address.Address, []address.Address) {
	lotusDevnet := LaunchDevnetDocker(t, numMiners, speed, ipfsMaddr, mountVolumes)
	cb, err := lotus.NewBuilder(util.MustParseAddr("/ip4/127.0.0.1/tcp/"+lotusDevnet.GetPort("7777/tcp")), "", 1)
	require.NoError(t, err)
	ctx, cls := context.WithTimeout(context.Background(), time.Second*10)
	defer cls()
	c, cls, err := cb(ctx)
	require.NoError(t, err)
	t.Cleanup(func() { cls() })
	ctx = context.Background()
	addr, err := c.WalletDefaultAddress(ctx)
	require.NoError(t, err)
	miners, err := c.StateListMiners(ctx, types.EmptyTSK)
	require.NoError(t, err)

	return cb, addr, miners
}

// CreateLocalDevnet returns an API client that targets a local devnet with numMiners number
// of miners. Refer to http://github.com/textileio/local-devnet for more information.
func CreateLocalDevnet(t TestingTWithCleanup, numMiners, speed int) (lotus.ClientBuilder, address.Address, []address.Address) {
	return CreateLocalDevnetWithIPFS(t, numMiners, speed, "", true)
}

func devnetEnv(name string, value interface{}) string {
	return fmt.Sprintf("TEXLOTUSDEVNET_%s=%s", name, value)
}
