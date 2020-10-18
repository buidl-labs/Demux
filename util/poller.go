package util

import (
	"context"
	"fmt"
	"math"
	"math/big"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/buidl-labs/Demux/dataservice"
	"github.com/buidl-labs/Demux/internal"
	"github.com/buidl-labs/Demux/model"

	"github.com/ipfs/go-cid"
	log "github.com/sirupsen/logrus"
	powc "github.com/textileio/powergate/api/client"
	"github.com/textileio/powergate/ffs"
)

// RunPoller runs a poller in the background which
// updates the state of storage deal jobs.
func RunPoller() {
	i, err := strconv.Atoi(os.Getenv("POLL_INTERVAL"))
	if err != nil {
		log.Error("Please set env variable `POLL_INTERVAL`")
		return
	}
	var pollInterval = time.Minute * time.Duration(i)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	pgClient, _ := powc.NewClient(InitialPowergateSetup.PowergateAddr)
	defer func() {
		if err := pgClient.Close(); err != nil {
			log.Errorf("closing powergate client: %s", err)
		}
	}()
	for {
		select {
		case <-ctx.Done():
			log.Info("shutting down archive tracker daemon")
			return
		case <-time.After(pollInterval):
			deals, err := dataservice.GetPendingDeals()
			if err != nil {
				return
			}
			log.Infof("Number of pending storage deals: %d\n", len(deals))
			for _, deal := range deals {
				cidcorrtype, _ := cid.Decode(deal.CID)
				b, s, err := pollStorageDealProgress(ctx, pgClient, ffs.JobID(deal.JobID), cidcorrtype, deal)
				log.Info("dealJobID", deal.JobID, "pollprog", b, s, err)
			}
		}
	}
}

func pollStorageDealProgress(ctx context.Context, pgClient *powc.Client, jid ffs.JobID, mycid cid.Cid, storageDeal model.StorageDeal) (bool, string, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	ctx = context.WithValue(ctx, powc.AuthKey, storageDeal.FFSToken)
	ch := make(chan powc.JobEvent, 1)

	if err := pgClient.FFS.WatchJobs(ctx, ch, jid); err != nil {
		// if error specifies that the auth token isn't found, powergate must have been reset.
		// return the error as fatal so the archive will be untracked
		if strings.Contains(err.Error(), "auth token not found") {
			return false, "", err
		}
		return true, fmt.Sprintf("watching current job %s for cid %s: %s", jid, mycid, err), nil
	}

	var aborted bool
	var abortMsg string
	var job ffs.StorageJob
	select {
	case <-ctx.Done():
		log.Infof("job %s status watching canceled\n", jid)
		return true, "watching cancelled", nil
	case s, ok := <-ch:
		if !ok {
			return true, "powergate closed communication chan", nil
		}
		if s.Err != nil {
			log.Errorf("job %s update: %s", jid, s.Err)
			aborted = true
			abortMsg = s.Err.Error()
		}
		job = s.Job
	}

	if !aborted && !isJobStatusFinal(job.Status) {
		return true, "no final status yet", nil
	}

	// On success, save Deal data in the underlying Bucket thread. On failure,
	// save the error message. Also update status on Mongo for the archive.
	if job.Status == ffs.Success {
		err := saveDealsInDB(ctx, pgClient, storageDeal.FFSToken, mycid)
		if err != nil {
			return true, fmt.Sprintf("saving deal data in archive: %s", err), nil
		}
	} else {
		log.Info("job.Status", ffs.JobStatusStr[job.Status], job.Status)
		dataservice.UpdateStorageDeal(mycid.String(), 2, "failed to create filecoin storage deal", "", "", 0)
	}

	msg := "reached final status"
	if aborted {
		msg = "aborted with reason " + abortMsg
	}

	return false, msg, nil
}

func saveDealsInDB(ctx context.Context, pgClient *powc.Client, ffsToken string, c cid.Cid) error {
	ctxFFS := context.WithValue(ctx, powc.AuthKey, ffsToken)
	sh, err := pgClient.FFS.Show(ctxFFS, c)
	if err != nil {
		return fmt.Errorf("getting cid info: %s", err)
	}

	proposals := sh.GetCidInfo().GetCold().GetFilecoin().GetProposals()

	log.Info("proposals", proposals)

	if len(proposals) > 0 {
		for _, prop := range proposals {
			priceAttoFIL := prop.EpochPrice * uint64(prop.Duration)
			priceAttoFILBigInt := new(big.Int).SetUint64(priceAttoFIL)
			priceFIL := float64(priceAttoFIL) * math.Pow(10, -18)
			log.Info("priceAttoFIL", priceAttoFIL, "priceAttoFILBigInt", priceAttoFILBigInt, "priceFIL", priceFIL)

			log.Info("ProposalCid", prop.ProposalCid)
			log.Info("Renewed", prop.Renewed)
			log.Info("Miner", prop.Miner)
			log.Info("StartEpoch", prop.StartEpoch)
			log.Info("ActivationEpoch", prop.ActivationEpoch)
			log.Info("Duration", prop.Duration)
			log.Info("EpochPrice", prop.EpochPrice)

			dataservice.UpdateStorageDeal(c.String(), 1, internal.AssetStatusMap[4], prop.Miner, priceAttoFILBigInt.String(), 0)
			dataservice.UpdateAssetStatusByCID(c.String(), 4, internal.AssetStatusMap[4])
		}
	}

	return nil
}

func isJobStatusFinal(js ffs.JobStatus) bool {
	return js == ffs.Success ||
		js == ffs.Canceled ||
		js == ffs.Failed
}
