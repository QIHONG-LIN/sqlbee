package sqlbee_go

// SqlBeeE的method type (枚举)
type SqlBeeE_MethodType int

const (
	// 第一个元素为None
	None SqlBeeE_MethodType = iota
	Insert
	Delete
	Update
)
