package bevtree

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/GodYY/gutils/assert"
	"github.com/pkg/errors"
)

// help function to create xml.Name in namespace bevtree.
func createXMLName(name string) xml.Name {
	return xml.Name{Space: "", Local: name}
}

// xml name for BevTree.
var xmlNameBevTree = ("bevtree")

// xml name for NodeType.
var xmlNameNodeType = ("nodetype")

// xml name for BevType
var xmlNameBevType = "bevtype"

// xml name for root node.
var xmlNameRoot = ("root")

// xml name for childs.
var xmlNameChilds = ("childs")

// xml name for child.
var xmlNameChild = ("child")

// xml name for limited.
var xmlNameLimited = ("limited")

// xml name for success on fail.
var xmlNameSuccessOnFail = ("successonfail")

// XMLUnmarshal is the interface implemented by objects that can marshal
// themselves into valid bevtree XML elements.
type XMLMarshaler interface {
	MarshalBTXML(*XMLEncoder, xml.StartElement) error
}

// XMLUnmarshaler is the interface implemented by objects that can unmarshal
// an bevtree XML element description of themselves.
type XMLUnmarshaler interface {
	UnmarshalBTXML(*XMLDecoder, xml.StartElement) error
}

// An Encoder writes bevtree XML data to an output stream.
type XMLEncoder struct {
	*xml.Encoder
}

// NewXMLEncoder returns a new encoder that writes to w.
func NewXMLEncoder(w io.Writer) *XMLEncoder {
	assert.Assert(w != nil, "writer nil")

	return &XMLEncoder{
		Encoder: xml.NewEncoder(w),
	}
}

// EncodeElement writes the bevtree XML encoding of v to the stream,
// using start as the outermost tag in the encoding.
//
// EncodeElement calls Flush before returning.
func (e *XMLEncoder) EncodeElement(v interface{}, start xml.StartElement) error {
	if marshaler, ok := v.(XMLMarshaler); ok {
		if err := marshaler.MarshalBTXML(e, start); err != nil {
			return err
		}

		return e.Flush()
	} else {
		return e.Encoder.EncodeElement(v, start)
	}
}

// EncodeSE writes the bevtree XML encoding of v to the stream
// between start and start.End(), using start as the outermost tag
// in the encoding.
//
// EncodeSE calls Flush before returning.
func (e *XMLEncoder) EncodeSE(start xml.StartElement, callback func(*XMLEncoder) error) error {
	if callback == nil {
		return errors.New("callback nil")
	}

	var err error

	if err = e.EncodeToken(start); err == nil {
		if err = callback(e); err == nil {
			if err = e.EncodeToken(start.End()); err == nil {
				if err = e.Flush(); err == nil {
					return nil
				}
			}
		}
	}

	return NewXMLTokenError(start, err)
}

// EncodeElementSE writes the bevtree XML encoding of v to the stream
// between start and start.End(), using start as the outermost tag
// in the encoding.
//
// EncodeElementSE calls Flush before returning.
func (e *XMLEncoder) EncodeElementSE(v interface{}, start xml.StartElement) error {
	if err := e.EncodeToken(start); err != nil {
		return NewXMLTokenError(start, err)
	}

	if err := e.EncodeElement(v, start); err != nil {
		return err
	}

	if err := e.EncodeToken(start.End()); err != nil {
		return NewXMLTokenError(start.End(), err)
	}

	if err := e.Flush(); err != nil {
		return NewXMLTokenError(start, err)
	}

	return nil
}

// EncodeNode invoke EncodeStartEnd to write the bevtree XML encoding of
// node n to the stream with name as element name:
// <name.Space:name.Local nodetype="nodetype">
// ...
// </name.Space:name.Local>
func (e *XMLEncoder) EncodeNode(n Node, start xml.StartElement) error {
	if ntAttr, err := n.NodeType().MarshalXMLAttr(createXMLName(xmlNameNodeType)); err == nil {
		start.Attr = append(start.Attr, ntAttr)
	} else {
		return errors.WithMessagef(err, "EncodeNode %s", xmlTokenToString(start))
	}

	return e.EncodeElement(n, start)
}

// A XMLDecoder represents an bevtree XML parser reading a particular
// input stream.
type XMLDecoder struct {
	*xml.Decoder
}

// NewXMLDecoder creates a new bevtree XML parser reading from r.
// If r does not implement io.ByteReader, NewXMLDecoder will
// do its own buffering.
func NewXMLDecoder(r io.Reader) *XMLDecoder {
	assert.Assert(r != nil, "reader nil")

	return &XMLDecoder{
		Decoder: xml.NewDecoder(r),
	}
}

