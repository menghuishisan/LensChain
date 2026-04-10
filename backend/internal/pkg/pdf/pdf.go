// pdf.go
// PDF 生成工具
// 用于成绩单 PDF 生成（模块06）
// 支持学校Logo、学生信息、成绩表格、GPA汇总等内容

package pdf

import (
	"bytes"
	"fmt"
	"io"
)

// TranscriptData 成绩单数据
type TranscriptData struct {
	SchoolName   string            // 学校名称
	SchoolLogo   io.Reader         // 学校Logo（可选）
	StudentName  string            // 学生姓名
	StudentNo    string            // 学号
	College      string            // 学院
	Major        string            // 专业
	Semesters    []SemesterGrades  // 各学期成绩
	CumulativeGPA float64          // 累计GPA
	TotalCredits float64           // 总学分
	GeneratedAt  string            // 生成时间
}

// SemesterGrades 学期成绩
type SemesterGrades struct {
	SemesterName string          // 学期名称
	Courses      []CourseGrade   // 课程成绩列表
	SemesterGPA  float64         // 学期GPA
}

// CourseGrade 课程成绩
type CourseGrade struct {
	CourseName string  // 课程名称
	Credits    float64 // 学分
	Score      float64 // 百分制成绩
	GradeLevel string  // 等级（A/B/C/D/F）
	GPAPoint   float64 // 绩点
}

// Generator PDF生成器接口
type Generator interface {
	GenerateTranscript(data *TranscriptData) (*bytes.Buffer, error)
}

// 全局生成器
var generator Generator

// Init 初始化 PDF 生成器
func Init() {
	generator = &simplePDFGenerator{}
}

// GenerateTranscript 生成成绩单 PDF
func GenerateTranscript(data *TranscriptData) (*bytes.Buffer, error) {
	if generator == nil {
		return nil, fmt.Errorf("PDF生成器未初始化")
	}
	return generator.GenerateTranscript(data)
}

// ---- 简易 PDF 生成器实现 ----
// 使用纯文本格式作为初始实现，后续可替换为 wkhtmltopdf 或 gofpdf

type simplePDFGenerator struct{}

func (g *simplePDFGenerator) GenerateTranscript(data *TranscriptData) (*bytes.Buffer, error) {
	buf := new(bytes.Buffer)

	// 标题
	fmt.Fprintf(buf, "========================================\n")
	fmt.Fprintf(buf, "          %s\n", data.SchoolName)
	fmt.Fprintf(buf, "            学生成绩单\n")
	fmt.Fprintf(buf, "========================================\n\n")

	// 学生信息
	fmt.Fprintf(buf, "姓名: %s    学号: %s\n", data.StudentName, data.StudentNo)
	fmt.Fprintf(buf, "学院: %s    专业: %s\n", data.College, data.Major)
	fmt.Fprintf(buf, "----------------------------------------\n\n")

	// 各学期成绩
	for _, semester := range data.Semesters {
		fmt.Fprintf(buf, "【%s】 学期GPA: %.2f\n", semester.SemesterName, semester.SemesterGPA)
		fmt.Fprintf(buf, "%-30s %6s %6s %6s %6s\n", "课程名称", "学分", "成绩", "等级", "绩点")
		fmt.Fprintf(buf, "%-30s %6s %6s %6s %6s\n", "-----", "----", "----", "----", "----")
		for _, course := range semester.Courses {
			fmt.Fprintf(buf, "%-30s %6.1f %6.1f %6s %6.2f\n",
				course.CourseName, course.Credits, course.Score, course.GradeLevel, course.GPAPoint)
		}
		fmt.Fprintf(buf, "\n")
	}

	// 汇总
	fmt.Fprintf(buf, "========================================\n")
	fmt.Fprintf(buf, "累计GPA: %.2f    总学分: %.1f\n", data.CumulativeGPA, data.TotalCredits)
	fmt.Fprintf(buf, "生成时间: %s\n", data.GeneratedAt)
	fmt.Fprintf(buf, "========================================\n")

	return buf, nil
}
