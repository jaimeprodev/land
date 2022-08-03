//go:build rpctest
// +build rpctest

package itest

var allTestCases = []*testCase{
	{
		name: "list addresses",
		test: testListAddresses,
	},
	{
		name: "recovery info",
		test: testGetRecoveryInfo,
	},
	{
		name: "onchain fund recovery",
		test: testOnchainFundRecovery,
	},
	{
		name: "basic funding flow with all input types",
		test: testChannelFundingInputTypes,
	},
	{
		name: "unconfirmed channel funding",
		test: testUnconfirmedChannelFunding,
	},
	{
		name: "update channel policy",
		test: testUpdateChannelPolicy,
	},
	{
		name: "update channel policy fee rate accuracy",
		test: testUpdateChannelPolicyFeeRateAccuracy,
	},
	{
		name: "open channel reorg test",
		test: testOpenChannelAfterReorg,
	},
	{
		name: "reconnect after ip change",
		test: testReconnectAfterIPChange,
	},
	{
		name: "graph topology notifications",
		test: testGraphTopologyNotifications,
	},
	{
		name: "funding flow persistence",
		test: testChannelFundingPersistence,
	},
	{
		name: "channel force closure",
		test: testChannelForceClosure,
	},
	{
		name: "channel balance",
		test: testChannelBalance,
	},
	{
		name: "channel unsettled balance",
		test: testChannelUnsettledBalance,
	},
	{
		name: "single hop invoice",
		test: testSingleHopInvoice,
	},
	{
		name: "test update node announcement rpc",
		test: testUpdateNodeAnnouncement,
	},
	{
		name: "list outgoing payments",
		test: testListPayments,
	},
	{
		name: "multi-hop payments",
		test: testMultiHopPayments,
	},
	{
		name: "single-hop send to route",
		test: testSingleHopSendToRoute,
	},
	{
		name: "multi-hop send to route",
		test: testMultiHopSendToRoute,
	},
	{
		name: "send to route error propagation",
		test: testSendToRouteErrorPropagation,
	},
	{
		name: "unannounced channels",
		test: testUnannouncedChannels,
	},
	{
		name: "private channels",
		test: testPrivateChannels,
	},
	{
		name: "private channel update policy",
		test: testUpdateChannelPolicyForPrivateChannel,
	},
	{
		name: "invoice routing hints",
		test: testInvoiceRoutingHints,
	},
	{
		name: "multi-hop payments over private channels",
		test: testMultiHopOverPrivateChannels,
	},
	{
		name: "multiple channel creation and update subscription",
		test: testBasicChannelCreationAndUpdates,
	},
	{
		name: "invoice update subscription",
		test: testInvoiceSubscriptions,
	},
	{
		name: "multi-hop htlc error propagation",
		test: testHtlcErrorPropagation,
	},
	{
		name: "reject onward htlc",
		test: testRejectHTLC,
	},
	// TODO(roasbeef): multi-path integration test
	{
		name: "node announcement",
		test: testNodeAnnouncement,
	},
	{
		name: "node sign verify",
		test: testNodeSignVerify,
	},
	{
		name: "derive shared key",
		test: testDeriveSharedKey,
	},
	{
		name: "sign output raw",
		test: testSignOutputRaw,
	},
	{
		name: "sign verify message",
		test: testSignVerifyMessage,
	},
	{
		name: "async payments benchmark",
		test: testAsyncPayments,
	},
	{
		name: "async bidirectional payments",
		test: testBidirectionalAsyncPayments,
	},
	{
		name: "switch circuit persistence",
		test: testSwitchCircuitPersistence,
	},
	{
		name: "switch offline delivery",
		test: testSwitchOfflineDelivery,
	},
	{
		name: "switch offline delivery persistence",
		test: testSwitchOfflineDeliveryPersistence,
	},
	{
		name: "switch offline delivery outgoing offline",
		test: testSwitchOfflineDeliveryOutgoingOffline,
	},
	{
		// TODO(roasbeef): test always needs to be last as Bob's state
		// is borked since we trick him into attempting to cheat Alice?
		name: "revoked uncooperative close retribution",
		test: testRevokedCloseRetribution,
	},
	{
		name: "failing link",
		test: testFailingChannel,
	},
	{
		name: "garbage collect link nodes",
		test: testGarbageCollectLinkNodes,
	},
	{
		name: "abandonchannel",
		test: testAbandonChannel,
	},
	{
		name: "revoked uncooperative close retribution zero value remote output",
		test: testRevokedCloseRetributionZeroValueRemoteOutput,
	},
	{
		name: "revoked uncooperative close retribution remote hodl",
		test: testRevokedCloseRetributionRemoteHodl,
	},
	{
		name: "revoked uncooperative close retribution altruist watchtower",
		test: testRevokedCloseRetributionAltruistWatchtower,
	},
	{
		name: "query routes",
		test: testQueryRoutes,
	},
	{
		name: "route fee cutoff",
		test: testRouteFeeCutoff,
	},
	{
		name: "send update disable channel",
		test: testSendUpdateDisableChannel,
	},
	{
		name: "streaming channel backup update",
		test: testChannelBackupUpdates,
	},
	{
		name: "export channel backup",
		test: testExportChannelBackup,
	},
	{
		name: "hold invoice sender persistence",
		test: testHoldInvoicePersistence,
	},
	{
		name: "hold invoice force close",
		test: testHoldInvoiceForceClose,
	},
	{
		name: "commitment deadline",
		test: testCommitmentTransactionDeadline,
	},
	{
		name: "cpfp",
		test: testCPFP,
	},
	{
		name: "anchors reserved value",
		test: testAnchorReservedValue,
	},
	{
		name: "macaroon authentication",
		test: testMacaroonAuthentication,
	},
	{
		name: "bake macaroon",
		test: testBakeMacaroon,
	},
	{
		name: "delete macaroon id",
		test: testDeleteMacaroonID,
	},
	{
		name: "immediate payment after channel opened",
		test: testPaymentFollowingChannelOpen,
	},
	{
		name: "psbt channel funding",
		test: testPsbtChanFunding,
	},
	{
		name: "psbt channel funding external",
		test: testPsbtChanFundingExternal,
	},
	{
		name: "sign psbt",
		test: testSignPsbt,
	},
	{
		name: "batch channel funding",
		test: testBatchChanFunding,
	},
	{
		name: "psbt channel funding single step",
		test: testPsbtChanFundingSingleStep,
	},
	{
		name: "sendtoroute multi path payment",
		test: testSendToRouteMultiPath,
	},
	{
		name: "sendtoroute amp",
		test: testSendToRouteAMP,
	},
	{
		name: "sendpayment amp",
		test: testSendPaymentAMP,
	},
	{
		name: "sendpayment amp invoice",
		test: testSendPaymentAMPInvoice,
	},
	{
		name: "sendpayment amp invoice repeat",
		test: testSendPaymentAMPInvoiceRepeat,
	},
	{
		name: "send multi path payment",
		test: testSendMultiPathPayment,
	},
	{
		name: "REST API",
		test: testRestAPI,
	},
	{
		name: "forward interceptor",
		test: testForwardInterceptorBasic,
	},
	{
		name: "forward interceptor dedup htlcs",
		test: testForwardInterceptorDedupHtlc,
	},
	{
		name: "wumbo channels",
		test: testWumboChannels,
	},
	{
		name: "maximum channel size",
		test: testMaxChannelSize,
	},
	{
		name: "connection timeout",
		test: testNetworkConnectionTimeout,
	},
	{
		name: "stateless init",
		test: testStatelessInit,
	},
	{
		name: "wallet import account",
		test: testWalletImportAccount,
	},
	{
		name: "wallet import pubkey",
		test: testWalletImportPubKey,
	},
	{
		name: "etcd_failover",
		test: testEtcdFailover,
	},
	{
		name: "max htlc pathfind",
		test: testMaxHtlcPathfind,
	},
	{
		name: "rpc middleware interceptor",
		test: testRPCMiddlewareInterceptor,
	},
	{
		name: "wipe forwarding packages",
		test: testWipeForwardingPackages,
	},
	{
		name: "remote signer",
		test: testRemoteSigner,
	},
	{
		name: "3rd party anchor spend",
		test: testAnchorThirdPartySpend,
	},
	{
		name: "taproot",
		test: testTaproot,
	},
	{
		name: "addpeer config",
		test: testAddPeerConfig,
	},
	{
		name: "resolution handoff",
		test: testResHandoff,
	},
	{
		name: "zero conf channel open",
		test: testZeroConfChannelOpen,
	},
	{
		name: "option scid alias",
		test: testOptionScidAlias,
	},
	{
		name: "scid alias channel update",
		test: testUpdateChannelPolicyScidAlias,
	},
	{
		name: "scid alias upgrade",
		test: testOptionScidUpgrade,
	},
	{
		name: "nonstd sweep",
		test: testNonstdSweep,
	},
	{
		name: "taproot coop close",
		test: testTaprootCoopClose,
	},
	{
		name: "trackpayments",
		test: testTrackPayments,
	},
	{
		name: "open channel fee policy",
		test: testOpenChannelUpdateFeePolicy,
	},
}
