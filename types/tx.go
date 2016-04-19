package types

import (
	"bytes"
	"encoding/json"

	. "github.com/tendermint/go-common"
	"github.com/tendermint/go-crypto"
	"github.com/tendermint/go-wire"
	tmsp "github.com/tendermint/tmsp/types"
)

/*
Tx (Transaction) is an atomic operation on the ledger state.

Account Types:
 - SendTx         Send coins to address
 - AppTx          Send a msg to a contract that runs in the vm
 - SignedCobaltTx       Send a Cobalt tx
*/

type Tx interface {
	AssertIsTx()
	SignBytes(chainID string) []byte
}

// Types of Tx implementations
const (
	// Account transactions
	TxTypeSend               = byte(0x01)
	TxTypeApp                = byte(0x02)
	TxTypeBinarySignedCobalt = byte(0x03)
)

func (_ *SendTx) AssertIsTx()               {}
func (_ *AppTx) AssertIsTx()                {}
func (_ *BinarySignedCobaltTx) AssertIsTx() {}

var _ = wire.RegisterInterface(
	struct{ Tx }{},
	wire.ConcreteType{&SendTx{}, TxTypeSend},
	wire.ConcreteType{&AppTx{}, TxTypeApp},
	wire.ConcreteType{&BinarySignedCobaltTx{}, TxTypeBinarySignedCobalt},
)

//-----------------------------------------------------------------------------

type TxInput struct {
	Address   []byte           `json:"address"`   // Hash of the PubKey
	Coins     Coins            `json:"coins"`     //
	Sequence  int              `json:"sequence"`  // Must be 1 greater than the last committed TxInput
	Signature crypto.Signature `json:"signature"` // Depends on the PubKey type and the whole Tx
	PubKey    crypto.PubKey    `json:"pub_key"`   // Is present iff Sequence == 0
}

func (txIn TxInput) ValidateBasic() tmsp.Result {
	if len(txIn.Address) != 20 {
		return tmsp.ErrBaseInvalidInput.AppendLog("Invalid address length")
	}
	if !txIn.Coins.IsValid() {
		return tmsp.ErrBaseInvalidInput.AppendLog(Fmt("Invalid coins %v", txIn.Coins))
	}
	if txIn.Coins.IsZero() {
		return tmsp.ErrBaseInvalidInput.AppendLog("Coins cannot be zero")
	}
	if txIn.Sequence <= 0 {
		return tmsp.ErrBaseInvalidInput.AppendLog("Sequence must be greater than 0")
	}
	if txIn.Sequence == 1 && txIn.PubKey == nil {
		return tmsp.ErrBaseInvalidInput.AppendLog("PubKey must be present when Sequence == 1")
	}
	if txIn.Sequence > 1 && txIn.PubKey != nil {
		return tmsp.ErrBaseInvalidInput.AppendLog("PubKey must be nil when Sequence > 1")
	}
	return tmsp.OK
}

func (txIn TxInput) String() string {
	return Fmt("TxInput{%X,%v,%v,%v,%v}", txIn.Address, txIn.Coins, txIn.Sequence, txIn.Signature, txIn.PubKey)
}

//-----------------------------------------------------------------------------

type TxOutput struct {
	Address []byte `json:"address"` // Hash of the PubKey
	Coins   Coins  `json:"coins"`   //
}

func (txOut TxOutput) ValidateBasic() tmsp.Result {
	if len(txOut.Address) != 20 {
		return tmsp.ErrBaseInvalidOutput.AppendLog("Invalid address length")
	}
	if !txOut.Coins.IsValid() {
		return tmsp.ErrBaseInvalidOutput.AppendLog(Fmt("Invalid coins %v", txOut.Coins))
	}
	if txOut.Coins.IsZero() {
		return tmsp.ErrBaseInvalidOutput.AppendLog("Coins cannot be zero")
	}
	return tmsp.OK
}

func (txOut TxOutput) String() string {
	return Fmt("TxOutput{%X,%v}", txOut.Address, txOut.Coins)
}

//-----------------------------------------------------------------------------

