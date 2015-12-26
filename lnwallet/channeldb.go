package lnwallet

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"time"

	"li.lan/labs/plasma/shachain"

	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil"
	"github.com/btcsuite/btcwallet/waddrmgr"
	"github.com/btcsuite/btcwallet/walletdb"
)

var (
	// Namespace bucket keys.
	lightningNamespaceKey = []byte("ln-wallet")
	waddrmgrNamespaceKey  = []byte("waddrmgr")
	wtxmgrNamespaceKey    = []byte("wtxmgr")

	openChannelBucket   = []byte("o")
	closedChannelBucket = []byte("c")
	activeChanKey       = []byte("a")

	endian = binary.BigEndian
)

// ChannelDB...
// TODO(roasbeef): CHECKSUMS, REDUNDANCY, etc etc.
type ChannelDB struct {
	// TODO(roasbeef): caching, etc?
	addrmgr *waddrmgr.Manager

	namespace walletdb.Namespace
}

// PutOpenChannel...
func (c *ChannelDB) PutOpenChannel(channel *OpenChannelState) error {
	return c.namespace.Update(func(tx walletdb.Tx) error {
		// Get the bucket dedicated to storing the meta-data for open
		// channels.
		rootBucket := tx.RootBucket()
		openChanBucket, err := rootBucket.CreateBucketIfNotExists(openChannelBucket)
		if err != nil {
			return err
		}

		return dbPutOpenChannel(openChanBucket, channel, c.addrmgr)
	})
}

// GetOpenChannel...
// TODO(roasbeef): assumes only 1 active channel per-node
func (c *ChannelDB) FetchOpenChannel(nodeID [32]byte) (*OpenChannelState, error) {
	var channel *OpenChannelState

	dbErr := c.namespace.View(func(tx walletdb.Tx) error {
		// Get the bucket dedicated to storing the meta-data for open
		// channels.
		rootBucket := tx.RootBucket()
		openChanBucket := rootBucket.Bucket(openChannelBucket)
		if openChannelBucket == nil {
			return fmt.Errorf("open channel bucket does not exist")
		}

		oChannel, err := dbGetOpenChannel(openChanBucket, nodeID, c.addrmgr)
		if err != nil {
			return err
		}
		channel = oChannel
		return nil
	})

	return channel, dbErr
}

// dbPutChannel...
func dbPutOpenChannel(activeChanBucket walletdb.Bucket, channel *OpenChannelState,
	addrmgr *waddrmgr.Manager) error {

	// Generate a serialized version of the open channel. The addrmgr is
	// required in order to encrypt densitive data.
	var b bytes.Buffer
	if err := channel.Encode(&b, addrmgr); err != nil {
		return err
	}

	// Grab the bucket dedicated to storing data related to this particular
	// node.
	nodeBucket, err := activeChanBucket.CreateBucketIfNotExists(channel.TheirLNID[:])
	if err != nil {
		return err
	}

	return nodeBucket.Put(activeChanKey, b.Bytes())
}

// dbPutChannel...
func dbGetOpenChannel(bucket walletdb.Bucket, nodeID [32]byte,
	addrmgr *waddrmgr.Manager) (*OpenChannelState, error) {

	// Grab the bucket dedicated to storing data related to this particular
	// node.
	nodeBucket := bucket.Bucket(nodeID[:])
	if nodeBucket == nil {
		return nil, fmt.Errorf("channel bucket for node does not exist")
	}

	serializedChannel := nodeBucket.Get(activeChanKey)
	if serializedChannel == nil {
		// TODO(roasbeef): make proper in error.go
		return nil, fmt.Errorf("node has no open channels")
	}

	// Decode the serialized channel state, using the addrmgr to decrypt
	// sensitive information.
	channel := &OpenChannelState{}
	reader := bytes.NewReader(serializedChannel)
	if err := channel.Decode(reader, addrmgr); err != nil {
		return nil, err
	}

	return channel, nil
}

// NewChannelDB...
// TODO(roasbeef): re-visit this dependancy...
func NewChannelDB(addrmgr *waddrmgr.Manager, namespace walletdb.Namespace) *ChannelDB {
	// TODO(roasbeef): create buckets if not created?
	return &ChannelDB{addrmgr, namespace}
}

