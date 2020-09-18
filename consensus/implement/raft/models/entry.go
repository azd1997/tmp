/**********************************************************************
* @Author: Eiger (201820114847@mail.scut.edu.cn)
* @Date: 2020/8/28 12:55
* @Description: 介绍各数据结构
***********************************************************************/

package models


// EntryType 条目类型
type EntryType uint64

const (
	EntryType_DataChange EntryType = 0  // 数据变化
	EntryType_ConfChange EntryType = 1	// 配置变化
)

var EntryType_name = map[uint64]string{
	0: "EntryNormal",
	1: "EntryConfChange",
}
var EntryType_value = map[string]uint64{
	"EntryNormal":     0,
	"EntryConfChange": 1,
}

func (e EntryType) String() string {
	x := uint64(e)
	return EntryType_name[x]
}

// Entry 日志。 数据的载体
type Entry struct {
	// 条目类型
	EntryType EntryType
	// 任期
	Term uint64
	// Entry索引
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

func (m *Entry) GetTerm() uint64 {
	if m != nil {
		return m.Term
	}
	return 0
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