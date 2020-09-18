/**********************************************************************
* @Author: Eiger (201820114847@mail.scut.edu.cn)
* @Date: 2020/9/13 15:56
* @Description: The file is for
***********************************************************************/

package potpb

// EntryType 条目类型
type EntryType uint64

const (
	EntryType_DataChange  EntryType = 0 // 数据变化 （增删改查）
	EntryType_ConfChange  EntryType = 1 // 配置变化
	EntryType_AppendBlock EntryType = 2 // 区块。单独列出来，区块只需要追加
)

var EntryType_name = map[uint64]string{
	0: "EntryNormal",
	1: "EntryConfChange",
	2: "EntryAppendBlock",
}
var EntryType_value = map[string]uint64{
	"EntryNormal":      0,
	"EntryConfChange":  1,
	"EntryAppendBlock": 2,
}

func (e EntryType) String() string {
	x := uint64(e)
	return EntryType_name[x]
}

// Entry 日志。 数据的载体
type Entry struct {
	// 条目类型
	EntryType EntryType
	// Entry索引，从1开始。0预留给异常/特殊情况
	Index uint64
	// 数据
	Data []byte
}

func (m *Entry) GetEntryType() EntryType {
	if m != nil {
		return m.EntryType
	}
	return EntryType_DataChange
}

func (m *Entry) GetIndex() uint64 {
	if m != nil {
		return m.Index
	}
	return 0
}

func (m *Entry) GetData() []byte {
	if m != nil {
		return m.Data
	}
	return nil
}