// DecodeElement read element from start to parse info v.
func (d *XMLDecoder) DecodeElement(v interface{}, start xml.StartElement) error {
	if unmarshal, ok := v.(XMLUnmarshaler); ok {
		return unmarshal.UnmarshalBTXML(d, start)
	} else {
		return d.Decoder.DecodeElement(v, &start)
	}
}

// DecodeAt looks for the start element named startName, and invoke
// callback with it.
func (d *XMLDecoder) DecodeAt(startName xml.Name, callback func(*XMLDecoder, xml.StartElement) error) error {
	// var start xml.StartElement
	// startWasFound := false

	var err error
	var token xml.Token
	for token, err = d.Token(); err == nil; token, err = d.Token() {
		switch t := token.(type) {
		case xml.StartElement:
			if t.Name == startName {
				if err = callback(d, t); err != nil {
					return err
				} else {
					return nil
				}
			}

		case xml.EndElement:
			if t.Name == startName {
				return errors.Errorf("xml: DecodeAt <%s...>: primarily found end element", xmlNameToString(startName))
			}
		}
	}

	return errors.WithMessagef(err, "xml: DecodeAt <%s...>", xmlNameToString(startName))
}

// DecodeElementAt looks for the start element named startName, and invoke
// DecodeElement with it.
func (d *XMLDecoder) DecodeElementAt(v interface{}, name xml.Name) error {
	var err error
	var token xml.Token
	for token, err = d.Token(); err == nil; token, err = d.Token() {
		switch t := token.(type) {
		case xml.StartElement:
			if t.Name == name {
				if err = d.DecodeElement(v, t); err != nil {
					return err
				} else {
					return nil
				}
			}

		case xml.EndElement:
			if t.Name == name {
				return errors.Errorf("xml: DecodeElementAt <%s...>: primarily found end element", xmlNameToString(name))
			}
		}
	}

	return errors.WithMessagef(err, "xml: DecodeElementAt <%s...>", xmlNameToString(name))
}

// DecodeNodeAt first looks for the start element named startName of node;
// then, parse attr NodeType from it and create new node; finally, invoke
// DecodeElement with the start element and new node.
func (d *XMLDecoder) DecodeNodeAt(pnode *Node, name xml.Name) error {
	var err error
	var token xml.Token
	for token, err = d.Token(); err == nil; token, err = d.Token() {
		switch t := token.(type) {
		case xml.StartElement:
			if t.Name == name {
				nodeTypeXMLName := createXMLName(xmlNameNodeType)
				var node Node
				for _, attr := range t.Attr {
					if attr.Name == nodeTypeXMLName {
						var nodeType NodeType
						if err = nodeType.UnmarshalXMLAttr(attr); err == nil {
							node = getNodeMETAByType(nodeType).createNode()
						} else {
							return errors.WithMessagef(err, "xml: DecodeNodeAt %s: ", xmlTokenToString(t))
						}

						break
					}
				}

				if node == nil {
					return NewXMLAttrNotFoundError(t, nodeTypeXMLName)
				}

				if err = d.DecodeElement(node, t); err != nil {
					return err
				} else {
					*pnode = node
					return nil
				}
			}
		case xml.EndElement:
			if t.Name == name {
				return errors.Errorf("xml: DecodeNodeAt <%s...>: primarily found end element", xmlNameToString(name))
			}
		}
	}

	return errors.WithMessagef(err, "xml: DecodeNodeAt <%s...>", xmlNameToString(name))
}

// marshal t to valid XML attribute.
func (t NodeType) MarshalXMLAttr(name xml.Name) (xml.Attr, error) {
	return xml.Attr{Name: name, Value: t.String()}, nil
}

// unmarshal an XML attribute to t.
func (t *NodeType) UnmarshalXMLAttr(attr xml.Attr) error {
	if meta, ok := nodeName2META[attr.Value]; ok {
		*t = meta.typ
		return nil
	} else {
		return errors.Errorf("invalid nodeType %s", attr.Value)
	}
}