// OpenChannelState...
// TODO(roasbeef): store only the essentials? optimize space...
// TODO(roasbeef): switch to "column store"
type OpenChannelState struct {
	// Hash? or Their current pubKey?
	// TODO(roasbeef): switch to Tadge's LNId
	TheirLNID [wire.HashSize]byte

	// The ID of a channel is the txid of the funding transaction.
	ChanID [wire.HashSize]byte

	MinFeePerKb btcutil.Amount
	// Our reserve. Assume symmetric reserve amounts. Only needed if the
	// funding type is CLTV.
	//ReserveAmount btcutil.Amount

	// Keys for both sides to be used for the commitment transactions.
	OurCommitKey   *btcec.PrivateKey
	TheirCommitKey *btcec.PublicKey

	// Tracking total channel capacity, and the amount of funds allocated
	// to each side.
	Capacity     btcutil.Amount
	OurBalance   btcutil.Amount
	TheirBalance btcutil.Amount

	// Commitment transactions for both sides (they're asymmetric). Also
	// their signature which lets us spend our version of the commitment
	// transaction.
	TheirCommitTx  *wire.MsgTx
	OurCommitTx    *wire.MsgTx // TODO(roasbeef): store hash instead?
	TheirCommitSig []byte      // TODO(roasbeef): fixed length?, same w/ redeem

	// The final funding transaction. Kept wallet-related records.
	FundingTx *wire.MsgTx

	MultiSigKey         *btcec.PrivateKey
	FundingRedeemScript []byte

	// Current revocation for their commitment transaction. However, since
	// this is the hash, and not the pre-image, we can't yet verify that
	// it's actually in the chain.
	TheirCurrentRevocation [wire.HashSize]byte
	TheirShaChain          *shachain.HyperShaChain
	OurShaChain            *shachain.HyperShaChain

	// Final delivery address
	OurDeliveryAddress   btcutil.Address
	TheirDeliveryAddress btcutil.Address

	// In blocks
	CsvDelay uint32

	// TODO(roasbeef): track fees, other stats?
	NumUpdates            uint64
	TotalSatoshisSent     uint64
	TotalSatoshisReceived uint64
	CreationTime          time.Time
}

// Encode...
// TODO(roasbeef): checksum
func (o *OpenChannelState) Encode(b io.Writer, addrManager *waddrmgr.Manager) error {
	if _, err := b.Write(o.TheirLNID[:]); err != nil {
		return err
	}
	if _, err := b.Write(o.ChanID[:]); err != nil {
		return err
	}

	if err := binary.Write(b, endian, uint64(o.MinFeePerKb)); err != nil {
		return err
	}

	encryptedPriv, err := addrManager.Encrypt(waddrmgr.CKTPrivate,
		o.OurCommitKey.Serialize())
	if err != nil {
		return err
	}
	if _, err := b.Write(encryptedPriv); err != nil {
		return err
	}
	if _, err := b.Write(o.TheirCommitKey.SerializeCompressed()); err != nil {
		return err
	}

	if err := binary.Write(b, endian, uint64(o.Capacity)); err != nil {
		return err
	}
	if err := binary.Write(b, endian, uint64(o.OurBalance)); err != nil {
		return err
	}
	if err := binary.Write(b, endian, uint64(o.TheirBalance)); err != nil {
		return err
	}

	if err := o.TheirCommitTx.Serialize(b); err != nil {
		return err
	}
	if err := o.OurCommitTx.Serialize(b); err != nil {
		return err
	}
	if _, err := b.Write(o.TheirCommitSig[:]); err != nil {
		return err
	}

	if err := o.FundingTx.Serialize(b); err != nil {
		return err
	}

	encryptedPriv, err = addrManager.Encrypt(waddrmgr.CKTPrivate,
		o.MultiSigKey.Serialize())
	if err != nil {
		return err
	}
	if _, err := b.Write(encryptedPriv); err != nil {
		return err
	}
	if _, err := b.Write(o.FundingRedeemScript); err != nil {
		return err
	}

	if _, err := b.Write(o.TheirCurrentRevocation[:]); err != nil {
		return err
	}
	// TODO(roasbeef): serialize shachains

	if _, err := b.Write([]byte(o.OurDeliveryAddress.EncodeAddress())); err != nil {
		return err
	}
	if _, err := b.Write([]byte(o.TheirDeliveryAddress.EncodeAddress())); err != nil {
		return err
	}

	if err := binary.Write(b, endian, o.CsvDelay); err != nil {
		return err
	}
	if err := binary.Write(b, endian, o.NumUpdates); err != nil {
		return err
	}
	if err := binary.Write(b, endian, o.TotalSatoshisSent); err != nil {
		return err
	}
	if err := binary.Write(b, endian, o.TotalSatoshisReceived); err != nil {
		return err
	}

	if err := binary.Write(b, endian, o.CreationTime.Unix()); err != nil {
		return err
	}

	return nil
}

