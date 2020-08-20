/**********************************************************************
* @Author: Eiger (201820114847@mail.scut.edu.cn)
* @Date: 2020/4/13 20:21
* @Description: The file is for
***********************************************************************/

package crypto

import (
	"fmt"
	"github.com/azd1997/ecoin/account/role"
	"testing"
)

func TestID(t *testing.T) {
	priv, err := NewPrivateKeyS256()
	if err != nil {
		t.Error(err)
	}
	fmt.Println(priv)

	id := PrivateKey2ID(priv, role.PATIENT)
	fmt.Println(id)
}
