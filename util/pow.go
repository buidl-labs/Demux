package util

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/ipfs/go-cid"
	logging "github.com/ipfs/go-log/v2"
	"github.com/textileio/powergate/api/client"
	"github.com/textileio/powergate/ffs"
	"github.com/textileio/powergate/health"
	"google.golang.org/grpc"
)

// PowergateSetup initializes stuff
type PowergateSetup struct {
	PowergateAddr string
	MinerAddr     string
	SampleSize    int64
	MaxParallel   int
	TotalSamples  int
	RandSeed      int
}

var (
	log = logging.Logger("runner")
)

// RunPow runs the pow client
func RunPow(ctx context.Context, setup PowergateSetup, fileName string) (cid.Cid, string, error) {
	var somecid cid.Cid
	c, err := client.NewClient(setup.PowergateAddr, grpc.WithInsecure(), grpc.WithPerRPCCredentials(client.TokenAuth{}))
	defer func() {
		if err := c.Close(); err != nil {
			log.Errorf("closing powergate client: %s", err)
		}
	}()
	if err != nil {
		return somecid, "", fmt.Errorf("creating client: %s", err)
	}

	if err := sanityCheck(ctx, c); err != nil {
		return somecid, "", fmt.Errorf("sanity check with client: %s", err)
	}

	if currcid, fname, err := runSetup(ctx, c, setup, fileName); err != nil {
		return somecid, "", fmt.Errorf("running test setup: %s", err)
	} else {
		return currcid, fname, nil
	}
}

func sanityCheck(ctx context.Context, c *client.Client) error {
	s, _, err := c.Health.Check(ctx)
	if err != nil {
		return fmt.Errorf("health check call: %s", err)
	}
	if s != health.Ok {
		return fmt.Errorf("reported health check not Ok: %s", s)
	}
	return nil
}

func runSetup(ctx context.Context, c *client.Client, setup PowergateSetup, fileName string) (cid.Cid, string, error) {

	var currcid cid.Cid
	var fname string

	_, tok, err := c.FFS.Create(ctx)
	if err != nil {
		return currcid, "", fmt.Errorf("creating ffs instance: %s", err)
	}
	fmt.Println("ffs tok", tok)
	ctx = context.WithValue(ctx, client.AuthKey, tok)
	info, err := c.FFS.Info(ctx)
	if err != nil {
		return currcid, "", fmt.Errorf("getting instance info: %s", err)
	}
	fmt.Println("ffs info", info)

	// *******************************
	// TODO: miner selection algorithm
	// asks, err := c.Asks.Get(ctx)
	// fmt.Println(asks)
	// *******************************

	addr := info.Balances[0].Addr
	time.Sleep(time.Second * 5)

	chLimit := make(chan struct{}, setup.MaxParallel)
	chErr := make(chan error, setup.TotalSamples)
	for i := 0; i < setup.TotalSamples; i++ {
		chLimit <- struct{}{}
		go func(i int) {
			defer func() { <-chLimit }()
			if currcid, fname, err = run(ctx, c, i, setup.RandSeed+i, setup.SampleSize, addr, setup.MinerAddr, fileName); err != nil {
				chErr <- fmt.Errorf("failed run %d: %s", i, err)
			} else {
				fmt.Printf("cid: %s fname: %s\n", currcid, fname)
			}
		}(i)
	}
	for i := 0; i < setup.MaxParallel; i++ {
		chLimit <- struct{}{}
	}
	close(chErr)
	for err := range chErr {
		return currcid, "", fmt.Errorf("sample run errored: %s", err)
	}
	return currcid, fname, nil
}

func run(ctx context.Context, c *client.Client, id int, seed int, size int64, addr string, minerAddr string, fileName string) (cid.Cid, string, error) {
	log.Infof("[%d] Executing run...", id)
	defer log.Infof("[%d] Done", id)

	ior, _ := os.Open(fileName)
	fmt.Println("insidefName", fileName, ior)

	log.Infof("[%d] Adding to hot layer...", id)
	fmt.Printf("[%d] Adding to hot layer...", id)
	ci, err := c.FFS.Stage(ctx, ior)
	if err != nil {
		return *ci, "", fmt.Errorf("importing data to hot storage (ipfs node): %s", err)
	}

	log.Infof("[%d] Pushing %s to FFS...", id, *ci)
	fmt.Printf("[%d] Pushing %s to FFS...", id, *ci)

	// TODO: tweak config
	cidConfig := ffs.StorageConfig{
		Repairable: false,
		Hot: ffs.HotConfig{
			Enabled:       true,
			AllowUnfreeze: false,
			Ipfs: ffs.IpfsConfig{
				AddTimeout: 30,
			},
		},
		Cold: ffs.ColdConfig{
			Enabled: true,
			Filecoin: ffs.FilConfig{
				RepFactor:       1,
				DealMinDuration: 1000,
				Addr:            addr,
				CountryCodes:    nil,
				ExcludedMiners:  nil,
				TrustedMiners:   []string{minerAddr},
				Renew:           ffs.FilRenew{Enabled: true, Threshold: 100},
				MaxPrice:        500000000, // to be set using different algorithm on testnet
			},
		},
	}

	jid, err := c.FFS.PushStorageConfig(ctx, *ci, client.WithStorageConfig(cidConfig))
	if err != nil {
		return *ci, "", fmt.Errorf("pushing to FFS: %s", err)
	}

	log.Infof("[%d] Pushed successfully, queued job %s. Waiting for termination...", id, jid)
	fmt.Printf("[%d] Pushed successfully, queued job %s. Waiting for termination...\n", id, jid)
	chJob := make(chan client.JobEvent, 1)
	ctxWatch, cancel := context.WithCancel(ctx)
	defer cancel()
	err = c.FFS.WatchJobs(ctxWatch, chJob, jid)
	if err != nil {
		return *ci, "", fmt.Errorf("opening listening job status: %s", err)
	}
	var s client.JobEvent
	for s = range chJob {
		if s.Err != nil {
			return *ci, "", fmt.Errorf("job watching: %s", s.Err)
		}
		log.Infof("[%d] Job changed to status %s", id, ffs.JobStatusStr[s.Job.Status])
		if s.Job.Status == ffs.Failed || s.Job.Status == ffs.Canceled {
			return *ci, "", fmt.Errorf("job execution failed or was canceled")
		}
		if s.Job.Status == ffs.Success {
			fmt.Printf("success!!! cid: %s filename: %s\n", *ci, fileName)
			return *ci, fileName, nil
		}
	}
	return *ci, "", fmt.Errorf("unexpected Job status watcher")
}
