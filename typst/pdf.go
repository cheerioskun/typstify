package typst

import "fmt"

type PdfSpec interface {
	fmt.Stringer
	Argument() string
}

type PdfVersion string
type PdfStandard string

const (
	// versions
	PDF1_4 PdfVersion = "PDF 1.4"
	PDF1_5 PdfVersion = "PDF 1.5"
	PDF1_6 PdfVersion = "PDF 1.6"
	PDF1_7 PdfVersion = "PDF 1.7"
	PDF2_0 PdfVersion = "PDF 2.0"
	// standards
	A1b PdfStandard = "PDF/A-1b"
	A1a PdfStandard = "PDF/A-1a"
	A2b PdfStandard = "PDF/A-2b"
	A2u PdfStandard = "PDF/A-2u"
	A2a PdfStandard = "PDF/A-2a"
	A3b PdfStandard = "PDF/A-3b"
	A3u PdfStandard = "PDF/A-3u"
	A3a PdfStandard = "PDF/A-3a"
	A4  PdfStandard = "PDF/A-4"
	A4f PdfStandard = "PDF/A-4f"
	A4e PdfStandard = "PDF/A-4e"
	Ua1 PdfStandard = "PDF/UA-1"
)

var (
	PdfVersions  = []PdfVersion{PDF1_4, PDF1_5, PDF1_6, PDF1_7, PDF2_0}
	PdfStandards = []PdfStandard{A1b, A1a, A2b, A2u, A2a, A3b, A3u, A3a, A4, A4f, A4e, Ua1}
)

func (p PdfVersion) String() string {
	return string(p)
}

func (p PdfVersion) Argument() string {
	switch p {
	case PDF1_4:
		return "1.4"
	case PDF1_5:
		return "1.5"
	case PDF1_6:
		return "1.6"
	case PDF1_7:
		return "1.7"
	case PDF2_0:
		return "2.0"
	default:
		return ""
	}
}

func (p PdfStandard) String() string {
	return string(p)
}

func (p PdfStandard) Argument() string {
	switch p {
	case A1b:
		return "a-1b"
	case A1a:
		return "a-1a"
	case A2b:
		return "a-2b"
	case A2u:
		return "a-2u"
	case A2a:
		return "a-2a"
	case A3b:
		return "a-3b"
	case A3u:
		return "a-3u"
	case A3a:
		return "a-3a"
	case A4:
		return "a-4"
	case A4f:
		return "a-4f"
	case A4e:
		return "a-4e"
	case Ua1:
		return "ua-1"
	default:
		return ""
	}
}

func (p PdfStandard) Features() []string {
	switch p {
	case A1b:
		return []string{
			"Preserves visual appearance for long-term archiving",
			"Does not support text extraction or accessibility",
			"Based on PDF 1.4",
			"Recommended for scanned or image-based documents",
		}
	case A1a:
		return []string{
			"Preserves visual appearance and document structure",
			"Supports text search, copying, and accessibility",
			"Based on PDF 1.4",
			"Recommended for digital documents requiring accessibility",
		}
	case A2b:
		return []string{
			"Improved compression and smaller file size",
			"Supports transparency, layers, and JPEG2000 images",
			"Ensures visual consistency for archiving",
			"Based on PDF 1.5",
			"Recommended for modern, visually complex documents",
		}
	case A2u:
		return []string{
			"Ensures visual consistency and searchable text",
			"All text must have Unicode mapping",
			"Based on PDF 1.5",
			"Recommended for documents needing reliable text extraction",
		}
	case A2a:
		return []string{
			"Combines full accessibility with visual preservation",
			"Fully tagged and structured for assistive technologies",
			"Based on PDF 1.5",
			"Recommended for accessible archives of digital documents",
		}
	case A3b:
		return []string{
			"Like A-2b but allows embedding source files (e.g., XML, CSV)",
			"Ensures long-term visual preservation",
			"Based on PDF 1.7",
			"Ideal for reports requiring attached data or metadata files",
		}
	case A3u:
		return []string{
			"Like A-3b but ensures searchable, Unicode text",
			"Based on PDF 1.7",
			"Recommended for data-rich, text-based documents with attachments",
		}
	case A3a:
		return []string{
			"Fully accessible version of A-3u",
			"Supports embedded files and complete tagging",
			"Based on PDF 1.7",
			"Recommended for accessible reports with embedded data",
		}
	case A4:
		return []string{
			"Modern archival standard based on PDF 2.0",
			"More flexible structure and simplified conformance levels",
			"Recommended for general-purpose long-term archiving",
		}
	case A4f:
		return []string{
			"Like A-4 but allows embedded source or data files",
			"Based on PDF 2.0",
			"Recommended for modern archives with attached content",
		}
	case A4e:
		return []string{
			"Like A-4 but supports 3D models and engineering data",
			"Based on PDF 2.0",
			"Recommended for technical and engineering documentation",
		}
	case Ua1:
		return []string{
			"Accessibility-focused standard based on PDF 1.7",
			"Requires tagging and proper reading order",
			"Ensures compatibility with screen readers and assistive tools",
			"Recommended for accessible, publication-ready PDFs",
		}
	default:
		return nil
	}
}

// MinVersion returns the minimum PDF version that supports this standard.
func (p PdfStandard) MinVersion() PdfVersion {
	switch p {
	case A1b, A1a:
		return PDF1_4
	case A2b, A2u, A2a:
		return PDF1_5
	case A3b, A3u, A3a, Ua1:
		return PDF1_7
	case A4, A4f, A4e:
		return PDF2_0
	default:
		return ""
	}
}

// Compatible checks if the given PDF version is compatible with the standard.
func (p PdfStandard) Compatible(version PdfVersion) bool {
	switch p {
	case A1a, A1b:
		// PDF/A-1 (ISO 19005-1) is based on and restricted to PDF 1.4.
		return version == PDF1_4

	case A2a, A2b, A2u:
		// PDF/A-2 (ISO 19005-2) is based on and restricted to PDF 1.7.
		// Files must declare PDF 1.7.
		return version == PDF1_7

	case A3a, A3b, A3u:
		// PDF/A-3 (ISO 19005-3) is based on and restricted to PDF 1.7.
		// Files must declare PDF 1.7.
		return version == PDF1_7

	case Ua1:
		// PDF/UA-1 (ISO 14289-1) is based on and restricted to PDF 1.7.
		// Files must declare PDF 1.7.
		return version == PDF1_7

	case A4, A4f, A4e:
		// PDF/A-4 (ISO 19005-5) is based on and restricted to PDF 2.0.
		return version == PDF2_0

	default:
		return false
	}
}
