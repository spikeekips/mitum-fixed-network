package element

import (
	"encoding/json"

	"github.com/Masterminds/semver"
	"github.com/spikeekips/mitum/common"
)

var (
	CurrentTransactionVersion semver.Version = *semver.MustParse("0.1.0-proto")
)

func NewTransactionHash(t Transaction) (common.Hash, error) {
	return common.NewHashFromObject("tx", t)
}

type Transaction struct {
	Version    semver.Version
	Source     common.Address
	Checkpoint []byte // NOTE account state root
	Fee        common.Big
	CreatedAt  common.Time
	Operations []Operation
}

func NewTransaction(source common.Address, checkpoint []byte, baseFee common.Big, operations []Operation) Transaction {
	return Transaction{
		Version:    CurrentTransactionVersion,
		Source:     source,
		Checkpoint: checkpoint,
		Fee:        baseFee.Mul(common.NewBig(uint64(len(operations)))),
		Operations: operations,
		CreatedAt:  common.Now(),
	}
}

func (t Transaction) Hash() (common.Hash, error) {
	return NewTransactionHash(t)
}

func (t Transaction) MarshalJSON() ([]byte, error) {
	m := map[string]interface{}{
		"version":    &t.Version,
		"source":     t.Source,
		"checkpoint": t.Checkpoint,
		"created_at": t.CreatedAt,
		"operations": t.Operations,
	}

	if !t.Fee.IsZero() { // NOTE fee can be omitted
		m["fee"] = t.Fee
	}

	return json.Marshal(m)
}

func (t *Transaction) UnmarshalJSON(b []byte) error {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}

	var version semver.Version
	if err := json.Unmarshal(raw["version"], &version); err != nil {
		return err
	}

	var source common.Address
	if err := json.Unmarshal(raw["source"], &source); err != nil {
		return err
	}

	var checkpoint []byte
	if err := json.Unmarshal(raw["checkpoint"], &checkpoint); err != nil {
		return err
	}

	var fee common.Big
	if r, ok := raw["fee"]; ok {
		if err := json.Unmarshal(r, &fee); err != nil {
			return err
		}
	}

	var createdAt common.Time
	if err := json.Unmarshal(raw["created_at"], &createdAt); err != nil {
		return err
	}

	t.Version = version
	t.Source = source
	t.Checkpoint = checkpoint
	t.Fee = fee
	t.CreatedAt = createdAt

	return nil
}

func (t Transaction) String() string {
	b, _ := json.MarshalIndent(t, "", "  ")
	return string(b)
}
