package main

import (
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
	// 	возникли затруднения. Все попадающиеся анализаторы сделаны под golangci
	// Собственный анализатор
	//	...

	multichecker.Main(mychecks...)
}
