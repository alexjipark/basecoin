package main

import (
	"encoding/json"
	"fmt"

	"github.com/tendermint/basecoin/app"
	"github.com/tendermint/basecoin/tests"
	"github.com/tendermint/basecoin/types"
	. "github.com/tendermint/go-common"
	"github.com/tendermint/go-wire"
	eyescli "github.com/tendermint/merkleeyes/client"
)

func main() {
	//testSendTx()
	//testSequence()
	testCobalt()
}

func testSendTx() {
	eyesCli := eyescli.NewLocalClient()
	chainID := "test_chain_id"
	bcApp := app.NewBasecoin(eyesCli)
	bcApp.SetOption("base/chainID", chainID)
	fmt.Println(bcApp.Info())

	test1PrivAcc := tests.PrivAccountFromSecret("test1")
	test2PrivAcc := tests.PrivAccountFromSecret("test2")

	// Seed Basecoin with account
	test1Acc := test1PrivAcc.Account
	test1Acc.Balance = types.Coins{{"", 1000}}
	fmt.Println(bcApp.SetOption("base/account", string(wire.JSONBytes(test1Acc))))

	// Construct a SendTx signature
	tx := &types.SendTx{
		Fee: 0,
		Gas: 0,
		Inputs: []types.TxInput{
			types.TxInput{
				Address:  test1PrivAcc.Account.PubKey.Address(),
				PubKey:   test1PrivAcc.Account.PubKey, // TODO is this needed?
				Coins:    types.Coins{{"", 1}},
				Sequence: 1,
			},
		},
		Outputs: []types.TxOutput{
			types.TxOutput{
				Address: test2PrivAcc.Account.PubKey.Address(),
				Coins:   types.Coins{{"", 1}},
			},
		},
	}

	// Sign request
	signBytes := tx.SignBytes(chainID)
	fmt.Printf("Sign bytes: %X\n", signBytes)
	sig := test1PrivAcc.PrivKey.Sign(signBytes)
	tx.Inputs[0].Signature = sig
	//fmt.Println("tx:", tx)
	fmt.Printf("Signed TX bytes: %X\n", wire.BinaryBytes(struct{ types.Tx }{tx}))

	// Write request
	txBytes := wire.BinaryBytes(struct{ types.Tx }{tx})
	res := bcApp.AppendTx(txBytes)
	fmt.Println(res)
	if res.IsErr() {
		Exit(Fmt("Failed: %v", res.Error()))
	}
}

func testSequence() {
	eyesCli := eyescli.NewLocalClient()
	chainID := "test_chain_id"
	bcApp := app.NewBasecoin(eyesCli)
	bcApp.SetOption("base/chainID", chainID)
	fmt.Println(bcApp.Info())

	// Get the test account
	test1PrivAcc := tests.PrivAccountFromSecret("test1")
	test1Acc := test1PrivAcc.Account
	test1Acc.Balance = types.Coins{{"", 1 << 53}}
	fmt.Println(bcApp.SetOption("base/account", string(wire.JSONBytes(test1Acc))))

	sequence := int(1)
	// Make a bunch of PrivAccounts
	privAccounts := tests.RandAccounts(1000, 1000000, 0)
	privAccountSequences := make(map[string]int)

	// Send coins to each account
	for i := 0; i < len(privAccounts); i++ {
		privAccount := privAccounts[i]
		tx := &types.SendTx{
			Fee: 2,
			Gas: 2,
			Inputs: []types.TxInput{
				types.TxInput{
					Address:  test1Acc.PubKey.Address(),
					PubKey:   test1Acc.PubKey, // TODO is this needed?
					Coins:    types.Coins{{"", 1000002}},
					Sequence: sequence,
				},
			},
			Outputs: []types.TxOutput{
				types.TxOutput{
					Address: privAccount.Account.PubKey.Address(),
					Coins:   types.Coins{{"", 1000000}},
				},
			},
		}
		sequence += 1

		// Sign request
		signBytes := tx.SignBytes(chainID)
		sig := test1PrivAcc.PrivKey.Sign(signBytes)
		tx.Inputs[0].Signature = sig
		// fmt.Printf("ADDR: %X -> %X\n", tx.Inputs[0].Address, tx.Outputs[0].Address)

		// Write request
		txBytes := wire.BinaryBytes(struct{ types.Tx }{tx})
		res := bcApp.AppendTx(txBytes)
		if res.IsErr() {
			Exit("AppendTx error: " + res.Error())
		}
	}

	fmt.Println("-------------------- RANDOM SENDS --------------------")

	// Now send coins between these accounts
	for i := 0; i < 10000; i++ {
		randA := RandInt() % len(privAccounts)
		randB := RandInt() % len(privAccounts)
		if randA == randB {
			continue
		}

		privAccountA := privAccounts[randA]
		privAccountASequence := privAccountSequences[privAccountA.Account.PubKey.KeyString()]
		privAccountSequences[privAccountA.Account.PubKey.KeyString()] = privAccountASequence + 1
		privAccountB := privAccounts[randB]

		tx := &types.SendTx{
			Fee: 2,
			Gas: 2,
			Inputs: []types.TxInput{
				types.TxInput{
					Address:  privAccountA.Account.PubKey.Address(),
					PubKey:   privAccountA.Account.PubKey,
					Coins:    types.Coins{{"", 3}},
					Sequence: privAccountASequence + 1,
				},
			},
			Outputs: []types.TxOutput{
				types.TxOutput{
					Address: privAccountB.Account.PubKey.Address(),
					Coins:   types.Coins{{"", 1}},
				},
			},
		}

		// Sign request
		signBytes := tx.SignBytes(chainID)
		sig := privAccountA.PrivKey.Sign(signBytes)
		tx.Inputs[0].Signature = sig
		// fmt.Printf("ADDR: %X -> %X\n", tx.Inputs[0].Address, tx.Outputs[0].Address)

		// Write request
		txBytes := wire.BinaryBytes(struct{ types.Tx }{tx})
		res := bcApp.AppendTx(txBytes)
		if res.IsErr() {
			Exit("AppendTx error: " + res.Error())
		}
	}
}

