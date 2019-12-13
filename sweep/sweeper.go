package sweep

import (
	"errors"
	"fmt"
	"math/rand"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil"
	"github.com/davecgh/go-spew/spew"
	"github.com/lightningnetwork/lnd/chainntnfs"
	"github.com/lightningnetwork/lnd/input"
	"github.com/lightningnetwork/lnd/lnwallet"
	"github.com/lightningnetwork/lnd/lnwallet/chainfee"
)

const (
	// DefaultMaxFeeRate is the default maximum fee rate allowed within the
	// UtxoSweeper. The current value is equivalent to a fee rate of 10,000
	// sat/vbyte.
	DefaultMaxFeeRate = chainfee.FeePerKwFloor * 1e4

	// DefaultFeeRateBucketSize is the default size of fee rate buckets
	// we'll use when clustering inputs into buckets with similar fee rates
	// within the UtxoSweeper.
	//
	// Given a minimum relay fee rate of 1 sat/vbyte, a multiplier of 10
	// would result in the following fee rate buckets up to the maximum fee
	// rate:
	//
	//   #1: min = 1 sat/vbyte, max = 10 sat/vbyte
	//   #2: min = 11 sat/vbyte, max = 20 sat/vbyte...
	DefaultFeeRateBucketSize = 10
)

var (
	// ErrRemoteSpend is returned in case an output that we try to sweep is
	// confirmed in a tx of the remote party.
	ErrRemoteSpend = errors.New("remote party swept utxo")

	// ErrTooManyAttempts is returned in case sweeping an output has failed
	// for the configured max number of attempts.
	ErrTooManyAttempts = errors.New("sweep failed after max attempts")

	// ErrNoFeePreference is returned when we attempt to satisfy a sweep
	// request from a client whom did not specify a fee preference.
	ErrNoFeePreference = errors.New("no fee preference specified")

	// ErrSweeperShuttingDown is an error returned when a client attempts to
	// make a request to the UtxoSweeper, but it is unable to handle it as
	// it is/has already been stoppepd.
	ErrSweeperShuttingDown = errors.New("utxo sweeper shutting down")

	// DefaultMaxSweepAttempts specifies the default maximum number of times
	// an input is included in a publish attempt before giving up and
	// returning an error to the caller.
	DefaultMaxSweepAttempts = 10
)

// Params contains the parameters that control the sweeping process.
type Params struct {
	// Fee is the fee preference of the client who requested the input to be
	// swept. If a confirmation target is specified, then we'll map it into
	// a fee rate whenever we attempt to cluster inputs for a sweep.
	Fee FeePreference
}

// pendingInput is created when an input reaches the main loop for the first
// time. It wraps the input and tracks all relevant state that is needed for
// sweeping.
type pendingInput struct {
	input.Input

	// listeners is a list of channels over which the final outcome of the
	// sweep needs to be broadcasted.
	listeners []chan Result

	// ntfnRegCancel is populated with a function that cancels the chain
	// notifier spend registration.
	ntfnRegCancel func()

	// minPublishHeight indicates the minimum block height at which this
	// input may be (re)published.
	minPublishHeight int32

	// publishAttempts records the number of attempts that have already been
	// made to sweep this tx.
	publishAttempts int

	// params contains the parameters that control the sweeping process.
	params Params

	// lastFeeRate is the most recent fee rate used for this input within a
	// transaction broadcast to the network.
	lastFeeRate chainfee.SatPerKWeight
}

// parameters returns the sweep parameters for this input.
//
// NOTE: Part of the txInput interface.
func (p *pendingInput) parameters() Params {
	return p.params
}

// pendingInputs is a type alias for a set of pending inputs.
type pendingInputs = map[wire.OutPoint]*pendingInput

// inputCluster is a helper struct to gather a set of pending inputs that should
// be swept with the specified fee rate.
type inputCluster struct {
	sweepFeeRate chainfee.SatPerKWeight
	inputs       pendingInputs
}

// pendingSweepsReq is an internal message we'll use to represent an external
// caller's intent to retrieve all of the pending inputs the UtxoSweeper is
// attempting to sweep.
type pendingSweepsReq struct {
	respChan chan map[wire.OutPoint]*PendingInput
}

// PendingInput contains information about an input that is currently being
// swept by the UtxoSweeper.
type PendingInput struct {
	// OutPoint is the identify outpoint of the input being swept.
	OutPoint wire.OutPoint

	// WitnessType is the witness type of the input being swept.
	WitnessType input.WitnessType

	// Amount is the amount of the input being swept.
	Amount btcutil.Amount

	// LastFeeRate is the most recent fee rate used for the input being
	// swept within a transaction broadcast to the network.
	LastFeeRate chainfee.SatPerKWeight

	// BroadcastAttempts is the number of attempts we've made to sweept the
	// input.
	BroadcastAttempts int

	// NextBroadcastHeight is the next height of the chain at which we'll
	// attempt to broadcast a transaction sweeping the input.
	NextBroadcastHeight uint32
}

