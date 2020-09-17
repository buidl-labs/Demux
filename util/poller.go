package util

import (
	"context"
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/buidl-labs/Demux/dataservice"
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
		log.Fatalln("Please set env variable `POLL_INTERVAL`")
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
			deals := dataservice.GetPendingDeals()
			fmt.Println("Pending storage deals:")
			fmt.Println(deals)
			for _, deal := range deals {
				fmt.Println("dealjid", deal.JID)
				cidcorrtype, _ := cid.Decode(deal.CID)
				b, s, err := pollStorageDealProgress(ctx, pgClient, ffs.JobID(deal.JID), cidcorrtype, deal)
				fmt.Println("pollprog", b, s, err)
			}
		}
	}
}

func pollStorageDealProgress(ctx context.Context, pgClient *powc.Client, jid ffs.JobID, mycid cid.Cid, storageDeal model.StorageDeal) (bool, string, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	ctx = context.WithValue(ctx, powc.AuthKey, storageDeal.Token)
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
		log.Infof("job %s status watching canceled", jid)
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

	// Step 2: On success, save Deal data in the underlying Bucket thread. On
	// failure save the error message. Also update status on Mongo for the archive.
	if job.Status == ffs.Success {
		err := saveDealsInDB(ctx, pgClient, storageDeal.Token, mycid)
		if err != nil {
			return true, fmt.Sprintf("saving deal data in archive: %s", err), nil
		}
		dataservice.UpdateStorageDealStatus(fmt.Sprintf("%s", mycid), 1)
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

	fmt.Println("proposals", proposals)
	if len(proposals) > 0 {
		for _, prop := range proposals {
			priceAttoFIL := prop.EpochPrice * uint64(prop.Duration)
			priceFIL := float64(priceAttoFIL) * math.Pow(10, -18)
			fmt.Println("***********************")
			fmt.Println("ProposalCid", prop.ProposalCid)
			fmt.Println("Renewed", prop.Renewed)
			fmt.Println("Miner", prop.Miner)
			fmt.Println("StartEpoch", prop.StartEpoch)
			fmt.Println("ActivationEpoch", prop.ActivationEpoch)
			fmt.Println("Duration", prop.Duration)
			fmt.Println("EpochPrice", prop.EpochPrice)

			dataservice.UpdateStorageDeal(c.String(), priceFIL, 0, prop.Miner)
		}
	}

	return nil
}

func isJobStatusFinal(js ffs.JobStatus) bool {
	return js == ffs.Success ||
		js == ffs.Canceled ||
		js == ffs.Failed
}
