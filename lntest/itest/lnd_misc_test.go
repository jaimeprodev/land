package itest

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"strings"
	"time"

	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcwallet/wallet"
	"github.com/lightningnetwork/lnd/chainreg"
	"github.com/lightningnetwork/lnd/funding"
	"github.com/lightningnetwork/lnd/input"
	"github.com/lightningnetwork/lnd/lncfg"
	"github.com/lightningnetwork/lnd/lnrpc"
	"github.com/lightningnetwork/lnd/lnrpc/invoicesrpc"
	"github.com/lightningnetwork/lnd/lnrpc/routerrpc"
	"github.com/lightningnetwork/lnd/lnrpc/walletrpc"
	"github.com/lightningnetwork/lnd/lntemp"
	"github.com/lightningnetwork/lnd/lntest"
	"github.com/lightningnetwork/lnd/lntest/wait"
	"github.com/lightningnetwork/lnd/lnwallet"
	"github.com/lightningnetwork/lnd/lnwire"
	"github.com/stretchr/testify/require"
)

// testDisconnectingTargetPeer performs a test which disconnects Alice-peer
// from Bob-peer and then re-connects them again. We expect Alice to be able to
// disconnect at any point.
//
// TODO(yy): move to lnd_network_test.
func testDisconnectingTargetPeer(ht *lntemp.HarnessTest) {
	// We'll start both nodes with a high backoff so that they don't
	// reconnect automatically during our test.
	args := []string{
		"--minbackoff=1m",
		"--maxbackoff=1m",
	}

	alice, bob := ht.Alice, ht.Bob
	ht.RestartNodeWithExtraArgs(alice, args)
	ht.RestartNodeWithExtraArgs(bob, args)

	// Start by connecting Alice and Bob with no channels.
	ht.EnsureConnected(alice, bob)

	chanAmt := funding.MaxBtcFundingAmount
	pushAmt := btcutil.Amount(0)

	// Create a new channel that requires 1 confs before it's considered
	// open, then broadcast the funding transaction
	const numConfs = 1
	p := lntemp.OpenChannelParams{
		Amt:     chanAmt,
		PushAmt: pushAmt,
	}
	stream := ht.OpenChannelAssertPending(alice, bob, p)

	// At this point, the channel's funding transaction will have been
	// broadcast, but not confirmed. Alice and Bob's nodes should reflect
	// this when queried via RPC.
	ht.AssertNumPendingOpenChannels(alice, 1)
	ht.AssertNumPendingOpenChannels(bob, 1)

	// Disconnect Alice-peer from Bob-peer should have no error.
	ht.DisconnectNodes(alice, bob)

	// Assert that the connection was torn down.
	ht.AssertNotConnected(alice, bob)

	// Mine a block, then wait for Alice's node to notify us that the
	// channel has been opened.
	ht.MineBlocksAndAssertNumTxes(numConfs, 1)

	// At this point, the channel should be fully opened and there should
	// be no pending channels remaining for either node.
	ht.AssertNumPendingOpenChannels(alice, 0)
	ht.AssertNumPendingOpenChannels(bob, 0)

	// Reconnect the nodes so that the channel can become active.
	ht.ConnectNodes(alice, bob)

	// The channel should be listed in the peer information returned by
	// both peers.
	chanPoint := ht.WaitForChannelOpenEvent(stream)

	// Check both nodes to ensure that the channel is ready for operation.
	ht.AssertChannelExists(alice, chanPoint)
	ht.AssertChannelExists(bob, chanPoint)

	// Disconnect Alice-peer from Bob-peer should have no error.
	ht.DisconnectNodes(alice, bob)

	// Check existing connection.
	ht.AssertNotConnected(alice, bob)

	// Reconnect both nodes before force closing the channel.
	ht.ConnectNodes(alice, bob)

	// Finally, immediately close the channel. This function will also
	// block until the channel is closed and will additionally assert the
	// relevant channel closing post conditions.
	ht.ForceCloseChannel(alice, chanPoint)

	// Disconnect Alice-peer from Bob-peer should have no error.
	ht.DisconnectNodes(alice, bob)

	// Check that the nodes not connected.
	ht.AssertNotConnected(alice, bob)

	// Finally, re-connect both nodes.
	ht.ConnectNodes(alice, bob)

	// Check existing connection.
	ht.AssertConnected(alice, bob)
}