func testCobalt() {
	eyesCli := eyescli.NewLocalClient()
	chainID := "test_chain_id"
	bcApp := app.NewBasecoin(eyesCli)
	bcApp.SetOption("base/chainID", chainID)
	fmt.Println(bcApp.Info())

	// Set Citibank
	citiPrivAcc := tests.PrivAccountFromSecret("citi")
	citiAcc := citiPrivAcc.Account
	citiAccount := types.Account{
		Name:     "Citibank",
		PubKey:   citiAcc.PubKey,
		Sequence: 0,
		Balance:  types.Coins{{"", 1 << 53}},
	}
	log := bcApp.SetOption("base/account", string(wire.JSONBytes(citiAccount)))
	if log != "Success" {
		Exit(Fmt("Failed to set option: %v", log))
	}
	fmt.Printf("Citibank pubkey:  %X\n", citiAcc.PubKey.Bytes())
	fmt.Printf("Citibank privkey: %X\n", citiPrivAcc.PrivKey.Bytes())

	// Set Deutsche Bank
	deutschePrivAcc := tests.PrivAccountFromSecret("deutsche")
	deutscheAcc := deutschePrivAcc.Account
	deutscheAccount := types.Account{
		Name:     "Deutsche Bank",
		PubKey:   deutscheAcc.PubKey,
		Sequence: 0,
		Balance:  types.Coins{{"", 1 << 53}},
	}
	log = bcApp.SetOption("base/account", string(wire.JSONBytes(deutscheAccount)))
	if log != "Success" {
		Exit(Fmt("Failed to set option: %v", log))
	}
	fmt.Printf("Deutsche Bank pubkey:  %X\n", deutscheAcc.PubKey.Bytes())
	fmt.Printf("Deutsche Bank privkey: %X\n", deutschePrivAcc.PrivKey.Bytes())

	normalizedTx := []byte(`{
	"entityA": "Citibank",
	"entityB": "Deutsche Bank",
	"product": "FX Spot",
	"currencyPair": "EURUSD",
	"quantity1": 1000000,
	"currency1": "EUR",
	"currency2": "USD",
	"rate": "1.0965",
	"tradeDate": "9 March 2016",
	"valueDate": "11 March 2016",
	"venueTradeExecutionTime": "10 Mar 2016 09:34:04.2323",
	"id": "d64cbbeb-31fe-446a-8017-397696cdf2d0"
}`)

	// Mutate the validator set.
	normalizedTxRaw := json.RawMessage(normalizedTx)
	tx := &types.SignedCobaltTx{
		Transaction: types.CobaltTx{
			NormalizedTx: &normalizedTxRaw,
			CipherText: []string{
				"04d69f6fcd5521065c6b05e044e9d566cc1ca30450e5dce6f50e3fd11ebc223b9c1733431c1b64257b808c08ab9877ff6a1d5afc9ff353afd1493992aef1665d277488156f8588a738a6d18d0afd192ba6060cc3d503439367788d0a5f7de904a792559438714dfc1a0a3b297aadf96b9362d3d9c5f24b55c985c3000bc1f53d742ce6eaadb9a9b1b5d988cd51c5e514f1d34ab8f5756348c185afe675b689b447bbf9688e32c2d27897b9d818ca2de9456343c19ea842cf578131df068593e289eef61536547aad49cee79706a9f3c25f4d6415d28263cd79b92201b99e1230b83f085ca99d12aa8510e71f3cfa5ed0a7e16a0d36d50abe8caa41f142b2f17859503dc572b56b436bdb9a298478a382700489a3d86355e01ed011f846c0ca3a59b6570f155d16efc15458dc4910d7f3d000328e813fa8a72d4ca0e3d7344ce6298e93726792d26b36c5979abbac56544b7ef5827c658effe8f781ac22aab0ea4b658e55eb446794f5bfcad47026f190fa4b592ec0585ab7e733b8060595445ee143af4b362360f6a5e6fee0a2aa2d86ca2441fa82864885fa88f23797b41bd8015c6d00de9dd9856317ce27baa438e9952c5b2b76504fe8c40d68bf046cd6161afb66beeb0c806166472fd05b250f2fe96d10ad887d38b699cc74310aeca93ed97cee87fbdf9f1bc981dd7c54a6200a287c0805f34e309273e0a357ad69872a7c07074592b49f03852a38d57cdacbb44967e90aad990976286476090faec2eb58e894eba2405cbf2c808cb1376149ea96d2296bbd53fa7252d98b7d4a65321f8f205aa370959012e6a1177ca9bcf496c96f",
				"0448d2f1f8ef80ffb0699bcd63494598c7a7f7e0e6c4b54263a135151d746275ca32e8685356ba3ef708e8b61d9cfb7e9bd224e2fccb107160b43fa4877685a69985e7a0714e0ae4115493b5bd02601a3e37d9722b4cc9d66cb5ce484aa4f22362467244cc58568e83efecb8fce57848844dd08057317e1f47cdabde6afbd87c0e9dbaa20587d7c93e80c13fff8c2cc65336fc0cf7a8bcff47d02c844a61731c1c0a76ec9326686eedccf479e1250f3c057620a5104696e487dabe9c1880cb32a2ac6f51395bb19f63f3ee5f75f1f6ab4cd796dcb2727d12e1412857eaa76b66bd55b8ace581bda17fd5add2f3240c58f4b4eb0247b4582f9753dd6063d84cc761cf84d09eae332e61e71c846ef7cd57f37026f230e7ad8a8868a6e6b8d7dcb8f194975ed43f481012951f40f66b3158495aefe4f9b5a102f3346e28fd9fa6b4906cfafe6711e8b68976b1a7283d771d26b5b216576be9db344324607624e0796b1c4969be833385ddd1a8f74fc638bd1827ab3cf9cf6a00a3091a763e96fa3700ef55a673721ece6e8a6375e906d60c8857ecc8e588701f68d7d0f079a97d300288d8d97deb30abcb8b87f189f28ce3693c23927425cd270e8832f8cddd153a2b68f7207e3ccef7c11ed693ef059c3c202bae42db876044a3d7d7084a9d83bafc1eb20a02511dbf0384334c3ee15d1d9dabce1d5c6f751a17b9d4beacccda262b40db5b1827abd322668b8fc4dab9d5f2aafff02f72d25c6e3452a8bcded5e41e29edcdc7a59067bb9c17522144b90634b0efaa297f8365ab341af3f22c4960cd45c06d54d1cf5bd9b14000ef9253f18a80b386c16e4f8a",
			},
			Participants: []string{
				"Citibank",
				"Deutsche Bank",
			},
		},
		Signatures: nil,
	}
	fmt.Printf("Tx sign string: %v\n", string(tx.SignBytes(chainID)))
	tx.SetSignature("Citibank", citiPrivAcc.Sign(tx.SignBytes(chainID))) // XXX What is the right sign bytes?
	tx.SetSignature("Deutsche Bank", deutschePrivAcc.Sign(tx.SignBytes(chainID)))
	txJSON, err := json.Marshal(tx)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Fully signed tx: %v\n", string(txJSON))
	btx := &types.BinarySignedCobaltTx{txJSON}
	res := bcApp.AppendTx(wire.BinaryBytes(struct{ types.Tx }{btx}))
	if res.IsErr() {
		Exit(Fmt("Failed to run Cobalt tx: %v", res.Error()))
	}
	fmt.Println(res)

	// TODO more tests...
}
