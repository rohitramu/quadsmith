package cel2sql

import (
	"cel.dev/cel-go/cel"
	"fmt"
)

func Explore() {
	env, _ := cel.NewEnv(cel.Variable("motor.kv_rating", cel.IntType))
	celAst, _ := env.Compile("-motor.kv_rating < -1000")
	e := celAst.NativeRep().Expr().AsCall().Args()[0]
	fmt.Printf("Unary Minus Func: %s\n", e.AsCall().FunctionName())
}