// testSphinxReplayPersistence verifies that replayed onion packets are
// rejected by a remote peer after a restart. We use a combination of unsafe
// configuration arguments to force Carol to replay the same sphinx packet
// after reconnecting to Dave, and compare the returned failure message with
// what we expect for replayed onion packets.
func testSphinxReplayPersistence(ht *lntemp.HarnessTest) {
	// Open a channel with 100k satoshis between Carol and Dave with Carol
	// being the sole funder of the channel.
	chanAmt := btcutil.Amount(100000)

	// First, we'll create Dave, the receiver, and start him in hodl mode.
	dave := ht.NewNode("Dave", []string{"--hodl.exit-settle"})

	// Next, we'll create Carol and establish a channel to from her to
	// Dave. Carol is started in both unsafe-replay which will cause her to
	// replay any pending Adds held in memory upon reconnection.
	carol := ht.NewNode("Carol", []string{"--unsafe-replay"})
	ht.FundCoins(btcutil.SatoshiPerBitcoin, carol)

	ht.ConnectNodes(carol, dave)
	chanPoint := ht.OpenChannel(
		carol, dave, lntemp.OpenChannelParams{
			Amt: chanAmt,
		},
	)

	// Next, we'll create Fred who is going to initiate the payment and
	// establish a channel to from him to Carol. We can't perform this test
	// by paying from Carol directly to Dave, because the '--unsafe-replay'
	// setup doesn't apply to locally added htlcs. In that case, the
	// mailbox, that is responsible for generating the replay, is bypassed.
	fred := ht.NewNode("Fred", nil)
	ht.FundCoins(btcutil.SatoshiPerBitcoin, fred)

	ht.ConnectNodes(fred, carol)
	chanPointFC := ht.OpenChannel(
		fred, carol, lntemp.OpenChannelParams{
			Amt: chanAmt,
		},
	)
	defer ht.CloseChannel(fred, chanPointFC)

	// Now that the channel is open, create an invoice for Dave which
	// expects a payment of 1000 satoshis from Carol paid via a particular
	// preimage.
	const paymentAmt = 1000
	preimage := ht.Random32Bytes()
	invoice := &lnrpc.Invoice{
		Memo:      "testing",
		RPreimage: preimage,
		Value:     paymentAmt,
	}
	invoiceResp := dave.RPC.AddInvoice(invoice)

	// Wait for all channels to be recognized and advertized.
	ht.AssertTopologyChannelOpen(carol, chanPoint)
	ht.AssertTopologyChannelOpen(dave, chanPoint)
	ht.AssertTopologyChannelOpen(carol, chanPointFC)
	ht.AssertTopologyChannelOpen(fred, chanPointFC)

	// With the invoice for Dave added, send a payment from Fred paying
	// to the above generated invoice.
	req := &routerrpc.SendPaymentRequest{
		PaymentRequest: invoiceResp.PaymentRequest,
		TimeoutSeconds: 60,
		FeeLimitMsat:   noFeeLimitMsat,
	}
	payStream := fred.RPC.SendPayment(req)

	// Dave's invoice should not be marked as settled.
	msg := &invoicesrpc.LookupInvoiceMsg{
		InvoiceRef: &invoicesrpc.LookupInvoiceMsg_PaymentAddr{
			PaymentAddr: invoiceResp.PaymentAddr,
		},
	}
	dbInvoice := dave.RPC.LookupInvoiceV2(msg)
	require.NotEqual(ht, lnrpc.InvoiceHTLCState_SETTLED, dbInvoice.State,
		"dave's invoice should not be marked as settled")

	// With the payment sent but hedl, all balance related stats should not
	// have changed.
	ht.AssertAmountPaid("carol => dave", carol, chanPoint, 0, 0)
	ht.AssertAmountPaid("dave <= carol", dave, chanPoint, 0, 0)

	// Before we restart Dave, make sure both Carol and Dave have added the
	// HTLC.
	ht.AssertNumActiveHtlcs(carol, 2)
	ht.AssertNumActiveHtlcs(dave, 1)

	// With the first payment sent, restart dave to make sure he is
	// persisting the information required to detect replayed sphinx
	// packets.
	ht.RestartNode(dave)

	// Carol should retransmit the Add hedl in her mailbox on startup. Dave
	// should not accept the replayed Add, and actually fail back the
	// pending payment. Even though he still holds the original settle, if
	// he does fail, it is almost certainly caused by the sphinx replay
	// protection, as it is the only validation we do in hodl mode.
	//
	// Assert that Fred receives the expected failure after Carol sent a
	// duplicate packet that fails due to sphinx replay detection.
	ht.AssertPaymentStatusFromStream(payStream, lnrpc.Payment_FAILED)
	ht.AssertLastHTLCError(fred, lnrpc.Failure_INVALID_ONION_KEY)

	// Since the payment failed, the balance should still be left
	// unaltered.
	ht.AssertAmountPaid("carol => dave", carol, chanPoint, 0, 0)
	ht.AssertAmountPaid("dave <= carol", dave, chanPoint, 0, 0)

	// Cleanup by mining the force close and sweep transaction.
	ht.ForceCloseChannel(carol, chanPoint)
}

// testListChannels checks that the response from ListChannels is correct. It
// tests the values in all ChannelConstraints are returned as expected. Once
// ListChannels becomes mature, a test against all fields in ListChannels
// should be performed.
func testListChannels(ht *lntemp.HarnessTest) {
	const aliceRemoteMaxHtlcs = 50
	const bobRemoteMaxHtlcs = 100

	// Get the standby nodes and open a channel between them.
	alice, bob := ht.Alice, ht.Bob

	args := []string{fmt.Sprintf(
		"--default-remote-max-htlcs=%v",
		bobRemoteMaxHtlcs,
	)}
	ht.RestartNodeWithExtraArgs(bob, args)

	// Connect Alice to Bob.
	ht.EnsureConnected(alice, bob)

	// Open a channel with 100k satoshis between Alice and Bob with Alice
	// being the sole funder of the channel. The minial HTLC amount is set
	// to 4200 msats.
	const customizedMinHtlc = 4200

	chanAmt := btcutil.Amount(100000)
	pushAmt := btcutil.Amount(1000)
	p := lntemp.OpenChannelParams{
		Amt:            chanAmt,
		PushAmt:        pushAmt,
		MinHtlc:        customizedMinHtlc,
		RemoteMaxHtlcs: aliceRemoteMaxHtlcs,
	}
	chanPoint := ht.OpenChannel(alice, bob, p)
	defer ht.CloseChannel(alice, chanPoint)

	// Alice should have one channel opened with Bob.
	ht.AssertNodeNumChannels(alice, 1)
	// Bob should have one channel opened with Alice.
	ht.AssertNodeNumChannels(bob, 1)

	// Check the returned response is correct.
	aliceChannel := ht.QueryChannelByChanPoint(alice, chanPoint)

	// Since Alice is the initiator, she pays the commit fee.
	aliceBalance := int64(chanAmt) - aliceChannel.CommitFee - int64(pushAmt)

	// Check the balance related fields are correct.
	require.Equal(ht, aliceBalance, aliceChannel.LocalBalance)
	require.EqualValues(ht, pushAmt, aliceChannel.RemoteBalance)
	require.EqualValues(ht, pushAmt, aliceChannel.PushAmountSat)

	// Calculate the dust limit we'll use for the test.
	dustLimit := lnwallet.DustLimitForSize(input.UnknownWitnessSize)

	// defaultConstraints is a ChannelConstraints with default values. It
	// is used to test against Alice's local channel constraints.
	defaultConstraints := &lnrpc.ChannelConstraints{
		CsvDelay:          4,
		ChanReserveSat:    1000,
		DustLimitSat:      uint64(dustLimit),
		MaxPendingAmtMsat: 99000000,
		MinHtlcMsat:       1,
		MaxAcceptedHtlcs:  bobRemoteMaxHtlcs,
	}
	assertChannelConstraintsEqual(
		ht, defaultConstraints, aliceChannel.LocalConstraints,
	)

	// customizedConstraints is a ChannelConstraints with customized
	// values. Ideally, all these values can be passed in when creating the
	// channel. Currently, only the MinHtlcMsat is customized. It is used
	// to check against Alice's remote channel constratins.
	customizedConstraints := &lnrpc.ChannelConstraints{
		CsvDelay:          4,
		ChanReserveSat:    1000,
		DustLimitSat:      uint64(dustLimit),
		MaxPendingAmtMsat: 99000000,
		MinHtlcMsat:       customizedMinHtlc,
		MaxAcceptedHtlcs:  aliceRemoteMaxHtlcs,
	}
	assertChannelConstraintsEqual(
		ht, customizedConstraints, aliceChannel.RemoteConstraints,
	)

	// Get the ListChannel response for Bob.
	bobChannel := ht.QueryChannelByChanPoint(bob, chanPoint)
	require.Equal(ht, aliceChannel.ChannelPoint, bobChannel.ChannelPoint,
		"Bob's channel point mismatched")

	// Check the balance related fields are correct.
	require.Equal(ht, aliceBalance, bobChannel.RemoteBalance)
	require.EqualValues(ht, pushAmt, bobChannel.LocalBalance)
	require.EqualValues(ht, pushAmt, bobChannel.PushAmountSat)

	// Check channel constraints match. Alice's local channel constraint
	// should be equal to Bob's remote channel constraint, and her remote
	// one should be equal to Bob's local one.
	assertChannelConstraintsEqual(
		ht, aliceChannel.LocalConstraints, bobChannel.RemoteConstraints,
	)
	assertChannelConstraintsEqual(
		ht, aliceChannel.RemoteConstraints, bobChannel.LocalConstraints,
	)
}