// bumpFeeReq is an internal message we'll use to represent an external caller's
// intent to bump the fee rate of a given input.
type bumpFeeReq struct {
	input         wire.OutPoint
	feePreference FeePreference
	responseChan  chan *bumpFeeResp
}

// bumpFeeResp is an internal message we'll use to hand off the response of a
// bumpFeeReq from the UtxoSweeper's main event loop back to the caller.
type bumpFeeResp struct {
	resultChan chan Result
	err        error
}

// UtxoSweeper is responsible for sweeping outputs back into the wallet
type UtxoSweeper struct {
	started uint32 // To be used atomically.
	stopped uint32 // To be used atomically.

	cfg *UtxoSweeperConfig

	newInputs chan *sweepInputMessage
	spendChan chan *chainntnfs.SpendDetail

	// pendingSweepsReq is a channel that will be sent requests by external
	// callers in order to retrieve the set of pending inputs the
	// UtxoSweeper is attempting to sweep.
	pendingSweepsReqs chan *pendingSweepsReq

	// bumpFeeReqs is a channel that will be sent requests by external
	// callers who wish to bump the fee rate of a given input.
	bumpFeeReqs chan *bumpFeeReq

	// pendingInputs is the total set of inputs the UtxoSweeper has been
	// requested to sweep.
	pendingInputs pendingInputs

	// timer is the channel that signals expiry of the sweep batch timer.
	timer <-chan time.Time

	testSpendChan chan wire.OutPoint

	currentOutputScript []byte

	relayFeeRate chainfee.SatPerKWeight

	quit chan struct{}
	wg   sync.WaitGroup
}

// UtxoSweeperConfig contains dependencies of UtxoSweeper.
type UtxoSweeperConfig struct {
	// GenSweepScript generates a P2WKH script belonging to the wallet where
	// funds can be swept.
	GenSweepScript func() ([]byte, error)

	// FeeEstimator is used when crafting sweep transactions to estimate
	// the necessary fee relative to the expected size of the sweep
	// transaction.
	FeeEstimator chainfee.Estimator

	// Wallet contains the wallet functions that sweeper requires.
	Wallet Wallet

	// NewBatchTimer creates a channel that will be sent on when a certain
	// time window has passed. During this time window, new inputs can still
	// be added to the sweep tx that is about to be generated.
	NewBatchTimer func() <-chan time.Time

	// Notifier is an instance of a chain notifier we'll use to watch for
	// certain on-chain events.
	Notifier chainntnfs.ChainNotifier

	// Store stores the published sweeper txes.
	Store SweeperStore

	// Signer is used by the sweeper to generate valid witnesses at the
	// time the incubated outputs need to be spent.
	Signer input.Signer

	// MaxInputsPerTx specifies the default maximum number of inputs allowed
	// in a single sweep tx. If more need to be swept, multiple txes are
	// created and published.
	MaxInputsPerTx int

	// MaxSweepAttempts specifies the maximum number of times an input is
	// included in a publish attempt before giving up and returning an error
	// to the caller.
	MaxSweepAttempts int

	// NextAttemptDeltaFunc returns given the number of already attempted
	// sweeps, how many blocks to wait before retrying to sweep.
	NextAttemptDeltaFunc func(int) int32

	// MaxFeeRate is the the maximum fee rate allowed within the
	// UtxoSweeper.
	MaxFeeRate chainfee.SatPerKWeight

	// FeeRateBucketSize is the default size of fee rate buckets we'll use
	// when clustering inputs into buckets with similar fee rates within the
	// UtxoSweeper.
	//
	// Given a minimum relay fee rate of 1 sat/vbyte, a fee rate bucket size
	// of 10 would result in the following fee rate buckets up to the
	// maximum fee rate:
	//
	//   #1: min = 1 sat/vbyte, max (exclusive) = 11 sat/vbyte
	//   #2: min = 11 sat/vbyte, max (exclusive) = 21 sat/vbyte...
	FeeRateBucketSize int
}

// Result is the struct that is pushed through the result channel. Callers can
// use this to be informed of the final sweep result. In case of a remote
// spend, Err will be ErrRemoteSpend.
type Result struct {
	// Err is the final result of the sweep. It is nil when the input is
	// swept successfully by us. ErrRemoteSpend is returned when another
	// party took the input.
	Err error

	// Tx is the transaction that spent the input.
	Tx *wire.MsgTx
}

// sweepInputMessage structs are used in the internal channel between the
// SweepInput call and the sweeper main loop.
type sweepInputMessage struct {
	input      input.Input
	params     Params
	resultChan chan Result
}

