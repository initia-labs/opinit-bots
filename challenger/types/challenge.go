package types

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	"github.com/initia-labs/opinit-bots/types"
)

type Challenge struct {
	EventType string      `json:"event_type"`
	Id        ChallengeId `json:"id"`
	Log       string      `json:"log"`
	Time      time.Time   `json:"timestamp"`
}

func (c Challenge) Marshal() ([]byte, error) {
	return json.Marshal(&c)
}

func (c *Challenge) Unmarshal(value []byte) error {
	return json.Unmarshal(value, c)
}

type ChallengeId struct {
	Type EventType `json:"type"`
	Id   uint64    `json:"id"`
}

func (c ChallengeId) String() string {
	return fmt.Sprintf("%s-%d", c.Type.String(), c.Id)
}

type ChallengeEvent interface {
	Equal(ChallengeEvent) (bool, error)
	String() string
	Marshal() ([]byte, error)
	Unmarshal([]byte) error
	Type() EventType
	EventTime() time.Time
	SetTimeout()
	IsTimeout() bool
	Id() ChallengeId
}

func UnmarshalChallengeEvent(eventType EventType, data []byte) (ChallengeEvent, error) {
	var event ChallengeEvent

	switch eventType {
	case EventTypeDeposit:
		event = &Deposit{}
	case EventTypeOutput:
		event = &Output{}
	case EventTypeOracle:
		event = &Oracle{}
	default:
		return nil, fmt.Errorf("invalid event type: %d", eventType)
	}
	if err := event.Unmarshal(data); err != nil {
		return nil, err
	}
	return event, nil
}

type EventType uint8

const (
	EventTypeDeposit EventType = iota
	EventTypeOutput
	EventTypeOracle
)

func (e EventType) Validate() error {
	if e != EventTypeDeposit && e != EventTypeOutput && e != EventTypeOracle {
		return fmt.Errorf("invalid event type: %d", e)
	}
	return nil
}

func (e EventType) String() string {
	switch e {
	case EventTypeDeposit:
		return "Deposit"
	case EventTypeOutput:
		return "Output"
	case EventTypeOracle:
		return "Oracle"
	default:
		return "Unknown"
	}
}

type Deposit struct {
	EventType     string    `json:"event_type"`
	Sequence      uint64    `json:"sequence"`
	L1BlockHeight int64     `json:"l1_block_height"`
	From          string    `json:"from"`
	To            string    `json:"to"`
	L1Denom       string    `json:"l1_denom"`
	Amount        string    `json:"amount"`
	Time          time.Time `json:"time"`
	Timeout       bool      `json:"timeout"`
}

var _ ChallengeEvent = &Deposit{}

func NewDeposit(sequence uint64, l1BlockHeight int64, from, to, l1Denom, amount string, time time.Time) *Deposit {
	d := &Deposit{
		Sequence:      sequence,
		L1BlockHeight: l1BlockHeight,
		From:          from,
		To:            to,
		L1Denom:       l1Denom,
		Amount:        amount,
		Time:          time,
	}
	d.EventType = d.Type().String()
	return d
}

func (d Deposit) Marshal() ([]byte, error) {
	return json.Marshal(&d)
}

func (d *Deposit) Unmarshal(data []byte) error {
	return json.Unmarshal(data, d)
}

func (d Deposit) Equal(another ChallengeEvent) (bool, error) {
	anotherDeposit, ok := another.(*Deposit)
	if !ok {
		return false, fmt.Errorf("invalid type: %T", another)
	}
	return d.Sequence == anotherDeposit.Sequence &&
		d.L1BlockHeight == anotherDeposit.L1BlockHeight &&
		d.From == anotherDeposit.From &&
		d.To == anotherDeposit.To &&
		d.L1Denom == anotherDeposit.L1Denom &&
		d.Amount == anotherDeposit.Amount, nil
}

func (d Deposit) String() string {
	return fmt.Sprintf("Deposit{Sequence: %d, L1BlockHeight: %d, From: %s, To: %s, L1Denom: %s, Amount: %s, Time: %s}", d.Sequence, d.L1BlockHeight, d.From, d.To, d.L1Denom, d.Amount, d.Time)
}

func (d Deposit) Type() EventType {
	return EventTypeDeposit
}

