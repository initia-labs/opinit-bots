package host

import (
	"slices"
	"testing"

	abcitypes "github.com/cometbft/cometbft/abci/types"
	ophosttypes "github.com/initia-labs/OPinit/x/ophost/types"
	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func TestParseMsgRecordBatch(t *testing.T) {
	fullAttributes := RecordBatchEvents("submitter")
	cases := []struct {
		name       string
		eventAttrs []abcitypes.EventAttribute
		submitter  string
		err        bool
	}{
		{
			name:       "success",
			eventAttrs: fullAttributes,
			submitter:  "submitter",
			err:        false,
		},
		{
			name:       "missing event attribute submitter",
			eventAttrs: fullAttributes[1:],
			err:        true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			submitter, err := ParseMsgRecordBatch(tc.eventAttrs)
			if tc.err {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.submitter, submitter)
			}
		})
	}
}

func TestParseUpdateBatchInfo(t *testing.T) {
	fullAttributes := UpdateBatchInfoEvents(1, ophosttypes.BatchInfo_INITIA, "submitter", 1, 1)
	cases := []struct {
		name          string
		eventAttrs    []abcitypes.EventAttribute
		bridgeId      uint64
		submitter     string
		chain         string
		outputIndex   uint64
		l2BlockNumber int64
		err           bool
	}{
		{
			name:          "success",
			eventAttrs:    fullAttributes,
			bridgeId:      1,
			submitter:     "submitter",
			chain:         "INITIA",
			outputIndex:   1,
			l2BlockNumber: 1,
			err:           false,
		},
		{
			name:       "missing event attribute bridge id",
			eventAttrs: fullAttributes[1:],
			err:        true,
		},
		{
			name:       "missing event attribute batch chain type",
			eventAttrs: append(slices.Clone(fullAttributes)[:1], fullAttributes[2:]...),
			err:        true,
		},
		{
			name:       "missing event attribute submitter",
			eventAttrs: append(slices.Clone(fullAttributes)[:2], fullAttributes[3:]...),
			err:        true,
		},
		{
			name:       "missing event attribute output index",
			eventAttrs: append(slices.Clone(fullAttributes)[:3], fullAttributes[4:]...),
			err:        true,
		},
		{
			name:       "missing event attribute l2 block number",
			eventAttrs: fullAttributes[:4],
			err:        true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			bridgeId, submitter, chain, outputIndex, l2BlockNumber, err := ParseMsgUpdateBatchInfo(tc.eventAttrs)
			if tc.err {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.bridgeId, bridgeId)
				require.Equal(t, tc.submitter, submitter)
				require.Equal(t, tc.chain, chain)
				require.Equal(t, tc.outputIndex, outputIndex)
				require.Equal(t, tc.l2BlockNumber, l2BlockNumber)
			}
		})
	}
}

func TestParseInitiateTokenDeposit(t *testing.T) {
	fullAttributes := InitiateTokenDepositEvents(1, "init1hrasklz3tr6s9rls4r8fjuf0k4zuha6w9rude5", "init1z3689ct7pc72yr5an97nsj89dnlefydxwdhcv0", sdk.NewInt64Coin("l1Denom", 100), []byte("databytes"), 1, "l2denom")
	cases := []struct {
		name       string
		eventAttrs []abcitypes.EventAttribute
		bridgeId   uint64
		sender     string
		to         string
		amount     sdk.Coin
		data       []byte
		l1Sequence uint64
		l2Denom    string
		err        bool
	}{
		{
			name:       "success",
			eventAttrs: fullAttributes,
			bridgeId:   1,
			sender:     "init1hrasklz3tr6s9rls4r8fjuf0k4zuha6w9rude5",
			to:         "init1z3689ct7pc72yr5an97nsj89dnlefydxwdhcv0",
			amount:     sdk.NewInt64Coin("l1Denom", 100),
			data:       []byte("databytes"),
			l1Sequence: 1,
			l2Denom:    "l2denom",
			err:        false,
		},
		{
			name:       "missing event attribute bridge id",
			eventAttrs: fullAttributes[1:],
			err:        true,
		},
		{
			name:       "missing event attribute l1 sequence",
			eventAttrs: append(slices.Clone(fullAttributes)[:1], fullAttributes[2:]...),
			err:        true,
		},
		{
			name:       "missing event attribute from",
			eventAttrs: append(slices.Clone(fullAttributes)[:2], fullAttributes[3:]...),
			err:        true,
		},
		{
			name:       "missing event attribute to",
			eventAttrs: append(slices.Clone(fullAttributes)[:3], fullAttributes[4:]...),
			err:        true,
		},
		{
			name:       "missing event attribute l1 denom",
			eventAttrs: append(slices.Clone(fullAttributes)[:4], fullAttributes[5:]...),
			err:        true,
		},
		{
			name:       "missing event attribute l2 denom",
			eventAttrs: append(slices.Clone(fullAttributes)[:5], fullAttributes[6:]...),
			err:        true,
		},
		{
			name:       "missing event attribute amount",
			eventAttrs: append(slices.Clone(fullAttributes)[:6], fullAttributes[7:]...),
			err:        true,
		},
		{
			name:       "missing event attribute data",
			eventAttrs: fullAttributes[8:],
			err:        true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			bridgeId, l1Sequence, from, to, l1Denom, l2Denom, amount, data, err := ParseMsgInitiateDeposit(tc.eventAttrs)
			if tc.err {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.bridgeId, bridgeId)
				require.Equal(t, tc.l1Sequence, l1Sequence)
				require.Equal(t, tc.sender, from)
				require.Equal(t, tc.to, to)
				require.Equal(t, tc.amount.Denom, l1Denom)
				require.Equal(t, tc.l2Denom, l2Denom)
				require.Equal(t, tc.amount.Amount.String(), amount)
				require.Equal(t, tc.data, data)
			}
		})
	}
}

