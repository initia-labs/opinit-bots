package types

import (
	"bytes"
	"encoding/json"
	"fmt"
	"time"
)

type Challenge struct {
	Id  ChallengeId `json:"id"`
	Log string      `json:"log"`
}

func NewChallenge(id ChallengeId, log string) Challenge {
	return Challenge{
		Id:  id,
		Log: log,
	}
}

func (c Challenge) Marshal() ([]byte, error) {
	return json.Marshal(&c)
}

func (c *Challenge) Unmarshal(data []byte) error {
	return json.Unmarshal(data, c)
}

type ChallengeElem struct {
	Node  NodeType       `json:"node"`
	Id    uint64         `json:"id"`
	Event ChallengeEvent `json:"event"`
}

func (e ChallengeElem) ChallengeId() ChallengeId {
	return ChallengeId{
		Type: e.Event.Type(),
		Id:   e.Id,
	}
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
}

type ChallengeState uint8

const (
	ChallengeStatePending ChallengeState = iota
	ChallengeStateFailed
	ChallengeStatePassed
)

func (c ChallengeState) String() string {
	switch c {
	case ChallengeStatePending:
		return "Pending"
	case ChallengeStateFailed:
		return "Failed"
	case ChallengeStatePassed:
		return "Passed"
	default:
		return "Unknown"
	}
}

type NodeType uint8

const (
	NodeTypeHost NodeType = iota
	NodeTypeChild
)

type EventType uint8

const (
	EventTypeDeposit EventType = iota
	EventTypeOutput
)

func (e EventType) String() string {
	switch e {
	case EventTypeDeposit:
		return "Deposit"
	case EventTypeOutput:
		return "Output"
	default:
		return "Unknown"
	}
}

type Deposit struct {
	Sequence      uint64    `json:"sequence"`
	L1BlockHeight uint64    `json:"l1_block_height"`
	EventTime     time.Time `json:"event_time"`
	From          string    `json:"from"`
	To            string    `json:"to"`
	L1Denom       string    `json:"l1_denom"`
	Amount        string    `json:"amount"`
}

var _ ChallengeEvent = &Deposit{}

func NewDeposit(sequence, l1BlockHeight uint64, eventTime time.Time, from, to, l1Denom, amount string) *Deposit {
	return &Deposit{
		Sequence:      sequence,
		L1BlockHeight: l1BlockHeight,
		EventTime:     eventTime,
		From:          from,
		To:            to,
		L1Denom:       l1Denom,
		Amount:        amount,
	}
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
	return fmt.Sprintf("Deposit{Sequence: %d, L1BlockHeight: %d, From: %s, To: %s, L1Denom: %s, Amount: %s, EventTime: %s}", d.Sequence, d.L1BlockHeight, d.From, d.To, d.L1Denom, d.Amount, d.EventTime)
}

func (d Deposit) Type() EventType {
	return EventTypeDeposit
}

type Output struct {
	L2BlockNumber uint64    `json:"l2_block_number"`
	EventTime     time.Time `json:"event_time"`
	OutputIndex   uint64    `json:"output_index"`
	OutputRoot    []byte    `json:"output_root"`
}

var _ ChallengeEvent = &Output{}

func NewOutput(l2BlockNumber, outputIndex uint64, outputRoot []byte) *Output {
	return &Output{
		L2BlockNumber: l2BlockNumber,
		OutputIndex:   outputIndex,
		OutputRoot:    outputRoot,
	}
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
	return fmt.Sprintf("Output{L2BlockNumber: %d, OutputIndex: %d, OutputRoot: %s, EventTime: %s}", o.L2BlockNumber, o.OutputIndex, o.OutputRoot, o.EventTime)
}

func (o Output) Type() EventType {
	return EventTypeOutput
}