func (d Deposit) EventTime() time.Time {
	return d.Time
}

func (d Deposit) Id() ChallengeId {
	return ChallengeId{
		Type: EventTypeDeposit,
		Id:   d.Sequence,
	}
}

func (d *Deposit) SetTimeout() {
	d.Timeout = true
}

func (d Deposit) IsTimeout() bool {
	return d.Timeout
}

type Output struct {
	EventType     string    `json:"event_type"`
	L2BlockNumber int64     `json:"l2_block_number"`
	OutputIndex   uint64    `json:"output_index"`
	OutputRoot    []byte    `json:"output_root"`
	Time          time.Time `json:"time"`
	Timeout       bool      `json:"timeout"`
}

var _ ChallengeEvent = &Output{}

func NewOutput(l2BlockNumber int64, outputIndex uint64, outputRoot []byte, time time.Time) *Output {
	o := &Output{
		L2BlockNumber: l2BlockNumber,
		OutputIndex:   outputIndex,
		OutputRoot:    outputRoot,
		Time:          time,
	}
	o.EventType = o.Type().String()
	return o
}

func (o Output) Marshal() ([]byte, error) {
	return json.Marshal(&o)
}

func (o *Output) Unmarshal(data []byte) error {
	return json.Unmarshal(data, o)
}

func (o Output) Equal(another ChallengeEvent) (bool, error) {
	anotherOutput, ok := another.(*Output)
	if !ok {
		return false, fmt.Errorf("invalid type: %T", another)
	}
	return o.L2BlockNumber == anotherOutput.L2BlockNumber &&
		o.OutputIndex == anotherOutput.OutputIndex &&
		bytes.Equal(o.OutputRoot, anotherOutput.OutputRoot), nil
}

func (o Output) String() string {
	return fmt.Sprintf("Output{L2BlockNumber: %d, OutputIndex: %d, OutputRoot: %s, Time: %s}", o.L2BlockNumber, o.OutputIndex, base64.RawStdEncoding.EncodeToString(o.OutputRoot), o.Time)
}

func (o Output) Type() EventType {
	return EventTypeOutput
}

func (o Output) EventTime() time.Time {
	return o.Time
}

func (o Output) Id() ChallengeId {
	return ChallengeId{
		Type: EventTypeOutput,
		Id:   o.OutputIndex,
	}
}
func (o *Output) SetTimeout() {
	o.Timeout = true
}

func (o Output) IsTimeout() bool {
	return o.Timeout
}

type Oracle struct {
	EventType string    `json:"event_type"`
	L1Height  int64     `json:"l1_height"`
	Data      []byte    `json:"data"`
	Time      time.Time `json:"time"`
	Timeout   bool      `json:"timeout"`
}

func NewOracle(l1Height int64, data []byte, time time.Time) *Oracle {
	o := &Oracle{
		L1Height: l1Height,
		Data:     data,
		Time:     time,
	}
	o.EventType = o.Type().String()
	return o
}

func (o Oracle) Marshal() ([]byte, error) {
	return json.Marshal(&o)
}

func (o *Oracle) Unmarshal(data []byte) error {
	return json.Unmarshal(data, o)
}

func (o Oracle) Equal(another ChallengeEvent) (bool, error) {
	anotherOracle, ok := another.(*Oracle)
	if !ok {
		return false, fmt.Errorf("invalid type: %T", another)
	}
	return o.L1Height == anotherOracle.L1Height &&
		bytes.Equal(o.Data, anotherOracle.Data), nil
}

func (o Oracle) String() string {
	return fmt.Sprintf("Oracle{L1Height: %d, Data: %s, Time: %s}", o.L1Height, base64.RawStdEncoding.EncodeToString(o.Data), o.Time)
}

func (o Oracle) Type() EventType {
	return EventTypeOracle
}

func (o Oracle) EventTime() time.Time {
	return o.Time
}

func (o Oracle) Id() ChallengeId {
	return ChallengeId{
		Type: EventTypeOracle,
		Id:   types.MustInt64ToUint64(o.L1Height),
	}
}

func (o *Oracle) SetTimeout() {
	o.Timeout = true
}

func (o Oracle) IsTimeout() bool {
	return o.Timeout
}
