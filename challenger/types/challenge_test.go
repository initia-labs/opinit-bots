package types

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestOracleRelayEqual(t *testing.T) {
	hash1 := []byte("oracle_price_hash_1")
	hash2 := []byte("oracle_price_hash_2")
	blockTime := time.Unix(0, 100).UTC()

	cases := []struct {
		name     string
		oracle1  *OracleRelay
		oracle2  *OracleRelay
		expected bool
	}{
		{
			name:     "equal oracle relays",
			oracle1:  NewOracleRelay(100, hash1, blockTime),
			oracle2:  NewOracleRelay(100, hash1, blockTime),
			expected: true,
		},
		{
			name:     "different L1 heights",
			oracle1:  NewOracleRelay(100, hash1, blockTime),
			oracle2:  NewOracleRelay(200, hash1, blockTime),
			expected: false,
		},
		{
			name:     "different oracle price hashes",
			oracle1:  NewOracleRelay(100, hash1, blockTime),
			oracle2:  NewOracleRelay(100, hash2, blockTime),
			expected: false,
		},
		{
			name:     "different times but same data - should be equal",
			oracle1:  NewOracleRelay(100, hash1, time.Unix(0, 100).UTC()),
			oracle2:  NewOracleRelay(100, hash1, time.Unix(0, 200).UTC()),
			expected: true, // Time is not compared in Equal
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := tc.oracle1.Equal(tc.oracle2)
			require.NoError(t, err)
			require.Equal(t, tc.expected, result)
		})
	}
}

func TestOracleRelayMarshalUnmarshal(t *testing.T) {
	hash := []byte("test_oracle_price_hash")
	blockTime := time.Unix(0, 100).UTC()

	original := NewOracleRelay(100, hash, blockTime)

	data, err := original.Marshal()
	require.NoError(t, err)

	var unmarshaled OracleRelay
	err = unmarshaled.Unmarshal(data)
	require.NoError(t, err)

	require.Equal(t, original.EventType, unmarshaled.EventType)
	require.Equal(t, original.L1Height, unmarshaled.L1Height)
	require.Equal(t, original.OraclePriceHash, unmarshaled.OraclePriceHash)
	require.Equal(t, original.Time, unmarshaled.Time)
	require.Equal(t, original.Timeout, unmarshaled.Timeout)
}

func TestOracleRelayType(t *testing.T) {
	oracle := NewOracleRelay(100, []byte("hash"), time.Now())
	require.Equal(t, EventTypeOracleRelay, oracle.Type())
	require.Equal(t, "OracleRelay", oracle.Type().String())
}

func TestOracleRelayId(t *testing.T) {
	oracle := NewOracleRelay(100, []byte("hash"), time.Now())
	id := oracle.Id()
	require.Equal(t, EventTypeOracleRelay, id.Type)
	require.Equal(t, uint64(100), id.Id)
}

func TestOracleRelayTimeout(t *testing.T) {
	oracle := NewOracleRelay(100, []byte("hash"), time.Now())
	require.False(t, oracle.IsTimeout())

	oracle.SetTimeout()
	require.True(t, oracle.IsTimeout())
}

func TestUnmarshalChallengeEventOracleRelay(t *testing.T) {
	original := NewOracleRelay(100, []byte("hash"), time.Unix(0, 100).UTC())

	data, err := original.Marshal()
	require.NoError(t, err)

	event, err := UnmarshalChallengeEvent(EventTypeOracleRelay, data)
	require.NoError(t, err)

	oracleRelay, ok := event.(*OracleRelay)
	require.True(t, ok)
	require.Equal(t, original.L1Height, oracleRelay.L1Height)
	require.Equal(t, original.OraclePriceHash, oracleRelay.OraclePriceHash)
}

func TestEventTypeOracleRelayValidate(t *testing.T) {
	err := EventTypeOracleRelay.Validate()
	require.NoError(t, err)
}
