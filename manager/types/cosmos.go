package types

import (
	types2 "github.com/cometbft/cometbft/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"time"
)

type GenesisChunkedResponse struct {
	Chunk string `json:"chunk"`
	Total string `json:"total"`
	Data  []byte `json:"data"`
}

type KiraStatus struct {
	NodeInfo      NodeInfo      `json:"node_info,omitempty"`
	SyncInfo      SyncInfo      `json:"sync_info,omitempty"`
	ValidatorInfo ValidatorInfo `json:"validator_info,omitempty"`
}

type GenesisInfo struct {
	GenesisDoc  *types2.GenesisDoc
	GenesisData []byte
}

type Coin struct {
	Amount string `json:"amount"`
	Denom  string `json:"denom"`
}

type QueryBalancesResponse struct {
	Balances []Coin `json:"balances"`
}

type Delegation struct {
	ValidatorInfo struct {
		Moniker string `json:"moniker,omitempty"`
		Address string `json:"address,omitempty"`
		ValKey  string `json:"valkey,omitempty"`
		Website string `json:"website,omitempty"`
		Logo    string `json:"logo,omitempty"`
	} `json:"validator_info"`
	PoolInfo struct {
		ID         int64    `json:"id,omitempty"`
		Commission string   `json:"commission,omitempty"`
		Status     string   `json:"status,omitempty"`
		Tokens     []string `json:"tokens,omitempty"`
	} `json:"pool_info"`
}

type QueryDelegationsResult struct {
	Delegations []Delegation `json:"delegations"`
	Pagination  struct {
		Total int `json:"total,string,omitempty"`
	} `json:"pagination,omitempty"`
}

type VerifyRecord struct {
	Address            string   `json:"address,omitempty"`
	Id                 string   `json:"id,omitempty"`
	LastRecordEditDate string   `json:"lastRecordEditDate,omitempty"`
	RecordIds          []string `json:"recordIds,omitempty"`
	Tip                string   `json:"tip,omitempty"`
	Verifier           string   `json:"verifier,omitempty"`
}

type IdVerifyResponse struct {
	VerifyRecords []VerifyRecord `json:"verifyRecords"`
	Pagination    interface{}    `json:"pagination"`
}