// New returns a new Sweeper instance.
func New(cfg *UtxoSweeperConfig) *UtxoSweeper {
	return &UtxoSweeper{
		cfg:               cfg,
		newInputs:         make(chan *sweepInputMessage),
		spendChan:         make(chan *chainntnfs.SpendDetail),
		bumpFeeReqs:       make(chan *bumpFeeReq),
		pendingSweepsReqs: make(chan *pendingSweepsReq),
		quit:              make(chan struct{}),
		pendingInputs:     make(pendingInputs),
	}
}

// Start starts the process of constructing and publish sweep txes.
func (s *UtxoSweeper) Start() error {
	if !atomic.CompareAndSwapUint32(&s.started, 0, 1) {
		return nil
	}

	log.Tracef("Sweeper starting")

	// Retrieve last published tx from database.
	lastTx, err := s.cfg.Store.GetLastPublishedTx()
	if err != nil {
		return fmt.Errorf("get last published tx: %v", err)
	}

	// Republish in case the previous call crashed lnd. We don't care about
	// the return value, because inputs will be re-offered and retried
	// anyway. The only reason we republish here is to prevent the corner
	// case where lnd goes into a restart loop because of a crashing publish
	// tx where we keep deriving new output script. By publishing and
	// possibly crashing already now, we haven't derived a new output script
	// yet.
	if lastTx != nil {
		log.Debugf("Publishing last tx %v", lastTx.TxHash())

		// Error can be ignored. Because we are starting up, there are
		// no pending inputs to update based on the publish result.
		err := s.cfg.Wallet.PublishTransaction(lastTx)
		if err != nil && err != lnwallet.ErrDoubleSpend {
			log.Errorf("last tx publish: %v", err)
		}
	}

	// Retrieve relay fee for dust limit calculation. Assume that this will
	// not change from here on.
	s.relayFeeRate = s.cfg.FeeEstimator.RelayFeePerKW()

	// We need to register for block epochs and retry sweeping every block.
	// We should get a notification with the current best block immediately
	// if we don't provide any epoch. We'll wait for that in the collector.
	blockEpochs, err := s.cfg.Notifier.RegisterBlockEpochNtfn(nil)
	if err != nil {
		return fmt.Errorf("register block epoch ntfn: %v", err)
	}

	// Start sweeper main loop.
	s.wg.Add(1)
	go func() {
		defer blockEpochs.Cancel()
		defer s.wg.Done()

		s.collector(blockEpochs.Epochs)
	}()

	return nil
}

// Stop stops sweeper from listening to block epochs and constructing sweep
// txes.
func (s *UtxoSweeper) Stop() error {
	if !atomic.CompareAndSwapUint32(&s.stopped, 0, 1) {
		return nil
	}

	log.Debugf("Sweeper shutting down")

	close(s.quit)
	s.wg.Wait()

	log.Debugf("Sweeper shut down")

	return nil
}

// SweepInput sweeps inputs back into the wallet. The inputs will be batched and
// swept after the batch time window ends. A custom fee preference can be
// provided to determine what fee rate should be used for the input. Note that
// the input may not always be swept with this exact value, as its possible for
// it to be batched under the same transaction with other similar fee rate
// inputs.
//
// NOTE: Extreme care needs to be taken that input isn't changed externally.
// Because it is an interface and we don't know what is exactly behind it, we
// cannot make a local copy in sweeper.
func (s *UtxoSweeper) SweepInput(input input.Input,
	params Params) (chan Result, error) {

	if input == nil || input.OutPoint() == nil || input.SignDesc() == nil {
		return nil, errors.New("nil input received")
	}

	// Ensure the client provided a sane fee preference.
	if _, err := s.feeRateForPreference(params.Fee); err != nil {
		return nil, err
	}

	log.Infof("Sweep request received: out_point=%v, witness_type=%v, "+
		"time_lock=%v, amount=%v, fee_preference=%v", input.OutPoint(),
		input.WitnessType(), input.BlocksToMaturity(),
		btcutil.Amount(input.SignDesc().Output.Value),
		params.Fee)

	sweeperInput := &sweepInputMessage{
		input:      input,
		params:     params,
		resultChan: make(chan Result, 1),
	}

	// Deliver input to main event loop.
	select {
	case s.newInputs <- sweeperInput:
	case <-s.quit:
		return nil, ErrSweeperShuttingDown
	}

	return sweeperInput.resultChan, nil
}

// feeRateForPreference returns a fee rate for the given fee preference. It
// ensures that the fee rate respects the bounds of the UtxoSweeper.
func (s *UtxoSweeper) feeRateForPreference(
	feePreference FeePreference) (chainfee.SatPerKWeight, error) {

	// Ensure a type of fee preference is specified to prevent using a
	// default below.
	if feePreference.FeeRate == 0 && feePreference.ConfTarget == 0 {
		return 0, ErrNoFeePreference
	}

	feeRate, err := DetermineFeePerKw(s.cfg.FeeEstimator, feePreference)
	if err != nil {
		return 0, err
	}
	if feeRate < s.relayFeeRate {
		return 0, fmt.Errorf("fee preference resulted in invalid fee "+
			"rate %v, mininum is %v", feeRate, s.relayFeeRate)
	}
	if feeRate > s.cfg.MaxFeeRate {
		return 0, fmt.Errorf("fee preference resulted in invalid fee "+
			"rate %v, maximum is %v", feeRate, s.cfg.MaxFeeRate)
	}

	return feeRate, nil
}