// testMaxPendingChannels checks that error is returned from remote peer if
// max pending channel number was exceeded and that '--maxpendingchannels' flag
// exists and works properly.
func testMaxPendingChannels(net *lntest.NetworkHarness, t *harnessTest) {
	maxPendingChannels := lncfg.DefaultMaxPendingChannels + 1
	amount := funding.MaxBtcFundingAmount

	// Create a new node (Carol) with greater number of max pending
	// channels.
	args := []string{
		fmt.Sprintf("--maxpendingchannels=%v", maxPendingChannels),
	}
	carol := net.NewNode(t.t, "Carol", args)
	defer shutdownAndAssert(net, t, carol)

	net.ConnectNodes(t.t, net.Alice, carol)

	carolBalance := btcutil.Amount(maxPendingChannels) * amount
	net.SendCoins(t.t, carolBalance, carol)

	// Send open channel requests without generating new blocks thereby
	// increasing pool of pending channels. Then check that we can't open
	// the channel if the number of pending channels exceed max value.
	openStreams := make([]lnrpc.Lightning_OpenChannelClient, maxPendingChannels)
	for i := 0; i < maxPendingChannels; i++ {
		stream := openChannelStream(
			t, net, net.Alice, carol,
			lntest.OpenChannelParams{
				Amt: amount,
			},
		)
		openStreams[i] = stream
	}

	// Carol exhausted available amount of pending channels, next open
	// channel request should cause ErrorGeneric to be sent back to Alice.
	_, err := net.OpenChannel(
		net.Alice, carol, lntest.OpenChannelParams{
			Amt: amount,
		},
	)

	if err == nil {
		t.Fatalf("error wasn't received")
	} else if !strings.Contains(
		err.Error(), lnwire.ErrMaxPendingChannels.Error(),
	) {

		t.Fatalf("not expected error was received: %v", err)
	}

	// For now our channels are in pending state, in order to not interfere
	// with other tests we should clean up - complete opening of the
	// channel and then close it.

	// Mine 6 blocks, then wait for node's to notify us that the channel has
	// been opened. The funding transactions should be found within the
	// first newly mined block. 6 blocks make sure the funding transaction
	// has enough confirmations to be announced publicly.
	block := mineBlocks(t, net, 6, maxPendingChannels)[0]

	chanPoints := make([]*lnrpc.ChannelPoint, maxPendingChannels)
	for i, stream := range openStreams {
		fundingChanPoint, err := net.WaitForChannelOpen(stream)
		if err != nil {
			t.Fatalf("error while waiting for channel open: %v", err)
		}

		fundingTxID, err := lnrpc.GetChanPointFundingTxid(fundingChanPoint)
		if err != nil {
			t.Fatalf("unable to get txid: %v", err)
		}

		// Ensure that the funding transaction enters a block, and is
		// properly advertised by Alice.
		assertTxInBlock(t, block, fundingTxID)
		err = net.Alice.WaitForNetworkChannelOpen(fundingChanPoint)
		if err != nil {
			t.Fatalf("channel not seen on network before "+
				"timeout: %v", err)
		}

		// The channel should be listed in the peer information
		// returned by both peers.
		chanPoint := wire.OutPoint{
			Hash:  *fundingTxID,
			Index: fundingChanPoint.OutputIndex,
		}
		err = net.AssertChannelExists(net.Alice, &chanPoint)
		require.NoError(t.t, err, "unable to assert channel existence")

		chanPoints[i] = fundingChanPoint
	}

	// Next, close the channel between Alice and Carol, asserting that the
	// channel has been properly closed on-chain.
	for _, chanPoint := range chanPoints {
		closeChannelAndAssert(t, net, net.Alice, chanPoint, false)
	}
}

