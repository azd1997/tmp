/**********************************************************************
* @Author: Eiger (201820114847@mail.scut.edu.cn)
* @Date: 2020/4/1 17:42
* @Description: The file is for
***********************************************************************/

package bc

import "fmt"

type ErrAlreadyUpToDate struct {
	reqHash []byte
}

func (a ErrAlreadyUpToDate) Error() string {
	return fmt.Sprintf("%X is already up to date", a.reqHash)
}

type ErrFlushingCache struct {
	reqHash []byte
}

func (f ErrFlushingCache) Error() string {
	return fmt.Sprintf("flushing happens while handling %X, give up", f.reqHash)
}

type ErrHashNotFound struct {
	reqHash []byte
}

func (s ErrHashNotFound) Error() string {
	return fmt.Sprintf("sync %X not found", s.reqHash)
}

type ErrInvalidBlockRange struct {
	info string
}

func (s ErrInvalidBlockRange) Error() string {
	return s.info
}

type ErrTxAlreadyExist struct {
	txId []byte
}

func (e ErrTxAlreadyExist) Error() string {
	return fmt.Sprintf("tx %X exists", e.txId)
}