// collector is the sweeper main loop. It processes new inputs, spend
// notifications and counts down to publication of the sweep tx.
func (s *UtxoSweeper) collector(blockEpochs <-chan *chainntnfs.BlockEpoch) {
	// We registered for the block epochs with a nil request. The notifier
	// should send us the current best block immediately. So we need to wait
	// for it here because we need to know the current best height.
	var bestHeight int32
	select {
	case bestBlock := <-blockEpochs:
		bestHeight = bestBlock.Height

	case <-s.quit:
		return
	}

	for {
		select {
		// A new inputs is offered to the sweeper. We check to see if we
		// are already trying to sweep this input and if not, set up a
		// listener for spend and schedule a sweep.
		case input := <-s.newInputs:
			outpoint := *input.input.OutPoint()
			pendInput, pending := s.pendingInputs[outpoint]
			if pending {
				log.Debugf("Already pending input %v received",
					outpoint)

				// Add additional result channel to signal
				// spend of this input.
				pendInput.listeners = append(
					pendInput.listeners, input.resultChan,
				)
				continue
			}

			// Create a new pendingInput and initialize the
			// listeners slice with the passed in result channel. If
			// this input is offered for sweep again, the result
			// channel will be appended to this slice.
			pendInput = &pendingInput{
				listeners:        []chan Result{input.resultChan},
				Input:            input.input,
				minPublishHeight: bestHeight,
				params:           input.params,
			}
			s.pendingInputs[outpoint] = pendInput

			// Start watching for spend of this input, either by us
			// or the remote party.
			cancel, err := s.waitForSpend(
				outpoint,
				input.input.SignDesc().Output.PkScript,
				input.input.HeightHint(),
			)
			if err != nil {
				err := fmt.Errorf("wait for spend: %v", err)
				s.signalAndRemove(&outpoint, Result{Err: err})
				continue
			}
			pendInput.ntfnRegCancel = cancel

			// Check to see if with this new input a sweep tx can be
			// formed.
			if err := s.scheduleSweep(bestHeight); err != nil {
				log.Errorf("schedule sweep: %v", err)
			}

		// A spend of one of our inputs is detected. Signal sweep
		// results to the caller(s).
		case spend := <-s.spendChan:
			// For testing purposes.
			if s.testSpendChan != nil {
				s.testSpendChan <- *spend.SpentOutPoint
			}

			// Query store to find out if we every published this
			// tx.
			spendHash := *spend.SpenderTxHash
			isOurTx, err := s.cfg.Store.IsOurTx(spendHash)
			if err != nil {
				log.Errorf("cannot determine if tx %v "+
					"is ours: %v", spendHash, err,
				)
				continue
			}

			log.Debugf("Detected spend related to in flight inputs "+
				"(is_ours=%v): %v",
				newLogClosure(func() string {
					return spew.Sdump(spend.SpendingTx)
				}), isOurTx,
			)

			// Signal sweep results for inputs in this confirmed
			// tx.
			for _, txIn := range spend.SpendingTx.TxIn {
				outpoint := txIn.PreviousOutPoint

				// Check if this input is known to us. It could
				// probably be unknown if we canceled the
				// registration, deleted from pendingInputs but
				// the ntfn was in-flight already. Or this could
				// be not one of our inputs.
				_, ok := s.pendingInputs[outpoint]
				if !ok {
					continue
				}

				// Return either a nil or a remote spend result.
				var err error
				if !isOurTx {
					err = ErrRemoteSpend
				}

				// Signal result channels.
				s.signalAndRemove(&outpoint, Result{
					Tx:  spend.SpendingTx,
					Err: err,
				})
			}

			// Now that an input of ours is spent, we can try to
			// resweep the remaining inputs.
			if err := s.scheduleSweep(bestHeight); err != nil {
				log.Errorf("schedule sweep: %v", err)
			}

		// A new external request has been received to retrieve all of
		// the inputs we're currently attempting to sweep.
		case req := <-s.pendingSweepsReqs:
			req.respChan <- s.handlePendingSweepsReq(req)

		// A new external request has been received to bump the fee rate
		// of a given input.
		case req := <-s.bumpFeeReqs:
			resultChan, err := s.handleBumpFeeReq(req, bestHeight)
			req.responseChan <- &bumpFeeResp{
				resultChan: resultChan,
				err:        err,
			}

		// The timer expires and we are going to (re)sweep.
		case <-s.timer:
			log.Debugf("Sweep timer expired")

			// Set timer to nil so we know that a new timer needs to
			// be started when new inputs arrive.
			s.timer = nil

			// We'll attempt to cluster all of our inputs with
			// similar fee rates. Before attempting to sweep them,
			// we'll sort them in descending fee rate order. We do
			// this to ensure any inputs which have had their fee
			// rate bumped are broadcast first in order enforce the
			// RBF policy.
			inputClusters := s.clusterBySweepFeeRate()
			sort.Slice(inputClusters, func(i, j int) bool {
				return inputClusters[i].sweepFeeRate >
					inputClusters[j].sweepFeeRate
			})
			for _, cluster := range inputClusters {
				err := s.sweepCluster(cluster, bestHeight)
				if err != nil {
					log.Errorf("input cluster sweep: %v",
						err)
				}
			}

		// A new block comes in. Things may have changed, so we retry a
		// sweep.
		case epoch, ok := <-blockEpochs:
			if !ok {
				return
			}

			bestHeight = epoch.Height

			log.Debugf("New block: height=%v, sha=%v",
				epoch.Height, epoch.Hash)

			if err := s.scheduleSweep(bestHeight); err != nil {
				log.Errorf("schedule sweep: %v", err)
			}

		case <-s.quit:
			return
		}
	}
}