func TestParseMsgProposeOutput(t *testing.T) {
	fullAttributes := ProposeOutputEvents("proposer", 1, 2, 3, []byte("output_root"))

	cases := []struct {
		name          string
		eventAttrs    []abcitypes.EventAttribute
		proposer      string
		bridgeId      uint64
		outputIndex   uint64
		l2BlockNumber int64
		outputRoot    []byte
		err           bool
	}{
		{
			name:          "success",
			eventAttrs:    fullAttributes,
			proposer:      "proposer",
			bridgeId:      1,
			outputIndex:   2,
			l2BlockNumber: 3,
			outputRoot:    []byte("output_root"),
			err:           false,
		},
		{
			name:       "missing event attribute proposer",
			eventAttrs: fullAttributes[1:],
			err:        true,
		},
		{
			name:       "missing event attribute bridge id",
			eventAttrs: append(slices.Clone(fullAttributes)[:1], fullAttributes[2:]...),
			err:        true,
		},
		{
			name:       "missing event attribute output index",
			eventAttrs: append(slices.Clone(fullAttributes)[:2], fullAttributes[3:]...),
			err:        true,
		},
		{
			name:       "missing event attribute l2 block number",
			eventAttrs: append(slices.Clone(fullAttributes)[:3], fullAttributes[4:]...),
			err:        true,
		},
		{
			name:       "missing event attribute output root",
			eventAttrs: fullAttributes[:4],
			err:        true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			bridgeId, l2BlockNumber, outputIndex, proposer, outputRoot, err := ParseMsgProposeOutput(tc.eventAttrs)
			if tc.err {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.bridgeId, bridgeId)
				require.Equal(t, tc.l2BlockNumber, l2BlockNumber)
				require.Equal(t, tc.outputIndex, outputIndex)
				require.Equal(t, tc.proposer, proposer)
				require.Equal(t, tc.outputRoot, outputRoot)
			}
		})
	}
}

func TestParseMsgFinalizeWithdrawal(t *testing.T) {
	fullAttributes := FinalizeWithdrawalEvents(1, 2, 3, "from", "to", "l1Denom", "l2Denom", sdk.NewInt64Coin("uinit", 10000))

	cases := []struct {
		name        string
		eventAttrs  []abcitypes.EventAttribute
		bridgeId    uint64
		outputIndex uint64
		l2Sequence  uint64
		from        string
		to          string
		l1Denom     string
		l2Denom     string
		amount      string
		err         bool
	}{
		{
			name:        "success",
			eventAttrs:  fullAttributes,
			bridgeId:    1,
			outputIndex: 2,
			l2Sequence:  3,
			from:        "from",
			to:          "to",
			l1Denom:     "l1Denom",
			l2Denom:     "l2Denom",
			amount:      "10000uinit",
			err:         false,
		},
		{
			name:       "missing event attribute bridge id",
			eventAttrs: fullAttributes[1:],
			err:        true,
		},
		{
			name:       "missing event attribute output index",
			eventAttrs: append(slices.Clone(fullAttributes)[:1], fullAttributes[2:]...),
			err:        true,
		},
		{
			name:       "missing event attribute l2 sequence",
			eventAttrs: append(slices.Clone(fullAttributes)[:2], fullAttributes[3:]...),
			err:        true,
		},
		{
			name:       "missing event attribute from",
			eventAttrs: append(slices.Clone(fullAttributes)[:3], fullAttributes[4:]...),
			err:        true,
		},
		{
			name:       "missing event attribute to",
			eventAttrs: append(slices.Clone(fullAttributes)[:4], fullAttributes[5:]...),
			err:        true,
		},
		{
			name:       "missing event attribute l1 denom",
			eventAttrs: append(slices.Clone(fullAttributes)[:5], fullAttributes[6:]...),
			err:        true,
		},
		{
			name:       "missing event attribute l2 denom",
			eventAttrs: append(slices.Clone(fullAttributes)[:6], fullAttributes[7:]...),
			err:        true,
		},
		{
			name:       "missing event attribute amount",
			eventAttrs: fullAttributes[:7],
			err:        true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			bridgeId, outputIndex, l2Sequence, from, to, l1Denom, l2Denom, amount, err := ParseMsgFinalizeWithdrawal(tc.eventAttrs)
			if tc.err {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.bridgeId, bridgeId)
				require.Equal(t, tc.outputIndex, outputIndex)
				require.Equal(t, tc.l2Sequence, l2Sequence)
				require.Equal(t, tc.from, from)
				require.Equal(t, tc.to, to)
				require.Equal(t, tc.l1Denom, l1Denom)
				require.Equal(t, tc.l2Denom, l2Denom)
				require.Equal(t, tc.amount, amount)
			}
		})
	}
}

func TestParseMsgUpdateOracleConfig(t *testing.T) {
	fullAttributes := UpdateOracleConfigEvents(1, true)

	cases := []struct {
		name          string
		eventAttrs    []abcitypes.EventAttribute
		bridgeId      uint64
		oracleEnabled bool
		err           bool
	}{
		{
			name:          "success",
			eventAttrs:    fullAttributes,
			bridgeId:      1,
			oracleEnabled: true,
			err:           false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			bridgeId, oracleEnabled, err := ParseMsgUpdateOracleConfig(tc.eventAttrs)
			if tc.err {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.bridgeId, bridgeId)
				require.Equal(t, tc.oracleEnabled, oracleEnabled)
			}
		})
	}
}
