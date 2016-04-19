package main

import (
	//"github.com/tendermint/basecoin/types"
	eyes "github.com/tendermint/merkleeyes/client"
)

type CobaltPlugin struct {
	eyesCli *eyes.Client
}

func NewCobaltPlugin(eyesCli *eyes.Client) *CobaltPlugin {
	return &CobaltPlugin{
		eyesCli: eyesCli,
	}
}

func (cp *CobaltPlugin) SetOption(key string, value string) (log string) {
	// When key="admin", set admin pubKey.
	// This pubKey is used to
	return ""
}

/*
// Value is any floating value.  It must be given to someone.
type Plugin interface {
	SetOption(key string, value string) (log string)
	RunTx(ctx CallContext, txBytes []byte) (res tmsp.Result)
	Query(query []byte) (res tmsp.Result)
	Commit() (res tmsp.Result)
}
*/
