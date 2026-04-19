// excel.go
// 该文件封装平台统一的表格导入导出能力，负责处理 Excel/CSV 的模板生成、原始行读取、
// 标准表头映射和二维数据导出。用户导入、成绩导出、失败明细下载、审计数据导出等功能都
// 应复用这里的能力，避免每个模块自己解析文件或自己拼导出格式。

package excel

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/xuri/excelize/v2"
)

// ExportConfig 导出配置
type ExportConfig struct {
	SheetName string    // 工作表名称
	Headers   []string  // 表头列名
	ColWidths []float64 // 列宽（可选，与 Headers 一一对应）
}

// Export 导出数据到 Excel。
// 该函数用于生成带表头和基础样式的标准 xlsx 文件，适合模板下载、失败明细导出、
// 成绩导出和审计导出等需要直接下载 Excel 的场景。
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

// ExportCSV 导出数据到 CSV。
// 当接口或文档要求导出 CSV 时使用该函数；它与 Export 共享表头和行数据输入形式，
// 方便上层在 Excel/CSV 两种格式之间切换而不用重新组织数据。
func ExportCSV(headers []string, rows [][]interface{}) (*bytes.Buffer, error) {
	buf := new(bytes.Buffer)
	writer := csv.NewWriter(buf)

	if len(headers) > 0 {
		if err := writer.Write(headers); err != nil {
			return nil, fmt.Errorf("写入 CSV 表头失败: %w", err)
		}
	}

	for _, row := range rows {
		record := make([]string, 0, len(row))
		for _, value := range row {
			record = append(record, fmt.Sprint(value))
		}
		if err := writer.Write(record); err != nil {
			return nil, fmt.Errorf("写入 CSV 数据失败: %w", err)
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, fmt.Errorf("生成 CSV 文件失败: %w", err)
	}
	return buf, nil
}

// ImportRow 导入行数据
type ImportRow struct {
	RowNum int               // 行号（从1开始，不含表头）
	Data   map[string]string // 列名 -> 值
	Errors []string          // 该行的校验错误
}

// Import 从 Excel 读取数据。
// 它负责根据期望表头把用户上传的列映射成结构化数据，供后续导入 service 做逐行校验、
// 去空格、去重和失败明细生成。
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

// ImportRawRows 从 Excel 或 CSV 读取原始数据行。
// 该函数只负责把首行表头之后的原始内容读出来，不做业务字段含义解释，适合导入前的统一
// 预处理流程，例如跳过空行、裁剪空格和生成失败明细。
func ImportRawRows(filename string, reader io.Reader) ([][]string, error) {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".csv":
		csvReader := csv.NewReader(reader)
		csvReader.FieldsPerRecord = -1
		rows, err := csvReader.ReadAll()
		if err != nil {
			return nil, fmt.Errorf("CSV文件解析失败")
		}
		if len(rows) <= 1 {
			return [][]string{}, nil
		}
		return rows[1:], nil
	case ".xlsx", ".xlsm", ".xltx", ".xltm", "":
		f, err := excelize.OpenReader(reader)
		if err != nil {
			return nil, fmt.Errorf("文件格式不正确，请上传 Excel 或 CSV 文件")
		}
		defer f.Close()

		sheetName := f.GetSheetName(0)
		if sheetName == "" {
			return nil, fmt.Errorf("读取文件内容失败")
		}
		rows, err := f.GetRows(sheetName)
		if err != nil {
			return nil, fmt.Errorf("读取文件内容失败")
		}
		if len(rows) <= 1 {
			return [][]string{}, nil
		}
		return rows[1:], nil
	default:
		return nil, fmt.Errorf("文件格式不正确，请上传 Excel 或 CSV 文件")
	}
}

// CreateTemplate 创建导入模板。
// 上层导入接口可以通过它快速生成标准表头和示例行一致的模板文件，减少各模块各自拼装
// 模板内容的重复工作。
func CreateTemplate(sheetName string, headers []string, sampleRows [][]interface{}) (*bytes.Buffer, error) {
	config := &ExportConfig{
		SheetName: sheetName,
		Headers:   headers,
	}
	return Export(config, sampleRows)
}