// testGarbageCollectLinkNodes tests that we properly garbage collect link nodes
// from the database and the set of persistent connections within the server.
func testGarbageCollectLinkNodes(net *lntest.NetworkHarness, t *harnessTest) {
	ctxb := context.Background()

	const (
		chanAmt = 1000000
	)

	// Open a channel between Alice and Bob which will later be
	// cooperatively closed.
	coopChanPoint := openChannelAndAssert(
		t, net, net.Alice, net.Bob, lntest.OpenChannelParams{
			Amt: chanAmt,
		},
	)

	// Create Carol's node and connect Alice to her.
	carol := net.NewNode(t.t, "Carol", nil)
	defer shutdownAndAssert(net, t, carol)
	net.ConnectNodes(t.t, net.Alice, carol)

	// Open a channel between Alice and Carol which will later be force
	// closed.
	forceCloseChanPoint := openChannelAndAssert(
		t, net, net.Alice, carol, lntest.OpenChannelParams{
			Amt: chanAmt,
		},
	)

	// Now, create Dave's a node and also open a channel between Alice and
	// him. This link will serve as the only persistent link throughout
	// restarts in this test.
	dave := net.NewNode(t.t, "Dave", nil)
	defer shutdownAndAssert(net, t, dave)

	net.ConnectNodes(t.t, net.Alice, dave)
	persistentChanPoint := openChannelAndAssert(
		t, net, net.Alice, dave, lntest.OpenChannelParams{
			Amt: chanAmt,
		},
	)

	// isConnected is a helper closure that checks if a peer is connected to
	// Alice.
	isConnected := func(pubKey string) bool {
		req := &lnrpc.ListPeersRequest{}
		ctxt, _ := context.WithTimeout(ctxb, defaultTimeout)
		resp, err := net.Alice.ListPeers(ctxt, req)
		if err != nil {
			t.Fatalf("unable to retrieve alice's peers: %v", err)
		}

		for _, peer := range resp.Peers {
			if peer.PubKey == pubKey {
				return true
			}
		}

		return false
	}

	// Restart both Bob and Carol to ensure Alice is able to reconnect to
	// them.
	if err := net.RestartNode(net.Bob, nil); err != nil {
		t.Fatalf("unable to restart bob's node: %v", err)
	}
	if err := net.RestartNode(carol, nil); err != nil {
		t.Fatalf("unable to restart carol's node: %v", err)
	}

	require.Eventually(t.t, func() bool {
		return isConnected(net.Bob.PubKeyStr)
	}, defaultTimeout, 20*time.Millisecond)
	require.Eventually(t.t, func() bool {
		return isConnected(carol.PubKeyStr)
	}, defaultTimeout, 20*time.Millisecond)

	// We'll also restart Alice to ensure she can reconnect to her peers
	// with open channels.
	if err := net.RestartNode(net.Alice, nil); err != nil {
		t.Fatalf("unable to restart alice's node: %v", err)
	}

	require.Eventually(t.t, func() bool {
		return isConnected(net.Bob.PubKeyStr)
	}, defaultTimeout, 20*time.Millisecond)
	require.Eventually(t.t, func() bool {
		return isConnected(carol.PubKeyStr)
	}, defaultTimeout, 20*time.Millisecond)
	require.Eventually(t.t, func() bool {
		return isConnected(dave.PubKeyStr)
	}, defaultTimeout, 20*time.Millisecond)
	err := wait.Predicate(func() bool {
		return isConnected(dave.PubKeyStr)
	}, defaultTimeout)

	// testReconnection is a helper closure that restarts the nodes at both
	// ends of a channel to ensure they do not reconnect after restarting.
	// When restarting Alice, we'll first need to ensure she has
	// reestablished her connection with Dave, as they still have an open
	// channel together.
	testReconnection := func(node *lntest.HarnessNode) {
		// Restart both nodes, to trigger the pruning logic.
		if err := net.RestartNode(node, nil); err != nil {
			t.Fatalf("unable to restart %v's node: %v",
				node.Name(), err)
		}

		if err := net.RestartNode(net.Alice, nil); err != nil {
			t.Fatalf("unable to restart alice's node: %v", err)
		}

		// Now restart both nodes and make sure they don't reconnect.
		if err := net.RestartNode(node, nil); err != nil {
			t.Fatalf("unable to restart %v's node: %v", node.Name(),
				err)
		}
		err = wait.Invariant(func() bool {
			return !isConnected(node.PubKeyStr)
		}, 5*time.Second)
		if err != nil {
			t.Fatalf("alice reconnected to %v", node.Name())
		}

		if err := net.RestartNode(net.Alice, nil); err != nil {
			t.Fatalf("unable to restart alice's node: %v", err)
		}
		err = wait.Predicate(func() bool {
			return isConnected(dave.PubKeyStr)
		}, defaultTimeout)
		if err != nil {
			t.Fatalf("alice didn't reconnect to Dave")
		}

		err = wait.Invariant(func() bool {
			return !isConnected(node.PubKeyStr)
		}, 5*time.Second)
		if err != nil {
			t.Fatalf("alice reconnected to %v", node.Name())
		}
	}

	// Now, we'll close the channel between Alice and Bob and ensure there
	// is no reconnection logic between the both once the channel is fully
	// closed.
	closeChannelAndAssert(t, net, net.Alice, coopChanPoint, false)

	testReconnection(net.Bob)

	// We'll do the same with Alice and Carol, but this time we'll force
	// close the channel instead.
	closeChannelAndAssert(t, net, net.Alice, forceCloseChanPoint, true)

	// Cleanup by mining the force close and sweep transaction.
	cleanupForceClose(t, net, net.Alice, forceCloseChanPoint)

	// We'll need to mine some blocks in order to mark the channel fully
	// closed.
	_, err = net.Miner.Client.Generate(chainreg.DefaultBitcoinTimeLockDelta - defaultCSV)
	if err != nil {
		t.Fatalf("unable to generate blocks: %v", err)
	}

	// Before we test reconnection, we'll ensure that the channel has been
	// fully cleaned up for both Carol and Alice.
	var predErr error
	pendingChansRequest := &lnrpc.PendingChannelsRequest{}
	err = wait.Predicate(func() bool {
		ctxt, _ := context.WithTimeout(ctxb, defaultTimeout)
		pendingChanResp, err := net.Alice.PendingChannels(
			ctxt, pendingChansRequest,
		)
		if err != nil {
			predErr = fmt.Errorf("unable to query for pending "+
				"channels: %v", err)
			return false
		}

		predErr = checkNumForceClosedChannels(pendingChanResp, 0)
		if predErr != nil {
			return false
		}

		ctxt, _ = context.WithTimeout(ctxb, defaultTimeout)
		pendingChanResp, err = carol.PendingChannels(
			ctxt, pendingChansRequest,
		)
		if err != nil {
			predErr = fmt.Errorf("unable to query for pending "+
				"channels: %v", err)
			return false
		}

		predErr = checkNumForceClosedChannels(pendingChanResp, 0)

		return predErr == nil
	}, defaultTimeout)
	if err != nil {
		t.Fatalf("channels not marked as fully resolved: %v", predErr)
	}

	testReconnection(carol)

	// Finally, we'll ensure that Bob and Carol no longer show in Alice's
	// channel graph.
	describeGraphReq := &lnrpc.ChannelGraphRequest{
		IncludeUnannounced: true,
	}
	ctxt, _ := context.WithTimeout(ctxb, defaultTimeout)
	channelGraph, err := net.Alice.DescribeGraph(ctxt, describeGraphReq)
	if err != nil {
		t.Fatalf("unable to query for alice's channel graph: %v", err)
	}
	for _, node := range channelGraph.Nodes {
		if node.PubKey == net.Bob.PubKeyStr {
			t.Fatalf("did not expect to find bob in the channel " +
				"graph, but did")
		}
		if node.PubKey == carol.PubKeyStr {
			t.Fatalf("did not expect to find carol in the channel " +
				"graph, but did")
		}
	}

	// Now that the test is done, we can also close the persistent link.
	closeChannelAndAssert(t, net, net.Alice, persistentChanPoint, false)
}

