package params


type CodeVersion uint16		// 用三位数字表示版本号

const (
	// NodeVersionV1 starts from v1.0.0
	NodeVersionV1 = CodeVersion(1)
)

var CurrentCodeVersion = NodeVersionV1
var MinimizeVersionRequired = NodeVersionV1
