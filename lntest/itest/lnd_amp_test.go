package itest

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"sort"
	"testing"
	"time"

	"github.com/btcsuite/btcd/btcutil"
	"github.com/lightningnetwork/lnd/amp"
	"github.com/lightningnetwork/lnd/chainreg"
	"github.com/lightningnetwork/lnd/lnrpc"
	"github.com/lightningnetwork/lnd/lnrpc/invoicesrpc"
	"github.com/lightningnetwork/lnd/lnrpc/routerrpc"
	"github.com/lightningnetwork/lnd/lntemp"
	"github.com/lightningnetwork/lnd/lntest"
	"github.com/lightningnetwork/lnd/lntypes"
	"github.com/stretchr/testify/require"
)

// testSendPaymentAMPInvoice tests that we can send an AMP payment to a
// specified AMP invoice using SendPaymentV2.
func testSendPaymentAMPInvoice(ht *lntemp.HarnessTest) {
	succeed := ht.Run("native payaddr", func(t *testing.T) {
		tt := ht.Subtest(t)
		testSendPaymentAMPInvoiceCase(tt, false)
	})

	// Abort the test if failed.
	if !succeed {
		return
	}

	ht.Run("external payaddr", func(t *testing.T) {
		tt := ht.Subtest(t)
		testSendPaymentAMPInvoiceCase(tt, true)
	})
}

