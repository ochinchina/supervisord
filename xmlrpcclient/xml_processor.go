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

// NewXMLPath creates new XMLPath object
func NewXMLPath() *XMLPath {
	return &XMLPath{ElemNames: make([]string, 0)}
}

// AddChildren appends paths to the XMLPath
func (xp *XMLPath) AddChildren(names ...string) {
	for _, name := range names {
		xp.ElemNames = append(xp.ElemNames, name)
	}
}

// AddChild adds child to the XMLPath
func (xp *XMLPath) AddChild(elemName string) {
	xp.ElemNames = append(xp.ElemNames, elemName)
}

// RemoveLast removes last element from XMLPath
func (xp *XMLPath) RemoveLast() {
	if len(xp.ElemNames) > 0 {
		xp.ElemNames = xp.ElemNames[0 : len(xp.ElemNames)-1]
	}
}

// Equals checks if XMLPath object equals given XMLPath object
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

// String converts XMLPath to string
func (xp *XMLPath) String() string {
	return strings.Join(xp.ElemNames, "/")
}

// XMLLeafProcessor the XML leaf element process function
type XMLLeafProcessor func(value string)

// XMLSwitchTypeProcessor the switch type process function
type XMLSwitchTypeProcessor func()

// XMLProcessorManager the xml processor based on the XMLPath
type XMLProcessorManager struct {
	leafProcessors       map[string]XMLLeafProcessor
	switchTypeProcessors map[string]XMLSwitchTypeProcessor
}

// NewXMLProcessorManager creates new XMLProcessorManager object
func NewXMLProcessorManager() *XMLProcessorManager {
	return &XMLProcessorManager{leafProcessors: make(map[string]XMLLeafProcessor),
		switchTypeProcessors: make(map[string]XMLSwitchTypeProcessor)}
}

// AddLeafProcessor adds leaf processor for the xml path
func (xpm *XMLProcessorManager) AddLeafProcessor(path string, processor XMLLeafProcessor) {
	xpm.leafProcessors[path] = processor
}

// AddSwitchTypeProcessor adds switch type processor for the xml path
func (xpm *XMLProcessorManager) AddSwitchTypeProcessor(path string, processor XMLSwitchTypeProcessor) {
	xpm.switchTypeProcessors[path] = processor
}

// ProcessLeafNode processes leaf element with xml path and its value
func (xpm *XMLProcessorManager) ProcessLeafNode(path string, data string) {
	if processor, ok := xpm.leafProcessors[path]; ok {
		processor(data)
	}
}

// ProcessSwitchTypeNode processes switch type based on the xml path
func (xpm *XMLProcessorManager) ProcessSwitchTypeNode(path string) {
	if processor, ok := xpm.switchTypeProcessors[path]; ok {
		processor()
	}
}

// ProcessXML reads xml from reader and process it
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
			}
			xpm.ProcessSwitchTypeNode(curPath.String())
			curPath.RemoveLast()
		}
	}
}
