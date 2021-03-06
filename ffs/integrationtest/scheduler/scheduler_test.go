package scheduler

import (
	"context"
	"math/rand"
	"os"
	"testing"
	"time"

	"github.com/ipfs/go-cid"
	logging "github.com/ipfs/go-log/v2"
	"github.com/stretchr/testify/require"
	"github.com/textileio/powergate/v2/ffs"
	it "github.com/textileio/powergate/v2/ffs/integrationtest"
	itmanager "github.com/textileio/powergate/v2/ffs/integrationtest/manager"
	"github.com/textileio/powergate/v2/tests"

	"github.com/textileio/powergate/v2/util"
)

func TestMain(m *testing.M) {
	util.AvgBlockTime = time.Millisecond * 500
	logging.SetAllLoggers(logging.LevelError)
	os.Exit(m.Run())
}

func TestJobCancelation(t *testing.T) {
	r := rand.New(rand.NewSource(22))
	ipfsAPI, _, fapi, cls := itmanager.NewAPI(t, 1, 300)
	defer cls()

	cid, _ := it.AddRandomFile(t, r, ipfsAPI)
	jid, err := fapi.PushStorageConfig(cid)
	require.NoError(t, err)
	it.RequireEventualJobState(t, fapi, jid, ffs.Executing)
	time.Sleep(time.Second * 2)

	err = fapi.CancelJob(jid)
	require.NoError(t, err)

	// Assert that the Job status is Canceled, *and* was
	// finished _fast_.
	before := time.Now()
	it.RequireEventualJobState(t, fapi, jid, ffs.Canceled)
	require.True(t, time.Since(before) < time.Second)
}

func TestParallelExecution(t *testing.T) {
	t.Parallel()
	ipfs, _, fapi, cls := itmanager.NewAPI(t, 1, 300)
	defer cls()

	r := rand.New(rand.NewSource(22))
	n := 3
	cids := make([]cid.Cid, n)
	jids := make([]ffs.JobID, n)
	for i := 0; i < n; i++ {
		cid, _ := it.AddRandomFile(t, r, ipfs)
		jid, err := fapi.PushStorageConfig(cid)
		require.NoError(t, err)
		cids[i] = cid
		jids[i] = jid
		// Add some sleep time to avoid all of them
		// being batched in the same scheduler run.
		time.Sleep(time.Millisecond * 100)
	}
	// Check that all jobs should be immediately in the Executing status, since
	// the default max parallel runs is 50. So all should get in.
	for i := 0; i < len(jids); i++ {
		it.RequireEventualJobState(t, fapi, jids[i], ffs.Executing)
	}

	// Now just check that all of them finish successfully.
	for i := 0; i < len(jids); i++ {
		it.RequireEventualJobState(t, fapi, jids[i], ffs.Executing)
		it.RequireStorageConfig(t, fapi, cids[i], nil)
	}
}

func TestResumeScheduler(t *testing.T) {
	t.Parallel()

	ds := tests.NewTxMapDatastore()
	ipfs, ipfsMAddr := it.CreateIPFS(t)
	addr, client, ms := itmanager.NewDevnet(t, 1, 300, ipfsMAddr)
	manager, closeManager := itmanager.NewFFSManager(t, ds, client, addr, ms, ipfs)
	auth, err := manager.Create(context.Background())
	require.NoError(t, err)
	time.Sleep(time.Second * 3) // Wait for funding txn to finish.
	fapi, err := manager.GetByAuthToken(auth.Token)
	require.NoError(t, err)

	r := rand.New(rand.NewSource(22))
	c, _ := it.AddRandomFile(t, r, ipfs)
	jid, err := fapi.PushStorageConfig(c)
	require.NoError(t, err)

	time.Sleep(time.Second * 3)
	ds2, err := ds.Clone()
	require.NoError(t, err)
	closeManager()

	manager, closeManager = itmanager.NewFFSManager(t, ds2, client, addr, ms, ipfs)
	defer closeManager()
	fapi, err = manager.GetByAuthToken(auth.Token) // Get same FFS instance again
	require.NoError(t, err)
	it.RequireEventualJobState(t, fapi, jid, ffs.Success)

	sh, err := fapi.StorageInfo(c)
	require.NoError(t, err)
	require.Equal(t, 1, len(sh.Cold.Filecoin.Proposals)) // Check only one deal still exists.
}

func TestFailedJobMessage(t *testing.T) {
	t.Parallel()
	ipfs, _, fapi, cls := itmanager.NewAPI(t, 1, 300)
	defer cls()

	r := rand.New(rand.NewSource(22))
	// Add a file size that would be bigger than the
	// sector size. This should make the deal fail on the miner.
	c1, _ := it.AddRandomFileSize(t, r, ipfs, 2000)

	jid, err := fapi.PushStorageConfig(c1)
	require.NoError(t, err)
	job := it.RequireEventualJobState(t, fapi, jid, ffs.Failed)
	require.NotEmpty(t, job.ErrCause)
	require.Len(t, job.DealErrors, 1)
	de := job.DealErrors[0]
	require.False(t, de.ProposalCid.Defined())
	require.NotEmpty(t, de.Miner)
	require.Equal(t, "data doesn't fit in a sector", de.Message)
}