func testSendPaymentAMPInvoiceCase(ht *lntemp.HarnessTest,
	useExternalPayAddr bool) {

	mts := newMppTestScenario(ht)

	// Subscribe to bob's invoices. Do this early in the test to make sure
	// that the subscription has actually been completed when we add an
	// invoice. Otherwise the notification will be missed.
	req := &lnrpc.InvoiceSubscription{}
	bobInvoiceSubscription := mts.bob.RPC.SubscribeInvoices(req)

	const paymentAmt = btcutil.Amount(300000)

	// Set up a network with three different paths Alice <-> Bob. Channel
	// capacities are set such that the payment can only succeed if (at
	// least) three paths are used.
	//
	//              _ Eve _
	//             /       \
	// Alice -- Carol ---- Bob
	//      \              /
	//       \__ Dave ____/
	//
	mppReq := &mppOpenChannelRequest{
		amtAliceCarol: 235000,
		amtAliceDave:  135000,
		amtCarolBob:   135000,
		amtCarolEve:   135000,
		amtDaveBob:    135000,
		amtEveBob:     135000,
	}
	mts.openChannels(mppReq)
	chanPointAliceDave := mts.channelPoints[1]
	chanPointDaveBob := mts.channelPoints[4]

	invoice := &lnrpc.Invoice{
		Value: int64(paymentAmt),
		IsAmp: true,
	}
	addInvoiceResp := mts.bob.RPC.AddInvoice(invoice)

	// Ensure we get a notification of the invoice being added by Bob.
	rpcInvoice := ht.ReceiveInvoiceUpdate(bobInvoiceSubscription)

	require.False(ht, rpcInvoice.Settled) // nolint:staticcheck
	require.Equal(ht, lnrpc.Invoice_OPEN, rpcInvoice.State)
	require.Equal(ht, int64(0), rpcInvoice.AmtPaidSat)
	require.Equal(ht, int64(0), rpcInvoice.AmtPaidMsat)
	require.Equal(ht, 0, len(rpcInvoice.Htlcs))

	// Increase Dave's fee to make the test deterministic. Otherwise it
	// would be unpredictable whether pathfinding would go through Charlie
	// or Dave for the first shard.
	expectedPolicy := mts.updateDaveGlobalPolicy()

	// Make sure Alice has heard it for both Dave's channels.
	ht.AssertChannelPolicyUpdate(
		mts.alice, mts.dave, expectedPolicy, chanPointAliceDave, false,
	)
	ht.AssertChannelPolicyUpdate(
		mts.alice, mts.dave, expectedPolicy, chanPointDaveBob, false,
	)

	// Generate an external payment address when attempting to pseudo-reuse
	// an AMP invoice. When using an external payment address, we'll also
	// expect an extra invoice to appear in the ListInvoices response, since
	// a new invoice will be JIT inserted under a different payment address
	// than the one in the invoice.
	var (
		expNumInvoices  = 1
		externalPayAddr []byte
	)
	if useExternalPayAddr {
		expNumInvoices = 2
		externalPayAddr = ht.Random32Bytes()
	}

	sendReq := &routerrpc.SendPaymentRequest{
		PaymentRequest: addInvoiceResp.PaymentRequest,
		PaymentAddr:    externalPayAddr,
		TimeoutSeconds: 60,
		FeeLimitMsat:   noFeeLimitMsat,
	}
	payment := ht.SendPaymentAssertSettled(mts.alice, sendReq)

	// Check that Alice split the payment in at least three shards. Because
	// the hand-off of the htlc to the link is asynchronous (via a mailbox),
	// there is some non-determinism in the process. Depending on whether
	// the new pathfinding round is started before or after the htlc is
	// locked into the channel, different sharding may occur. Therefore we
	// can only check if the number of shards isn't below the theoretical
	// minimum.
	succeeded := 0
	for _, htlc := range payment.Htlcs {
		if htlc.Status == lnrpc.HTLCAttempt_SUCCEEDED {
			succeeded++
		}
	}

	const minExpectedShards = 3
	require.GreaterOrEqual(ht, succeeded, minExpectedShards,
		"expected num of shards not reached")

	// When an external payment address is supplied, we'll get an extra
	// notification for the JIT inserted invoice, since it differs from the
	// original.
	if useExternalPayAddr {
		ht.ReceiveInvoiceUpdate(bobInvoiceSubscription)
	}

	// There should now be a settle event for the invoice.
	rpcInvoice = ht.ReceiveInvoiceUpdate(bobInvoiceSubscription)

	// Also fetch Bob's invoice from ListInvoices and assert it is equal to
	// the one received via the subscription.
	invoices := ht.AssertNumInvoices(mts.bob, expNumInvoices)
	assertInvoiceEqual(ht.T, rpcInvoice, invoices[expNumInvoices-1])

	// Assert that the invoice is settled for the total payment amount and
	// has the correct payment address.
	require.True(ht, rpcInvoice.Settled) // nolint:staticcheck
	require.Equal(ht, lnrpc.Invoice_SETTLED, rpcInvoice.State)
	require.Equal(ht, int64(paymentAmt), rpcInvoice.AmtPaidSat)
	require.Equal(ht, int64(paymentAmt*1000), rpcInvoice.AmtPaidMsat)

	// Finally, assert that the same set id is recorded for each htlc, and
	// that the preimage hash pair is valid.
	var setID []byte
	require.Equal(ht, succeeded, len(rpcInvoice.Htlcs))
	for _, htlc := range rpcInvoice.Htlcs {
		require.NotNil(ht, htlc.Amp)
		if setID == nil {
			setID = make([]byte, 32)
			copy(setID, htlc.Amp.SetId)
		}
		require.Equal(ht, setID, htlc.Amp.SetId)

		// Parse the child hash and child preimage, and assert they are
		// well-formed.
		childHash, err := lntypes.MakeHash(htlc.Amp.Hash)
		require.NoError(ht, err)
		childPreimage, err := lntypes.MakePreimage(htlc.Amp.Preimage)
		require.NoError(ht, err)

		// Assert that the preimage actually matches the hashes.
		validPreimage := childPreimage.Matches(childHash)
		require.True(ht, validPreimage)
	}

	// The set ID we extract above should be shown in the final settled
	// state.
	ampState := rpcInvoice.AmpInvoiceState[hex.EncodeToString(setID)]
	require.Equal(ht, lnrpc.InvoiceHTLCState_SETTLED, ampState.State)

	// Finally, close all channels.
	mts.closeChannels()
}