// sweepCluster tries to sweep the given input cluster.
func (s *UtxoSweeper) sweepCluster(cluster inputCluster,
	currentHeight int32) error {

	// Examine pending inputs and try to construct lists of inputs.
	inputLists, err := s.getInputLists(cluster, currentHeight)
	if err != nil {
		return fmt.Errorf("unable to examine pending inputs: %v", err)
	}

	// Sweep selected inputs.
	for _, inputs := range inputLists {
		err := s.sweep(inputs, cluster.sweepFeeRate, currentHeight)
		if err != nil {
			return fmt.Errorf("unable to sweep inputs: %v", err)
		}
	}

	return nil
}

// bucketForFeeReate determines the proper bucket for a fee rate. This is done
// in order to batch inputs with similar fee rates together.
func (s *UtxoSweeper) bucketForFeeRate(
	feeRate chainfee.SatPerKWeight) int {

	return int(feeRate-s.relayFeeRate) / s.cfg.FeeRateBucketSize
}

// clusterBySweepFeeRate takes the set of pending inputs within the UtxoSweeper
// and clusters those together with similar fee rates. Each cluster contains a
// sweep fee rate, which is determined by calculating the average fee rate of
// all inputs within that cluster.
func (s *UtxoSweeper) clusterBySweepFeeRate() []inputCluster {
	bucketInputs := make(map[int]pendingInputs)
	inputFeeRates := make(map[wire.OutPoint]chainfee.SatPerKWeight)

	// First, we'll group together all inputs with similar fee rates. This
	// is done by determining the fee rate bucket they should belong in.
	for op, input := range s.pendingInputs {
		feeRate, err := s.feeRateForPreference(input.params.Fee)
		if err != nil {
			log.Warnf("Skipping input %v: %v", op, err)
			continue
		}
		feeGroup := s.bucketForFeeRate(feeRate)

		inputs, ok := bucketInputs[feeGroup]
		if !ok {
			inputs = make(pendingInputs)
			bucketInputs[feeGroup] = inputs
		}

		input.lastFeeRate = feeRate
		inputs[op] = input
		inputFeeRates[op] = feeRate
	}

	// We'll then determine the sweep fee rate for each set of inputs by
	// calculating the average fee rate of the inputs within each set.
	inputClusters := make([]inputCluster, 0, len(bucketInputs))
	for _, inputs := range bucketInputs {
		var sweepFeeRate chainfee.SatPerKWeight
		for op := range inputs {
			sweepFeeRate += inputFeeRates[op]
		}
		sweepFeeRate /= chainfee.SatPerKWeight(len(inputs))
		inputClusters = append(inputClusters, inputCluster{
			sweepFeeRate: sweepFeeRate,
			inputs:       inputs,
		})
	}

	return inputClusters
}

// scheduleSweep starts the sweep timer to create an opportunity for more inputs
// to be added.
func (s *UtxoSweeper) scheduleSweep(currentHeight int32) error {
	// The timer is already ticking, no action needed for the sweep to
	// happen.
	if s.timer != nil {
		log.Debugf("Timer still ticking")
		return nil
	}

	// We'll only start our timer once we have inputs we're able to sweep.
	startTimer := false
	for _, cluster := range s.clusterBySweepFeeRate() {
		// Examine pending inputs and try to construct lists of inputs.
		inputLists, err := s.getInputLists(cluster, currentHeight)
		if err != nil {
			return fmt.Errorf("get input lists: %v", err)
		}

		log.Infof("Sweep candidates at height=%v with fee_rate=%v, "+
			"yield %v distinct txns", currentHeight,
			cluster.sweepFeeRate, len(inputLists))

		if len(inputLists) != 0 {
			startTimer = true
			break
		}
	}
	if !startTimer {
		return nil
	}

	// Start sweep timer to create opportunity for more inputs to be added
	// before a tx is constructed.
	s.timer = s.cfg.NewBatchTimer()

	log.Debugf("Sweep timer started")

	return nil
}