// testRejectHTLC tests that a node can be created with the flag --rejecthtlc.
// This means that the node will reject all forwarded HTLCs but can still
// accept direct HTLCs as well as send HTLCs.
func testRejectHTLC(net *lntest.NetworkHarness, t *harnessTest) {
	//             RejectHTLC
	// Alice ------> Carol ------> Bob
	//
	const chanAmt = btcutil.Amount(1000000)
	ctxb := context.Background()

	// Create Carol with reject htlc flag.
	carol := net.NewNode(t.t, "Carol", []string{"--rejecthtlc"})
	defer shutdownAndAssert(net, t, carol)

	// Connect Alice to Carol.
	net.ConnectNodes(t.t, net.Alice, carol)

	// Connect Carol to Bob.
	net.ConnectNodes(t.t, carol, net.Bob)

	// Send coins to Carol.
	net.SendCoins(t.t, btcutil.SatoshiPerBitcoin, carol)

	// Send coins to Alice.
	net.SendCoins(t.t, btcutil.SatoshiPerBitcent, net.Alice)

	// Open a channel between Alice and Carol.
	chanPointAlice := openChannelAndAssert(
		t, net, net.Alice, carol,
		lntest.OpenChannelParams{
			Amt: chanAmt,
		},
	)

	// Open a channel between Carol and Bob.
	chanPointCarol := openChannelAndAssert(
		t, net, carol, net.Bob,
		lntest.OpenChannelParams{
			Amt: chanAmt,
		},
	)

	// Channel should be ready for payments.
	const payAmt = 100

	// Helper closure to generate a random pre image.
	genPreImage := func() []byte {
		preimage := make([]byte, 32)

		_, err := rand.Read(preimage)
		if err != nil {
			t.Fatalf("unable to generate preimage: %v", err)
		}

		return preimage
	}

	// Create an invoice from Carol of 100 satoshis.
	// We expect Alice to be able to pay this invoice.
	preimage := genPreImage()

	carolInvoice := &lnrpc.Invoice{
		Memo:      "testing - alice should pay carol",
		RPreimage: preimage,
		Value:     payAmt,
	}

	// Carol adds the invoice to her database.
	resp, err := carol.AddInvoice(ctxb, carolInvoice)
	if err != nil {
		t.Fatalf("unable to add invoice: %v", err)
	}

	// Alice pays Carols invoice.
	err = completePaymentRequests(
		net.Alice, net.Alice.RouterClient,
		[]string{resp.PaymentRequest}, true,
	)
	if err != nil {
		t.Fatalf("unable to send payments from alice to carol: %v", err)
	}

	// Create an invoice from Bob of 100 satoshis.
	// We expect Carol to be able to pay this invoice.
	preimage = genPreImage()

	bobInvoice := &lnrpc.Invoice{
		Memo:      "testing - carol should pay bob",
		RPreimage: preimage,
		Value:     payAmt,
	}

	// Bob adds the invoice to his database.
	resp, err = net.Bob.AddInvoice(ctxb, bobInvoice)
	if err != nil {
		t.Fatalf("unable to add invoice: %v", err)
	}

	// Carol pays Bobs invoice.
	err = completePaymentRequests(
		carol, carol.RouterClient,
		[]string{resp.PaymentRequest}, true,
	)
	if err != nil {
		t.Fatalf("unable to send payments from carol to bob: %v", err)
	}

	// Create an invoice from Bob of 100 satoshis.
	// Alice attempts to pay Bob but this should fail, since we are
	// using Carol as a hop and her node will reject onward HTLCs.
	preimage = genPreImage()

	bobInvoice = &lnrpc.Invoice{
		Memo:      "testing - alice tries to pay bob",
		RPreimage: preimage,
		Value:     payAmt,
	}

	// Bob adds the invoice to his database.
	resp, err = net.Bob.AddInvoice(ctxb, bobInvoice)
	if err != nil {
		t.Fatalf("unable to add invoice: %v", err)
	}

	// Alice attempts to pay Bobs invoice. This payment should be rejected since
	// we are using Carol as an intermediary hop, Carol is running lnd with
	// --rejecthtlc.
	err = completePaymentRequests(
		net.Alice, net.Alice.RouterClient,
		[]string{resp.PaymentRequest}, true,
	)
	if err == nil {
		t.Fatalf(
			"should have been rejected, carol will not accept forwarded htlcs",
		)
	}

	assertLastHTLCError(t, net.Alice, lnrpc.Failure_CHANNEL_DISABLED)

	// Close all channels.
	closeChannelAndAssert(t, net, net.Alice, chanPointAlice, false)
	closeChannelAndAssert(t, net, carol, chanPointCarol, false)
}

func testNodeSignVerify(net *lntest.NetworkHarness, t *harnessTest) {
	ctxb := context.Background()

	chanAmt := funding.MaxBtcFundingAmount
	pushAmt := btcutil.Amount(100000)

	// Create a channel between alice and bob.
	aliceBobCh := openChannelAndAssert(
		t, net, net.Alice, net.Bob,
		lntest.OpenChannelParams{
			Amt:     chanAmt,
			PushAmt: pushAmt,
		},
	)

	aliceMsg := []byte("alice msg")

	// alice signs "alice msg" and sends her signature to bob.
	sigReq := &lnrpc.SignMessageRequest{Msg: aliceMsg}
	ctxt, _ := context.WithTimeout(ctxb, defaultTimeout)
	sigResp, err := net.Alice.SignMessage(ctxt, sigReq)
	if err != nil {
		t.Fatalf("SignMessage rpc call failed: %v", err)
	}
	aliceSig := sigResp.Signature

	// bob verifying alice's signature should succeed since alice and bob are
	// connected.
	verifyReq := &lnrpc.VerifyMessageRequest{Msg: aliceMsg, Signature: aliceSig}
	ctxt, _ = context.WithTimeout(ctxb, defaultTimeout)
	verifyResp, err := net.Bob.VerifyMessage(ctxt, verifyReq)
	if err != nil {
		t.Fatalf("VerifyMessage failed: %v", err)
	}
	if !verifyResp.Valid {
		t.Fatalf("alice's signature didn't validate")
	}
	if verifyResp.Pubkey != net.Alice.PubKeyStr {
		t.Fatalf("alice's signature doesn't contain alice's pubkey.")
	}

	// carol is a new node that is unconnected to alice or bob.
	carol := net.NewNode(t.t, "Carol", nil)
	defer shutdownAndAssert(net, t, carol)

	carolMsg := []byte("carol msg")

	// carol signs "carol msg" and sends her signature to bob.
	sigReq = &lnrpc.SignMessageRequest{Msg: carolMsg}
	ctxt, _ = context.WithTimeout(ctxb, defaultTimeout)
	sigResp, err = carol.SignMessage(ctxt, sigReq)
	if err != nil {
		t.Fatalf("SignMessage rpc call failed: %v", err)
	}
	carolSig := sigResp.Signature

	// bob verifying carol's signature should fail since they are not connected.
	verifyReq = &lnrpc.VerifyMessageRequest{Msg: carolMsg, Signature: carolSig}
	ctxt, _ = context.WithTimeout(ctxb, defaultTimeout)
	verifyResp, err = net.Bob.VerifyMessage(ctxt, verifyReq)
	if err != nil {
		t.Fatalf("VerifyMessage failed: %v", err)
	}
	if verifyResp.Valid {
		t.Fatalf("carol's signature should not be valid")
	}
	if verifyResp.Pubkey != carol.PubKeyStr {
		t.Fatalf("carol's signature doesn't contain her pubkey")
	}

	// Close the channel between alice and bob.
	closeChannelAndAssert(t, net, net.Alice, aliceBobCh, false)
}