// testSendPaymentAMPInvoiceRepeat tests that it's possible to pay an AMP
// invoice multiple times by having the client generate a new setID each time.
func testSendPaymentAMPInvoiceRepeat(ht *lntemp.HarnessTest) {
	// In this basic test, we'll only need two nodes as we want to
	// primarily test the recurring payment feature. So we'll re-use the
	carol := ht.NewNode("Carol", nil)

	// Send Carol enough coins to be able to open a channel to Dave.
	ht.FundCoins(btcutil.SatoshiPerBitcoin, carol)

	dave := ht.NewNode("Dave", nil)

	// Set up an invoice subscription so we can be notified when Dave
	// receives his repeated payments.
	req := &lnrpc.InvoiceSubscription{}
	invSubscription := dave.RPC.SubscribeInvoices(req)

	// Before we start the test, we'll ensure both sides are connected to
	// the funding flow can properly be executed.
	ht.EnsureConnected(carol, dave)

	// Establish a channel between Carol and Dave.
	chanAmt := btcutil.Amount(100_000)
	ht.OpenChannel(
		carol, dave, lntemp.OpenChannelParams{Amt: chanAmt},
	)

	// Create an AMP invoice of a trivial amount, that we'll pay repeatedly
	// in this integration test.
	paymentAmt := 10000
	invoice := &lnrpc.Invoice{
		Value: int64(paymentAmt),
		IsAmp: true,
	}
	addInvoiceResp := dave.RPC.AddInvoice(invoice)

	// We should get an initial notification that the HTLC has been added.
	rpcInvoice := ht.ReceiveInvoiceUpdate(invSubscription)
	require.False(ht, rpcInvoice.Settled) // nolint:staticcheck
	require.Equal(ht, lnrpc.Invoice_OPEN, rpcInvoice.State)
	require.Equal(ht, int64(0), rpcInvoice.AmtPaidSat)
	require.Equal(ht, int64(0), rpcInvoice.AmtPaidMsat)
	require.Equal(ht, 0, len(rpcInvoice.Htlcs))

	// Now we'll use Carol to pay the invoice that Dave created.
	ht.CompletePaymentRequests(
		carol, []string{addInvoiceResp.PaymentRequest},
	)

	// Dave should get a notification that the invoice has been settled.
	invoiceNtfn := ht.ReceiveInvoiceUpdate(invSubscription)

	// The notification should signal that the invoice is now settled, and
	// should also include the set ID, and show the proper amount paid.
	require.True(ht, invoiceNtfn.Settled) // nolint:staticcheck
	require.Equal(ht, lnrpc.Invoice_SETTLED, invoiceNtfn.State)
	require.Equal(ht, paymentAmt, int(invoiceNtfn.AmtPaidSat))
	require.Equal(ht, 1, len(invoiceNtfn.AmpInvoiceState))
	var firstSetID []byte
	for setIDStr, ampState := range invoiceNtfn.AmpInvoiceState {
		firstSetID, _ = hex.DecodeString(setIDStr)
		require.Equal(ht, lnrpc.InvoiceHTLCState_SETTLED,
			ampState.State)
	}

	// Pay the invoice again, we should get another notification that Dave
	// has received another payment.
	ht.CompletePaymentRequests(
		carol, []string{addInvoiceResp.PaymentRequest},
	)

	// Dave should get another notification.
	invoiceNtfn = ht.ReceiveInvoiceUpdate(invSubscription)

	// The invoice should still be shown as settled, and also include the
	// information about this newly generated setID, showing 2x the amount
	// paid.
	require.True(ht, invoiceNtfn.Settled) // nolint:staticcheck
	require.Equal(ht, paymentAmt*2, int(invoiceNtfn.AmtPaidSat))

	var secondSetID []byte
	for setIDStr, ampState := range invoiceNtfn.AmpInvoiceState {
		secondSetID, _ = hex.DecodeString(setIDStr)
		require.Equal(ht, lnrpc.InvoiceHTLCState_SETTLED,
			ampState.State)
	}

	// The returned invoice should only include a single HTLC since we
	// return the "projected" sub-invoice for a given setID.
	require.Equal(ht, 1, len(invoiceNtfn.Htlcs))

	// However the AMP state index should show that there've been two
	// repeated payments to this invoice so far.
	require.Equal(ht, 2, len(invoiceNtfn.AmpInvoiceState))

	// Now we'll look up the invoice using the new LookupInvoice2 RPC call
	// by the set ID of each of the invoices.
	msg := &invoicesrpc.LookupInvoiceMsg{
		InvoiceRef: &invoicesrpc.LookupInvoiceMsg_SetId{
			SetId: firstSetID,
		},
		LookupModifier: invoicesrpc.LookupModifier_HTLC_SET_ONLY,
	}
	subInvoice1 := dave.RPC.LookupInvoiceV2(msg)
	msg = &invoicesrpc.LookupInvoiceMsg{
		InvoiceRef: &invoicesrpc.LookupInvoiceMsg_SetId{
			SetId: secondSetID,
		},
		LookupModifier: invoicesrpc.LookupModifier_HTLC_SET_ONLY,
	}
	subInvoice2 := dave.RPC.LookupInvoiceV2(msg)

	// Each invoice should only show a single HTLC present, as we passed
	// the HTLC set only modifier.
	require.Equal(ht, 1, len(subInvoice1.Htlcs))
	require.Equal(ht, 1, len(subInvoice2.Htlcs))

	// If we look up the same invoice, by its payment address, but now with
	// the HTLC blank modifier, then none of them should be returned.
	msg = &invoicesrpc.LookupInvoiceMsg{
		InvoiceRef: &invoicesrpc.LookupInvoiceMsg_PaymentAddr{
			PaymentAddr: addInvoiceResp.PaymentAddr,
		},
		LookupModifier: invoicesrpc.LookupModifier_HTLC_SET_BLANK,
	}
	rootInvoice := dave.RPC.LookupInvoiceV2(msg)
	require.Equal(ht, 0, len(rootInvoice.Htlcs))

	// If we look up the same invoice, by its payment address, but without
	// that modified, then we should get all the relevant HTLCs.
	msg = &invoicesrpc.LookupInvoiceMsg{
		InvoiceRef: &invoicesrpc.LookupInvoiceMsg_PaymentAddr{
			PaymentAddr: addInvoiceResp.PaymentAddr,
		},
	}
	rootInvoice = dave.RPC.LookupInvoiceV2(msg)
	require.Equal(ht, 2, len(rootInvoice.Htlcs))

	// Finally, we'll test that if we subscribe for notifications of
	// settled invoices, we get a backlog, which includes the invoice we
	// settled last (since you can only fetch from index 1 onwards), and
	// only the relevant set of HTLCs.
	req = &lnrpc.InvoiceSubscription{
		SettleIndex: 1,
	}
	invSub2 := dave.RPC.SubscribeInvoices(req)

	// The first invoice we get back should match the state of the invoice
	// after our second payment: amt updated, but only a single HTLC shown
	// through.
	backlogInv := ht.ReceiveInvoiceUpdate(invSub2)
	require.Equal(ht, 1, len(backlogInv.Htlcs))
	require.Equal(ht, 2, len(backlogInv.AmpInvoiceState))
	require.True(ht, backlogInv.Settled) // nolint:staticcheck
	require.Equal(ht, paymentAmt*2, int(backlogInv.AmtPaidSat))
}

