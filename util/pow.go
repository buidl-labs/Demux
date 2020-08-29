package util

import (
	"context"
	"fmt"
	golog "log"
	"os"
	"strconv"
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
	SampleSize    int64
	MaxParallel   int
	TotalSamples  int
	RandSeed      int
}

var (
	log                  = logging.Logger("runner")
	powergateAddr        = os.Getenv("POWERGATE_ADDR")
	ipfsRevProxyAddr     = os.Getenv("IPFS_REV_PROXY_ADDR")
	epochDurationSeconds = 30
	minDealDuration      = 180 * (24 * 60 * 60 / epochDurationSeconds)
)

// InitialPowergateSetup creates an instance of PowergateSetup
var InitialPowergateSetup = PowergateSetup{
	PowergateAddr: powergateAddr,
	SampleSize:    700,
	MaxParallel:   1,
	TotalSamples:  1,
	RandSeed:      22,
}

// RunPow runs the powergate client
func RunPow(ctx context.Context, setup PowergateSetup, fName string) (cid.Cid, string, string, int, int, error) {
	var currCid cid.Cid // CID for the file/folder that is being stored
	var minerName string
	var storagePrice int
	var expiry int // unix timestamp
	var powCloseError error

	// Create a new powergate client
	c, err := client.NewClient(setup.PowergateAddr, grpc.WithInsecure(), grpc.WithPerRPCCredentials(client.TokenAuth{}))
	defer func() {
		if err := c.Close(); err != nil {
			log.Errorf("closing powergate client: %s", err)
			powCloseError = err
		}
	}()
	if err != nil {
		return currCid, fName, minerName, storagePrice, expiry, fmt.Errorf("creating client: %s", err)
	}

	if err := sanityCheck(ctx, c); err != nil {
		return currCid, fName, minerName, storagePrice, expiry, fmt.Errorf("sanity check with client: %s", err)
	}

	if currCid, fName, minerName, storagePrice, expiry, err = runSetup(ctx, c, setup, fName); err != nil {
		return currCid, fName, minerName, storagePrice, expiry, fmt.Errorf("running test setup: %s", err)
	}

	if powCloseError != nil {
		return currCid, fName, minerName, storagePrice, expiry, powCloseError
	}
	return currCid, fName, minerName, storagePrice, expiry, nil
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

func runSetup(ctx context.Context, c *client.Client, setup PowergateSetup, fName string) (cid.Cid, string, string, int, int, error) {

	var currCid cid.Cid
	var minerName string
	var storagePrice int
	var expiry int

	// Create new ffs instance
	_, tok, err := c.FFS.Create(ctx)
	if err != nil {
		return currCid, fName, minerName, storagePrice, expiry, fmt.Errorf("creating ffs instance: %s", err)
	}

	golog.Printf("ffs tok: [%s]\n", tok)
	log.Infof("ffs tok: [%s]\n", tok)

	ctx = context.WithValue(ctx, client.AuthKey, tok)

	info, err := c.FFS.Info(ctx)
	if err != nil {
		return currCid, fName, minerName, storagePrice, expiry, fmt.Errorf("getting instance info: %s", err)
	}
	golog.Printf("ffs info: [%v]\n", info)
	log.Infof("ffs info: [%s]\n", info)

	// Asks index
	index, err := c.Asks.Get(ctx)
	if err != nil {
		return currCid, fName, minerName, storagePrice, expiry, fmt.Errorf("getting asks index: %s", err)
	}

	if len(index.Storage) > 0 {
		fmt.Printf("Storage median price: %v\n", index.StorageMedianPrice)
		fmt.Printf("Last updated: %v\n", index.LastUpdated.Format("01/02/06 15:04 MST"))
		data := make([][]string, len(index.Storage))
		i := 0
		for _, a := range index.Storage {
			minerName = a.Miner
			storagePrice = int(a.Price)
			expiry = int(a.Expiry)
			data[i] = []string{
				a.Miner,
				strconv.Itoa(int(a.Price)),
				strconv.Itoa(int(a.MinPieceSize)),
				strconv.FormatInt(a.Timestamp, 10),
				strconv.FormatInt(a.Expiry, 10),
			}
			i++
		}
	}

	// wallet address
	addr := info.Balances[0].Addr
	time.Sleep(time.Second * 5)

	chLimit := make(chan struct{}, setup.MaxParallel)
	chErr := make(chan error, setup.TotalSamples)
	for i := 0; i < setup.TotalSamples; i++ {
		chLimit <- struct{}{}
		go func(i int) {
			defer func() { <-chLimit }()
			if currCid, fName, minerName, storagePrice, expiry, err = run(ctx, c, i, setup.RandSeed+i, setup.SampleSize, addr, fName, minerName, storagePrice, expiry); err != nil {
				chErr <- fmt.Errorf("failed run %d: %s", i, err)
			} else {
				// no errors
				golog.Printf("cid: [%s] fName: [%s]\n", currCid, fName)
				log.Infof("cid: [%s] fName: [%s]\n", currCid, fName)
			}
		}(i)
	}
	for i := 0; i < setup.MaxParallel; i++ {
		chLimit <- struct{}{}
	}
	close(chErr)
	for err := range chErr {
		return currCid, fName, minerName, storagePrice, expiry, fmt.Errorf("sample run errored: %s", err)
	}

	return currCid, fName, minerName, storagePrice, expiry, nil
}

func run(ctx context.Context, c *client.Client, id int, seed int, size int64, addr string, fName string, minerAddr string, storagePrice int, expiry int) (cid.Cid, string, string, int, int, error) {
	golog.Printf("[%d] Executing run...", id)
	log.Infof("[%d] Executing run...", id)
	defer golog.Printf("[%d] Done", id)
	defer log.Infof("[%d] Done", id)

	var ci cid.Cid

	fi, err := os.Stat(fName)
	if os.IsNotExist(err) {
		return ci, fName, minerAddr, storagePrice, expiry, fmt.Errorf("file/folder doesn't exist: %s", err)
	}
	if err != nil {
		return ci, fName, minerAddr, storagePrice, expiry, fmt.Errorf("getting file/folder information: %s", err)
	}

	if fi.IsDir() {
		// if a folder has been pushed
		ci, err = c.FFS.StageFolder(ctx, ipfsRevProxyAddr, fName)
		if err != nil {
			return ci, fName, minerAddr, storagePrice, expiry, fmt.Errorf("importing folder to hot storage (ipfs node): %s", err)
		}
	} else {
		// if a file has been pushed
		f, err := os.Open(fName)
		if err != nil {
			return ci, fName, minerAddr, storagePrice, expiry, fmt.Errorf("importing file to hot storage (ipfs node): %s", err)
		}
		defer func() {
			e := f.Close()
			if e != nil {
				golog.Printf("closing file: %s", e)
				log.Infof("closing file: %s", e)
			}
		}()

		ptrCid, err := c.FFS.Stage(ctx, f)
		if err != nil {
			return ci, fName, minerAddr, storagePrice, expiry, fmt.Errorf("importing file to hot storage (ipfs node): %s", err)
		}
		ci = *ptrCid
	}

	golog.Printf("[%d] Pushing %s to FFS...", id, ci)
	log.Infof("[%d] Pushing %s to FFS...", id, ci)

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
				DealMinDuration: int64(minDealDuration),
				Addr:            addr,
				CountryCodes:    nil,
				ExcludedMiners:  nil,
				TrustedMiners:   []string{minerAddr},
				Renew:           ffs.FilRenew{Enabled: false, Threshold: 0},
				MaxPrice:        uint64(storagePrice),
			},
		},
	}

	jobID, err := c.FFS.PushStorageConfig(ctx, ci, client.WithStorageConfig(cidConfig))
	if err != nil {
		return ci, fName, minerAddr, storagePrice, expiry, fmt.Errorf("pushing to FFS: %s", err)
	}

	golog.Printf("[%d] Pushed successfully, queued job %s. Waiting for termination...", id, jobID)
	log.Infof("[%d] Pushed successfully, queued job %s. Waiting for termination...", id, jobID)

	chJob := make(chan client.JobEvent, 1)
	ctxWatch, cancel := context.WithCancel(ctx)
	defer cancel()
	err = c.FFS.WatchJobs(ctxWatch, chJob, jobID)
	if err != nil {
		return ci, "", minerAddr, storagePrice, expiry, fmt.Errorf("opening listening job status: %s", err)
	}
	var s client.JobEvent
	for s = range chJob {
		if s.Err != nil {
			return ci, fName, minerAddr, storagePrice, expiry, fmt.Errorf("job watching: %s", s.Err)
		}
		golog.Printf("[%d] Job changed to status %s", id, ffs.JobStatusStr[s.Job.Status])
		log.Infof("[%d] Job changed to status %s", id, ffs.JobStatusStr[s.Job.Status])
		if s.Job.Status == ffs.Failed || s.Job.Status == ffs.Canceled {
			return ci, fName, minerAddr, storagePrice, expiry, fmt.Errorf("job execution failed or was canceled")
		}
		if s.Job.Status == ffs.Success {
			golog.Printf("success!!! cid: [%s] fName: [%s]\n", ci, fName)
			log.Infof("success!!! cid: [%s] fName: [%s]\n", ci, fName)
			return ci, fName, minerAddr, storagePrice, expiry, nil
		}
	}
	return ci, fName, minerAddr, storagePrice, expiry, fmt.Errorf("unexpected Job status watcher")
}
