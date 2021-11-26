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

// xml name for Bev.
var xmlNameBev = ("bev")

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

// EncodeStartEnd writes the bevtree XML encoding of v to the stream
// between start and start.End(), using start as the outermost tag
// in the encoding.
//
// EncodeStartEnd calls Flush before returning.
func (e *XMLEncoder) EncodeStartEnd(v interface{}, start xml.StartElement) error {
	var err error

	if err = e.EncodeToken(start); err != nil {
		return err
	}

	if err = e.EncodeElement(v, start); err != nil {
		return err
	}

	if err = e.EncodeToken(start.End()); err != nil {
		return err
	}

	return e.Flush()
}

// EncodeTree invoke EncodeStartEnd to write the bevtree XML encoding of
// tree to the stream.
func (e *XMLEncoder) EncodeTree(tree *BevTree) error {
	if tree == nil {
		return nil
	}

	return e.EncodeStartEnd(tree, xml.StartElement{Name: createXMLName(xmlNameBevTree)})
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
		return err
	}

	return e.EncodeStartEnd(n, start)
}

// encodeBev writes the bevtree XML encoding of Bev bd to the stream.
func (e *XMLEncoder) encodeBev(b Bev, start xml.StartElement) error {
	if btAttr, err := b.BevType().MarshalXMLAttr(createXMLName(xmlNameBevType)); err == nil {
		start.Attr = append(start.Attr, btAttr)
	} else {
		return err
	}
	return e.EncodeStartEnd(b, start)
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

var errDecodeXMLStop = errors.New("decode XML stop")

type DecodeXMLCallback func(d *XMLDecoder, start xml.StartElement) error

// DecodeEndTo read element end to endTo unless f return error.
// If f return errDecodeXMLStop, DecodeEndTo return nil.
func (d *XMLDecoder) DecodeEndTo(endTo xml.EndElement, f DecodeXMLCallback) error {
	var err error
	var token xml.Token
	for token, err = d.Token(); err == nil; token, err = d.Token() {
		switch t := token.(type) {
		case xml.StartElement:
			if err = f(d, t); err == nil || err == errDecodeXMLStop {
				if err == errDecodeXMLStop {
					return nil
				}

				continue
			} else {
				return err
			}

		case xml.EndElement:
			if t == endTo {
				if debug {
					log.Printf("decodeEndTo %v end", endTo)
				}
				return nil
			}

		}
	}

	return err
}

// DecodeTree read element from input stream and parse into t.
func (d *XMLDecoder) DecodeTree(t *BevTree) error {
	if t == nil {
		return nil
	}

	return d.DecodeEndTo(xml.EndElement{Name: createXMLName(xmlNameBevTree)}, func(d *XMLDecoder, start xml.StartElement) error {
		if start.Name != createXMLName(xmlNameBevTree) {
			return nil
		}

		if err := t.UnmarshalBTXML(d, start); err != nil {
			return err
		}

		return errDecodeXMLStop
	})
}

// DecodeNode parse nodetype attr from start, then create node
// based on nodetype, finally invoke node.UnmarshalBTXML and
// set *pnode to node.
func (d *XMLDecoder) DecodeNode(pnode *Node, start xml.StartElement) error {
	var err error

	nodeTypeXMLName := createXMLName(xmlNameNodeType)
	var node Node
	for _, attr := range start.Attr {
		if attr.Name == nodeTypeXMLName {
			var nodeType NodeType
			if err = nodeType.UnmarshalXMLAttr(attr); err == nil {
				node = getNodeMETAByType(nodeType).createNode()
			} else {
				return errXMLElemErr(err, start)
			}

			break
		}
	}

	if node == nil {
		return errXMLAttrNotFound(start, nodeTypeXMLName)
	}

	nodeUnmarshaler, ok := node.(XMLUnmarshaler)
	if !ok {
		return errXMLElem(start, "node not implements XMLUnmarshaler")
	}

	if err = nodeUnmarshaler.UnmarshalBTXML(d, start); err != nil {
		return err
	}

	*pnode = node
	return nil
}

// decodeBev parse element from start to create Bev.
func (d *XMLDecoder) decodeBev(pb *Bev, start xml.StartElement) error {
	xmlNameBevType := createXMLName(xmlNameBevType)
	var b Bev
	for _, attr := range start.Attr {
		if attr.Name == xmlNameBevType {
			var bevType BevType
			if err := bevType.UnmarshalXMLAttr(attr); err == nil {
				b = getBevMETAByType(bevType).createTemplate()
			} else {
				return errXMLElemErr(err, start)
			}
			break
		}
	}

	if b == nil {
		return errXMLAttrNotFound(start, xmlNameBevType)
	}

	if err := b.UnmarshalBTXML(d, start); err != nil {
		return err
	}

	*pb = b

	return nil
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

	if err := e.EncodeTree(t); err != nil {
		return nil, err
	}

	e.Flush()

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
	return d.DecodeTree(t)
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
	if err = enc.EncodeTree(t); err != nil {
		return err
	}

	return enc.Flush()
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

	return dec.DecodeTree(t)
}

func (t *BevTree) MarshalBTXML(e *XMLEncoder, start xml.StartElement) error {
	if err := e.EncodeStartEnd(t.root, xml.StartElement{Name: createXMLName(xmlNameRoot)}); err != nil {
		return errXMLElemErr(err, start)
	}

	return nil
}

func (t *BevTree) UnmarshalBTXML(d *XMLDecoder, start xml.StartElement) error {
	if debug {
		log.Printf("BevTree.UnmarshalBTXML start:%v", start)
	}

	rootXMLName := createXMLName(xmlNameRoot)
	if err := d.DecodeEndTo(start.End(), func(d *XMLDecoder, s xml.StartElement) error {
		if s.Name == start.Name {
			return errXMLElemSelfNested(start)
		}

		if s.Name == rootXMLName {
			if t.root != nil {
				return errXMLMultiElem(start, s.Name)
			}

			if debug {
				log.Printf("BevTree.UnmarshalBTXML root:%v", s)
			}

			root := newRootNode()
			if err := root.UnmarshalBTXML(d, s); err != nil {
				return errXMLElemErr(err, start)
			}

			t.root = root
		}

		return nil
	}); err != nil {
		return err
	} else if t.root == nil {
		return errXMLElem(start, "no root")
	} else {
		return nil
	}
}

func (r *rootNode) MarshalBTXML(e *XMLEncoder, start xml.StartElement) error {
	if r.child == nil {
		return nil
	}

	return e.EncodeNode(r.child, xml.StartElement{Name: createXMLName(xmlNameChild)})
}

func (r *rootNode) UnmarshalBTXML(d *XMLDecoder, start xml.StartElement) error {
	return d.DecodeEndTo(start.End(), func(d *XMLDecoder, s xml.StartElement) error {
		if s.Name == start.Name {
			return errXMLElemSelfNested(start)
		}

		if s.Name == createXMLName(xmlNameChild) {
			if r.Child() != nil {
				return errXMLMultiElem(start, s.Name)
			}

			var child Node
			if err := d.DecodeNode(&child, s); err != nil {
				return errXMLElemErr(err, start)
			}

			r.SetChild(child)
		}

		return nil
	})
}

func (d *decoratorNode) MarshalBTXML(e *XMLEncoder, start xml.StartElement) error {
	if d.child == nil {
		return nil
	}

	return e.EncodeNode(d.child, xml.StartElement{Name: createXMLName(xmlNameChild)})
}

func (d *decoratorNode) UnmarshalBTXML(dec *XMLDecoder, start xml.StartElement, f DecodeXMLCallback) error {
	return dec.DecodeEndTo(start.End(), func(dec *XMLDecoder, s xml.StartElement) error {
		if s.Name == createXMLName(xmlNameChild) {
			if d.Child() != nil {
				return errXMLMultiElem(start, s.Name)
			}

			var child Node
			if err := dec.DecodeNode(&child, s); err != nil {
				return errXMLElemErr(err, start)
			}

			d.child = child

			return nil
		} else if f != nil {
			return f(dec, s)
		} else {
			return nil
		}
	})
}

func (i *InverterNode) UnmarshalBTXML(d *XMLDecoder, start xml.StartElement) error {
	if debug {
		log.Printf("InverterNode.UnmarshalBTXML start:%v", start)
	}

	err := i.decoratorNode.UnmarshalBTXML(d, start, nil)
	if err == nil {
		i.child.SetParent(i)
		return err
	}

	return err
}

func (s *SucceederNode) UnmarshalBTXML(d *XMLDecoder, start xml.StartElement) error {
	if debug {
		log.Printf("SucceederNode.UnmarshalBTXML start:%v", start)
	}

	err := s.decoratorNode.UnmarshalBTXML(d, start, nil)
	if err == nil {
		s.child.SetParent(s)
		return err
	}

	return err
}

func (r *RepeaterNode) MarshalBTXML(e *XMLEncoder, start xml.StartElement) error {
	if debug {
		log.Printf("RepeaterNode.MarshalBTXML start:%v", start)
	}

	var err error

	if err = e.EncodeElement(r.limited, xml.StartElement{Name: createXMLName(xmlNameLimited)}); err != nil {
		return errXMLChildElemErr(err, start, xml.StartElement{Name: createXMLName(xmlNameLimited)})
	}

	return r.decoratorNode.MarshalBTXML(e, start)
}

func (r *RepeaterNode) UnmarshalBTXML(d *XMLDecoder, start xml.StartElement) error {
	if debug {
		log.Printf("RepeaterNode.UnmarshalBTXML start:%v", start)
	}

	bLimited := false

	err := r.decoratorNode.UnmarshalBTXML(d, start, func(d *XMLDecoder, s xml.StartElement) error {
		if s.Name == createXMLName(xmlNameLimited) {
			if debug {
				log.Printf("RepeaterNode.unmarshalBTXML limited")
			}

			if bLimited {
				return errXMLMultiElem(start, s.Name)
			}

			if err := d.DecodeElement(&r.limited, s); err != nil {
				return errXMLChildElemErr(err, start, s)
			}

			bLimited = true
		}

		return nil
	})
	if err == nil {
		r.child.SetParent(r)
	}

	return err
}

func (r *RepeatUntilFailNode) MarshalBTXML(e *XMLEncoder, start xml.StartElement) error {
	if debug {
		log.Printf("RepeatUntilFailNode.MarshalBTXML start:%v", start)
	}

	var err error

	if err = e.EncodeElement(r.successOnFail, xml.StartElement{Name: createXMLName(xmlNameSuccessOnFail)}); err != nil {
		return errXMLChildElemErr(err, start, xml.StartElement{Name: createXMLName(xmlNameSuccessOnFail)})
	}

	return r.decoratorNode.MarshalBTXML(e, start)
}

func (r *RepeatUntilFailNode) UnmarshalBTXML(d *XMLDecoder, start xml.StartElement) error {
	if debug {
		log.Printf("RepeatUntilFailNode.UnmarshalBTXML start:%v", start)
	}

	bSuccessOnFail := false

	err := r.decoratorNode.UnmarshalBTXML(d, start, func(d *XMLDecoder, s xml.StartElement) error {
		if s.Name == createXMLName(xmlNameSuccessOnFail) {
			if debug {
				log.Printf("RepeatUntilFailNodee.unmarshalBTXML successOnFail start:%v", s)
			}

			if bSuccessOnFail {
				return errXMLMultiElem(start, s.Name)
			}

			if err := d.DecodeElement(&r.successOnFail, s); err != nil {
				return errXMLChildElemErr(err, start, s)
			}

			bSuccessOnFail = true
		}

		return nil
	})
	if err == nil {
		r.child.SetParent(r)
	}

	return err
}

func (c *compositeNode) MarshalBTXML(e *XMLEncoder, start xml.StartElement) error {
	childCount := c.ChildCount()
	if childCount > 0 {
		var err error

		childsStart := xml.StartElement{Name: createXMLName(xmlNameChilds)}
		childsStart.Attr = append(childsStart.Attr, xml.Attr{Name: createXMLName("count"), Value: strconv.Itoa(childCount)})

		if err = e.EncodeToken(childsStart); err != nil {
			return errXMLChildElemErr(err, start, childsStart)
		}

		for i := 0; i < childCount; i++ {
			child := c.Child(i)
			if err = e.EncodeNode(child, xml.StartElement{Name: createXMLName(xmlNameChild)}); err != nil {
				return errors.WithMessagef(err, "%s marshal No.%d child", xmlTokenToString(start), i)
			}
		}

		if err = e.EncodeToken(childsStart.End()); err != nil {
			return errXMLChildElemErr(err, start, childsStart)
		}

		return nil
	}

	return nil
}

func (c *compositeNode) UnmarshalBTXML(d *XMLDecoder, start xml.StartElement) error {
	var childCount int
	var childs []Node

	err := d.DecodeEndTo(start.End(), func(d *XMLDecoder, s xml.StartElement) error {
		if s.Name == createXMLName(xmlNameChilds) {
			if c.childs != nil {
				return errXMLMultiElem(start, s.Name)
			}

			xmlCountName := createXMLName("count")
			for _, attr := range s.Attr {
				if attr.Name == xmlCountName {
					var err error
					if childCount, err = strconv.Atoi(attr.Value); err != nil {
						return errors.WithMessagef(err, "%s: %s: unmarshal attribute \"%s\"", xmlTokenToString(start), xmlTokenToString(s), xmlNameToString(xmlCountName))
					} else if childCount <= 0 {
						return errXMLChildElemF(start, s, "attribute \"%s\" invalid", xmlNameToString(xmlCountName))
					} else {
						childs = make([]Node, 0, childCount)
					}
					break
				}
			}

			if childs == nil {
				return errXMLAttrNotFound(s, xmlCountName)
			}
		} else if s.Name == createXMLName(xmlNameChild) {
			if len(childs) == childCount {
				return errXMLChildElemF(start, s, "number of \"%s\" greater than %d", xmlNameToString(createXMLName(xmlNameChild)), childCount)
			}

			var node Node
			if err := d.DecodeNode(&node, s); err != nil {
				return errors.WithMessagef(err, "%s unmarshal No.%d child", xmlTokenToString(start), c.ChildCount())
			}

			childs = append(childs, node)

			return nil
		}

		return nil
	})
	if err == nil {
		c.childs = childs
	}

	return err
}

func (s *SequenceNode) UnmarshalBTXML(d *XMLDecoder, start xml.StartElement) error {
	if debug {
		log.Printf("SequenceNode.MarshalBTXML start:%v", start)
	}

	err := s.compositeNode.UnmarshalBTXML(d, start)
	if err == nil {
		for _, v := range s.childs {
			v.SetParent(s)
		}
	}

	return err
}

func (s *SelectorNode) UnmarshalBTXML(d *XMLDecoder, start xml.StartElement) error {
	if debug {
		log.Printf("SelectorNode.MarshalBTXML start:%v", start)
	}

	err := s.compositeNode.UnmarshalBTXML(d, start)
	if err == nil {
		for _, v := range s.childs {
			v.SetParent(s)
		}
	}

	return err
}

func (r *RandSequenceNode) UnmarshalBTXML(d *XMLDecoder, start xml.StartElement) error {
	if debug {
		log.Printf("RandSequenceNode.MarshalBTXML start:%v", start)
	}

	err := r.compositeNode.UnmarshalBTXML(d, start)
	if err == nil {
		for _, v := range r.childs {
			v.SetParent(r)
		}
	}

	return err
}

func (r *RandSelectorNode) UnmarshalBTXML(d *XMLDecoder, start xml.StartElement) error {
	if debug {
		log.Printf("RandSelectorNode.MarshalBTXML start:%v", start)
	}

	err := r.compositeNode.UnmarshalBTXML(d, start)
	if err == nil {
		for _, v := range r.childs {
			v.SetParent(r)
		}
	}

	return err
}

func (p *ParallelNode) UnmarshalBTXML(d *XMLDecoder, start xml.StartElement) error {
	if debug {
		log.Printf("ParallelNode.MarshalBTXML start:%v", start)
	}

	err := p.compositeNode.UnmarshalBTXML(d, start)
	if err == nil {
		for _, v := range p.childs {
			v.SetParent(p)
		}
	}

	return err
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

	if err := e.encodeBev(b.bev, xml.StartElement{Name: createXMLName(xmlNameBev)}); err != nil {
		return errors.WithMessage(err, xmlTokenToString(start))
	}

	return nil
}

func (b *BevNode) UnmarshalBTXML(d *XMLDecoder, start xml.StartElement) error {
	if debug {
		log.Printf("BevNode.UnmarshalBTXML start:%v", start)
	}

	if err := d.DecodeEndTo(start.End(), func(d *XMLDecoder, s xml.StartElement) error {
		if s.Name == createXMLName(xmlNameBev) {
			if b.bev != nil {
				return errXMLMultiElem(start, s.Name)
			}

			if err := d.decodeBev(&b.bev, s); err != nil {
				return errors.WithMessage(err, xmlTokenToString(start))
			}

			return nil
		}

		return nil
	}); err != nil {
		return err
	} else if b.bev == nil {
		return errXMLElem(start, "no bev")
	} else {
		return nil
	}
}

func xmlNameToString(name xml.Name) string {
	if name.Space == "" {
		return name.Local
	} else {
		return name.Space + name.Local
	}
}

func xmlTokenToString(token xml.Token) string {
	switch o := token.(type) {
	case xml.StartElement:
		var sb strings.Builder

		nameStr := xmlNameToString(o.Name)
		sb.WriteString(fmt.Sprintf("<%s", nameStr))

		for _, attr := range o.Attr {
			sb.WriteString(fmt.Sprintf(" %s=\"%s\"", xmlNameToString(attr.Name), attr.Value))
		}

		sb.WriteString(">")

		return sb.String()

	default:
		panic("not support yet")
	}
}

func errXMLAttrNotFound(start xml.StartElement, attrName xml.Name) error {
	return errors.Errorf("%s: attribute \"%s\" not found", xmlTokenToString(start), xmlNameToString(attrName))
}

func errXMLElemSelfNested(start xml.StartElement) error {
	return errXMLElem(start, "self nested")
}

func errXMLElemErr(err error, start xml.StartElement) error {
	return errors.WithMessage(err, xmlTokenToString(start))
}

func errXMLChildElemErr(err error, start xml.StartElement, child xml.StartElement) error {
	return errors.WithMessage(err, xmlTokenToString(start)+": "+xmlTokenToString(child))
}

func errXMLMultiElem(start xml.StartElement, elemName xml.Name) error {
	return errors.Errorf("%s: has multi element: %s", xmlTokenToString(start), xmlNameToString(elemName))
}

func errXMLElem(start xml.StartElement, msg string) error {
	return errors.New(xmlTokenToString(start) + ":" + msg)
}

func errXMLChildElemF(start xml.StartElement, child xml.StartElement, f string, args ...interface{}) error {
	return errors.Errorf(xmlTokenToString(start)+": "+xmlTokenToString(child)+": "+f, args...)
}
