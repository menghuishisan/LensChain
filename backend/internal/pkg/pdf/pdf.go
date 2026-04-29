// pdf.go
// 该文件负责生成模块06所需的正式成绩单 PDF，统一处理学校抬头、学生基本信息、学期成绩、
// GPA 汇总和防伪编号等版式内容。它是“成绩单输出格式”的基础实现，供成绩模块直接调用，
// 避免在 service 或 handler 里拼接二进制文档内容。

package pdf

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/color"
	imagedraw "image/draw"
	"image/jpeg"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	xdraw "golang.org/x/image/draw"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

const (
	pageWidthPx           = 1240
	pageMinHeightPx       = 1754
	pageHeaderHeightPx    = 240
	pageFooterHeightPx    = 120
	pageMarginLeftPx      = 72
	pageMarginRightPx     = 72
	pageMarginTopPx       = 64
	pageRowHeightPx       = 44
	pageSectionGapPx      = 36
	pageTableHeaderGapPx  = 52
	pageEstimatedDPI      = 150.0
	defaultSerialPrefix   = "TRANSCRIPT"
	defaultTranscriptDate = "2006-01-02 15:04:05"
)

var (
	colorWhite     = color.RGBA{255, 255, 255, 255}
	colorPrimary   = color.RGBA{25, 48, 84, 255}
	colorMuted     = color.RGBA{88, 102, 122, 255}
	colorLightLine = color.RGBA{205, 213, 225, 255}
	colorHeaderBg  = color.RGBA{240, 244, 248, 255}
	colorBlack     = color.RGBA{31, 41, 55, 255}
)

// TranscriptData 成绩单数据。
type TranscriptData struct {
	SchoolName    string           // 学校名称
	SchoolLogo    io.Reader        // 学校 Logo（可选）
	StudentName   string           // 学生姓名
	StudentNo     string           // 学号
	College       string           // 学院
	Major         string           // 专业
	Semesters     []SemesterGrades // 各学期成绩
	CumulativeGPA float64          // 累计 GPA
	TotalCredits  float64          // 总学分
	GeneratedAt   string           // 生成时间
	SerialNumber  string           // 成绩单唯一编号（可选）
}

// SemesterGrades 学期成绩。
type SemesterGrades struct {
	SemesterName string        // 学期名称
	Courses      []CourseGrade // 课程成绩列表
	SemesterGPA  float64       // 学期 GPA
}

// CourseGrade 课程成绩。
type CourseGrade struct {
	CourseName string  // 课程名称
	Credits    float64 // 学分
	Score      float64 // 百分制成绩
	GradeLevel string  // 等级（A/B/C/D/F）
	GPAPoint   float64 // 绩点
}

// Generator PDF 生成器接口。
type Generator interface {
	GenerateTranscript(data *TranscriptData) (*bytes.Buffer, error)
}

var generator Generator

// Init 初始化 PDF 生成器。
func Init() {
	generator = &imagePDFGenerator{
		fontPath: discoverFontPath(),
	}
}

// GenerateTranscript 生成成绩单 PDF。
func GenerateTranscript(data *TranscriptData) (*bytes.Buffer, error) {
	if generator == nil {
		Init()
	}
	if data == nil {
		return nil, fmt.Errorf("成绩单数据不能为空")
	}
	return generator.GenerateTranscript(data)
}

type imagePDFGenerator struct {
	fontPath string
}

// GenerateTranscript 先把成绩单渲染成高分辨率页面图像，再封装为单页 PDF。
// 这样可以在不引入额外重量级 PDF 库的前提下，稳定输出包含中文的正式成绩单文件。
func (g *imagePDFGenerator) GenerateTranscript(data *TranscriptData) (*bytes.Buffer, error) {
	rendered, err := g.renderTranscriptImage(data)
	if err != nil {
		return nil, err
	}
	return buildSingleImagePDF(rendered.jpegData, rendered.width, rendered.height), nil
}

type renderedImage struct {
	jpegData []byte
	width    int
	height   int
}