// MarshalXMLBevTree return an bevtree XML encoding of t.
func MarshalXMLBevTree(t *BevTree) ([]byte, error) {
	if t == nil {
		return nil, nil
	}

	var buf = bytes.NewBuffer(nil)
	e := NewXMLEncoder(buf)
	e.Indent("", "    ")

	start := xml.StartElement{Name: createXMLName(xmlNameBevTree)}
	if err := e.EncodeElementSE(t, start); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// UnmarshalXMLBevTree parses the bevtree XML-encoded BevTree
// data and stores the result in the BevTree pointed to by t.
func UnmarshalXMLBevTree(data []byte, t *BevTree) error {
	if data == nil || t == nil {
		return nil
	}

	var buf = bytes.NewReader(data)
	d := NewXMLDecoder(buf)

	return d.DecodeElementAt(t, createXMLName(xmlNameBevTree))
}

// EncodeBevTreeXMLFile works like MarshalXMLBevTree but write
// encoded data to file.
func EncodeBevTreeXMLFile(path string, t *BevTree) (err error) {
	if t == nil {
		return nil
	}

	var file *os.File

	file, err = os.Create(path)
	if err != nil {
		return err
	}

	defer func() {
		if e := file.Close(); err == nil {
			err = e
		}
	}()

	enc := NewXMLEncoder(file)

	start := xml.StartElement{Name: createXMLName(xmlNameBevTree)}
	if err := enc.EncodeElementSE(t, start); err != nil {
		return err
	}

	return nil
}

// DecodeBevTreeXMLFile works like UnmarshalXMLBevTree but read
// encoded data from file.
func DecodeBevTreeXMLFile(path string, t *BevTree) error {
	if t == nil {
		return nil
	}

	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	dec := NewXMLDecoder(file)

	return dec.DecodeElementAt(t, createXMLName(xmlNameBevTree))
}

func (t *BevTree) MarshalBTXML(e *XMLEncoder, start xml.StartElement) error {
	rootStart := xml.StartElement{Name: createXMLName(xmlNameRoot)}
	if err := e.EncodeElementSE(t.root, rootStart); err != nil {
		return NewXMLTokenError(start, err)
	}

	return nil
}

func (t *BevTree) UnmarshalBTXML(d *XMLDecoder, start xml.StartElement) error {
	if debug {
		log.Printf("BevTree.UnmarshalBTXML start:%v", start)
	}

	root := newRootNode()
	if err := d.DecodeElementAt(root, createXMLName(xmlNameRoot)); err != nil {
		return NewXMLTokenError(start, err)
	}

	t.root = root

	return nil
}

func (r *rootNode) MarshalBTXML(e *XMLEncoder, start xml.StartElement) error {
	if r.child == nil {
		return nil
	}

	if err := e.EncodeNode(r.child, xml.StartElement{Name: createXMLName(xmlNameChild)}); err != nil {
		return NewXMLTokenError(start, err)
	}

	return nil
}

func (r *rootNode) UnmarshalBTXML(d *XMLDecoder, start xml.StartElement) error {
	var child Node
	var err error

	if err = d.DecodeNodeAt(&child, createXMLName(xmlNameChild)); err == nil {
		if err = d.Skip(); err == nil {
			r.SetChild(child)
			return nil
		}
	}

	return NewXMLTokenError(start, err)
}

func (d *decoratorNode) marshalXML(e *XMLEncoder) error {
	if d.child == nil {
		return nil
	}

	return e.EncodeNode(d.child, xml.StartElement{Name: createXMLName(xmlNameChild)})
}

func (d *decoratorNode) unmarshalXML(dec *XMLDecoder) error {
	var child Node
	if err := dec.DecodeNodeAt(&child, createXMLName(xmlNameChild)); err != nil {
		return err
	}

	d.SetChild(child)
	return nil
}

func (i *InverterNode) MarshalBTXML(e *XMLEncoder, start xml.StartElement) error {
	if debug {
		log.Printf("InverterNode.MarshalBTXML start:%v", start)
	}

	if err := e.EncodeSE(start, func(e *XMLEncoder) error {
		return i.decoratorNode.marshalXML(e)
	}); err != nil {
		return NewXMLTokenError(start, err)
	}

	return nil
}

func (i *InverterNode) UnmarshalBTXML(d *XMLDecoder, start xml.StartElement) error {
	if debug {
		log.Printf("InverterNode.UnmarshalBTXML start:%v", start)
	}

	var err error

	if err = i.decoratorNode.unmarshalXML(d); err == nil {
		if err = d.Skip(); err == nil {
			i.child.SetParent(i)
			return nil
		}
	}

	return NewXMLTokenError(start, err)
}

func (s *SucceederNode) MarshalBTXML(e *XMLEncoder, start xml.StartElement) error {
	if debug {
		log.Printf("SucceederNode.MarshalBTXML start:%v", start)
	}

	if err := e.EncodeSE(start, func(e *XMLEncoder) error {
		return s.decoratorNode.marshalXML(e)
	}); err != nil {
		return NewXMLTokenError(start, err)
	}

	return nil
}

func (s *SucceederNode) UnmarshalBTXML(d *XMLDecoder, start xml.StartElement) error {
	if debug {
		log.Printf("SucceederNode.UnmarshalBTXML start:%v", start)
	}

	var err error

	if err = s.decoratorNode.unmarshalXML(d); err == nil {
		if err = d.Skip(); err == nil {
			s.child.SetParent(s)
			return nil
		}
	}

	return NewXMLTokenError(start, err)
}

func (r *RepeaterNode) MarshalBTXML(e *XMLEncoder, start xml.StartElement) error {
	if debug {
		log.Printf("RepeaterNode.MarshalBTXML start:%v", start)
	}

	if err := e.EncodeSE(start, func(e *XMLEncoder) error {

		if err := e.EncodeElement(r.limited, xml.StartElement{Name: createXMLName(xmlNameLimited)}); err != nil {
			return NewXMLTokenError(xml.StartElement{Name: createXMLName(xmlNameLimited)}, err)
		}

		return r.decoratorNode.marshalXML(e)

	}); err != nil {
		return NewXMLTokenError(start, err)
	}

	return nil
}

func (r *RepeaterNode) UnmarshalBTXML(d *XMLDecoder, start xml.StartElement) error {
	if debug {
		log.Printf("RepeaterNode.UnmarshalBTXML start:%v", start)
	}

	var err error

	if err = d.DecodeElementAt(&r.limited, createXMLName(xmlNameLimited)); err == nil {
		if err = r.decoratorNode.unmarshalXML(d); err == nil {
			if err = d.Skip(); err == nil {
				r.child.SetParent(r)
				return nil
			}
		}
	}

	return NewXMLTokenError(start, err)
}

func (r *RepeatUntilFailNode) MarshalBTXML(e *XMLEncoder, start xml.StartElement) error {
	if debug {
		log.Printf("RepeatUntilFailNode.MarshalBTXML start:%v", start)
	}

	if err := e.EncodeSE(start, func(e *XMLEncoder) error {

		if err := e.EncodeElement(r.successOnFail, xml.StartElement{Name: createXMLName(xmlNameSuccessOnFail)}); err != nil {
			return NewXMLTokenError(xml.StartElement{Name: createXMLName(xmlNameSuccessOnFail)}, err)
		}

		return r.decoratorNode.marshalXML(e)

	}); err != nil {
		return NewXMLTokenError(start, err)
	}

	return nil
}

func (r *RepeatUntilFailNode) UnmarshalBTXML(d *XMLDecoder, start xml.StartElement) error {
	if debug {
		log.Printf("RepeatUntilFailNode.UnmarshalBTXML start:%v", start)
	}

	var err error

	if err = d.DecodeElementAt(&r.successOnFail, createXMLName(xmlNameSuccessOnFail)); err == nil {
		if err = r.decoratorNode.unmarshalXML(d); err == nil {
			if err = d.Skip(); err == nil {
				r.child.SetParent(r)
				return nil
			}
		}
	}

	return NewXMLTokenError(start, err)
}

func (c *compositeNode) marshalXML(e *XMLEncoder) error {
	childCount := c.ChildCount()
	if childCount > 0 {
		var err error

		childsStart := xml.StartElement{Name: createXMLName(xmlNameChilds)}
		childsStart.Attr = append(childsStart.Attr, xml.Attr{Name: createXMLName("count"), Value: strconv.Itoa(childCount)})

		if err = e.EncodeToken(childsStart); err != nil {
			return NewXMLTokenError(childsStart, err)
		}

		for i := 0; i < childCount; i++ {
			child := c.Child(i)
			if err = e.EncodeNode(child, xml.StartElement{Name: createXMLName(xmlNameChild)}); err != nil {
				return errors.WithMessagef(err, "marshal No.%d child", i)
			}
		}

		if err = e.EncodeToken(childsStart.End()); err != nil {
			return NewXMLTokenError(childsStart, err)
		}

		return nil
	}

	return nil
}

func (c *compositeNode) unmarshalXML(d *XMLDecoder) error {
	return d.DecodeAt(createXMLName(xmlNameChilds), func(d *XMLDecoder, s xml.StartElement) error {
		xmlCountName := createXMLName("count")
		var childCount int
		var childs []Node
		for _, attr := range s.Attr {
			if attr.Name == xmlCountName {
				var err error
				if childCount, err = strconv.Atoi(attr.Value); err != nil {
					return errors.WithMessagef(err, "%s: unmarshal attribute \"%s\"", xmlTokenToString(s), xmlNameToString(xmlCountName))
				} else if childCount <= 0 {
					return XMLTokenErrorf(s, "attribute \"%s\" invalid", xmlNameToString(xmlCountName))
				} else {
					childs = make([]Node, 0, childCount)
				}
				break
			}
		}

		if childCount <= 0 {
			return nil
		}

		xmlChildName := createXMLName(xmlNameChild)
		for i := 0; i < childCount; i++ {
			var node Node
			if err := d.DecodeNodeAt(&node, xmlChildName); err != nil {
				return errors.WithMessagef(err, "unmarshal No.%d child", i)
			}

			childs = append(childs, node)
		}

		c.childs = childs
		return d.Skip()
	})
}

func (s *SequenceNode) MarshalBTXML(e *XMLEncoder, start xml.StartElement) error {
	if debug {
		log.Printf("SequenceNode.MarshalBTXML start:%v", start)
	}

	if err := e.EncodeSE(start, func(e *XMLEncoder) error {
		return s.compositeNode.marshalXML(e)
	}); err != nil {
		return NewXMLTokenError(start, err)
	}

	return nil
}

func (s *SequenceNode) UnmarshalBTXML(d *XMLDecoder, start xml.StartElement) error {
	if debug {
		log.Printf("SequenceNode.UnmarshalBTXML start:%v", start)
	}

	var err error

	if err = s.compositeNode.unmarshalXML(d); err == nil {
		if err = d.Skip(); err == nil {
			for _, v := range s.childs {
				v.SetParent(s)
			}
			return nil
		}
	}

	return NewXMLTokenError(start, err)
}

func (s *SelectorNode) MarshalBTXML(e *XMLEncoder, start xml.StartElement) error {
	if debug {
		log.Printf("SelectorNode.MarshalBTXML start:%v", start)
	}

	if err := e.EncodeSE(start, func(e *XMLEncoder) error {
		return s.compositeNode.marshalXML(e)
	}); err != nil {
		return NewXMLTokenError(start, err)
	}

	return nil
}

func (s *SelectorNode) UnmarshalBTXML(d *XMLDecoder, start xml.StartElement) error {
	if debug {
		log.Printf("SelectorNode.UnmarshalBTXML start:%v", start)
	}

	var err error

	if err = s.compositeNode.unmarshalXML(d); err == nil {
		if err = d.Skip(); err == nil {
			for _, v := range s.childs {
				v.SetParent(s)
			}
			return nil
		}
	}

	return NewXMLTokenError(start, err)
}

func (r *RandSequenceNode) MarshalBTXML(e *XMLEncoder, start xml.StartElement) error {
	if debug {
		log.Printf("RandSequenceNode.MarshalBTXML start:%v", start)
	}

	if err := e.EncodeSE(start, func(e *XMLEncoder) error {
		return r.compositeNode.marshalXML(e)
	}); err != nil {
		return NewXMLTokenError(start, err)
	}

	return nil
}

func (r *RandSequenceNode) UnmarshalBTXML(d *XMLDecoder, start xml.StartElement) error {
	if debug {
		log.Printf("RandSequenceNode.UnmarshalBTXML start:%v", start)
	}

	var err error

	if err = r.compositeNode.unmarshalXML(d); err == nil {
		if err = d.Skip(); err == nil {
			for _, v := range r.childs {
				v.SetParent(r)
			}
			return nil
		}
	}

	return NewXMLTokenError(start, err)
}

func (r *RandSelectorNode) MarshalBTXML(e *XMLEncoder, start xml.StartElement) error {
	if debug {
		log.Printf("RandSelectorNode.MarshalBTXML start:%v", start)
	}

	if err := e.EncodeSE(start, func(e *XMLEncoder) error {
		return r.compositeNode.marshalXML(e)
	}); err != nil {
		return NewXMLTokenError(start, err)
	}

	return nil
}

func (r *RandSelectorNode) UnmarshalBTXML(d *XMLDecoder, start xml.StartElement) error {
	if debug {
		log.Printf("RandSelectorNode.MarshalBTXML start:%v", start)
	}

	var err error

	if err = r.compositeNode.unmarshalXML(d); err == nil {
		if err = d.Skip(); err == nil {
			for _, v := range r.childs {
				v.SetParent(r)
			}
			return nil
		}
	}

	return NewXMLTokenError(start, err)
}

func (p *ParallelNode) MarshalBTXML(e *XMLEncoder, start xml.StartElement) error {
	if debug {
		log.Printf("ParallelNode.MarshalBTXML start:%v", start)
	}

	if err := e.EncodeSE(start, func(e *XMLEncoder) error {
		return p.compositeNode.marshalXML(e)
	}); err != nil {
		return NewXMLTokenError(start, err)
	}

	return nil
}

func (p *ParallelNode) UnmarshalBTXML(d *XMLDecoder, start xml.StartElement) error {
	if debug {
		log.Printf("ParallelNode.MarshalBTXML start:%v", start)
	}

	var err error

	if err = p.compositeNode.unmarshalXML(d); err == nil {
		if err = d.Skip(); err == nil {
			for _, v := range p.childs {
				v.SetParent(p)
			}
			return nil
		}
	}

	return NewXMLTokenError(start, err)
}

func (t BevType) MarshalXMLAttr(name xml.Name) (xml.Attr, error) {
	return xml.Attr{Name: name, Value: t.String()}, nil
}

func (t *BevType) UnmarshalXMLAttr(attr xml.Attr) error {
	if meta, ok := bevName2META[attr.Value]; ok {
		*t = meta.typ
		return nil
	} else {
		return fmt.Errorf("invalid bevType %s", attr.Value)
	}
}

func (b *BevNode) MarshalBTXML(e *XMLEncoder, start xml.StartElement) error {
	if debug {
		log.Printf("BevNode.MarshalBTXML start:%v", start)
	}

	var err error
	var bevTypeAttr xml.Attr
	if bevTypeAttr, err = b.bevParams.BevType().MarshalXMLAttr(createXMLName(xmlNameBevType)); err != nil {
		return NewXMLTokenError(start, err)
	}

	start.Attr = append(start.Attr, bevTypeAttr)

	switch o := b.bevParams.(type) {
	case XMLMarshaler:
		err = o.MarshalBTXML(e, start)
	default:
		err = e.Encoder.EncodeElement(b.bevParams, start)
	}

	return err
}

func (b *BevNode) UnmarshalBTXML(d *XMLDecoder, start xml.StartElement) error {
	if debug {
		log.Printf("BevNode.UnmarshalBTXML start:%v", start)
	}

	var bevParams BevParams

	xmlNameBevType := createXMLName(xmlNameBevType)
	for _, attr := range start.Attr {
		if attr.Name == xmlNameBevType {
			var bevType BevType
			if err := bevType.UnmarshalXMLAttr(attr); err != nil {
				return NewXMLTokenError(start, err)
			} else {
				bevParams = getBevMETAByType(bevType).createParams()
				break
			}
		}
	}

	if bevParams == nil {
		return NewXMLAttrNotFoundError(start, xmlNameBevType)
	}

	if err := d.DecodeElement(bevParams, start); err != nil {
		return err
	}

	b.bevParams = bevParams
	return nil
}

func xmlNameToString(name xml.Name) string {
	if name.Space == "" {
		return name.Local
	} else {
		return name.Space + name.Local
	}
}

func xmlTokenToString(token xml.Token) string {
	var sb strings.Builder

	switch o := token.(type) {
	case xml.StartElement:
		nameStr := xmlNameToString(o.Name)
		sb.WriteString(fmt.Sprintf("<%s", nameStr))

		for _, attr := range o.Attr {
			sb.WriteString(fmt.Sprintf(" %s=\"%s\"", xmlNameToString(attr.Name), attr.Value))
		}

		sb.WriteString(">")

		return sb.String()

	case xml.EndElement:
		nameStr := xmlNameToString(o.Name)
		sb.WriteString(fmt.Sprintf("</%s>", nameStr))
		return sb.String()

	default:
		panic("not support yet")
	}
}

func NewXMLTokenError(token xml.Token, err error) error {
	return errors.WithMessage(err, xmlTokenToString(token))
}

func XMLTokenErrorf(token xml.Token, f string, args ...interface{}) error {
	return errors.Errorf(xmlTokenToString(token)+": "+f, args...)
}

func NewXMLAttrNotFoundError(start xml.StartElement, attrName xml.Name) error {
	return errors.Errorf("%s: attribute \"%s\" not found", xmlTokenToString(start), xmlNameToString(attrName))
}