// testSendPaymentAMP tests that we can send an AMP payment to a specified
// destination using SendPaymentV2.
func testSendPaymentAMP(ht *lntemp.HarnessTest) {
	mts := newMppTestScenario(ht)
	const paymentAmt = btcutil.Amount(300000)

	// Set up a network with three different paths Alice <-> Bob. Channel
	// capacities are set such that the payment can only succeed if (at
	// least) three paths are used.
	//
	//              _ Eve _
	//             /       \
	// Alice -- Carol ---- Bob
	//      \              /
	//       \__ Dave ____/
	//
	mppReq := &mppOpenChannelRequest{
		amtAliceCarol: 235000,
		amtAliceDave:  135000,
		amtCarolBob:   135000,
		amtCarolEve:   135000,
		amtDaveBob:    135000,
		amtEveBob:     135000,
	}
	mts.openChannels(mppReq)
	chanPointAliceDave := mts.channelPoints[1]

	// Increase Dave's fee to make the test deterministic. Otherwise it
	// would be unpredictable whether pathfinding would go through Charlie
	// or Dave for the first shard.
	expectedPolicy := mts.updateDaveGlobalPolicy()

	// Make sure Alice has heard it.
	ht.AssertChannelPolicyUpdate(
		mts.alice, mts.dave, expectedPolicy, chanPointAliceDave, false,
	)

	sendReq := &routerrpc.SendPaymentRequest{
		Dest:           mts.bob.PubKey[:],
		Amt:            int64(paymentAmt),
		FinalCltvDelta: chainreg.DefaultBitcoinTimeLockDelta,
		TimeoutSeconds: 60,
		FeeLimitMsat:   noFeeLimitMsat,
		Amp:            true,
	}
	payment := ht.SendPaymentAssertSettled(mts.alice, sendReq)

	// Check that Alice split the payment in at least three shards. Because
	// the hand-off of the htlc to the link is asynchronous (via a mailbox),
	// there is some non-determinism in the process. Depending on whether
	// the new pathfinding round is started before or after the htlc is
	// locked into the channel, different sharding may occur. Therefore we
	// can only check if the number of shards isn't below the theoretical
	// minimum.
	succeeded := 0
	for _, htlc := range payment.Htlcs {
		if htlc.Status == lnrpc.HTLCAttempt_SUCCEEDED {
			succeeded++
		}
	}

	const minExpectedShards = 3
	require.GreaterOrEqual(ht, succeeded, minExpectedShards,
		"expected num of shards not reached")

	// Fetch Bob's invoices. There should only be one invoice.
	invoices := ht.AssertNumInvoices(mts.bob, 1)
	rpcInvoice := invoices[0]

	// Assert that the invoice is settled for the total payment amount and
	// has the correct payment address.
	require.True(ht, rpcInvoice.Settled) // nolint:staticcheck
	require.Equal(ht, lnrpc.Invoice_SETTLED, rpcInvoice.State)
	require.Equal(ht, int64(paymentAmt), rpcInvoice.AmtPaidSat)
	require.Equal(ht, int64(paymentAmt*1000), rpcInvoice.AmtPaidMsat)

	// Finally, assert that the same set id is recorded for each htlc, and
	// that the preimage hash pair is valid.
	var setID []byte
	require.Equal(ht, succeeded, len(rpcInvoice.Htlcs))
	for _, htlc := range rpcInvoice.Htlcs {
		require.NotNil(ht, htlc.Amp)
		if setID == nil {
			setID = make([]byte, 32)
			copy(setID, htlc.Amp.SetId)
		}
		require.Equal(ht, setID, htlc.Amp.SetId)

		// Parse the child hash and child preimage, and assert they are
		// well-formed.
		childHash, err := lntypes.MakeHash(htlc.Amp.Hash)
		require.NoError(ht, err)
		childPreimage, err := lntypes.MakePreimage(htlc.Amp.Preimage)
		require.NoError(ht, err)

		// Assert that the preimage actually matches the hashes.
		validPreimage := childPreimage.Matches(childHash)
		require.True(ht, validPreimage)
	}

	// The set ID we extract above should be shown in the final settled
	// state.
	ampState := rpcInvoice.AmpInvoiceState[hex.EncodeToString(setID)]
	require.Equal(ht, lnrpc.InvoiceHTLCState_SETTLED, ampState.State)

	// Finally, close all channels.
	mts.closeChannels()
}