type NetworkProperties struct {
	MinTxFee                     string `json:"minTxFee"`
	MaxTxFee                     string `json:"maxTxFee"`
	VoteQuorum                   string `json:"voteQuorum"`
	MinimumProposalEndTime       string `json:"minimumProposalEndTime"`
	ProposalEnactmentTime        string `json:"proposalEnactmentTime"`
	MinProposalEndBlocks         string `json:"minProposalEndBlocks"`
	MinProposalEnactmentBlocks   string `json:"minProposalEnactmentBlocks"`
	EnableForeignFeePayments     bool   `json:"enableForeignFeePayments"`
	MischanceRankDecreaseAmount  string `json:"mischanceRankDecreaseAmount"`
	MaxMischance                 string `json:"maxMischance"`
	MischanceConfidence          string `json:"mischanceConfidence"`
	InactiveRankDecreasePercent  string `json:"inactiveRankDecreasePercent"`
	MinValidators                string `json:"minValidators"`
	PoorNetworkMaxBankSend       string `json:"poorNetworkMaxBankSend"`
	UnjailMaxTime                string `json:"unjailMaxTime"`
	EnableTokenWhitelist         bool   `json:"enableTokenWhitelist"`
	EnableTokenBlacklist         bool   `json:"enableTokenBlacklist"`
	MinIdentityApprovalTip       string `json:"minIdentityApprovalTip"`
	UniqueIdentityKeys           string `json:"uniqueIdentityKeys"`
	UbiHardcap                   string `json:"ubiHardcap"`
	ValidatorsFeeShare           string `json:"validatorsFeeShare"`
	InflationRate                string `json:"inflationRate"`
	InflationPeriod              string `json:"inflationPeriod"`
	UnstakingPeriod              string `json:"unstakingPeriod"`
	MaxDelegators                string `json:"maxDelegators"`
	MinDelegationPushout         string `json:"minDelegationPushout"`
	SlashingPeriod               string `json:"slashingPeriod"`
	MaxJailedPercentage          string `json:"maxJailedPercentage"`
	MaxSlashingPercentage        string `json:"maxSlashingPercentage"`
	MinCustodyReward             string `json:"minCustodyReward"`
	MaxCustodyBufferSize         string `json:"maxCustodyBufferSize"`
	MaxCustodyTxSize             string `json:"maxCustodyTxSize"`
	AbstentionRankDecreaseAmount string `json:"abstentionRankDecreaseAmount"`
	MaxAbstention                string `json:"maxAbstention"`
	MinCollectiveBond            string `json:"minCollectiveBond"`
	MinCollectiveBondingTime     string `json:"minCollectiveBondingTime"`
	MaxCollectiveOutputs         string `json:"maxCollectiveOutputs"`
	MinCollectiveClaimPeriod     string `json:"minCollectiveClaimPeriod"`
	ValidatorRecoveryBond        string `json:"validatorRecoveryBond"`
	MaxAnnualInflation           string `json:"maxAnnualInflation"`
	MaxProposalTitleSize         string `json:"maxProposalTitleSize"`
	MaxProposalDescriptionSize   string `json:"maxProposalDescriptionSize"`
	MaxProposalPollOptionSize    string `json:"maxProposalPollOptionSize"`
	MaxProposalPollOptionCount   string `json:"maxProposalPollOptionCount"`
	MaxProposalReferenceSize     string `json:"maxProposalReferenceSize"`
	MaxProposalChecksumSize      string `json:"maxProposalChecksumSize"`
	MinDappBond                  string `json:"minDappBond"`
	MaxDappBond                  string `json:"maxDappBond"`
	DappBondDuration             string `json:"dappBondDuration"`
	DappVerifierBond             string `json:"dappVerifierBond"`
	DappAutoDenounceTime         string `json:"dappAutoDenounceTime"`
}

type NetworkPropertiesResponse struct {
	Properties *NetworkProperties `json:"properties"`
}

type QueryStakingPoolDelegatorsResponse struct {
	Pool       ValidatorPool `json:"pool"`
	Delegators []string      `json:"delegators,omitempty"`
}

type QueryValidatorPoolResult struct {
	ID              int64      `json:"id,omitempty"`
	Slashed         string     `json:"slashed"`
	Commission      string     `json:"commission"`
	TotalDelegators int64      `json:"total_delegators"`
	VotingPower     []sdk.Coin `json:"voting_power"`
	Tokens          []string   `json:"tokens"`
}

type Undelegation struct {
	ID         uint64   `json:"id,string"`
	Address    string   `json:"address"`
	ValAddress string   `json:"valaddress"`
	Expiry     string   `json:"expiry"`
	Amount     []string `json:"amount"`
}

type QueryUndelegationsResult struct {
	Undelegations []Undelegation `json:"undelegations"`
}

type BlocksResultResponse struct {
	Blocks     []map[string]interface{} `json:"blocks"`
	Pagination Pagination               `json:"pagination"`
}

type TxsResultResponse struct {
	Transactions []map[string]interface{} `json:"transactions"`
	Pagination   Pagination               `json:"pagination"`
}

type TransactionResultResponse struct {
	Time      int64         `json:"time"`
	Hash      string        `json:"hash"`
	Status    string        `json:"status"`
	Direction string        `json:"direction"`
	Memo      string        `json:"memo"`
	Fee       sdk.Coins     `json:"fee"`
	Txs       []interface{} `json:"txs"`
}

