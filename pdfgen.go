// pdfgen generates PDF 1.7 files to an io.Writer
package pdfgen

import (
	"fmt"
	"io"
	"strings"
)

type PDFDoc struct {
	Writer        io.Writer
	width, height float64
	fontnames     []string
	objectcount	int
}

var fontmap = map[string]string{"sans":"Helvetica", "serif":"Times-Roman", "mono":"Courier", "symbol":"Zapf-Dingbats"}

func NewDoc(w io.Writer, pagewidth, pageheight float64) *PDFDoc {
	return &PDFDoc{
		Writer:    w,
		width:     pagewidth,
		height:    pageheight,
		fontnames: []string{fontmap["sans"], fontmap["serif"], fontmap["mono"], fontmap["symbol"]},
		objectcount: 0,
	}
}

const (
	rectfmt = "%s rg %v %v %v %v re f\n"
	linefmt = "%v w %s RG %v %v m %v %v l S\n"
	curvefmt = "%v w %s RG %v %v m %v %v %v %v v S\n"
	endfmt = "trailer\n<</Size %d /Root 1 0 R >>\n%%%%EOF\n"
	textfmt = "BT /%s %v Tf %v %v Td %s rg (%s) Tj ET\n"
	newpagefmt = "%d 0 obj\n<</Type /Page /Parent 1 0 R /Resources 2 0 R /Contents %d 0 R>>\nendobj\n\n%d 0 obj\n<</Length 0>>\nstream\n"
	colorfmt = "%.1f %.1f %.1f"
	imagefmt = "<</Type /XObject /Subtype /Image /Width %d /Height %d /ColorSpace /DeviceRGB /BitsPerComponent 8 /Filter /DCTDecode /Length %d>>\n"
	pagefmt = "] /Count %d /MediaBox [0 0 %v %v]>>\nendobj\n\n"
	resfmt = "2 0 obj\n<< /Font\n"
	fontfmt = "<< /%s << /Type /Font /Subtype /Type1 /BaseFont /%s >>\n"
)

func (p *PDFDoc) Init(n int) {
	fmt.Fprintln(p.Writer, "%PDF-1.4")
	p.root(n)
	p.resources()
}

func pdfstring(s string) string {
	s = strings.Replace(s, "\\", "\\\\", -1)
	s = strings.Replace(s, "(", "\\(", -1)
	s = strings.Replace(s, ")", "\\)", -1)
	s = strings.Replace(s, "\r", "\\r", -1)
	return s
}

func (p *PDFDoc) root(npages int) {
	// Object 1 is the root, object 2 is resources). 
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

func (p *PDFDoc) resources() {
	f := p.fontnames[0]
	fmt.Fprint(p.Writer, resfmt)
	//for _, f := range p.fontnames {
		fmt.Fprintf(p.Writer, fontfmt, f, f)
	//}
	fmt.Fprintln(p.Writer, ">>\n>>\nendobj\n")
	p.objectcount++
}

func (p *PDFDoc) EndPage() {
	fmt.Fprintf(p.Writer, "endstream\nendobj\n\n")
	p.objectcount++
}

func (p *PDFDoc) EndDoc() {
	fmt.Fprintf(p.Writer, endfmt, p.objectcount)
}

func (p *PDFDoc) NewPage(n int) {
	fmt.Fprintf(p.Writer, newpagefmt, n+2, n+3, n+3)
	p.objectcount++
}

func pdfcolor(color string) string {
	r, g, b := colorlookup(color)
	return fmt.Sprintf(colorfmt, float64(r)/255.0, float64(g)/255.0, float64(b)/255.0)
}

func (p *PDFDoc) Text(x, y float64, s, font string, size float64, color string) {
	fmt.Fprintf(p.Writer, textfmt, fontmap[font], size, x, y, pdfcolor(color), pdfstring(s))
}

func (p *PDFDoc) Image(x, y float64, width, height int, name string) {
	length := 0
	fmt.Fprintf(p.Writer, imagefmt, width, height, length)
}

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

func (p *PDFDoc) Line(x1, y1, x2, y2, sw float64, color string) {
	fmt.Fprintf(p.Writer, linefmt, sw, pdfcolor(color), x1, y1, x2, y2)
}

func (p *PDFDoc) Ellipse(x, y, w, h float64) {
}

func (p *PDFDoc) Rect(x, y, w, h float64, color string) {
	fmt.Fprintf(p.Writer, rectfmt, pdfcolor(color), x, y, w, h)
}

func (p *PDFDoc) Arc(x, y, w, h, a1, a2 float64) {
}

func (p *PDFDoc) Curve(x1, y1, x2, y2, x3, y3, sw float64, color string) {
	fmt.Fprintf(p.Writer, curvefmt, sw, pdfcolor(color), x1, y1, x2, y2, x3, y3)
}