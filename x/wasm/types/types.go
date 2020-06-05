package types

import (
	tmBytes "github.com/tendermint/tendermint/libs/bytes"

	wasmTypes "github.com/confio/go-cosmwasm/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	auth "github.com/cosmos/cosmos-sdk/x/auth/exported"
	"github.com/cosmos/cosmos-sdk/x/bank"
)

const defaultLRUCacheSize = uint64(0)
const defaultQueryGasLimit = uint64(3000000)

// Model is a struct that holds a KV pair
type Model struct {
	// hex-encode key to read it better (this is often ascii)
	Key tmBytes.HexBytes `json:"key"`
	// base64-encode raw value
	Value []byte `json:"val"`
}

// NewCodeInfo fills a new Contract struct
func NewCodeInfo(codeHash []byte, creator sdk.AccAddress, source string, builder string) *CodeInfo {
	return &CodeInfo{
		CodeHash: codeHash,
		Creator:  creator,
		Source:   source,
		Builder:  builder,
	}
}

// LessThan can be used to sort
func (a *CreatedAt) LessThan(b *CreatedAt) bool {
	if a == nil {
		return true
	}
	if b == nil {
		return false
	}
	return a.BlockHeight < b.BlockHeight || (a.BlockHeight == b.BlockHeight && a.TxIndex < b.TxIndex)
}

// NewCreatedAt gets a timestamp from the context
func NewCreatedAt(ctx sdk.Context) *CreatedAt {
	// we must safely handle nil gas meters
	var index uint64
	meter := ctx.BlockGasMeter()
	if meter != nil {
		index = meter.GasConsumed()
	}
	return &CreatedAt{
		BlockHeight: ctx.BlockHeight(),
		TxIndex:     index,
	}
}

// NewContractInfo creates a new instance of a given WASM contract info
func NewContractInfo(codeID uint64, creator sdk.AccAddress, initMsg []byte, label string, createdAt *CreatedAt) *ContractInfo {
	return &ContractInfo{
		CodeId:  codeID,
		Creator: creator,
		InitMsg: initMsg,
		Label:   label,
		Created: createdAt,
	}
}

// NewParams initializes params for a contract instance
func NewParams(ctx sdk.Context, creator sdk.AccAddress, deposit sdk.Coins, contractAcct auth.Account, k bank.ViewKeeper) wasmTypes.Env {
	contractAddr := contractAcct.GetAddress()
	return wasmTypes.Env{
		Block: wasmTypes.BlockInfo{
			Height:  ctx.BlockHeight(),
			Time:    ctx.BlockTime().Unix(),
			ChainID: ctx.ChainID(),
		},
		Message: wasmTypes.MessageInfo{
			Signer:    wasmTypes.CanonicalAddress(creator),
			SentFunds: NewWasmCoins(deposit),
		},
		Contract: wasmTypes.ContractInfo{
			Address: wasmTypes.CanonicalAddress(contractAddr),
			Balance: NewWasmCoins(k.GetAllBalances(ctx, contractAddr)),
		},
	}
}

// NewWasmCoins translates between Cosmos SDK coins and Wasm coins
func NewWasmCoins(cosmosCoins sdk.Coins) (wasmCoins []wasmTypes.Coin) {
	for _, coin := range cosmosCoins {
		wasmCoin := wasmTypes.Coin{
			Denom:  coin.Denom,
			Amount: coin.Amount.String(),
		}
		wasmCoins = append(wasmCoins, wasmCoin)
	}
	return wasmCoins
}

const CustomEventType = "wasm"
const AttributeKeyContractAddr = "contract_address"

type WasmResult struct {
	Data   []byte
	Events sdk.Events
}

// WasmResult converts from a Wasm Result type
func CosmosResult(wasmResult wasmTypes.Result, contractAddr sdk.AccAddress) WasmResult {
	var events sdk.Events
	if len(wasmResult.Log) > 0 {
		// we always tag with the contract address issuing this event
		attrs := []sdk.Attribute{sdk.NewAttribute(AttributeKeyContractAddr, contractAddr.String())}
		for _, l := range wasmResult.Log {
			// and reserve the contract_address key for our use (not contract)
			if l.Key != AttributeKeyContractAddr {
				attr := sdk.NewAttribute(l.Key, l.Value)
				attrs = append(attrs, attr)
			}
		}
		events = sdk.Events{sdk.NewEvent(CustomEventType, attrs...)}
	}
	return WasmResult{
		Data:   []byte(wasmResult.Data),
		Events: events,
	}
}

// WasmConfig is the extra config required for wasm
type WasmConfig struct {
	SmartQueryGasLimit uint64 `mapstructure:"query_gas_limit"`
	CacheSize          uint64 `mapstructure:"lru_size"`
}

// DefaultWasmConfig returns the default settings for WasmConfig
func DefaultWasmConfig() WasmConfig {
	return WasmConfig{
		SmartQueryGasLimit: defaultQueryGasLimit,
		CacheSize:          defaultLRUCacheSize,
	}
}