// renderTranscriptImage 负责把成绩单业务数据排版到画布上。
// 这里统一处理页眉、学生信息、学期表格和 GPA 汇总，避免上层 service 自己拼版式。
func (g *imagePDFGenerator) renderTranscriptImage(data *TranscriptData) (*renderedImage, error) {
	normalFace, titleFace, smallFace, err := g.loadFaces()
	if err != nil {
		return nil, err
	}

	estimatedRows := 0
	for _, semester := range data.Semesters {
		estimatedRows += len(semester.Courses) + 3
	}
	pageHeightPx := max(pageMinHeightPx, pageHeaderHeightPx+pageFooterHeightPx+estimatedRows*pageRowHeightPx+pageSectionGapPx*4)

	canvas := image.NewRGBA(image.Rect(0, 0, pageWidthPx, pageHeightPx))
	imagedraw.Draw(canvas, canvas.Bounds(), image.NewUniform(colorWhite), image.Point{}, imagedraw.Src)

	y := pageMarginTopPx
	if data.SchoolLogo != nil {
		if nextY, drawErr := drawLogo(canvas, data.SchoolLogo, pageMarginLeftPx, y); drawErr == nil {
			y = nextY
		}
	}

	drawText(canvas, titleFace, pageMarginLeftPx, y+52, data.SchoolName, colorPrimary)
	drawText(canvas, titleFace, pageMarginLeftPx, y+110, "学生成绩单", colorPrimary)
	serial := strings.TrimSpace(data.SerialNumber)
	if serial == "" {
		serial = buildSerialNumber()
	}
	drawRightText(canvas, smallFace, pageWidthPx-pageMarginRightPx, y+42, "编号："+serial, colorMuted)
	generatedAt := strings.TrimSpace(data.GeneratedAt)
	if generatedAt == "" {
		generatedAt = time.Now().UTC().Format(defaultTranscriptDate)
	}
	drawRightText(canvas, smallFace, pageWidthPx-pageMarginRightPx, y+82, "生成时间："+generatedAt, colorMuted)

	y += pageHeaderHeightPx
	drawLine(canvas, pageMarginLeftPx, y, pageWidthPx-pageMarginRightPx, y, colorLightLine)
	y += 50

	infoPairs := [][2]string{
		{"姓名", data.StudentName},
		{"学号", data.StudentNo},
		{"学院", data.College},
		{"专业", data.Major},
	}
	for i, pair := range infoPairs {
		lineY := y + i*40
		drawText(canvas, normalFace, pageMarginLeftPx, lineY, pair[0]+"："+safeText(pair[1]), colorBlack)
	}
	y += 200

	tableColumns := []struct {
		title string
		x     int
	}{
		{title: "课程名称", x: pageMarginLeftPx},
		{title: "学分", x: pageMarginLeftPx + 620},
		{title: "成绩", x: pageMarginLeftPx + 760},
		{title: "等级", x: pageMarginLeftPx + 900},
		{title: "绩点", x: pageMarginLeftPx + 1040},
	}

	for _, semester := range data.Semesters {
		drawText(canvas, titleFace, pageMarginLeftPx, y, semester.SemesterName, colorPrimary)
		drawRightText(canvas, normalFace, pageWidthPx-pageMarginRightPx, y, fmt.Sprintf("学期 GPA：%.2f", semester.SemesterGPA), colorPrimary)
		y += pageSectionGapPx

		drawRect(canvas, image.Rect(pageMarginLeftPx-10, y-34, pageWidthPx-pageMarginRightPx+10, y+18), colorHeaderBg)
		for _, column := range tableColumns {
			drawText(canvas, normalFace, column.x, y, column.title, colorBlack)
		}
		y += pageTableHeaderGapPx
		drawLine(canvas, pageMarginLeftPx-10, y-26, pageWidthPx-pageMarginRightPx+10, y-26, colorLightLine)

		for _, course := range semester.Courses {
			drawText(canvas, normalFace, tableColumns[0].x, y, truncateText(course.CourseName, 26), colorBlack)
			drawText(canvas, normalFace, tableColumns[1].x, y, fmt.Sprintf("%.1f", course.Credits), colorBlack)
			drawText(canvas, normalFace, tableColumns[2].x, y, fmt.Sprintf("%.1f", course.Score), colorBlack)
			drawText(canvas, normalFace, tableColumns[3].x, y, safeText(course.GradeLevel), colorBlack)
			drawText(canvas, normalFace, tableColumns[4].x, y, fmt.Sprintf("%.2f", course.GPAPoint), colorBlack)
			drawLine(canvas, pageMarginLeftPx-10, y+12, pageWidthPx-pageMarginRightPx+10, y+12, colorLightLine)
			y += pageRowHeightPx
		}
		y += pageSectionGapPx
	}

	drawLine(canvas, pageMarginLeftPx, y, pageWidthPx-pageMarginRightPx, y, colorLightLine)
	y += 56
	drawText(canvas, titleFace, pageMarginLeftPx, y, fmt.Sprintf("累计 GPA：%.2f", data.CumulativeGPA), colorPrimary)
	drawRightText(canvas, titleFace, pageWidthPx-pageMarginRightPx, y, fmt.Sprintf("总学分：%.1f", data.TotalCredits), colorPrimary)

	var jpegBuffer bytes.Buffer
	if err := jpeg.Encode(&jpegBuffer, canvas, &jpeg.Options{Quality: 92}); err != nil {
		return nil, fmt.Errorf("生成成绩单页面图像失败: %w", err)
	}

	return &renderedImage{
		jpegData: jpegBuffer.Bytes(),
		width:    canvas.Bounds().Dx(),
		height:   canvas.Bounds().Dy(),
	}, nil
}

