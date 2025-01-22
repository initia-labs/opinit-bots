package child

import (
	"slices"
	"testing"

	abcitypes "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

func TestParseDepositEvents(t *testing.T) {
	fullAttributes := FinalizeDepositEvents(1, "sender", "recipient", "uinit", "baseDenom", sdk.NewInt64Coin("uinit", 10000), 2)
	cases := []struct {
		name           string
		eventAttrs     []abcitypes.EventAttribute
		l1Sequence     uint64
		from           string
		to             string
		baseDenom      string
		amount         sdk.Coin
		finalizeHeight int64
		err            bool
	}{
		{
			name:           "success",
			eventAttrs:     fullAttributes,
			l1Sequence:     1,
			from:           "sender",
			to:             "recipient",
			baseDenom:      "baseDenom",
			amount:         sdk.NewInt64Coin("uinit", 10000),
			finalizeHeight: 2,
			err:            false,
		},
		{
			name:       "missing event attribute l1 sequence",
			eventAttrs: fullAttributes[1:],
			err:        true,
		},
		{
			name:       "missing event attribute sender",
			eventAttrs: append(slices.Clone(fullAttributes[:1]), fullAttributes[2:]...),
			err:        true,
		},
		{
			name:       "missing event attribute recipient",
			eventAttrs: append(slices.Clone(fullAttributes[:2]), fullAttributes[3:]...),
			err:        true,
		},
		{
			name:       "missing event attribute l1 denom",
			eventAttrs: append(slices.Clone(fullAttributes[:3]), fullAttributes[4:]...),
			err:        true,
		},
		{
			name:       "missing event attribute l2 denom",
			eventAttrs: append(slices.Clone(fullAttributes[:4]), fullAttributes[5:]...),
			err:        true,
		},
		{
			name:       "missing event attribute amount",
			eventAttrs: append(slices.Clone(fullAttributes[:5]), fullAttributes[6:]...),
			err:        true,
		},
		{
			name:       "missing event attribute finalize height",
			eventAttrs: fullAttributes[:6],
			err:        true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			l1BlockHeight, l1Sequence, from, to, baseDenom, amount, err := ParseFinalizeDeposit(tc.eventAttrs)
			if tc.err {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.l1Sequence, l1Sequence)
				require.Equal(t, tc.from, from)
				require.Equal(t, tc.to, to)
				require.Equal(t, tc.baseDenom, baseDenom)
				require.Equal(t, tc.amount, amount)
				require.Equal(t, tc.finalizeHeight, l1BlockHeight)
			}
		})
	}
}

func TestParseUpdateOracle(t *testing.T) {
	fullAttributes := UpdateOracleEvents(1, "sender")
	cases := []struct {
		name          string
		eventAttrs    []abcitypes.EventAttribute
		l1BlockHeight int64
		from          string
		err           bool
	}{
		{
			name:          "success",
			eventAttrs:    fullAttributes,
			l1BlockHeight: 1,
			from:          "sender",
			err:           false,
		},
		{
			name:       "missing event attribute l1 block height",
			eventAttrs: fullAttributes[1:],
			err:        true,
		},
		{
			name:       "missing event attribute from",
			eventAttrs: fullAttributes[:1],
			err:        true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			l1BlockHeight, from, err := ParseUpdateOracle(tc.eventAttrs)
			if tc.err {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.l1BlockHeight, l1BlockHeight)
				require.Equal(t, tc.from, from)
			}
		})
	}
}

func TestParseInitiateWithdrawal(t *testing.T) {
	fullAttributes := InitiateWithdrawalEvents("from", "to", "denom", "uinit", sdk.NewInt64Coin("denom", 10000), 1)

	cases := []struct {
		name       string
		eventAttrs []abcitypes.EventAttribute
		l2Sequence uint64
		amount     sdk.Coin
		from       string
		to         string
		baseDenom  string
		err        bool
	}{
		{
			name:       "success",
			eventAttrs: fullAttributes,
			l2Sequence: 1,
			amount:     sdk.NewInt64Coin("denom", 10000),
			from:       "from",
			to:         "to",
			baseDenom:  "uinit",
			err:        false,
		},
		{
			name:       "missing event attribute from",
			eventAttrs: fullAttributes[1:],
			err:        true,
		},
		{
			name:       "missing event attribute to",
			eventAttrs: append(slices.Clone(fullAttributes[:1]), fullAttributes[2:]...),
			err:        true,
		},
		{
			name:       "missing event attribute denom",
			eventAttrs: append(slices.Clone(fullAttributes[:2]), fullAttributes[3:]...),
			err:        true,
		},
		{
			name:       "missing event attribute base denom",
			eventAttrs: append(slices.Clone(fullAttributes[:3]), fullAttributes[4:]...),
			err:        true,
		},
		{
			name:       "missing event attribute amount",
			eventAttrs: append(slices.Clone(fullAttributes[:4]), fullAttributes[5:]...),
			err:        true,
		},
		{
			name:       "missing event attribute l2 sequence",
			eventAttrs: fullAttributes[:5],
			err:        true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			l2Sequence, amount, from, to, denom, baseDenom, err := ParseInitiateWithdrawal(tc.eventAttrs)
			if tc.err {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.l2Sequence, l2Sequence)
				require.Equal(t, tc.amount.Amount.Uint64(), amount)
				require.Equal(t, tc.from, from)
				require.Equal(t, tc.to, to)
				require.Equal(t, tc.amount.Denom, denom)
				require.Equal(t, tc.baseDenom, baseDenom)
			}
		})
	}
}
