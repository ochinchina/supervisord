package xmlrpcclient

import (
	"encoding/xml"
	"io"
	"strings"
)

type XmlPath struct {
	ElemNames []string
}

func NewXmlPath() *XmlPath {
	return &XmlPath{ElemNames: make([]string, 0)}
}

func (xp *XmlPath) AddChildren(names ...string) {
	for _, name := range names {
		xp.ElemNames = append(xp.ElemNames, name)
	}
}
func (xp *XmlPath) AddChild(elemName string) {
	xp.ElemNames = append(xp.ElemNames, elemName)
}

func (xp *XmlPath) RemoveLast() {
	if len(xp.ElemNames) > 0 {
		xp.ElemNames = xp.ElemNames[0 : len(xp.ElemNames)-1]
	}
}

func (xp *XmlPath) Equals(other *XmlPath) bool {
	if len(xp.ElemNames) != len(other.ElemNames) {
		return false
	}

	for i := len(xp.ElemNames) - 1; i >= 0; i -= 1 {
		if xp.ElemNames[i] != other.ElemNames[i] {
			return false
		}
	}
	return true
}
func (xp *XmlPath) String() string {
	return strings.Join(xp.ElemNames, "/")
}

type XmlLeafProcessor func(value string)
type XmlNonLeafProcessor func()

type XmlProcessorManager struct {
	leafProcessors    map[string]XmlLeafProcessor
	nonLeafProcessors map[string]XmlNonLeafProcessor
}

func NewXmlProcessorManager() *XmlProcessorManager {
	return &XmlProcessorManager{leafProcessors: make(map[string]XmlLeafProcessor),
		nonLeafProcessors: make(map[string]XmlNonLeafProcessor)}
}

func (xpm *XmlProcessorManager) AddLeafProcessor(path string, processor XmlLeafProcessor) {
	xpm.leafProcessors[path] = processor
}

func (xpm *XmlProcessorManager) AddNonLeafProcessor(path string, processor XmlNonLeafProcessor) {
	xpm.nonLeafProcessors[path] = processor
}

func (xpm *XmlProcessorManager) ProcessLeafNode(path string, data string) {
	if processor, ok := xpm.leafProcessors[path]; ok {
		processor(data)
	}
}

func (xpm *XmlProcessorManager) ProcessNonLeafNode(path string) {
	if processor, ok := xpm.nonLeafProcessors[path]; ok {
		processor()
	}
}

func (xpm *XmlProcessorManager) ProcessXml(reader io.Reader) {
	decoder := xml.NewDecoder(reader)
	var curData xml.CharData
	curPath := NewXmlPath()

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
