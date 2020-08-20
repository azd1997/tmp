package storage

import (
	"encoding/gob"
	"github.com/pkg/errors"
	"io"

	"github.com/azd1997/ecoin/protocol/core"
)

// Tx 交易体
type Tx struct {
	*core.Tx
}

func (tx *Tx) Decode(data io.Reader) error {
	if err := gob.NewDecoder(data).Decode(tx); err != nil {
		return errors.Wrap(err, "Tx_Decode")
	}
	return nil
}
