// pdfgen generates PDF 1.7 files to an io.Writer
package pdfgen

import (
	"fmt"
	"io"
	"io/ioutil"
	"math"
	"os"
	"strings"
)

// PDFDoc defines the document structure.
type PDFDoc struct {
	Writer        io.Writer
	width, height float64
	fontnames     []string
	objectcount   int
}

var fontmap = map[string]string{"sans": "Helvetica", "serif": "Times-Roman", "mono": "Courier", "symbol": "Zapf-Dingbats"}

const (
	rectfmt    = "%s rg %.3f %.3f %.3f %.3f re f\n"
	linefmt    = "%.3f w %s RG %.3f %.3f m %.3f %.3f l S\n"
	curvefmt   = "%.3f w %s RG %.3f %.3f m %.3f %.3f %.3f %.3f v S\n"
	arcfmt     = "%.3f %.3f m %.3f %.3f %.3f %.3f v S\n"
	fillarcfmt = "%.3f %.3f m %.3f %.3f l %.3f %.3f %.3f %.3f v F\n"
	endfmt     = "trailer\n<</Size %d /Root 1 0 R >>\n%%%%EOF\n"
	textfmt    = "BT /%s %.3f Tf %.3f %.3f Td %s rg (%s) Tj ET\n"
	newpagefmt = "%d 0 obj\n<</Type /Page /Parent 1 0 R /Resources 2 0 R /Contents %d 0 R>>\nendobj\n\n%d 0 obj\n<</Length 0>>\nstream\n"
	colorfmt   = "%.3f %.3f %.3f"
	imagefmt   = "<</Type /XObject\n/Subtype /Image\n/Width %d\n/Height %d\n/ColorSpace /DeviceRGB\n/BitsPerComponent 8\n/Length %d>>\n"
	pagefmt    = "] /Count %d /MediaBox [0 0 %v %v]>>\nendobj\n\n"
	resfmt     = "2 0 obj\n<< /Font\n"
	fontfmt    = "<< /%s << /Type /Font /Subtype /Type1 /BaseFont /%s >>\n"
)

// NewDoc initializes the document structure.
func NewDoc(w io.Writer, pagewidth, pageheight float64) *PDFDoc {
	return &PDFDoc{
		Writer:      w,
		width:       pagewidth,
		height:      pageheight,
		fontnames:   []string{fontmap["sans"], fontmap["serif"], fontmap["mono"], fontmap["symbol"]},
		objectcount: 0,
	}
}

// Init begins the document.
func (p *PDFDoc) Init(n int) {
	fmt.Fprintln(p.Writer, "%PDF-1.5")
	p.root(n)
	p.resources()
}

// pdfstring returns an escaped string
func pdfstring(s string) string {
	s = strings.Replace(s, "\\", "\\\\", -1)
	s = strings.Replace(s, "(", "\\(", -1)
	s = strings.Replace(s, ")", "\\)", -1)
	s = strings.Replace(s, "\r", "\\r", -1)
	return s
}

// root defines the document root
func (p *PDFDoc) root(npages int) {
	// Object 1 is the root, object 2 is resources.
	// page references begin at 3, with the contents as the next sequential reference.
	// For example 3 -> 4, 5 -> 6, etc.
	fmt.Fprintf(p.Writer, "1 0 obj\n<</Type /Catalog /Pages 3 0 R /Kids [")
	for i, objref := 0, 3; i < npages; i++ {
		fmt.Fprintf(p.Writer, "%d 0 R ", objref)
		objref += 2
	}
	fmt.Fprintf(p.Writer, pagefmt, npages, p.width, p.height)
	p.objectcount++
}

// Resources defines page resources: fonts, etc.
func (p *PDFDoc) resources() {
	f := p.fontnames[0]
	fmt.Fprint(p.Writer, resfmt)
	//for _, f := range p.fontnames {
	fmt.Fprintf(p.Writer, fontfmt, f, f)
	//}
	fmt.Fprintln(p.Writer, ">>\n>>\nendobj")
	p.objectcount++
}

// EndPage closes out a page
func (p *PDFDoc) EndPage() {
	fmt.Fprintf(p.Writer, "endstream\nendobj\n\n")
	p.objectcount++
}

// EndDoc closes out the document
func (p *PDFDoc) EndDoc() {
	fmt.Fprintf(p.Writer, endfmt, p.objectcount)
}

// NewPage sets up a new page
// page references begin at 3, with the contents as the next sequential reference.
func (p *PDFDoc) NewPage(n int) {
	obj := (2 * n) + 1
	ref := obj + 1
	fmt.Fprintf(p.Writer, newpagefmt, obj, ref, ref)
	p.objectcount++
}

// pdfcolor converts a color string to the PDF (RGB) format
func pdfcolor(color string) string {
	r, g, b := colorlookup(color)
	return fmt.Sprintf(colorfmt, float64(r)/255.0, float64(g)/255.0, float64(b)/255.0)
}

