// excel.go
// Excel 导入导出工具
// 基于 excelize/v2 封装
// 用于：用户批量导入（模块01）、成绩导出（模块06）、审计日志导出（模块08）

package excel

import (
	"bytes"
	"fmt"
	"io"

	"github.com/xuri/excelize/v2"
)

// ExportConfig 导出配置
type ExportConfig struct {
	SheetName string     // 工作表名称
	Headers   []string   // 表头列名
	ColWidths []float64  // 列宽（可选，与 Headers 一一对应）
}

// Export 导出数据到 Excel
// rows 为二维字符串数组，每行对应一条数据
func Export(config *ExportConfig, rows [][]interface{}) (*bytes.Buffer, error) {
	f := excelize.NewFile()
	defer f.Close()

	sheet := config.SheetName
	if sheet == "" {
		sheet = "Sheet1"
	}

	// 创建工作表
	index, err := f.NewSheet(sheet)
	if err != nil {
		return nil, fmt.Errorf("创建工作表失败: %w", err)
	}
	f.SetActiveSheet(index)

	// 删除默认的 Sheet1（如果不是目标表）
	if sheet != "Sheet1" {
		f.DeleteSheet("Sheet1")
	}

	// 设置表头样式
	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 12},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"#E8E8E8"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "#D4D4D4", Style: 1},
			{Type: "top", Color: "#D4D4D4", Style: 1},
			{Type: "bottom", Color: "#D4D4D4", Style: 1},
			{Type: "right", Color: "#D4D4D4", Style: 1},
		},
	})

	// 写入表头
	for i, header := range config.Headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheet, cell, header)
		f.SetCellStyle(sheet, cell, cell, headerStyle)

		// 设置列宽
		colName, _ := excelize.ColumnNumberToName(i + 1)
		if i < len(config.ColWidths) && config.ColWidths[i] > 0 {
			f.SetColWidth(sheet, colName, colName, config.ColWidths[i])
		} else {
			f.SetColWidth(sheet, colName, colName, 15) // 默认列宽
		}
	}

	// 写入数据行
	for rowIdx, row := range rows {
		for colIdx, val := range row {
			cell, _ := excelize.CoordinatesToCellName(colIdx+1, rowIdx+2)
			f.SetCellValue(sheet, cell, val)
		}
	}

	// 写入缓冲区
	buf := new(bytes.Buffer)
	if err := f.Write(buf); err != nil {
		return nil, fmt.Errorf("写入Excel失败: %w", err)
	}

	return buf, nil
}

// ImportRow 导入行数据
type ImportRow struct {
	RowNum int                    // 行号（从1开始，不含表头）
	Data   map[string]string      // 列名 -> 值
	Errors []string               // 该行的校验错误
}

// Import 从 Excel 读取数据
// reader 为文件读取器
// headers 为期望的表头列名（用于映射）
func Import(reader io.Reader, headers []string) ([]ImportRow, error) {
	f, err := excelize.OpenReader(reader)
	if err != nil {
		return nil, fmt.Errorf("打开Excel文件失败: %w", err)
	}
	defer f.Close()

	// 获取第一个工作表
	sheetName := f.GetSheetName(0)
	rows, err := f.GetRows(sheetName)
	if err != nil {
		return nil, fmt.Errorf("读取工作表失败: %w", err)
	}

	if len(rows) < 2 {
		return nil, fmt.Errorf("Excel文件为空或仅有表头")
	}

	// 解析表头映射（实际列位置 -> 期望列名）
	headerMap := make(map[int]string)
	for colIdx, cell := range rows[0] {
		for _, h := range headers {
			if cell == h {
				headerMap[colIdx] = h
				break
			}
		}
	}

	// 解析数据行
	var result []ImportRow
	for rowIdx := 1; rowIdx < len(rows); rowIdx++ {
		row := rows[rowIdx]
		data := make(map[string]string)
		for colIdx, header := range headerMap {
			if colIdx < len(row) {
				data[header] = row[colIdx]
			} else {
				data[header] = ""
			}
		}
		result = append(result, ImportRow{
			RowNum: rowIdx,
			Data:   data,
		})
	}

	return result, nil
}

// CreateTemplate 创建导入模板
// headers 为表头列名
// sampleRows 为示例数据行（可选）
func CreateTemplate(sheetName string, headers []string, sampleRows [][]interface{}) (*bytes.Buffer, error) {
	config := &ExportConfig{
		SheetName: sheetName,
		Headers:   headers,
	}
	return Export(config, sampleRows)
}