type SendTx struct {
	Fee     int64      `json:"fee"` // Fee
	Gas     int64      `json:"gas"` // Gas
	Inputs  []TxInput  `json:"inputs"`
	Outputs []TxOutput `json:"outputs"`
}

func (tx *SendTx) SignBytes(chainID string) []byte {
	signBytes := wire.BinaryBytes(chainID)
	sigz := make([]crypto.Signature, len(tx.Inputs))
	for i, input := range tx.Inputs {
		sigz[i] = input.Signature
		tx.Inputs[i].Signature = nil
	}
	signBytes = append(signBytes, wire.BinaryBytes(tx)...)
	for i := range tx.Inputs {
		tx.Inputs[i].Signature = sigz[i]
	}
	return signBytes
}

func (tx *SendTx) SetSignature(addr []byte, sig crypto.Signature) bool {
	for i, input := range tx.Inputs {
		if bytes.Equal(input.Address, addr) {
			tx.Inputs[i].Signature = sig
			return true
		}
	}
	return false
}

func (tx *SendTx) String() string {
	return Fmt("SendTx{%v/%v %v->%v}", tx.Fee, tx.Gas, tx.Inputs, tx.Outputs)
}

//-----------------------------------------------------------------------------

type AppTx struct {
	Fee   int64   `json:"fee"`   // Fee
	Gas   int64   `json:"gas"`   // Gas
	Type  byte    `json:"type"`  // Which app
	Input TxInput `json:"input"` // Hmmm do we want coins?
	Data  []byte  `json:"data"`
}

func (tx *AppTx) SignBytes(chainID string) []byte {
	signBytes := wire.BinaryBytes(chainID)
	sig := tx.Input.Signature
	tx.Input.Signature = nil
	signBytes = append(signBytes, wire.BinaryBytes(tx)...)
	tx.Input.Signature = sig
	return signBytes
}

func (tx *AppTx) SetSignature(sig crypto.Signature) bool {
	tx.Input.Signature = sig
	return true
}

func (tx *AppTx) String() string {
	return Fmt("AppTx{%v/%v %v %v %X}", tx.Fee, tx.Gas, tx.Type, tx.Input, tx.Data)
}

//-----------------------------------------------------------------------------

type CobaltTx struct {
	NormalizedTx *json.RawMessage `json:"cobaltNormalizedRepresentation"`
	CipherText   []string         `json:"cipherText"`
	Participants []string         `json:"participants"`
}

type SignedCobaltTx struct {
	Transaction CobaltTx `json:"transaction"`
	Signatures  []string `json:"signatures"`
}

func (tx *SignedCobaltTx) SignBytes(chainID string) []byte {
	txJSON, err := json.Marshal(tx.Transaction)
	if err != nil {
		panic(err)
	}
	return txJSON
}

func (tx *SignedCobaltTx) SetSignature(name string, sig crypto.Signature) bool {
	var idx = -1
	for i, participant := range tx.Transaction.Participants {
		if participant == name {
			idx = i
			break
		}
	}
	if idx == -1 {
		panic("Couldn't set signature")
	}
	if len(tx.Signatures) == 0 {
		tx.Signatures = make([]string, len(tx.Transaction.Participants))
	}
	tx.Signatures[idx] = Fmt("%X", sig.Bytes())
	return true
}

func (tx *SignedCobaltTx) String() string {
	return Fmt("SignedCobaltTx{%v %v}",
		tx.Transaction.Participants, tx.Signatures)
}

type BinarySignedCobaltTx struct {
	Bytes []byte
}

func (tx *BinarySignedCobaltTx) SignBytes(chainID string) []byte {
	return tx.Bytes
}

//-----------------------------------------------------------------------------

func TxID(chainID string, tx Tx) []byte {
	signBytes := tx.SignBytes(chainID)
	return wire.BinaryRipemd160(signBytes)
}

//--------------------------------------------------------------------------------

// Contract: This function is deterministic and completely reversible.
func jsonEscape(str string) string {
	escapedBytes, err := json.Marshal(str)
	if err != nil {
		PanicSanity(Fmt("Error json-escaping a string", str))
	}
	return string(escapedBytes)
}