// testAbandonChannel abandones a channel and asserts that it is no
// longer open and not in one of the pending closure states. It also
// verifies that the abandoned channel is reported as closed with close
// type 'abandoned'.
func testAbandonChannel(net *lntest.NetworkHarness, t *harnessTest) {
	ctxb := context.Background()

	// First establish a channel between Alice and Bob.
	channelParam := lntest.OpenChannelParams{
		Amt:     funding.MaxBtcFundingAmount,
		PushAmt: btcutil.Amount(100000),
	}

	chanPoint := openChannelAndAssert(
		t, net, net.Alice, net.Bob, channelParam,
	)
	txid, err := lnrpc.GetChanPointFundingTxid(chanPoint)
	require.NoError(t.t, err, "alice bob get channel funding txid")
	chanPointStr := fmt.Sprintf("%v:%v", txid, chanPoint.OutputIndex)

	// Wait for channel to be confirmed open.
	err = net.Alice.WaitForNetworkChannelOpen(chanPoint)
	require.NoError(t.t, err, "alice wait for network channel open")
	err = net.Bob.WaitForNetworkChannelOpen(chanPoint)
	require.NoError(t.t, err, "bob wait for network channel open")

	// Now that the channel is open, we'll obtain its channel ID real quick
	// so we can use it to query the graph below.
	listReq := &lnrpc.ListChannelsRequest{}
	ctxt, _ := context.WithTimeout(ctxb, defaultTimeout)
	aliceChannelList, err := net.Alice.ListChannels(ctxt, listReq)
	require.NoError(t.t, err)
	var chanID uint64
	for _, channel := range aliceChannelList.Channels {
		if channel.ChannelPoint == chanPointStr {
			chanID = channel.ChanId
		}
	}

	require.NotZero(t.t, chanID, "unable to find channel")

	// To make sure the channel is removed from the backup file as well when
	// being abandoned, grab a backup snapshot so we can compare it with the
	// later state.
	bkupBefore, err := ioutil.ReadFile(net.Alice.ChanBackupPath())
	require.NoError(t.t, err, "channel backup before abandoning channel")

	// Send request to abandon channel.
	abandonChannelRequest := &lnrpc.AbandonChannelRequest{
		ChannelPoint: chanPoint,
	}

	ctxt, _ = context.WithTimeout(ctxb, defaultTimeout)
	_, err = net.Alice.AbandonChannel(ctxt, abandonChannelRequest)
	require.NoError(t.t, err, "abandon channel")

	// Assert that channel in no longer open.
	ctxt, _ = context.WithTimeout(ctxb, defaultTimeout)
	aliceChannelList, err = net.Alice.ListChannels(ctxt, listReq)
	require.NoError(t.t, err, "list channels")
	require.Zero(t.t, len(aliceChannelList.Channels), "alice open channels")

	// Assert that channel is not pending closure.
	pendingReq := &lnrpc.PendingChannelsRequest{}
	ctxt, _ = context.WithTimeout(ctxb, defaultTimeout)
	alicePendingList, err := net.Alice.PendingChannels(ctxt, pendingReq)
	require.NoError(t.t, err, "alice list pending channels")
	require.Zero(
		t.t, len(alicePendingList.PendingClosingChannels), //nolint:staticcheck
		"alice pending channels",
	)
	require.Zero(
		t.t, len(alicePendingList.PendingForceClosingChannels),
		"alice pending force close channels",
	)
	require.Zero(
		t.t, len(alicePendingList.WaitingCloseChannels),
		"alice waiting close channels",
	)

	// Assert that channel is listed as abandoned.
	closedReq := &lnrpc.ClosedChannelsRequest{
		Abandoned: true,
	}
	ctxt, _ = context.WithTimeout(ctxb, defaultTimeout)
	aliceClosedList, err := net.Alice.ClosedChannels(ctxt, closedReq)
	require.NoError(t.t, err, "alice list closed channels")
	require.Len(t.t, aliceClosedList.Channels, 1, "alice closed channels")

	// Ensure that the channel can no longer be found in the channel graph.
	err = wait.NoError(func() error {
		_, err := net.Alice.GetChanInfo(ctxb, &lnrpc.ChanInfoRequest{
			ChanId: chanID,
		})
		if err == nil {
			return fmt.Errorf("expected error but got nil")
		}

		if !strings.Contains(err.Error(), "marked as zombie") {
			return fmt.Errorf("expected error to contain '%s' but "+
				"was '%v'", "marked as zombie", err)
		}

		return nil
	}, defaultTimeout)
	require.NoError(t.t, err, "marked as zombie")

	// Make sure the channel is no longer in the channel backup list.
	err = wait.NoError(func() error {
		bkupAfter, err := ioutil.ReadFile(net.Alice.ChanBackupPath())
		if err != nil {
			return fmt.Errorf("could not get channel backup "+
				"before abandoning channel: %v", err)
		}

		if len(bkupAfter) >= len(bkupBefore) {
			return fmt.Errorf("expected backups after to be less "+
				"than %d but was %d", bkupBefore, bkupAfter)
		}

		return nil
	}, defaultTimeout)
	require.NoError(t.t, err, "channel removed from backup file")

	// Calling AbandonChannel again, should result in no new errors, as the
	// channel has already been removed.
	ctxt, _ = context.WithTimeout(ctxb, defaultTimeout)
	_, err = net.Alice.AbandonChannel(ctxt, abandonChannelRequest)
	require.NoError(t.t, err, "abandon channel second time")

	// Now that we're done with the test, the channel can be closed. This
	// is necessary to avoid unexpected outcomes of other tests that use
	// Bob's lnd instance.
	closeChannelAndAssert(t, net, net.Bob, chanPoint, true)

	// Cleanup by mining the force close and sweep transaction.
	cleanupForceClose(t, net, net.Bob, chanPoint)
}

