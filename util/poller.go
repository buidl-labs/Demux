package util

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/buidl-labs/Demux/dataservice"
	// "github.com/buidl-labs/Demux/internal"
	"github.com/buidl-labs/Demux/model"

	"github.com/ipfs/go-cid"
	log "github.com/sirupsen/logrus"
	"github.com/textileio/powergate/v2/api/client"
	powc "github.com/textileio/powergate/v2/api/client"
	userPb "github.com/textileio/powergate/v2/api/gen/powergate/user/v1"
	"github.com/textileio/powergate/v2/ffs"
	// "github.com/textileio/powergate/v2/api/client/admin"
	// "google.golang.org/protobuf/encoding/protojson"
)

// RunPoller runs a poller in the background which
// updates the state of storage deal jobs.
func RunPoller(db dataservice.DatabaseHelper) {
	assetDB := dataservice.NewAssetDatabase(db)
	storageDealDB := dataservice.NewStorageDealDatabase(db)

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
			deals, err := storageDealDB.GetPendingDeals()
			if err != nil {
				return
			}
			log.Infof("Number of pending storage deals: %d\n", len(deals))
			for _, deal := range deals {
				cidcorrtype, _ := cid.Decode(deal.CID)
				b, s, err := pollStorageDealProgress(ctx, pgClient, ffs.JobID(deal.JobID), cidcorrtype, deal, storageDealDB, assetDB)
				log.Info("dealJobID", deal.JobID, "pollprog", b, s, err)
			}
		}
	}
}

func pollStorageDealProgress(ctx context.Context, pgClient *powc.Client, jid ffs.JobID, mycid cid.Cid, storageDeal model.StorageDeal, storageDealDB dataservice.StorageDealDatabase, assetDB dataservice.AssetDatabase) (bool, string, error) {
	ctx = context.WithValue(ctx, powc.AuthKey, storageDeal.FFSToken)

	chJob := make(chan powc.WatchStorageJobsEvent, 1)
	ctxWatch, cancel := context.WithCancel(ctx)
	defer cancel()
	if err := pgClient.StorageJobs.Watch(ctxWatch, chJob, jid.String()); err != nil {
		// return fmt.Errorf("opening listening job status: %s", err)

		// if error specifies that the auth token isn't found, powergate must have been reset.
		// return the error as fatal so the archive will be untracked
		if strings.Contains(err.Error(), "auth token not found") {
			return false, "", err
		}
		return true, fmt.Sprintf("watching current job %s for cid %s: %s", jid, mycid, err), nil
	}

	// var s powc.WatchStorageJobsEvent

	var aborted bool
	var abortMsg string
	var storageJob *userPb.StorageJob
	select {
	case <-ctx.Done():
		log.Infof("job %s status watching canceled\n", jid)
		return true, "watching cancelled", nil
	case s, ok := <-chJob:
		if !ok {
			return true, "powergate closed communication chan", nil
		}
		if s.Err != nil {
			log.Errorf("job %s update: %s", jid, s.Err)
			aborted = true
			abortMsg = s.Err.Error()
		}
		storageJob = s.Res.StorageJob
	}

	if !aborted && !isJobStatusFinal(storageJob) {
		return true, "no final status yet", nil
	}

	// On success, save Deal data in the underlying Bucket thread. On failure,
	// save the error message. Also update status on Mongo for the archive.
	if storageJob.Status == userPb.JobStatus_JOB_STATUS_SUCCESS {
		err := saveDealsInDB(ctx, pgClient, storageDeal.FFSToken, mycid, storageDealDB, assetDB)
		if err != nil {
			return true, fmt.Sprintf("saving deal data in archive: %s", err), nil
		}
	} else {
		log.Info("job.Status", storageJob.Status, storageJob.Status.String())
		storageDealDB.UpdateStorageDeal(mycid.String(), 2, "failed to create filecoin storage deal", "", "", 0)
	}

	msg := "reached final status"
	if aborted {
		msg = "aborted with reason " + abortMsg
	}

	return false, msg, nil
}

func saveDealsInDB(ctx context.Context, pgClient *powc.Client, ffsToken string, c cid.Cid, storageDealDB dataservice.StorageDealDatabase, assetDB dataservice.AssetDatabase) error {
	ctx = context.WithValue(ctx, powc.AuthKey, ffsToken)
	conf := powc.ListConfig{
		Select: client.Executing,
	}
	res, err := pgClient.StorageJobs.List(ctx, conf)
	if err != nil {
		return fmt.Errorf("getting executing storage jobs: %s", err)
	}
	// sh, err := pgClient.FFS.Show(ctxFFS, c)
	// if err != nil {
	// 	return fmt.Errorf("getting cid info: %s", err)
	// }

	log.Info("sdidb res:", res)

	// proposals := sh.GetCidInfo().GetCold().GetFilecoin().GetProposals()

	// log.Info("proposals", proposals)

	// if len(proposals) > 0 {
	// 	for _, prop := range proposals {
	// 		priceAttoFIL := prop.EpochPrice * uint64(prop.Duration)
	// 		priceAttoFILBigInt := new(big.Int).SetUint64(priceAttoFIL)
	// 		priceFIL := float64(priceAttoFIL) * math.Pow(10, -18)
	// 		log.Info("priceAttoFIL", priceAttoFIL, "priceAttoFILBigInt", priceAttoFILBigInt, "priceFIL", priceFIL)

	// 		log.Info("ProposalCid", prop.ProposalCid)
	// 		log.Info("Renewed", prop.Renewed)
	// 		log.Info("Miner", prop.Miner)
	// 		log.Info("StartEpoch", prop.StartEpoch)
	// 		log.Info("ActivationEpoch", prop.ActivationEpoch)
	// 		log.Info("Duration", prop.Duration)
	// 		log.Info("EpochPrice", prop.EpochPrice)

	// 		storageDealDB.UpdateStorageDeal(c.String(), 1, internal.AssetStatusMap[4], prop.Miner, priceAttoFILBigInt.String(), 0)
	// 		sDeal, err := storageDealDB.GetStorageDealByCID(c.String())
	// 		if err == nil {
	// 			assetID := sDeal.AssetID
	// 			assetDB.UpdateAssetStatus(assetID, 4, internal.AssetStatusMap[4], false)
	// 		}
	// 	}
	// }

	return nil
}

func isJobStatusFinal(sJ *userPb.StorageJob) bool {
	return sJ.Status == userPb.JobStatus_JOB_STATUS_SUCCESS ||
		sJ.Status == userPb.JobStatus_JOB_STATUS_CANCELED ||
		sJ.Status == userPb.JobStatus_JOB_STATUS_FAILED
}
