package transaction

import "github.com/azd1997/ego/utils"

func init() {
	utils.GobRegister(&TxCoinbase{}, &TxGeneral{}, &TxR2P{}, &TxP2R{},
		&TxP2H{}, &TxH2P{}, &TxP2D{}, &TxD2P{}, &TxArbitrate{})
}
