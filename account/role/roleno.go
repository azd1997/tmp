package role

type No = uint8

const (
	A = iota	// 判断角色是否是A类节点(必须参与挖矿)
	B 			// 判断角色是否是B类节点(普通类型)
	ALL			// 判断角色是否是All节点（0~99，AB的并集）
	SINGLE		// 查询角色是否是指定的某一种
)

const (
	ALLROLE = iota				// 默认值
	HOSPITAL 					// 1
	RESEARCHER 		// 2
	_
	_
	_
	_
	_
	_
	_
	PATIENT						// 10
	DOCTOR						// 11
)

func IsARole(no uint8) bool {
	return no == HOSPITAL || no == RESEARCHER
}

func IsBRole(no uint8) bool {
	return no == PATIENT || no == DOCTOR
}

func IsRole(no uint8) bool {
	return no == HOSPITAL || no == RESEARCHER ||
		no == PATIENT || no == DOCTOR
}

func IsHospital(no uint8) bool {
	return no == HOSPITAL
}

func IsResearcher(no uint8) bool {
	return no == RESEARCHER
}

func IsPatient(no uint8) bool {
	return no == PATIENT
}

func IsDoctor(no uint8) bool {
	return no == DOCTOR
}