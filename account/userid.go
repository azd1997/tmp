package account

import (
	"github.com/azd1997/ecoin/common/crypto"
)

// UserId 用户身份标识符，这里直接简化为公钥的压缩信息。 首部第一个字节为角色编号
type ID = crypto.ID