// signalAndRemove notifies the listeners of the final result of the input
// sweep. It cancels any pending spend notification and removes the input from
// the list of pending inputs. When this function returns, the sweeper has
// completely forgotten about the input.
func (s *UtxoSweeper) signalAndRemove(outpoint *wire.OutPoint, result Result) {
	pendInput := s.pendingInputs[*outpoint]
	listeners := pendInput.listeners

	if result.Err == nil {
		log.Debugf("Dispatching sweep success for %v to %v listeners",
			outpoint, len(listeners),
		)
	} else {
		log.Debugf("Dispatching sweep error for %v to %v listeners: %v",
			outpoint, len(listeners), result.Err,
		)
	}

	// Signal all listeners. Channel is buffered. Because we only send once
	// on every channel, it should never block.
	for _, resultChan := range listeners {
		resultChan <- result
	}

	// Cancel spend notification with chain notifier. This is not necessary
	// in case of a success, except for that a reorg could still happen.
	if pendInput.ntfnRegCancel != nil {
		log.Debugf("Canceling spend ntfn for %v", outpoint)

		pendInput.ntfnRegCancel()
	}

	// Inputs are no longer pending after result has been sent.
	delete(s.pendingInputs, *outpoint)
}

// getInputLists goes through the given inputs and constructs multiple distinct
// sweep lists with the given fee rate, each up to the configured maximum number
// of inputs. Negative yield inputs are skipped. Transactions with an output
// below the dust limit are not published. Those inputs remain pending and will
// be bundled with future inputs if possible.
func (s *UtxoSweeper) getInputLists(cluster inputCluster,
	currentHeight int32) ([]inputSet, error) {

	// Filter for inputs that need to be swept. Create two lists: all
	// sweepable inputs and a list containing only the new, never tried
	// inputs.
	//
	// We want to create as large a tx as possible, so we return a final set
	// list that starts with sets created from all inputs. However, there is
	// a chance that those txes will not publish, because they already
	// contain inputs that failed before. Therefore we also add sets
	// consisting of only new inputs to the list, to make sure that new
	// inputs are given a good, isolated chance of being published.
	var newInputs, retryInputs []txInput
	for _, input := range cluster.inputs {
		// Skip inputs that have a minimum publish height that is not
		// yet reached.
		if input.minPublishHeight > currentHeight {
			continue
		}

		// Add input to the either one of the lists.
		if input.publishAttempts == 0 {
			newInputs = append(newInputs, input)
		} else {
			retryInputs = append(retryInputs, input)
		}
	}

	// If there is anything to retry, combine it with the new inputs and
	// form input sets.
	var allSets []inputSet
	if len(retryInputs) > 0 {
		var err error
		allSets, err = generateInputPartitionings(
			append(retryInputs, newInputs...), s.relayFeeRate,
			cluster.sweepFeeRate, s.cfg.MaxInputsPerTx,
		)
		if err != nil {
			return nil, fmt.Errorf("input partitionings: %v", err)
		}
	}

	// Create sets for just the new inputs.
	newSets, err := generateInputPartitionings(
		newInputs, s.relayFeeRate, cluster.sweepFeeRate,
		s.cfg.MaxInputsPerTx,
	)
	if err != nil {
		return nil, fmt.Errorf("input partitionings: %v", err)
	}

	log.Debugf("Sweep candidates at height=%v: total_num_pending=%v, "+
		"total_num_new=%v", currentHeight, len(allSets), len(newSets))

	// Append the new sets at the end of the list, because those tx likely
	// have a higher fee per input.
	return append(allSets, newSets...), nil
}

