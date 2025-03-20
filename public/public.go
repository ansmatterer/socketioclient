package public

import "reflect"

func IsString(v interface{}) bool {
	return reflect.TypeOf(v).Kind() == reflect.String
}

// GetType 返回变量类型的字符串表示
/*
int
*string
nil
[]int
map[string]bool
struct {}
*/
func GetType(v interface{}) string {
	t := reflect.TypeOf(v)
	if t == nil {
		// 处理接口类型的nil（如直接传入nil）
		return "nil"
	}
	return t.String()
}