type QueryTxsParams struct {
	Hash       string   `json:"hash,omitempty"`
	Height     string   `json:"height,omitempty"`
	Address    string   `json:"address,omitempty"`
	StartDate  int64    `json:"start_date,string,omitempty"`
	EndDate    int64    `json:"end_date,string,omitempty"`
	Directions []string `json:"directions,omitempty"`
	Statuses   []string `json:"statuses,omitempty"`
	Types      []string `json:"types,omitempty"`
	Offset     int      `json:"offset,string,omitempty"`
	Limit      int      `json:"limit,string,omitempty"`
}

type TxResponse struct {
	Height    string `json:"height"`
	TxHash    string `json:"txhash"`
	Codespace string `json:"codespace"`
	Code      uint32 `json:"code"`
	Data      string `json:"data"`
	RawLog    string `json:"raw_log"`
	Logs      []struct {
		Events []struct {
			Type       string `json:"type"`
			Attributes []struct {
				Key   string `json:"key"`
				Value string `json:"value"`
			} `json:"attributes"`
		} `json:"events"`
	} `json:"logs"`
	Info      string    `json:"info"`
	GasWanted string    `json:"gas_wanted"`
	GasUsed   string    `json:"gas_used"`
	Tx        Tx        `json:"tx"`
	Timestamp time.Time `json:"timestamp"`
}

type MempoolResponse struct {
	Txs []TxResponse `json:"txs"`
}

type Tx struct {
	Body struct {
		Messages []interface{} `json:"messages"`
		Memo     string        `json:"memo"`
	} `json:"body"`
	AuthInfo struct {
		SignerInfos []struct {
			PublicKey struct {
				Type string `json:"@type"`
				Key  string `json:"key"`
			} `json:"public_key"`
			ModeInfo struct {
				Single struct {
					Mode string `json:"mode"`
				} `json:"single"`
			} `json:"mode_info"`
			Sequence string `json:"sequence"`
		} `json:"signer_infos"`
		Fee struct {
			Amount   sdk.Coins `json:"amount"`
			GasLimit string    `json:"gas_limit"`
			Payer    string    `json:"payer"`
			Granter  string    `json:"granter"`
		} `json:"fee"`
	} `json:"auth_info"`
	Signatures []string `json:"signatures"`
}

type ProposalsRequest struct {
	Proposer   string   `json:"proposer,omitempty"`
	DateStart  int      `json:"date_start,string,omitempty"`
	DateEnd    int      `json:"date_end,string,omitempty"`
	SortBy     string   `json:"sort_by,omitempty"`
	Types      []string `json:"types,omitempty"`
	Statuses   []string `json:"statuses,omitempty"`
	Voter      string   `json:"voter,omitempty"`
	Offset     int64    `json:"offset,string,omitempty"`
	Limit      int64    `json:"limit,string,omitempty"`
	CountTotal int64    `json:"count_total,string,omitempty"`
}

type Proposal struct {
	ProposalID                 string      `json:"proposalId"`
	Title                      string      `json:"title"`
	Description                string      `json:"description"`
	Content                    interface{} `json:"content"`
	SubmitTime                 string      `json:"submitTime"`
	VotingEndTime              string      `json:"votingEndTime"`
	EnactmentEndTime           string      `json:"enactmentEndTime"`
	MinVotingEndBlockHeight    string      `json:"minVotingEndBlockHeight"`
	MinEnactmentEndBlockHeight string      `json:"minEnactmentEndBlockHeight"`
	ExecResult                 string      `json:"execResult"`
	Result                     string      `json:"result"`
	VotersCount                int         `json:"voters_count"`
	VotesCount                 int         `json:"votes_count"`
	Quorum                     string      `json:"quorum"`
	Metadata                   string      `json:"meta_data"`
	Hash                       string      `json:"transaction_hash,omitempty"`
	Timestamp                  int         `json:"timestamp,omitempty"`
	BlockHeight                int         `json:"block_height,omitempty"`
	Type                       string      `json:"type,omitempty"`
	Proposer                   string      `json:"proposer,omitempty"`
}