// sweep takes a set of preselected inputs, creates a sweep tx and publishes the
// tx. The output address is only marked as used if the publish succeeds.
func (s *UtxoSweeper) sweep(inputs inputSet, feeRate chainfee.SatPerKWeight,
	currentHeight int32) error {

	// Generate an output script if there isn't an unused script available.
	if s.currentOutputScript == nil {
		pkScript, err := s.cfg.GenSweepScript()
		if err != nil {
			return fmt.Errorf("gen sweep script: %v", err)
		}
		s.currentOutputScript = pkScript
	}

	// Create sweep tx.
	tx, err := createSweepTx(
		inputs, s.currentOutputScript, uint32(currentHeight), feeRate,
		s.cfg.Signer,
	)
	if err != nil {
		return fmt.Errorf("create sweep tx: %v", err)
	}

	// Add tx before publication, so that we will always know that a spend
	// by this tx is ours. Otherwise if the publish doesn't return, but did
	// publish, we loose track of this tx. Even republication on startup
	// doesn't prevent this, because that call returns a double spend error
	// then and would also not add the hash to the store.
	err = s.cfg.Store.NotifyPublishTx(tx)
	if err != nil {
		return fmt.Errorf("notify publish tx: %v", err)
	}

	// Publish sweep tx.
	log.Debugf("Publishing sweep tx %v, num_inputs=%v, height=%v",
		tx.TxHash(), len(tx.TxIn), currentHeight)

	log.Tracef("Sweep tx at height=%v: %v", currentHeight,
		newLogClosure(func() string {
			return spew.Sdump(tx)
		}),
	)

	err = s.cfg.Wallet.PublishTransaction(tx)

	// In case of an unexpected error, don't try to recover.
	if err != nil && err != lnwallet.ErrDoubleSpend {
		return fmt.Errorf("publish tx: %v", err)
	}

	// Keep the output script in case of an error, so that it can be reused
	// for the next transaction and causes no address inflation.
	if err == nil {
		s.currentOutputScript = nil
	}

	// Reschedule sweep.
	for _, input := range tx.TxIn {
		pi, ok := s.pendingInputs[input.PreviousOutPoint]
		if !ok {
			// It can be that the input has been removed because it
			// exceed the maximum number of attempts in a previous
			// input set.
			continue
		}

		// Record another publish attempt.
		pi.publishAttempts++

		// We don't care what the result of the publish call was. Even
		// if it is published successfully, it can still be that it
		// needs to be retried. Call NextAttemptDeltaFunc to calculate
		// when to resweep this input.
		nextAttemptDelta := s.cfg.NextAttemptDeltaFunc(
			pi.publishAttempts,
		)

		pi.minPublishHeight = currentHeight + nextAttemptDelta

		log.Debugf("Rescheduling input %v after %v attempts at "+
			"height %v (delta %v)", input.PreviousOutPoint,
			pi.publishAttempts, pi.minPublishHeight,
			nextAttemptDelta)

		if pi.publishAttempts >= s.cfg.MaxSweepAttempts {
			// Signal result channels sweep result.
			s.signalAndRemove(&input.PreviousOutPoint, Result{
				Err: ErrTooManyAttempts,
			})
		}
	}

	return nil
}

// waitForSpend registers a spend notification with the chain notifier. It
// returns a cancel function that can be used to cancel the registration.
func (s *UtxoSweeper) waitForSpend(outpoint wire.OutPoint,
	script []byte, heightHint uint32) (func(), error) {

	log.Debugf("Wait for spend of %v", outpoint)

	spendEvent, err := s.cfg.Notifier.RegisterSpendNtfn(
		&outpoint, script, heightHint,
	)
	if err != nil {
		return nil, fmt.Errorf("register spend ntfn: %v", err)
	}

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		select {
		case spend, ok := <-spendEvent.Spend:
			if !ok {
				log.Debugf("Spend ntfn for %v canceled",
					outpoint)
				return
			}

			log.Debugf("Delivering spend ntfn for %v",
				outpoint)
			select {
			case s.spendChan <- spend:
				log.Debugf("Delivered spend ntfn for %v",
					outpoint)

			case <-s.quit:
			}
		case <-s.quit:
		}
	}()

	return spendEvent.Cancel, nil
}

// PendingInputs returns the set of inputs that the UtxoSweeper is currently
// attempting to sweep.
func (s *UtxoSweeper) PendingInputs() (map[wire.OutPoint]*PendingInput, error) {
	respChan := make(chan map[wire.OutPoint]*PendingInput, 1)
	select {
	case s.pendingSweepsReqs <- &pendingSweepsReq{
		respChan: respChan,
	}:
	case <-s.quit:
		return nil, ErrSweeperShuttingDown
	}

	select {
	case pendingSweeps := <-respChan:
		return pendingSweeps, nil
	case <-s.quit:
		return nil, ErrSweeperShuttingDown
	}
}

// handlePendingSweepsReq handles a request to retrieve all pending inputs the
// UtxoSweeper is attempting to sweep.
func (s *UtxoSweeper) handlePendingSweepsReq(
	req *pendingSweepsReq) map[wire.OutPoint]*PendingInput {

	pendingInputs := make(map[wire.OutPoint]*PendingInput, len(s.pendingInputs))
	for _, pendingInput := range s.pendingInputs {
		// Only the exported fields are set, as we expect the response
		// to only be consumed externally.
		op := *pendingInput.OutPoint()
		pendingInputs[op] = &PendingInput{
			OutPoint:    op,
			WitnessType: pendingInput.WitnessType(),
			Amount: btcutil.Amount(
				pendingInput.SignDesc().Output.Value,
			),
			LastFeeRate:         pendingInput.lastFeeRate,
			BroadcastAttempts:   pendingInput.publishAttempts,
			NextBroadcastHeight: uint32(pendingInput.minPublishHeight),
		}
	}

	return pendingInputs
}

