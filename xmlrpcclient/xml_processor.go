package xmlrpcclient

import (
	"encoding/xml"
	"io"
	"strings"
)

// XMLPath represent the XML path in array
type XMLPath struct {
	ElemNames []string
}

// NewXMLPath create a new XMLPath object
func NewXMLPath() *XMLPath {
	return &XMLPath{ElemNames: make([]string, 0)}
}

// AddChildren append paths to the XMLPath
func (xp *XMLPath) AddChildren(names ...string) {
	for _, name := range names {
		xp.ElemNames = append(xp.ElemNames, name)
	}
}

// AddChild add a child to the path
func (xp *XMLPath) AddChild(elemName string) {
	xp.ElemNames = append(xp.ElemNames, elemName)
}

// RemoveLast remove the last element from path
func (xp *XMLPath) RemoveLast() {
	if len(xp.ElemNames) > 0 {
		xp.ElemNames = xp.ElemNames[0 : len(xp.ElemNames)-1]
	}
}

// Equals check if this XMLPath object equals other XMLPath object
func (xp *XMLPath) Equals(other *XMLPath) bool {
	if len(xp.ElemNames) != len(other.ElemNames) {
		return false
	}

	for i := len(xp.ElemNames) - 1; i >= 0; i-- {
		if xp.ElemNames[i] != other.ElemNames[i] {
			return false
		}
	}
	return true
}

// String convert the XMLPath to string
func (xp *XMLPath) String() string {
	return strings.Join(xp.ElemNames, "/")
}

// XMLLeafProcessor the XML leaf element process function
type XMLLeafProcessor func(value string)

// XMLNonLeafProcessor the non-leaf element process function
type XMLNonLeafProcessor func()

// XMLProcessorManager the xml processor based on the XMLPath
type XMLProcessorManager struct {
	leafProcessors    map[string]XMLLeafProcessor
	nonLeafProcessors map[string]XMLNonLeafProcessor
}

// NewXMLProcessorManager create a new XMLProcessorManager object
func NewXMLProcessorManager() *XMLProcessorManager {
	return &XMLProcessorManager{leafProcessors: make(map[string]XMLLeafProcessor),
		nonLeafProcessors: make(map[string]XMLNonLeafProcessor)}
}

// AddLeafProcessor add a leaf processor for the xml path
func (xpm *XMLProcessorManager) AddLeafProcessor(path string, processor XMLLeafProcessor) {
	xpm.leafProcessors[path] = processor
}

// AddNonLeafProcessor add a non-leaf processor for the xml path
func (xpm *XMLProcessorManager) AddNonLeafProcessor(path string, processor XMLNonLeafProcessor) {
	xpm.nonLeafProcessors[path] = processor
}

// ProcessLeafNode process the leaf element with xml path and its value
func (xpm *XMLProcessorManager) ProcessLeafNode(path string, data string) {
	if processor, ok := xpm.leafProcessors[path]; ok {
		processor(data)
	}
}

// ProcessNonLeafNode process the non-leaf element based on the xml path
func (xpm *XMLProcessorManager) ProcessNonLeafNode(path string) {
	if processor, ok := xpm.nonLeafProcessors[path]; ok {
		processor()
	}
}

// ProcessXML read the xml from reader and process it
func (xpm *XMLProcessorManager) ProcessXML(reader io.Reader) {
	decoder := xml.NewDecoder(reader)
	var curData xml.CharData
	curPath := NewXMLPath()

	for {
		tk, err := decoder.Token()
		if err != nil {
			break
		}

		switch tk.(type) {
		case xml.StartElement:
			startElem, _ := tk.(xml.StartElement)
			curPath.AddChild(startElem.Name.Local)
			curData = nil
		case xml.CharData:
			data, _ := tk.(xml.CharData)
			curData = data.Copy()
		case xml.EndElement:
			if curData != nil {
				xpm.ProcessLeafNode(curPath.String(), string(curData))
			} else {
				xpm.ProcessNonLeafNode(curPath.String())
			}
			curPath.RemoveLast()
		}
	}
}
