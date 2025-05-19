package wasmer

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strconv"
)

// 用于存储每次调用的指标
type CallMetrics struct {
	ExportMemoryTime int
	RealFuncTime     int
	ReturnResultTime int
	TotalFuncTime    int
	BigNumCalTime    int
}

// 用于存储累计的占比数据
type AverageRatios struct {
	TotalCalls          int
	ExportMemoryPercent float64
	RealFuncPercent     float64
	ReturnResultPercent float64
	TotalCostPercent    float64
}

var (
	// 正则表达式匹配模式
	exportMemoryRe = regexp.MustCompile(`export memory get param.*executionTime (\d+)`)
	realFuncRe     = regexp.MustCompile(`real func.*executionTime (\d+)`)
	returnResultRe = regexp.MustCompile(`return result.*executionTime (\d+)`)
	totalCostRe    = regexp.MustCompile(`totalCost.*executionTime (\d+)`)
	bigNumCalRe    = regexp.MustCompile(`contractResult.*executionTime (\d+)`)
)

func main() {
	file, err := os.Open("default.log.2025042723")
	if err != nil {
		panic(err)
	}
	defer file.Close()

	var calls []CallMetrics
	currentCall := new(CallMetrics)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()

		switch {
		case exportMemoryRe.MatchString(line):
			currentCall.ExportMemoryTime = extractNumber(exportMemoryRe, line)
		case realFuncRe.MatchString(line):
			currentCall.RealFuncTime = extractNumber(realFuncRe, line)
		case returnResultRe.MatchString(line):
			currentCall.ReturnResultTime = extractNumber(returnResultRe, line)
		case totalCostRe.MatchString(line):
			currentCall.TotalFuncTime = extractNumber(totalCostRe, line)
		case bigNumCalRe.MatchString(line):
			currentCall.BigNumCalTime = extractNumber(bigNumCalRe, line)
			calls = append(calls, *currentCall)
			currentCall = new(CallMetrics)
		}
	}

	// 计算平均占比
	averages := calculateAverages(calls)
	printAverages(averages)
}

func extractNumber(re *regexp.Regexp, line string) int {
	matches := re.FindStringSubmatch(line)
	if len(matches) < 2 {
		return 0
	}
	num, _ := strconv.Atoi(matches[1])
	return num
}

func calculateAverages(calls []CallMetrics) AverageRatios {
	var sums struct {
		ExportMemory float64
		RealFunc     float64
		ReturnResult float64
		TotalCost    float64
	}

	for _, call := range calls {
		// 计算各阶段占比
		totalCost := float64(call.TotalFuncTime)
		bigNumCal := float64(call.BigNumCalTime)

		sums.ExportMemory += float64(call.ExportMemoryTime) / totalCost * 100
		sums.RealFunc += float64(call.RealFuncTime) / totalCost * 100
		sums.ReturnResult += float64(call.ReturnResultTime) / totalCost * 100
		sums.TotalCost += totalCost / bigNumCal * 100
	}

	count := float64(len(calls))
	return AverageRatios{
		TotalCalls:          len(calls),
		ExportMemoryPercent: sums.ExportMemory / count,
		RealFuncPercent:     sums.RealFunc / count,
		ReturnResultPercent: sums.ReturnResult / count,
		TotalCostPercent:    sums.TotalCost / count,
	}
}

func printAverages(avg AverageRatios) {
	fmt.Printf("分析结果（基于 %d 次调用）：\n", avg.TotalCalls)
	fmt.Printf("Export Memory 平均占比: %.2f%%\n", avg.ExportMemoryPercent)
	fmt.Printf("Real Func 平均占比:     %.2f%%\n", avg.RealFuncPercent)
	fmt.Printf("Return Result 平均占比: %.2f%%\n", avg.ReturnResultPercent)
	fmt.Printf("TotalCost/BigNumCal 平均占比: %.2f%%\n", avg.TotalCostPercent)
}