// BumpFee allows bumping the fee of an input being swept by the UtxoSweeper
// according to the provided fee preference. The new fee preference will be used
// for a new sweep transaction of the input that will act as a replacement
// transaction (RBF) of the original sweeping transaction, if any.
//
// NOTE: This currently doesn't do any fee rate validation to ensure that a bump
// is actually successful. The responsibility of doing so should be handled by
// the caller.
func (s *UtxoSweeper) BumpFee(input wire.OutPoint,
	feePreference FeePreference) (chan Result, error) {

	// Ensure the client provided a sane fee preference.
	if _, err := s.feeRateForPreference(feePreference); err != nil {
		return nil, err
	}

	responseChan := make(chan *bumpFeeResp, 1)
	select {
	case s.bumpFeeReqs <- &bumpFeeReq{
		input:         input,
		feePreference: feePreference,
		responseChan:  responseChan,
	}:
	case <-s.quit:
		return nil, ErrSweeperShuttingDown
	}

	select {
	case response := <-responseChan:
		return response.resultChan, response.err
	case <-s.quit:
		return nil, ErrSweeperShuttingDown
	}
}

// handleBumpFeeReq handles a bump fee request by simply updating the inputs fee
// preference. Currently, no validation is done on the new fee preference to
// ensure it will properly create a replacement transaction.
//
// TODO(wilmer):
//   * Validate fee preference to ensure we'll create a valid replacement
//     transaction to allow the new fee rate to propagate throughout the
//     network.
//   * Ensure we don't combine this input with any other unconfirmed inputs that
//     did not exist in the original sweep transaction, resulting in an invalid
//     replacement transaction.
func (s *UtxoSweeper) handleBumpFeeReq(req *bumpFeeReq,
	bestHeight int32) (chan Result, error) {

	// If the UtxoSweeper is already trying to sweep this input, then we can
	// simply just increase its fee rate. This will allow the input to be
	// batched with others which also have a similar fee rate, creating a
	// higher fee rate transaction that replaces the original input's
	// sweeping transaction.
	pendingInput, ok := s.pendingInputs[req.input]
	if !ok {
		return nil, lnwallet.ErrNotMine
	}

	log.Debugf("Updating fee preference for %v from %v to %v", req.input,
		pendingInput.params.Fee, req.feePreference)

	pendingInput.params.Fee = req.feePreference

	// We'll reset the input's publish height to the current so that a new
	// transaction can be created that replaces the transaction currently
	// spending the input. We only do this for inputs that have been
	// broadcast at least once to ensure we don't spend an input before its
	// maturity height.
	//
	// NOTE: The UtxoSweeper is not yet offered time-locked inputs, so the
	// check for broadcast attempts is redundant at the moment.
	if pendingInput.publishAttempts > 0 {
		pendingInput.minPublishHeight = bestHeight
	}

	if err := s.scheduleSweep(bestHeight); err != nil {
		log.Errorf("Unable to schedule sweep: %v", err)
	}

	resultChan := make(chan Result, 1)
	pendingInput.listeners = append(pendingInput.listeners, resultChan)

	return resultChan, nil
}

// CreateSweepTx accepts a list of inputs and signs and generates a txn that
// spends from them. This method also makes an accurate fee estimate before
// generating the required witnesses.
//
// The created transaction has a single output sending all the funds back to
// the source wallet, after accounting for the fee estimate.
//
// The value of currentBlockHeight argument will be set as the tx locktime.
// This function assumes that all CLTV inputs will be unlocked after
// currentBlockHeight. Reasons not to use the maximum of all actual CLTV expiry
// values of the inputs:
//
// - Make handling re-orgs easier.
// - Thwart future possible fee sniping attempts.
// - Make us blend in with the bitcoind wallet.
func (s *UtxoSweeper) CreateSweepTx(inputs []input.Input, feePref FeePreference,
	currentBlockHeight uint32) (*wire.MsgTx, error) {

	feePerKw, err := DetermineFeePerKw(s.cfg.FeeEstimator, feePref)
	if err != nil {
		return nil, err
	}

	// Generate the receiving script to which the funds will be swept.
	pkScript, err := s.cfg.GenSweepScript()
	if err != nil {
		return nil, err
	}

	return createSweepTx(
		inputs, pkScript, currentBlockHeight, feePerKw, s.cfg.Signer,
	)
}

// DefaultNextAttemptDeltaFunc is the default calculation for next sweep attempt
// scheduling. It implements exponential back-off with some randomness. This is
// to prevent a stuck tx (for example because fee is too low and can't be bumped
// in btcd) from blocking all other retried inputs in the same tx.
func DefaultNextAttemptDeltaFunc(attempts int) int32 {
	return 1 + rand.Int31n(1<<uint(attempts-1))
}

// init initializes the random generator for random input rescheduling.
func init() {
	rand.Seed(time.Now().Unix())
}
