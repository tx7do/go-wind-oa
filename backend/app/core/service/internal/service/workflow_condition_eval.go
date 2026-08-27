package service

import (
	"encoding/json"

	"github.com/expr-lang/expr"
)

// evalCondition 在沙箱中求值条件表达式。
// formDataJSON 是实例的申请表单数据（JSON 文本），解析为 map 后作为表达式环境。
// 返回表达式的布尔结果；语法错/类型错/运行错均返回 error（调用方按 fail-closed 处理）。
func evalCondition(condition, formDataJSON string) (bool, error) {
	if condition == "" {
		return false, nil
	}

	var env map[string]any
	if formDataJSON != "" {
		if err := json.Unmarshal([]byte(formDataJSON), &env); err != nil {
			// form_data 不是合法 JSON 对象：环境为空 map。
			env = make(map[string]any)
		}
	} else {
		env = make(map[string]any)
	}

	// expr 默认禁止任意函数调用和全局访问，Env 锁定为 form_data map。
	program, err := expr.Compile(condition, expr.Env(env), expr.AsBool())
	if err != nil {
		return false, err
	}

	output, err := expr.Run(program, env)
	if err != nil {
		return false, err
	}

	result, ok := output.(bool)
	if !ok {
		return false, nil
	}
	return result, nil
}