type Pagination struct {
	NextKey string `json:"next_key"`
	Total   int    `json:"total,string"`
}

type ProposalsResponse = struct {
	Proposals  []Proposal `json:"proposals"`
	Pagination Pagination `json:"pagination"`
}

type TokenAlias struct {
	Denom             string `json:"denom"`
	TokenType         string `json:"tokenType"`
	FeeRate           string `json:"feeRate"`
	FeeEnabled        bool   `json:"feeEnabled"`
	Supply            string `json:"supply"`
	SupplyCap         string `json:"supplyCap"`
	StakeCap          string `json:"stakeCap"`
	StakeMin          string `json:"stakeMin"`
	StakeEnabled      bool   `json:"stakeEnabled"`
	Inactive          bool   `json:"inactive"`
	Symbol            string `json:"symbol"`
	Name              string `json:"name"`
	Icon              string `json:"icon"`
	Decimals          int    `json:"decimals"`
	Description       string `json:"description"`
	Website           string `json:"website"`
	Social            string `json:"social"`
	Holders           string `json:"holders"`
	MintingFee        string `json:"mintingFee"`
	Owner             string `json:"owner"`
	OwnerEditDisabled bool   `json:"ownerEditDisabled"`
	NftMetadata       string `json:"nftMetadata"`
	NftHash           string `json:"nftHash"`
}

type TokenAliasesData struct {
	Data TokenAlias `json:"data"`
}

type TokenAliasesRequest struct {
	Tokens []string `json:"tokens"`
}

type TokenAliasesGRPCResponse struct {
	Data []TokenAliasesData `json:"data"`
}

type TokenAliasesResponse struct {
	Data         []TokenAlias `json:"data"`
	DefaultDenom string       `json:"default_denom"`
	Bech32Prefix string       `json:"bech32_prefix"`
	Pagination   *Pagination  `json:"pagination,omitempty"`
}

type TokenSupply struct {
	Amount sdk.Int `json:"amount"`
	Denom  string  `json:"denom"`
}

type TokenSupplyResponse struct {
	Supply []TokenSupply `json:"supply"`
}

type CustomPrefixesResponse struct {
	DefaultDenom string `json:"defaultDenom"`
	Bech32Prefix string `json:"bech32Prefix"`
}

type FaucetAccountInfo struct {
	Address  string     `json:"address"`
	Balances []sdk.Coin `json:"balances"`
}

type BalancesResponse struct {
	Balances []sdk.Coin `json:"balances"`
}

type FaucetRequest struct {
	Claim string `json:"claim,omitempty"`
	Token string `json:"token,omitempty"`
}

type CosmosConfig struct {
	Node struct {
		JsonRpc    string `json:"json_rpc"`
		Tendermint string `json:"tendermint"`
	}
	TxModes map[string]bool `json:"tx_modes"`
	Faucet  struct {
		FaucetAmounts        map[string]int64 `json:"faucet_amounts"`
		FaucetMinimumAmounts map[string]int64 `json:"faucet_minimum_amounts"`
		FeeAmounts           map[string]int64 `json:"fee_amounts"`
		TimeLimit            int64            `json:"time_limit,float64"`
	}
	GWTimeout   int    `json:"gw_timeout,float64"`
	Interaction string `json:"interaction"`
	Token       string `json:"token"`
	Retries     int    `json:"retries,float64"`
	RetryDelay  int    `json:"retry_delay,float64"`
	RateLimit   int    `json:"rate_limit,float64"`
}

type AccountResponse struct {
	Account struct {
		Type          string      `json:"@type"`
		Address       string      `json:"address"`
		PubKey        interface{} `json:"pubKey"`
		AccountNumber string      `json:"accountNumber"`
		Sequence      string      `json:"sequence"`
	} `json:"account"`
}