// placeimage places an image
func (p *PDFDoc) placeimage(x, y, w, h float64, id string) {
	fmt.Fprintf(p.Writer, "q %.2f 0 0 %.2f %.2f %.2f cm /I%s Do Q\n", w, h, x, y, id)
}

// Text draws attributed (font, size, color) text at a (x,y) location
func (p *PDFDoc) Text(x, y float64, s, font string, size float64, color string) {
	fmt.Fprintf(p.Writer, textfmt, fontmap[font], size, x, y, pdfcolor(color), pdfstring(s))
}

// Image places an image at the (x,y) location
func (p *PDFDoc) Image(x, y float64, width, height int, name string) {
	r, err := os.Open(name)
	if err != nil {
		return
	}
	data, err := ioutil.ReadAll(r)
	fmt.Fprintf(p.Writer, imagefmt, width, height, len(data))
	fmt.Fprintf(p.Writer, "stream\n")
	io.WriteString(p.Writer, string(data))
	fmt.Fprintf(p.Writer, "\nendstream\n")
}

// Polygon draws a colored polygon
func (p *PDFDoc) Polygon(x []float64, y []float64, color string) {
	if len(x) != len(y) {
		return
	}
	fmt.Fprintf(p.Writer, "%s rg %v %v m", pdfcolor(color), x[0], y[0])
	for i := 1; i < len(x); i++ {
		fmt.Fprintf(p.Writer, " %v %v l", x[i], y[i])
	}
	fmt.Fprintf(p.Writer, " %v %v l f\n", x[0], y[0])
}

// Line draws a line with specified stroke color and width
func (p *PDFDoc) Line(x1, y1, x2, y2, sw float64, color string) {
	fmt.Fprintf(p.Writer, linefmt, sw, pdfcolor(color), x1, y1, x2, y2)
}

// Rect draws a colored rectangle with the upper left at (x,y)
func (p *PDFDoc) Rect(x, y, w, h float64, color string) {
	fmt.Fprintf(p.Writer, rectfmt, pdfcolor(color), x, y, w, h)
}

// Square draws a colored square with the upper left at (x,y)
func (p *PDFDoc) Square(x, y, w float64, color string) {
	p.Rect(x, y, w, w, color)
}

// Curve draws a quadratic Bezier curve at the specified stroke color and width
func (p *PDFDoc) Curve(x1, y1, x2, y2, x3, y3, sw float64, color string) {
	fmt.Fprintf(p.Writer, curvefmt, sw, pdfcolor(color), x1, y1, x2, y2, x3, y3)
}

// Circle draws a color filled circle
func (p *PDFDoc) Circle(x, y, r float64, color string) {
	p.FillArc(x, y, r, r, 0, 360, color)
}

// Ellipse draws an color filled ellipse
func (p *PDFDoc) Ellipse(x, y, w, h float64, color string) {
	p.FillArc(x, y, w, h, 0, 360, color)
}

func arcdata(i int, x, y, w, h, angle1, angle2 float64) (float64, float64, float64, float64, float64, float64) {
	const n = 16

	angle1 = angle1 * (math.Pi / 180)
	angle2 = angle2 * (math.Pi / 180)
	p1 := float64(i+0) / n
	p2 := float64(i+1) / n
	a1 := angle1 + (angle2-angle1)*p1
	a2 := angle1 + (angle2-angle1)*p2
	x0 := x + w*math.Cos(a1)
	y0 := y + h*math.Sin(a1)
	x1 := x + w*math.Cos(a1+(a2-a1)/2)
	y1 := y + h*math.Sin(a1+(a2-a1)/2)
	x2 := x + w*math.Cos(a2)
	y2 := y + h*math.Sin(a2)
	cx := 2*x1 - x0/2 - x2/2
	cy := 2*y1 - y0/2 - y2/2
	return x0, y0, cx, cy, x2, y2
}

// Arc draws an filled elliptical arc, using a series of quadratic Bezier curves
func (p *PDFDoc) FillArc(x, y, w, h, angle1, angle2 float64, color string) {
	const n = 16
	fmt.Fprintf(p.Writer, "%s rg\n", pdfcolor(color))
	for i := 0; i < n; i++ {
		x0, y0, cx, cy, x2, y2 := arcdata(i, x, y, w, h, angle1, angle2)
		fmt.Fprintf(p.Writer, fillarcfmt, x, y, x0, y0, cx, cy, x2, y2)
	}
}

// Arc strokes an elliptical arc, using a series of quadratic Bezier curves
func (p *PDFDoc) Arc(x, y, w, h, angle1, angle2, sw float64, color string) {
	const n = 16
	fmt.Fprintf(p.Writer, "%s RG %.3f w\n", pdfcolor(color), sw)
	for i := 0; i < n; i++ {
		x0, y0, cx, cy, x2, y2 := arcdata(i, x, y, w, h, angle1, angle2)
		fmt.Fprintf(p.Writer, arcfmt, x0, y0, cx, cy, x2, y2)
	}
}