// testSweepAllCoins tests that we're able to properly sweep all coins from the
// wallet into a single target address at the specified fee rate.
//
// TODO(yy): expand this test to also use P2TR.
func testSweepAllCoins(ht *lntemp.HarnessTest) {
	// First, we'll make a new node, ainz who'll we'll use to test wallet
	// sweeping.
	//
	// NOTE: we won't use standby nodes here since the test will change
	// each of the node's wallet state.
	ainz := ht.NewNode("Ainz", nil)

	// Next, we'll give Ainz exactly 2 utxos of 1 BTC each, with one of
	// them being p2wkh and the other being a n2wpkh address.
	ht.FundCoins(btcutil.SatoshiPerBitcoin, ainz)
	ht.FundCoinsNP2WKH(btcutil.SatoshiPerBitcoin, ainz)

	// Ensure that we can't send coins to our own Pubkey.
	info := ainz.RPC.GetInfo()

	// Create a label that we will used to label the transaction with.
	sendCoinsLabel := "send all coins"

	sweepReq := &lnrpc.SendCoinsRequest{
		Addr:    info.IdentityPubkey,
		SendAll: true,
		Label:   sendCoinsLabel,
	}
	ainz.RPC.SendCoinsAssertErr(sweepReq)

	// Ensure that we can't send coins to another user's Pubkey.
	info = ht.Alice.RPC.GetInfo()

	sweepReq = &lnrpc.SendCoinsRequest{
		Addr:    info.IdentityPubkey,
		SendAll: true,
		Label:   sendCoinsLabel,
	}
	ainz.RPC.SendCoinsAssertErr(sweepReq)

	// With the two coins above mined, we'll now instruct ainz to sweep all
	// the coins to an external address not under its control.  We will
	// first attempt to send the coins to addresses that are not compatible
	// with the current network. This is to test that the wallet will
	// prevent any onchain transactions to addresses that are not on the
	// same network as the user.

	// Send coins to a testnet3 address.
	sweepReq = &lnrpc.SendCoinsRequest{
		Addr:    "tb1qfc8fusa98jx8uvnhzavxccqlzvg749tvjw82tg",
		SendAll: true,
		Label:   sendCoinsLabel,
	}
	ainz.RPC.SendCoinsAssertErr(sweepReq)

	// Send coins to a mainnet address.
	sweepReq = &lnrpc.SendCoinsRequest{
		Addr:    "1MPaXKp5HhsLNjVSqaL7fChE3TVyrTMRT3",
		SendAll: true,
		Label:   sendCoinsLabel,
	}
	ainz.RPC.SendCoinsAssertErr(sweepReq)

	// Send coins to a compatible address.
	minerAddr := ht.Miner.NewMinerAddress()
	sweepReq = &lnrpc.SendCoinsRequest{
		Addr:    minerAddr.String(),
		SendAll: true,
		Label:   sendCoinsLabel,
	}
	ainz.RPC.SendCoins(sweepReq)

	// We'll mine a block which should include the sweep transaction we
	// generated above.
	block := ht.MineBlocksAndAssertNumTxes(1, 1)[0]

	// The sweep transaction should have exactly two inputs as we only had
	// two UTXOs in the wallet.
	sweepTx := block.Transactions[1]
	require.Len(ht, sweepTx.TxIn, 2, "expected 2 inputs")

	// assertTxLabel is a helper function which finds a target tx in our
	// set of transactions and checks that it has the desired label.
	assertTxLabel := func(targetTx, label string) error {
		// List all transactions relevant to our wallet, and find the
		// tx so that we can check the correct label has been set.
		txResp := ainz.RPC.GetTransactions()

		var target *lnrpc.Transaction

		// First we need to find the target tx.
		for _, txn := range txResp.Transactions {
			if txn.TxHash == targetTx {
				target = txn
			}
		}

		// If we cannot find it, return an error.
		if target == nil {
			return fmt.Errorf("target tx %v not found", targetTx)
		}

		// Otherwise, check the labels are matched.
		if target.Label == label {
			return nil
		}

		return fmt.Errorf("labels not match, want: "+
			"%v, got %v", label, target.Label)
	}

	// waitTxLabel waits until the desired tx label is found or timeout.
	waitTxLabel := func(targetTx, label string) {
		err := wait.NoError(func() error {
			return assertTxLabel(targetTx, label)
		}, defaultTimeout)

		require.NoError(ht, err, "timeout assertTxLabel")
	}

	sweepTxStr := sweepTx.TxHash().String()
	waitTxLabel(sweepTxStr, sendCoinsLabel)

	// While we are looking at labels, we test our label transaction
	// command to make sure it is behaving as expected. First, we try to
	// label our transaction with an empty label, and check that we fail as
	// expected.
	sweepHash := sweepTx.TxHash()
	req := &walletrpc.LabelTransactionRequest{
		Txid:      sweepHash[:],
		Label:     "",
		Overwrite: false,
	}
	err := ainz.RPC.LabelTransactionAssertErr(req)

	// Our error will be wrapped in a rpc error, so we check that it
	// contains the error we expect.
	errZeroLabel := "cannot label transaction with empty label"
	require.Contains(ht, err.Error(), errZeroLabel,
		"expected: zero label errorv")

	// Next, we try to relabel our transaction without setting the overwrite
	// boolean. We expect this to fail, because the wallet requires setting
	// of this param to prevent accidental overwrite of labels.
	req = &walletrpc.LabelTransactionRequest{
		Txid:      sweepHash[:],
		Label:     "label that will not work",
		Overwrite: false,
	}
	err = ainz.RPC.LabelTransactionAssertErr(req)

	// Our error will be wrapped in a rpc error, so we check that it
	// contains the error we expect.
	require.Contains(ht, err.Error(), wallet.ErrTxLabelExists.Error())

	// Finally, we overwrite our label with a new label, which should not
	// fail.
	newLabel := "new sweep tx label"
	req = &walletrpc.LabelTransactionRequest{
		Txid:      sweepHash[:],
		Label:     newLabel,
		Overwrite: true,
	}
	ainz.RPC.LabelTransaction(req)

	waitTxLabel(sweepTxStr, newLabel)

	// Finally, Ainz should now have no coins at all within his wallet.
	resp := ainz.RPC.WalletBalance()
	require.Zero(ht, resp.ConfirmedBalance, "wrong confirmed balance")
	require.Zero(ht, resp.UnconfirmedBalance, "wrong unconfirmed balance")

	// If we try again, but this time specifying an amount, then the call
	// should fail.
	sweepReq.Amount = 10000
	ainz.RPC.SendCoinsAssertErr(sweepReq)
}

