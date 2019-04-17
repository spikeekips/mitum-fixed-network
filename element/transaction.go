package element

import (
	"encoding/json"

	"github.com/Masterminds/semver"
	"github.com/spikeekips/mitum/common"
)

var (
	CurrentTransactionVersion semver.Version = *semver.MustParse("v0.1-proto")
)

func NewTransactionHash(t Transaction) (common.Hash, error) {
	return common.NewHashFromObject("tx", t)
}

type Transaction struct {
	Version    semver.Version
	Source     common.Address
	Checkpoint []byte // TODO account state root
	Fee        common.Amount
	Created    common.Time
	Operations []Operation
}

func NewTransaction(source common.Address, checkpoint []byte, baseFee common.Amount, operations []Operation) Transaction {
	return Transaction{
		Version:    CurrentTransactionVersion,
		Source:     source,
		Checkpoint: checkpoint,
		Fee:        baseFee.Mul(common.NewAmount(uint64(len(operations)))),
		Operations: operations,
		Created:    common.Now(),
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
		"created":    t.Created,
		"operations": t.Operations,
	}

	if !t.Fee.IsZero() {
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

	var fee common.Amount
	if err := json.Unmarshal(raw["fee"], &fee); err != nil {
		return err
	}

	var created common.Time
	if err := json.Unmarshal(raw["created"], &created); err != nil {
		return err
	}

	t.Version = version
	t.Source = source
	t.Checkpoint = checkpoint
	t.Fee = fee
	t.Created = created

	return nil
}

func (t Transaction) String() string {
	b, _ := json.MarshalIndent(t, "", "  ")
	return string(b)
}