func testSendToRouteAMP(net *lntest.NetworkHarness, t *harnessTest) {
	ctxb := context.Background()

	ctx := newMppTestContext(t, net)
	defer ctx.shutdownNodes()

	const (
		paymentAmt = btcutil.Amount(300000)
		numShards  = 3
		shardAmt   = paymentAmt / numShards
		chanAmt    = shardAmt * 3 / 2
	)

	// Set up a network with three different paths Alice <-> Bob.
	//              _ Eve _
	//             /       \
	// Alice -- Carol ---- Bob
	//      \              /
	//       \__ Dave ____/
	//
	ctx.openChannel(ctx.carol, ctx.bob, chanAmt)
	ctx.openChannel(ctx.dave, ctx.bob, chanAmt)
	ctx.openChannel(ctx.alice, ctx.dave, chanAmt)
	ctx.openChannel(ctx.eve, ctx.bob, chanAmt)
	ctx.openChannel(ctx.carol, ctx.eve, chanAmt)

	// Since the channel Alice-> Carol will have to carry two
	// shards, we make it larger.
	ctx.openChannel(ctx.alice, ctx.carol, chanAmt+shardAmt)

	defer ctx.closeChannels()

	ctx.waitForChannels()

	// Subscribe to bob's invoices.
	req := &lnrpc.InvoiceSubscription{}
	ctxc, cancelSubscription := context.WithCancel(ctxb)
	bobInvoiceSubscription, err := ctx.bob.SubscribeInvoices(ctxc, req)
	require.NoError(t.t, err)
	defer cancelSubscription()

	// We'll send shards along three routes from Alice.
	sendRoutes := [numShards][]*lntest.HarnessNode{
		{ctx.carol, ctx.bob},
		{ctx.dave, ctx.bob},
		{ctx.carol, ctx.eve, ctx.bob},
	}

	payAddr := make([]byte, 32)
	_, err = rand.Read(payAddr)
	require.NoError(t.t, err)

	setID := make([]byte, 32)
	_, err = rand.Read(setID)
	require.NoError(t.t, err)

	var sharer amp.Sharer
	sharer, err = amp.NewSeedSharer()
	require.NoError(t.t, err)

	childPreimages := make(map[lntypes.Preimage]uint32)
	responses := make(chan *lnrpc.HTLCAttempt, len(sendRoutes))

	// Define a closure for sending each of the three shards.
	sendShard := func(i int, hops []*lntest.HarnessNode) {
		// Build a route for the specified hops.
		r, err := ctx.buildRoute(ctxb, shardAmt, ctx.alice, hops)
		if err != nil {
			t.Fatalf("unable to build route: %v", err)
		}

		// Set the MPP records to indicate this is a payment shard.
		hop := r.Hops[len(r.Hops)-1]
		hop.TlvPayload = true
		hop.MppRecord = &lnrpc.MPPRecord{
			PaymentAddr:  payAddr,
			TotalAmtMsat: int64(paymentAmt * 1000),
		}

		var child *amp.Child
		if i < len(sendRoutes)-1 {
			var left amp.Sharer
			left, sharer, err = sharer.Split()
			require.NoError(t.t, err)

			child = left.Child(uint32(i))
		} else {
			child = sharer.Child(uint32(i))
		}
		childPreimages[child.Preimage] = child.Index

		hop.AmpRecord = &lnrpc.AMPRecord{
			RootShare:  child.Share[:],
			SetId:      setID,
			ChildIndex: child.Index,
		}

		// Send the shard.
		sendReq := &routerrpc.SendToRouteRequest{
			PaymentHash: child.Hash[:],
			Route:       r,
		}

		// We'll send all shards in their own goroutine, since SendToRoute will
		// block as long as the payment is in flight.
		go func() {
			ctxt, _ := context.WithTimeout(ctxb, defaultTimeout)
			resp, err := ctx.alice.RouterClient.SendToRouteV2(ctxt, sendReq)
			if err != nil {
				t.Fatalf("unable to send payment: %v", err)
			}

			responses <- resp
		}()
	}

	// Send the first shard, this cause Bob to JIT add an invoice.
	sendShard(0, sendRoutes[0])

	// Ensure we get a notification of the invoice being added by Bob.
	rpcInvoice, err := bobInvoiceSubscription.Recv()
	require.NoError(t.t, err)

	require.False(t.t, rpcInvoice.Settled) // nolint:staticcheck
	require.Equal(t.t, lnrpc.Invoice_OPEN, rpcInvoice.State)
	require.Equal(t.t, int64(0), rpcInvoice.AmtPaidSat)
	require.Equal(t.t, int64(0), rpcInvoice.AmtPaidMsat)
	require.Equal(t.t, payAddr, rpcInvoice.PaymentAddr)

	require.Equal(t.t, 0, len(rpcInvoice.Htlcs))

	sendShard(1, sendRoutes[1])
	sendShard(2, sendRoutes[2])

	// Assert that all of the child preimages are unique.
	require.Equal(t.t, len(sendRoutes), len(childPreimages))

	// Make a copy of the childPreimages map for validating the resulting
	// invoice.
	childPreimagesCopy := make(map[lntypes.Preimage]uint32)
	for preimage, childIndex := range childPreimages {
		childPreimagesCopy[preimage] = childIndex
	}

	// Wait for all responses to be back, and check that they all
	// succeeded.
	for range sendRoutes {
		var resp *lnrpc.HTLCAttempt
		select {
		case resp = <-responses:
		case <-time.After(defaultTimeout):
			t.Fatalf("response not received")
		}

		if resp.Failure != nil {
			t.Fatalf("received payment failure : %v", resp.Failure)
		}

		preimage, err := lntypes.MakePreimage(resp.Preimage)
		require.NoError(t.t, err)

		// Assert that the response includes one of our child preimages.
		_, ok := childPreimages[preimage]
		require.True(t.t, ok)

		// Remove this preimage from out set so that we ensure all
		// responses have a unique child preimage.
		delete(childPreimages, preimage)
	}
	childPreimages = childPreimagesCopy

	// There should now be a settle event for the invoice.
	rpcInvoice, err = bobInvoiceSubscription.Recv()
	require.NoError(t.t, err)

	// Also fetch Bob's invoice from ListInvoices and assert it is equal to
	// the one received via the subscription.
	invoiceResp, err := ctx.bob.ListInvoices(
		ctxb, &lnrpc.ListInvoiceRequest{},
	)
	require.NoError(t.t, err)
	require.Equal(t.t, 1, len(invoiceResp.Invoices))
	assertInvoiceEqual(t.t, rpcInvoice, invoiceResp.Invoices[0])

	// Assert that the invoice is settled for the total payment amount and
	// has the correct payment address.
	require.True(t.t, rpcInvoice.Settled) // nolint:staticcheck
	require.Equal(t.t, lnrpc.Invoice_SETTLED, rpcInvoice.State)
	require.Equal(t.t, int64(paymentAmt), rpcInvoice.AmtPaidSat)
	require.Equal(t.t, int64(paymentAmt*1000), rpcInvoice.AmtPaidMsat)
	require.Equal(t.t, payAddr, rpcInvoice.PaymentAddr)

	// Finally, assert that the proper set id is recorded for each htlc, and
	// that the preimage hash pair is valid.
	require.Equal(t.t, numShards, len(rpcInvoice.Htlcs))
	for _, htlc := range rpcInvoice.Htlcs {
		require.NotNil(t.t, htlc.Amp)
		require.Equal(t.t, setID, htlc.Amp.SetId)

		// Parse the child hash and child preimage, and assert they are
		// well-formed.
		childHash, err := lntypes.MakeHash(htlc.Amp.Hash)
		require.NoError(t.t, err)
		childPreimage, err := lntypes.MakePreimage(htlc.Amp.Preimage)
		require.NoError(t.t, err)

		// Assert that the preimage actually matches the hashes.
		validPreimage := childPreimage.Matches(childHash)
		require.True(t.t, validPreimage)

		// Assert that the HTLC includes one of our child preimages.
		childIndex, ok := childPreimages[childPreimage]
		require.True(t.t, ok)

		// Assert that the correct child index is reflected.
		require.Equal(t.t, childIndex, htlc.Amp.ChildIndex)

		// Remove this preimage from our set so that we ensure all HTLCs
		// have a unique child preimage.
		delete(childPreimages, childPreimage)
	}
}

