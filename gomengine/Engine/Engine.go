package Engine

import "fmt"

func Match(order OrderNode) bool {
	if false == ExistsPrePool(order) {
		return false
	}

	DeletePrePool(order)

	// 撮合计算逻辑
	fmt.Printf("%#v\n", order)
	fmt.Printf("%T\n", order)

	return true
}