// loadFaces 加载 PDF 渲染所需字体。
// 优先使用系统中文字体；当运行环境没有可用字体时退化为基础字体，至少保证文件可生成。
func (g *imagePDFGenerator) loadFaces() (font.Face, font.Face, font.Face, error) {
	if strings.TrimSpace(g.fontPath) == "" {
		return basicfont.Face7x13, basicfont.Face7x13, basicfont.Face7x13, nil
	}

	fontBytes, err := os.ReadFile(g.fontPath)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("读取 PDF 字体失败: %w", err)
	}

	parsedFont, err := opentype.Parse(fontBytes)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("解析 PDF 字体失败: %w", err)
	}

	buildFace := func(size float64) (font.Face, error) {
		return opentype.NewFace(parsedFont, &opentype.FaceOptions{
			Size:    size,
			DPI:     pageEstimatedDPI,
			Hinting: font.HintingFull,
		})
	}

	titleFace, err := buildFace(26)
	if err != nil {
		return nil, nil, nil, err
	}
	normalFace, err := buildFace(15)
	if err != nil {
		return nil, nil, nil, err
	}
	smallFace, err := buildFace(12)
	if err != nil {
		return nil, nil, nil, err
	}

	return normalFace, titleFace, smallFace, nil
}

// discoverFontPath 按常见部署环境搜索可用字体文件路径。
func discoverFontPath() string {
	candidates := []string{
		`C:\Windows\Fonts\simhei.ttf`,
		`C:\Windows\Fonts\msyh.ttc`,
		`C:\Windows\Fonts\simsun.ttc`,
		"/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf",
		"/usr/share/fonts/truetype/noto/NotoSansCJK-Regular.ttc",
		"/usr/share/fonts/opentype/noto/NotoSansCJK-Regular.ttc",
	}
	for _, candidate := range candidates {
		if candidate == "" {
			continue
		}
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	return ""
}

// drawLogo 把学校 Logo 缩放后绘制到成绩单页眉区域。
func drawLogo(canvas *image.RGBA, reader io.Reader, x, y int) (int, error) {
	if reader == nil {
		return y, fmt.Errorf("学校Logo为空")
	}
	logo, _, err := image.Decode(reader)
	if err != nil {
		return y, err
	}

	const targetSize = 120
	dstRect := image.Rect(x, y, x+targetSize, y+targetSize)
	xdraw.CatmullRom.Scale(canvas, dstRect, logo, logo.Bounds(), imagedraw.Over, nil)
	return y + targetSize + 12, nil
}

// drawText 在指定坐标绘制左对齐文本。
func drawText(canvas *image.RGBA, face font.Face, x, y int, value string, clr color.Color) {
	if face == nil {
		return
	}
	d := &font.Drawer{
		Dst:  canvas,
		Src:  image.NewUniform(clr),
		Face: face,
		Dot:  fixed.P(x, y),
	}
	d.DrawString(safeText(value))
}

// drawRightText 在指定坐标绘制右对齐文本。
func drawRightText(canvas *image.RGBA, face font.Face, x, y int, value string, clr color.Color) {
	if face == nil {
		return
	}
	d := &font.Drawer{
		Dst:  canvas,
		Src:  image.NewUniform(clr),
		Face: face,
	}
	width := d.MeasureString(safeText(value)).Round()
	d.Dot = fixed.P(x-width, y)
	d.DrawString(safeText(value))
}

// drawLine 在画布上绘制直线，用于表格分隔线和区域边界。
func drawLine(canvas *image.RGBA, x1, y1, x2, y2 int, clr color.Color) {
	if y1 == y2 {
		for x := x1; x <= x2; x++ {
			canvas.Set(x, y1, clr)
		}
		return
	}
	dx := float64(x2 - x1)
	dy := float64(y2 - y1)
	steps := int(math.Max(math.Abs(dx), math.Abs(dy)))
	for i := 0; i <= steps; i++ {
		x := x1 + int(float64(i)*dx/float64(steps))
		y := y1 + int(float64(i)*dy/float64(steps))
		canvas.Set(x, y, clr)
	}
}

// drawRect 绘制填充矩形，用于表头和块级背景区域。
func drawRect(canvas *image.RGBA, rect image.Rectangle, clr color.Color) {
	imagedraw.Draw(canvas, rect, image.NewUniform(clr), image.Point{}, imagedraw.Src)
}

// truncateText 对过长课程名做裁剪，避免表格列宽被破坏。
func truncateText(value string, maxChars int) string {
	runes := []rune(strings.TrimSpace(value))
	if len(runes) <= maxChars {
		return string(runes)
	}
	return string(runes[:maxChars-1]) + "…"
}

// safeText 统一处理空文本展示，避免 PDF 中出现空白占位。
func safeText(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "-"
	}
	return trimmed
}