// assertInvoiceEqual asserts that two lnrpc.Invoices are equivalent. A custom
// comparison function is defined for these tests, since proto message returned
// from unary and streaming RPCs (as of protobuf 1.23.0 and grpc 1.29.1) aren't
// consistent with the private fields set on the messages. As a result, we avoid
// using require.Equal and test only the actual data members.
func assertInvoiceEqual(t *testing.T, a, b *lnrpc.Invoice) {
	t.Helper()

	// Ensure the HTLCs are sorted properly before attempting to compare.
	sort.Slice(a.Htlcs, func(i, j int) bool {
		return a.Htlcs[i].ChanId < a.Htlcs[j].ChanId
	})
	sort.Slice(b.Htlcs, func(i, j int) bool {
		return b.Htlcs[i].ChanId < b.Htlcs[j].ChanId
	})

	require.Equal(t, a.Memo, b.Memo)
	require.Equal(t, a.RPreimage, b.RPreimage)
	require.Equal(t, a.RHash, b.RHash)
	require.Equal(t, a.Value, b.Value)
	require.Equal(t, a.ValueMsat, b.ValueMsat)
	require.Equal(t, a.CreationDate, b.CreationDate)
	require.Equal(t, a.SettleDate, b.SettleDate)
	require.Equal(t, a.PaymentRequest, b.PaymentRequest)
	require.Equal(t, a.DescriptionHash, b.DescriptionHash)
	require.Equal(t, a.Expiry, b.Expiry)
	require.Equal(t, a.FallbackAddr, b.FallbackAddr)
	require.Equal(t, a.CltvExpiry, b.CltvExpiry)
	require.Equal(t, a.RouteHints, b.RouteHints)
	require.Equal(t, a.Private, b.Private)
	require.Equal(t, a.AddIndex, b.AddIndex)
	require.Equal(t, a.SettleIndex, b.SettleIndex)
	require.Equal(t, a.AmtPaidSat, b.AmtPaidSat)
	require.Equal(t, a.AmtPaidMsat, b.AmtPaidMsat)
	require.Equal(t, a.State, b.State)
	require.Equal(t, a.Features, b.Features)
	require.Equal(t, a.IsKeysend, b.IsKeysend)
	require.Equal(t, a.PaymentAddr, b.PaymentAddr)
	require.Equal(t, a.IsAmp, b.IsAmp)

	require.Equal(t, len(a.Htlcs), len(b.Htlcs))
	for i := range a.Htlcs {
		htlcA, htlcB := a.Htlcs[i], b.Htlcs[i]
		require.Equal(t, htlcA.ChanId, htlcB.ChanId)
		require.Equal(t, htlcA.HtlcIndex, htlcB.HtlcIndex)
		require.Equal(t, htlcA.AmtMsat, htlcB.AmtMsat)
		require.Equal(t, htlcA.AcceptHeight, htlcB.AcceptHeight)
		require.Equal(t, htlcA.AcceptTime, htlcB.AcceptTime)
		require.Equal(t, htlcA.ResolveTime, htlcB.ResolveTime)
		require.Equal(t, htlcA.ExpiryHeight, htlcB.ExpiryHeight)
		require.Equal(t, htlcA.State, htlcB.State)
		require.Equal(t, htlcA.CustomRecords, htlcB.CustomRecords)
		require.Equal(t, htlcA.MppTotalAmtMsat, htlcB.MppTotalAmtMsat)
		require.Equal(t, htlcA.Amp, htlcB.Amp)
	}
}