// testListAddresses tests that we get all the addresses and their
// corresponding balance correctly.
func testListAddresses(net *lntest.NetworkHarness, t *harnessTest) {
	ctxb := context.Background()

	// First, we'll make a new node - Alice, which will be generating
	// new addresses.
	alice := net.NewNode(t.t, "Alice", nil)
	defer shutdownAndAssert(net, t, alice)

	// Next, we'll give Alice exactly 1 utxo of 1 BTC.
	net.SendCoins(t.t, btcutil.SatoshiPerBitcoin, alice)

	type addressDetails struct {
		Balance int64
		Type    walletrpc.AddressType
	}

	// A map of generated address and its balance.
	generatedAddr := make(map[string]addressDetails)

	// Create an address generated from internal keys.
	keyLoc := &walletrpc.KeyReq{KeyFamily: 123}
	keyDesc, err := alice.WalletKitClient.DeriveNextKey(ctxb, keyLoc)
	require.NoError(t.t, err)

	// Hex Encode the public key.
	pubkeyString := hex.EncodeToString(keyDesc.RawKeyBytes)

	// Create a p2tr address.
	resp, err := alice.NewAddress(ctxb, &lnrpc.NewAddressRequest{
		Type: lnrpc.AddressType_TAPROOT_PUBKEY,
	})
	require.NoError(t.t, err)
	generatedAddr[resp.Address] = addressDetails{
		Balance: 200_000,
		Type:    walletrpc.AddressType_TAPROOT_PUBKEY,
	}

	// Create a p2wkh address.
	resp, err = alice.NewAddress(ctxb, &lnrpc.NewAddressRequest{
		Type: lnrpc.AddressType_WITNESS_PUBKEY_HASH,
	})
	require.NoError(t.t, err)
	generatedAddr[resp.Address] = addressDetails{
		Balance: 300_000,
		Type:    walletrpc.AddressType_WITNESS_PUBKEY_HASH,
	}

	// Create a np2wkh address.
	resp, err = alice.NewAddress(ctxb, &lnrpc.NewAddressRequest{
		Type: lnrpc.AddressType_NESTED_PUBKEY_HASH,
	})
	require.NoError(t.t, err)
	generatedAddr[resp.Address] = addressDetails{
		Balance: 400_000,
		Type:    walletrpc.AddressType_HYBRID_NESTED_WITNESS_PUBKEY_HASH,
	}

	for addr, addressDetail := range generatedAddr {
		_, err := alice.SendCoins(ctxb, &lnrpc.SendCoinsRequest{
			Addr:             addr,
			Amount:           addressDetail.Balance,
			SpendUnconfirmed: true,
		})
		require.NoError(t.t, err)
	}

	mineBlocks(t, net, 1, 3)

	// Get all the accounts except LND's custom accounts.
	addressLists, err := alice.WalletKitClient.ListAddresses(
		ctxb, &walletrpc.ListAddressesRequest{},
	)
	require.NoError(t.t, err)

	foundAddresses := 0
	for _, addressList := range addressLists.AccountWithAddresses {
		addresses := addressList.Addresses
		derivationPath, err := parseDerivationPath(
			addressList.DerivationPath,
		)
		require.NoError(t.t, err)

		// Should not get an account with KeyFamily - 123.
		require.NotEqual(
			t.t, uint32(keyLoc.KeyFamily), derivationPath[2],
		)

		for _, address := range addresses {
			if _, ok := generatedAddr[address.Address]; ok {
				addrDetails := generatedAddr[address.Address]
				require.Equal(
					t.t, addrDetails.Balance,
					address.Balance,
				)
				require.Equal(
					t.t, addrDetails.Type,
					addressList.AddressType,
				)
				foundAddresses++
			}
		}
	}

	require.Equal(t.t, len(generatedAddr), foundAddresses)
	foundAddresses = 0

	// Get all the accounts (including LND's custom accounts).
	addressLists, err = alice.WalletKitClient.ListAddresses(
		ctxb, &walletrpc.ListAddressesRequest{
			ShowCustomAccounts: true,
		},
	)
	require.NoError(t.t, err)

	for _, addressList := range addressLists.AccountWithAddresses {
		addresses := addressList.Addresses
		derivationPath, err := parseDerivationPath(
			addressList.DerivationPath,
		)
		require.NoError(t.t, err)

		for _, address := range addresses {
			// Check if the KeyFamily in derivation path is 123.
			if uint32(keyLoc.KeyFamily) == derivationPath[2] {
				// For LND's custom accounts, the address
				// represents the public key.
				pubkey := address.Address
				require.Equal(t.t, pubkeyString, pubkey)
			} else if _, ok := generatedAddr[address.Address]; ok {
				addrDetails := generatedAddr[address.Address]
				require.Equal(
					t.t, addrDetails.Balance,
					address.Balance,
				)
				require.Equal(
					t.t, addrDetails.Type,
					addressList.AddressType,
				)
				foundAddresses++
			}
		}
	}

	require.Equal(t.t, len(generatedAddr), foundAddresses)
}

func assertChannelConstraintsEqual(ht *lntemp.HarnessTest,
	want, got *lnrpc.ChannelConstraints) {

	require.Equal(ht, want.CsvDelay, got.CsvDelay, "CsvDelay mismatched")

	require.Equal(ht, want.ChanReserveSat, got.ChanReserveSat,
		"ChanReserveSat mismatched")

	require.Equal(ht, want.DustLimitSat, got.DustLimitSat,
		"DustLimitSat mismatched")

	require.Equal(ht, want.MaxPendingAmtMsat, got.MaxPendingAmtMsat,
		"MaxPendingAmtMsat mismatched")

	require.Equal(ht, want.MinHtlcMsat, got.MinHtlcMsat,
		"MinHtlcMsat mismatched")

	require.Equal(ht, want.MaxAcceptedHtlcs, got.MaxAcceptedHtlcs,
		"MaxAcceptedHtlcs mismatched")
}