// buildSerialNumber 生成默认成绩单防伪编号。
func buildSerialNumber() string {
	raw := defaultSerialPrefix + "-" + time.Now().UTC().Format("20060102150405")
	return strings.ToUpper(raw)
}

// buildSingleImagePDF 将 JPEG 图像直接封装为单页 PDF 文档。
// 这个实现只负责当前成绩单场景所需的最小 PDF 结构，避免为单页导出引入额外复杂依赖。
func buildSingleImagePDF(jpegData []byte, widthPx, heightPx int) *bytes.Buffer {
	pageWidthPt := pxToPt(widthPx)
	pageHeightPt := pxToPt(heightPx)
	content := fmt.Sprintf(
		"q\n%.2f 0 0 %.2f 0 0 cm\n/Im0 Do\nQ\n",
		pageWidthPt,
		pageHeightPt,
	)

	var buf bytes.Buffer
	offsets := make([]int, 0, 6)
	writeObject := func(id int, body []byte) {
		offsets = append(offsets, buf.Len())
		_, _ = fmt.Fprintf(&buf, "%d 0 obj\n", id)
		_, _ = buf.Write(body)
		_, _ = buf.WriteString("\nendobj\n")
	}

	_, _ = buf.WriteString("%PDF-1.4\n")
	_, _ = buf.WriteString("%" + string([]byte{0xE2, 0xE3, 0xCF, 0xD3}) + "\n")

	writeObject(1, []byte("<< /Type /Catalog /Pages 2 0 R >>"))
	writeObject(2, []byte("<< /Type /Pages /Count 1 /Kids [3 0 R] >>"))
	writeObject(3, []byte(fmt.Sprintf(
		"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 %.2f %.2f] /Resources << /XObject << /Im0 4 0 R >> >> /Contents 5 0 R >>",
		pageWidthPt,
		pageHeightPt,
	)))
	writeObject(4, buildImageObject(jpegData, widthPx, heightPx))
	writeObject(5, []byte(fmt.Sprintf("<< /Length %d >>\nstream\n%sendstream", len(content), content)))

	xrefOffset := buf.Len()
	_, _ = fmt.Fprintf(&buf, "xref\n0 %d\n", len(offsets)+1)
	_, _ = buf.WriteString("0000000000 65535 f \n")
	for _, offset := range offsets {
		_, _ = fmt.Fprintf(&buf, "%010d 00000 n \n", offset)
	}
	_, _ = fmt.Fprintf(&buf, "trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF", len(offsets)+1, xrefOffset)
	return &buf
}

// buildImageObject 构建 PDF 中的图像对象定义。
func buildImageObject(jpegData []byte, widthPx, heightPx int) []byte {
	header := fmt.Sprintf(
		"<< /Type /XObject /Subtype /Image /Width %d /Height %d /ColorSpace /DeviceRGB /BitsPerComponent 8 /Filter /DCTDecode /Length %d >>\nstream\n",
		widthPx,
		heightPx,
		len(jpegData),
	)
	footer := "\nendstream"
	body := make([]byte, 0, len(header)+len(jpegData)+len(footer))
	body = append(body, []byte(header)...)
	body = append(body, jpegData...)
	body = append(body, []byte(footer)...)
	return body
}

// pxToPt 将渲染像素换算为 PDF 使用的 point 单位。
func pxToPt(px int) float64 {
	return float64(px) * 72.0 / pageEstimatedDPI
}

// max 返回两个整数中的较大值，用于动态页面高度估算。
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// ExportBase64 生成可直接嵌入 API 返回的 Base64 PDF 字符串。
func ExportBase64(data *TranscriptData) (string, error) {
	buf, err := GenerateTranscript(data)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(buf.Bytes()), nil
}

// SuggestObjectName 根据成绩单编号生成对象存储路径。
func SuggestObjectName(data *TranscriptData) string {
	serial := strings.TrimSpace(data.SerialNumber)
	if serial == "" {
		serial = buildSerialNumber()
	}
	normalized := strings.ToLower(strings.ReplaceAll(serial, " ", "_"))
	normalized = strings.ReplaceAll(normalized, ":", "_")
	normalized = strings.ReplaceAll(normalized, "/", "_")
	return filepath.ToSlash("transcripts/" + normalized + ".pdf")
}

// SuggestFileName 根据成绩单信息生成下载文件名。
func SuggestFileName(data *TranscriptData) string {
	name := strings.TrimSpace(data.StudentName)
	if name == "" {
		name = "student"
	}
	studentNo := strings.TrimSpace(data.StudentNo)
	if studentNo == "" {
		studentNo = strconv.FormatInt(time.Now().Unix(), 10)
	}
	return fmt.Sprintf("%s_%s_成绩单.pdf", name, studentNo)
}
