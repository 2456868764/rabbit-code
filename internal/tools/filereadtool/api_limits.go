package filereadtool

// Mirrors restored-src/src/constants/apiLimits.ts (PDF + image client caps used by FileReadTool).

const (
	PDFTargetRawSize            = 20 * 1024 * 1024
	APIPDFMaxPages              = 100
	PDFExtractSizeThreshold     = 3 * 1024 * 1024
	PDFMaxExtractSize           = 100 * 1024 * 1024
	PDFMaxPagesPerRead          = 20
	PDFAtMentionInlineThreshold = 10
	APIImageMaxBase64Size       = 5 * 1024 * 1024
	ImageTargetRawSize          = (APIImageMaxBase64Size * 3) / 4
	ImageResizeMaxDim           = 2000
)
