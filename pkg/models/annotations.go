package models

// AnnotationType represents the type of annotation
type AnnotationType string

const (
	AnnotationTypeURLCitation AnnotationType = "url_citation"
	AnnotationTypeFile        AnnotationType = "file"
)

// Annotation represents an annotation in a message response
type Annotation struct {
	Type        AnnotationType  `json:"type"`
	URLCitation *URLCitation    `json:"url_citation,omitempty"`
	File        *FileAnnotation `json:"file,omitempty"`
}

// URLCitation represents a web search result citation
type URLCitation struct {
	URL        string `json:"url"`
	Title      string `json:"title"`
	Content    string `json:"content,omitempty"`
	StartIndex int    `json:"start_index"`
	EndIndex   int    `json:"end_index"`
}

// FileAnnotation represents parsed file information
type FileAnnotation struct {
	Filename string                 `json:"filename"`
	FileData map[string]interface{} `json:"file_data"`
}
