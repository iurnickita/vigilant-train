// Анализаторы. Пакет main.
// Реализация analysis/multichecker
package main

import (
	"go/ast"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/multichecker"
	"golang.org/x/tools/go/analysis/passes/printf"
	"golang.org/x/tools/go/analysis/passes/shadow"
	"golang.org/x/tools/go/analysis/passes/structtag"
	"honnef.co/go/tools/staticcheck"
)

func main() {
	// Analysis
	// analysis/passes
	mychecks := []*analysis.Analyzer{printf.Analyzer,
		shadow.Analyzer,
		structtag.Analyzer}
	// staticcheck
	for _, v := range staticcheck.Analyzers {
		// Проверки класса SA
		if v.Analyzer.Name[0:2] == "SA" {
			mychecks = append(mychecks, v.Analyzer)
		}
		// Проверки остальных классов
		// "S1028" Simplify error construction with fmt.Errorf
		// "ST1016" Use consistent method receiver names
		if v.Analyzer.Name == "S1028" || v.Analyzer.Name == "ST1016" {
			mychecks = append(mychecks, v.Analyzer)
		}
	}
	// Другие анализаторы
	// 		возникли затруднения. Все попадающиеся анализаторы сделаны под golangci

	// Собственный анализатор
	mychecks = append(mychecks, OsExitAnalyzer)

	multichecker.Main(mychecks...)
}

// Анализатор OsExitAnalyzer проверяет отстуствие прямого вызова os.Exit() в функции main
var OsExitAnalyzer = &analysis.Analyzer{
	Name: "exitcheck",
	Doc:  "check for os.Exit() is not called from main()",
	Run:  runOsExitAnalyzer,
}

func runOsExitAnalyzer(pass *analysis.Pass) (interface{}, error) {
	for _, file := range pass.Files {
		if file.Name.Name != "main" {
			continue
		}
		ast.Inspect(file, func(node ast.Node) bool {
			switch x := node.(type) {
			case *ast.ExprStmt:
				if call, ok := x.X.(*ast.CallExpr); ok {
					if isOsExit(call) {
						pass.Reportf(x.Pos(), "call os.Exit() from main function")
					}
				}
			}
			return true
		})
	}
	return nil, nil
}

func isOsExit(call *ast.CallExpr) bool {
	if s, ok := call.Fun.(*ast.SelectorExpr); ok {
		if sX, ok := s.X.(*ast.Ident); ok {
			if sX.Name == "os" && s.Sel.Name == "Exit" {
				return true
			}
		}
	}
	return false
}
