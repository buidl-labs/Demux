package util

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/ipfs/go-cid"

	log "github.com/sirupsen/logrus"
	"github.com/textileio/powergate/api/client"
	"github.com/textileio/powergate/ffs"
	"github.com/textileio/powergate/health"
)

// PowergateSetup initializes stuff
type PowergateSetup struct {
	PowergateAddr string
	SampleSize    int64
	MaxParallel   int
	TotalSamples  int
}

var (
	powergateAddr        = os.Getenv("POWERGATE_ADDR")
	ipfsRevProxyAddr     = os.Getenv("IPFS_REV_PROXY_ADDR")
	trustedMinersStr     = os.Getenv("TRUSTED_MINERS")
	trustedMiners        = strings.Fields(trustedMinersStr)
	epochDurationSeconds = 30
	minDealDuration      = 180 * (24 * 60 * 60 / epochDurationSeconds)
)

// InitialPowergateSetup creates an instance of PowergateSetup
var InitialPowergateSetup = PowergateSetup{
	PowergateAddr: powergateAddr,
	SampleSize:    700,
	MaxParallel:   1,
	TotalSamples:  1,
}

// RunPow runs the powergate client
func RunPow(ctx context.Context, setup PowergateSetup, fName string) (cid.Cid, string, string, string, string, int, int, bool, error) {
	var currCid cid.Cid
	var minerName string
	var tok string
	var jid string
	var storagePrice int
	var expiry int
	var staged bool = false
	var powCloseError error
	fmt.Println("TM", trustedMiners)

	// Create a new powergate client
	c, err := client.NewClient(setup.PowergateAddr)
	defer func() {
		if err := c.Close(); err != nil {
			log.Errorf("closing powergate client: %s", err)
			powCloseError = err
		}
	}()
	if err != nil {
		return currCid, fName, minerName, tok, jid, storagePrice, expiry, staged, fmt.Errorf("creating client: %s", err)
	}

	if err := sanityCheck(ctx, c); err != nil {
		fmt.Println("issuesanity", err)
		return currCid, fName, minerName, tok, jid, storagePrice, expiry, staged, fmt.Errorf("sanity check with client: %s", err)
	}

	if currCid, fName, minerName, tok, jid, storagePrice, expiry, staged, err = runSetup(ctx, c, setup, fName); err != nil {
		return currCid, fName, minerName, tok, jid, storagePrice, expiry, staged, fmt.Errorf("running test setup: %s", err)
	}

	if powCloseError != nil {
		return currCid, fName, minerName, tok, jid, storagePrice, expiry, staged, powCloseError
	}
	return currCid, fName, minerName, tok, jid, storagePrice, expiry, staged, nil
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

func runSetup(ctx context.Context, c *client.Client, setup PowergateSetup, fName string) (cid.Cid, string, string, string, string, int, int, bool, error) {

	var currCid cid.Cid
	var jid string
	var minerName string
	var storagePrice int
	var expiry int
	var staged bool = false

	// Create new ffs instance
	_, tok, err := c.FFS.Create(ctx)
	if err != nil {
		return currCid, fName, minerName, tok, jid, storagePrice, expiry, staged, fmt.Errorf("creating ffs instance: %s", err)
	}

	log.Infof("ffs tok: [%s]\n", tok)

	ctx = context.WithValue(ctx, client.AuthKey, tok)

	info, err := c.FFS.Info(ctx)
	if err != nil {
		return currCid, fName, minerName, tok, jid, storagePrice, expiry, staged, fmt.Errorf("getting instance info: %s", err)
	}
	log.Infof("ffs info: [%v]\n", info)

	// Asks index
	index, err := c.Asks.Get(ctx)
	if err != nil {
		return currCid, fName, minerName, tok, jid, storagePrice, expiry, staged, fmt.Errorf("getting asks index: %s", err)
	}

	if len(index.Storage) > 0 {
		log.Printf("Storage median price: %v\n", index.StorageMedianPrice)
		log.Printf("Last updated: %v\n", index.LastUpdated.Format("01/02/06 15:04 MST"))
	}
	minerName = "nil"
	storagePrice = 0
	expiry = 0

	// wallet address
	addr := info.Balances[0].Addr
	time.Sleep(time.Second * 5)

	chLimit := make(chan struct{}, setup.MaxParallel)
	chErr := make(chan error, setup.TotalSamples)
	for i := 0; i < setup.TotalSamples; i++ {
		chLimit <- struct{}{}
		go func(i int) {
			defer func() { <-chLimit }()
			if currCid, fName, minerName, tok, jid, storagePrice, expiry, staged, err = run(ctx, c, i, setup.SampleSize, addr, fName, minerName, tok, storagePrice, expiry); err != nil {
				chErr <- fmt.Errorf("failed run %d: %s", i, err)
			} else {
				// no errors
				log.Infof("cid: [%s] fName: [%s]\n", currCid, fName)
			}
		}(i)
	}
	for i := 0; i < setup.MaxParallel; i++ {
		chLimit <- struct{}{}
	}
	close(chErr)
	for err := range chErr {
		return currCid, fName, minerName, tok, jid, storagePrice, expiry, staged, fmt.Errorf("sample run errored: %s", err)
	}

	return currCid, fName, minerName, tok, jid, storagePrice, expiry, staged, nil
}

func run(ctx context.Context, c *client.Client, id int, size int64, addr string, fName string, minerName string, tok string, storagePrice int, expiry int) (cid.Cid, string, string, string, string, int, int, bool, error) {
	log.Infof("[%d] Executing run...", id)
	defer log.Infof("[%d] Done", id)

	var ci cid.Cid
	var jid string
	var staged bool = false

	fi, err := os.Stat(fName)
	if os.IsNotExist(err) {
		return ci, fName, minerName, tok, jid, storagePrice, expiry, staged, fmt.Errorf("file/folder doesn't exist: %s", err)
	}
	if err != nil {
		return ci, fName, minerName, tok, jid, storagePrice, expiry, staged, fmt.Errorf("getting file/folder information: %s", err)
	}

	if fi.IsDir() {
		// if a folder has been pushed
		ci, err = c.FFS.StageFolder(ctx, ipfsRevProxyAddr, fName)
		if err != nil {
			return ci, fName, minerName, tok, jid, storagePrice, expiry, staged, fmt.Errorf("importing folder to hot storage (ipfs node): %s", err)
		}
	} else {
		// if a file has been pushed
		f, err := os.Open(fName)
		if err != nil {
			return ci, fName, minerName, tok, jid, storagePrice, expiry, staged, fmt.Errorf("importing file to hot storage (ipfs node): %s", err)
		}
		defer func() {
			e := f.Close()
			if e != nil {
				log.Infof("closing file: %s", e)
			}
		}()

		ptrCid, err := c.FFS.Stage(ctx, f)
		if err != nil {
			return ci, fName, minerName, tok, jid, storagePrice, expiry, staged, fmt.Errorf("importing file to hot storage (ipfs node): %s", err)
		}
		ci = *ptrCid
	}

	staged = true

	log.Infof("[%d] Pushing %s to FFS...", id, ci)

	// TODO: tweak config
	cidConfig := ffs.StorageConfig{
		Repairable: false,
		Hot: ffs.HotConfig{
			Enabled:          true,
			AllowUnfreeze:    false,
			UnfreezeMaxPrice: 0,
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
				TrustedMiners:   trustedMiners,
				Renew:           ffs.FilRenew{Enabled: false, Threshold: 0},
				MaxPrice:        uint64(storagePrice),
			},
		},
	}

	jobID, err := c.FFS.PushStorageConfig(ctx, ci, client.WithStorageConfig(cidConfig))
	jid = fmt.Sprintf("%s", jobID)
	if err != nil {
		return ci, fName, minerName, tok, jid, storagePrice, expiry, staged, fmt.Errorf("pushing to FFS: %s", err)
	}

	log.Infof("[%d] Pushed successfully, queued job %s. Waiting for termination...", id, jobID)

	return ci, fName, minerName, tok, jid, storagePrice, expiry, staged, nil
}
