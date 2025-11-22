package main_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/dop251/goja"
)

func Test_Plugin_Javascript(t *testing.T) {
	// 创建新的JavaScript虚拟机
	vm := goja.New()

	// 预编译JavaScript代码
	_, err := vm.RunString(`
        function calculate(x, y) {
            return x * y + x / y;
        }
        
        function greet(name) {
            return "Hello, " + name + "!";
        }
    `)
	if err != nil {
		panic(err)
	}

	// 获取函数引用
	calcFn, ok := goja.AssertFunction(vm.Get("calculate"))
	if !ok {
		panic("calculate is not a function")
	}

	greetFn, ok := goja.AssertFunction(vm.Get("greet"))
	if !ok {
		panic("greet is not a function")
	}

	// 测试直接函数调用的性能
	start := time.Now()

	// 多次调用函数而不使用RunString
	for i := 0; i < 10000; i++ {
		x := float64(i%100 + 1)
		y := float64(i%50 + 1)

		
		result, err := calcFn(goja.Undefined(), vm.ToValue(x), vm.ToValue(y))
		if err != nil {
			panic(err)
		}

		// 演示使用结果（在实际应用中可能不需要打印每次结果）
		if i%1000 == 0 {
			fmt.Printf("calculate(%.1f, %.1f) = %.2f\n", x, y, result.ToFloat())
		}
	}

	// 调用另一个函数
	nameResult, err := greetFn(goja.Undefined(), vm.ToValue("Goja Developer"))
	if err != nil {
		panic(err)
	}
	fmt.Println(nameResult.ToString())

	elapsed := time.Since(start)
	fmt.Printf("直接函数调用执行时间: %v\n", elapsed)

	// 对比使用RunString的性能
	start = time.Now()

	for i := 0; i < 10000; i++ {
		x := float64(i%100 + 1)
		y := float64(i%50 + 1)

		_, err := vm.RunString(fmt.Sprintf("calculate(%f, %f)", x, y))
		if err != nil {
			panic(err)
		}
	}

	elapsed = time.Since(start)
	fmt.Printf("使用RunString执行时间: %v\n", elapsed)
}