// Decode...
func (o *OpenChannelState) Decode(b io.Reader, addrManager *waddrmgr.Manager) error {
	var scratch [8]byte

	if _, err := b.Read(o.TheirLNID[:]); err != nil {
		return err
	}
	if _, err := b.Read(o.ChanID[:]); err != nil {
		return err
	}

	if _, err := b.Read(scratch[:]); err != nil {
		return err
	}
	o.MinFeePerKb = btcutil.Amount(endian.Uint64(scratch[:]))

	// nonce + serPrivKey + mac
	var encryptedPriv [24 + 32 + 16]byte
	if _, err := b.Read(encryptedPriv[:]); err != nil {
		return err
	}
	decryptedPriv, err := addrManager.Decrypt(waddrmgr.CKTPrivate, encryptedPriv[:])
	if err != nil {
		return err
	}
	o.OurCommitKey, _ = btcec.PrivKeyFromBytes(btcec.S256(), decryptedPriv)

	var serPubKey [33]byte
	if _, err := b.Read(serPubKey[:]); err != nil {
		return err
	}
	o.TheirCommitKey, err = btcec.ParsePubKey(serPubKey[:], btcec.S256())
	if err != nil {
		return err
	}

	if _, err := b.Read(scratch[:]); err != nil {
		return err
	}
	o.Capacity = btcutil.Amount(endian.Uint64(scratch[:]))
	if _, err := b.Read(scratch[:]); err != nil {
		return err
	}
	o.OurBalance = btcutil.Amount(endian.Uint64(scratch[:]))
	if _, err := b.Read(scratch[:]); err != nil {
		return err
	}
	o.TheirBalance = btcutil.Amount(endian.Uint64(scratch[:]))

	o.TheirCommitTx = wire.NewMsgTx()
	if err := o.TheirCommitTx.Deserialize(b); err != nil {
		return err
	}
	o.OurCommitTx = wire.NewMsgTx()
	if err := o.OurCommitTx.Deserialize(b); err != nil {
		return err
	}

	var sig [64]byte
	if _, err := b.Read(sig[:]); err != nil {
		return err
	}
	o.TheirCommitSig = sig[:]
	if err != nil {
		return err
	}

	o.FundingTx = wire.NewMsgTx()
	if err := o.FundingTx.Deserialize(b); err != nil {
		return err
	}

	if _, err := b.Read(encryptedPriv[:]); err != nil {
		return err
	}
	decryptedPriv, err = addrManager.Decrypt(waddrmgr.CKTPrivate, encryptedPriv[:])
	if err != nil {
		return err
	}
	o.MultiSigKey, _ = btcec.PrivKeyFromBytes(btcec.S256(), decryptedPriv)

	var redeemScript [71]byte
	if _, err := b.Read(redeemScript[:]); err != nil {
		return err
	}
	o.FundingRedeemScript = redeemScript[:]

	if _, err := b.Read(o.TheirCurrentRevocation[:]); err != nil {
		return err
	}

	var addr [34]byte
	if _, err := b.Read(addr[:]); err != nil {
		return err
	}
	o.OurDeliveryAddress, err = btcutil.DecodeAddress(string(addr[:]), ActiveNetParams)
	if err != nil {
		return err
	}

	if _, err := b.Read(addr[:]); err != nil {
		return err
	}
	o.TheirDeliveryAddress, err = btcutil.DecodeAddress(string(addr[:]), ActiveNetParams)
	if err != nil {
		return err
	}

	if err := binary.Read(b, endian, &o.CsvDelay); err != nil {
		return err
	}
	if err := binary.Read(b, endian, &o.NumUpdates); err != nil {
		return err
	}
	if err := binary.Read(b, endian, &o.TotalSatoshisSent); err != nil {
		return err
	}
	if err := binary.Read(b, endian, &o.TotalSatoshisReceived); err != nil {
		return err
	}

	var unix int64
	if err := binary.Read(b, endian, &unix); err != nil {
		return err
	}
	o.CreationTime = time.Unix(unix, 0)

	return nil
}
