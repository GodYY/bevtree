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
func XMLName(name string) xml.Name {
	return xml.Name{Space: "", Local: name}
}

// XML strings.
const (

	// xml name for BevTree.
	XMLStringBevTree = "bevtree"

	// xml name for name.
	XMLStringName = "name"

	// xml name for comment.
	XMLStringComment = "comment"

	// xml name for NodeType.
	XMLStringNodeType = "nodetype"

	// xml name for BevType
	XMLStringBevType = "bevtype"

	// xml name for root node.
	XMLStringRoot = "root"

	// xml name for childs.
	XMLStringChilds = "childs"

	// xml name for child.
	XMLStringChild = "child"

	// xml name for limited.
	XMLStringLimited = "limited"

	// xml name for success on fail.
	XMLStringSuccessOnFail = "successonfail"
)

func XMLNameToString(name xml.Name) string {
	if name.Space == "" {
		return name.Local
	} else {
		return name.Space + name.Local
	}
}

func XMLTokenToString(token xml.Token) string {
	var sb strings.Builder

	switch o := token.(type) {
	case xml.StartElement:
		nameStr := XMLNameToString(o.Name)
		sb.WriteString(fmt.Sprintf("<%s", nameStr))

		for _, attr := range o.Attr {
			sb.WriteString(fmt.Sprintf(" %s=\"%s\"", XMLNameToString(attr.Name), attr.Value))
		}

		sb.WriteString(">")

		return sb.String()

	case xml.EndElement:
		nameStr := XMLNameToString(o.Name)
		sb.WriteString(fmt.Sprintf("</%s>", nameStr))
		return sb.String()

	default:
		panic("not support yet")
	}
}

func XMLTokenError(token xml.Token, err error) error {
	return errors.WithMessage(err, XMLTokenToString(token))
}

func XMLTokenErrorf(token xml.Token, f string, args ...interface{}) error {
	return errors.Errorf(XMLTokenToString(token)+": "+f, args...)
}

func XMLAttrNotFoundError(attrName xml.Name) error {
	return errors.Errorf("attribute \"%s\" not found", XMLNameToString(attrName))
}

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
func (e *XMLEncoder) EncodeSE(start xml.StartElement, f func(*XMLEncoder) error) error {
	if f == nil {
		return errors.New("f nil")
	}

	if err := e.EncodeToken(start); err != nil {
		return err
	}

	if err := f(e); err != nil {
		return err
	}

	if err := e.EncodeToken(start.End()); err != nil {
		return err
	}

	return e.Flush()
}

// EncodeElementSE writes the bevtree XML encoding of v to the stream
// between start and start.End(), using start as the outermost tag
// in the encoding.
//
// EncodeElementSE calls Flush before returning.
func (e *XMLEncoder) EncodeElementSE(v interface{}, start xml.StartElement) error {
	if err := e.EncodeToken(start); err != nil {
		return err
	}

	if err := e.EncodeElement(v, start); err != nil {
		return err
	}

	if err := e.EncodeToken(start.End()); err != nil {
		return err
	}

	return e.Flush()
}

// EncodeNode encode the behavior tree node to the stream with start as
// start element. EncodeNode automatically encode the type of the node
// as xml.Attr and append it to start.
// <start ... nodetype="nodetype">
// ...
// </end>
func (e *XMLEncoder) EncodeNode(n Node, start xml.StartElement) error {
	if ntAttr, err := n.NodeType().MarshalXMLAttr(XMLName(XMLStringNodeType)); err == nil {
		start.Attr = append(start.Attr, ntAttr)
	} else {
		return err
	}

	if n.Comment() != "" {
		start.Attr = append(start.Attr, xml.Attr{Name: XMLName(XMLStringComment), Value: n.Comment()})
	}

	return e.EncodeElement(n, start)
}

// A XMLDecoder represents an bevtree XML parser reading a particular
// input stream.
type XMLDecoder struct {
	*xml.Decoder

	// The cached token for next decoding.
	tokenCached xml.Token
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

func (d *XMLDecoder) Token() (xml.Token, error) {
	if d.tokenCached != nil {
		token := d.tokenCached
		d.tokenCached = nil
		return token, nil
	} else {
		return d.Decoder.Token()
	}
}

func (d *XMLDecoder) Skip() error {
	if d.tokenCached != nil {
		if _, ok := d.tokenCached.(xml.EndElement); ok {
			d.tokenCached = nil
			return nil
		}

		d.tokenCached = nil
	}

	return d.Decoder.Skip()
}

// DecodeElement read element from start to parse into v.
func (d *XMLDecoder) DecodeElement(v interface{}, start xml.StartElement) error {
	if unmarshal, ok := v.(XMLUnmarshaler); ok {
		return unmarshal.UnmarshalBTXML(d, start)
	} else {
		return d.Decoder.DecodeElement(v, &start)
	}
}

// DecodeAt search the start element with name at. If the
// start elment was found, DecodeAt invoke f with the start
// element and return the result of f.
func (d *XMLDecoder) DecodeAt(at xml.Name, f func(*XMLDecoder, xml.StartElement) error) error {
	var err error
	var token xml.Token

	for token, err = d.Token(); err == nil; token, err = d.Token() {
		switch t := token.(type) {
		case xml.StartElement:
			if t.Name == at {
				if err = f(d, t); err != nil {
					return err
				} else {
					return nil
				}
			}

		case xml.EndElement:
			if t.Name == at {
				return fmt.Errorf("primarily found %s", XMLTokenToString(t))
			}
		}
	}

	return err
}

// The error used to stop decoding.
var ErrXMLDecodeStop = errors.New("XML decode stop")

// DecodeUntil invokes f with every start element, until
// until end element has been readed, or the result of f
// is non-nil or ErrXMLDecodeStop.
func (d *XMLDecoder) DecodeUntil(until xml.EndElement, f func(*XMLDecoder, xml.StartElement) error) error {
	var err error
	var token xml.Token

	for token, err = d.Token(); err == nil; token, err = d.Token() {
		switch t := token.(type) {
		case xml.StartElement:
			if err = f(d, t); err == nil {
				continue
			} else if err != ErrXMLDecodeStop {
				return err
			} else {
				return nil
			}

		case xml.EndElement:
			if t == until {
				d.tokenCached = t
				return nil
			}
		}
	}

	return err
}

// DecodeAtUntil invokes f with every start element named at,
// until until end element has been read, or the result of
// f is non-nil or ErrXMLDecodeStop.
func (d *XMLDecoder) DecodeAtUntil(at xml.Name, until xml.EndElement, f func(*XMLDecoder, xml.StartElement) error) error {
	var err error
	var token xml.Token
	found := false

	for token, err = d.Token(); err == nil; token, err = d.Token() {
		switch t := token.(type) {
		case xml.StartElement:
			if t.Name == at {
				if found {
					return errors.Errorf("<%s...> self nested", XMLNameToString(at))
				} else {
					found = true

					if err = f(d, t); err == nil {
						found = false
						continue
					} else if err == ErrXMLDecodeStop {
						return nil
					} else {
						return err
					}
				}
			}

		case xml.EndElement:
			if t.Name == at {
				return fmt.Errorf("primarily found %s", XMLTokenToString(t))
			} else if t == until {
				d.tokenCached = t
				return nil
			}
		}
	}

	return err
}

// DecodeElementAt looks for the start element named at, and invoke
// DecodeElement with it.
func (d *XMLDecoder) DecodeElementAt(v interface{}, at xml.Name) error {
	return d.DecodeAt(at, func(d *XMLDecoder, s xml.StartElement) error {
		return d.DecodeElement(v, s)
	})
}

// DecodeElementUntil looks for the start element named name to decode
// to v, until the until end element has been read.
func (d *XMLDecoder) DecodeElementUntil(v interface{}, name xml.Name, until xml.EndElement) error {
	return d.DecodeUntil(until, func(d *XMLDecoder, t xml.StartElement) error {
		if t.Name == name {
			if err := d.DecodeElement(v, t); err != nil {
				return err
			} else {
				return ErrXMLDecodeStop
			}
		}

		return nil
	})
}

// DecodeNode decode the node type from start, then create the node
// with the type to decode into it.
func (d *XMLDecoder) DecodeNode(start xml.StartElement) (Node, error) {
	nodeTypeXMLName := XMLName(XMLStringNodeType)
	commentXMLName := XMLName(XMLStringComment)
	var node Node
	var comment string
	var err error
	for _, attr := range start.Attr {
		if attr.Name == nodeTypeXMLName {
			var nodeType NodeType
			if err = nodeType.UnmarshalXMLAttr(attr); err == nil {
				node = getNodeMETAByType(nodeType).createNode()
			} else {
				return nil, XMLTokenError(start, err)
			}
		} else if attr.Name == commentXMLName {
			comment = attr.Value
		}
	}

	if node == nil {
		return nil, XMLTokenError(start, XMLAttrNotFoundError(nodeTypeXMLName))
	}

	node.SetComment(comment)

	if err := d.DecodeElement(node, start); err != nil {
		return nil, err
	}

	return node, nil
}

// DecodeNodeAt first looks for the start element named startName of node;
// then, parse attr NodeType from it and create new node; finally, invoke
// DecodeElement with the start element and new node.
func (d *XMLDecoder) DecodeNodeAt(pnode *Node, at xml.Name) error {
	return d.DecodeAt(at, func(d *XMLDecoder, s xml.StartElement) error {

		node, err := d.DecodeNode(s)
		if err != nil {
			return err
		} else {
			*pnode = node
			return nil
		}

	})
}

// DecodeNodeUntil works like DecodeUntil, except it works on behavior tree
// node.
func (d *XMLDecoder) DecodeNodeUntil(pnode *Node, name xml.Name, until xml.EndElement) error {
	return d.DecodeUntil(until, func(d *XMLDecoder, s xml.StartElement) error {

		node, err := d.DecodeNode(s)
		if err != nil {
			return err
		} else {
			*pnode = node
			return ErrXMLDecodeStop
		}

	})
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

	start := xml.StartElement{Name: XMLName(XMLStringBevTree)}
	if err := e.EncodeElement(t, start); err != nil {
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

	return d.DecodeElementAt(t, XMLName(XMLStringBevTree))
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
	enc.Indent("", "    ")

	start := xml.StartElement{Name: XMLName(XMLStringBevTree)}
	if err := enc.EncodeElement(t, start); err != nil {
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

	return dec.DecodeElementAt(t, XMLName(XMLStringBevTree))
}

func (t *BevTree) MarshalBTXML(e *XMLEncoder, start xml.StartElement) error {
	if debug {
		log.Printf("BevTree.MarshalBTXML start:%v", start)
	}

	if t.name != "" {
		start.Attr = append(start.Attr, xml.Attr{Name: XMLName(XMLStringName), Value: t.name})
	}

	if t.comment != "" {
		start.Attr = append(start.Attr, xml.Attr{Name: XMLName(XMLStringComment), Value: t.comment})
	}

	if err := e.EncodeSE(start, func(x *XMLEncoder) error {
		rootStart := xml.StartElement{Name: XMLName(XMLStringRoot)}
		if err := e.EncodeElementSE(t.root, rootStart); err != nil {
			return errors.WithMessagef(err, "Marshal root")
		}

		return nil
	}); err != nil {
		return errors.WithMessagef(err, "BevTree %s Marshal", XMLTokenToString(start))
	}

	return nil
}

func (t *BevTree) UnmarshalBTXML(d *XMLDecoder, start xml.StartElement) error {
	if debug {
		log.Printf("BevTree.UnmarshalBTXML start:%v", start)
	}

	for _, attr := range start.Attr {
		if attr.Name == XMLName(XMLStringName) {
			t.name = attr.Value
		} else if attr.Name == XMLName(XMLStringComment) {
			t.comment = attr.Value
		}
	}

	root := newRootNode()
	if err := d.DecodeElementAt(root, XMLName(XMLStringRoot)); err != nil {
		return errors.WithMessagef(err, "BevTree %s Unmarshal root", XMLTokenToString(start))
	}

	t.root = root

	return d.Skip()
}

func (r *rootNode) MarshalBTXML(e *XMLEncoder, start xml.StartElement) error {
	if debug {
		log.Printf("rootNode.MarshalBTXML start:%v", start)
	}

	if r.child == nil {
		return nil
	}

	if err := e.EncodeNode(r.child, xml.StartElement{Name: XMLName(XMLStringChild)}); err != nil {
		return errors.WithMessage(err, "rootNode Marshal child")
	}

	return nil
}

func (r *rootNode) UnmarshalBTXML(d *XMLDecoder, start xml.StartElement) error {
	if debug {
		log.Printf("rootNode.UnmarshalBTXML start:%v", start)
	}

	var child Node
	if err := d.DecodeNodeUntil(&child, XMLName(XMLStringChild), start.End()); err != nil {
		return errors.WithMessage(err, "rootNode Unmarshal child")
	}

	if child != nil {
		r.SetChild(child)
	}

	return d.Skip()
}

func (d *decoratorNode) marshalXML(e *XMLEncoder) error {
	if d.child == nil {
		return nil
	}

	if err := e.EncodeNode(d.child, xml.StartElement{Name: XMLName(XMLStringChild)}); err != nil {
		return errors.WithMessage(err, "Marshal child")
	}

	return nil
}

func (d *decoratorNode) unmarshalXML(dec *XMLDecoder, start xml.StartElement) error {
	var child Node
	if err := dec.DecodeNodeUntil(&child, XMLName(XMLStringChild), start.End()); err != nil {
		return errors.WithMessage(err, "Unmarshal child")
	}

	if child != nil {
		d.setChild(child)
	}

	return nil
}

func (i *InverterNode) MarshalBTXML(e *XMLEncoder, start xml.StartElement) error {
	if debug {
		log.Printf("InverterNode.MarshalBTXML start:%v", start)
	}

	if err := e.EncodeSE(start, func(e *XMLEncoder) error {
		return i.decoratorNode.marshalXML(e)
	}); err != nil {
		return errors.WithMessagef(err, "InverterNode %s Marshal", XMLTokenToString(start))
	}

	return nil
}

func (i *InverterNode) UnmarshalBTXML(d *XMLDecoder, start xml.StartElement) error {
	if debug {
		log.Printf("InverterNode.UnmarshalBTXML start:%v", start)
	}

	if err := i.decoratorNode.unmarshalXML(d, start); err != nil {
		return errors.WithMessagef(err, "InverterNode %s Unmarshal", XMLTokenToString(start))
	}

	if i.child != nil {
		i.child.SetParent(i)
	}

	return d.Skip()
}

func (s *SucceederNode) MarshalBTXML(e *XMLEncoder, start xml.StartElement) error {
	if debug {
		log.Printf("SucceederNode.MarshalBTXML start:%v", start)
	}

	if err := e.EncodeSE(start, func(e *XMLEncoder) error {
		return s.decoratorNode.marshalXML(e)
	}); err != nil {
		return errors.WithMessagef(err, "SucceederNode %s Marshal", XMLTokenToString(start))
	}

	return nil
}

func (s *SucceederNode) UnmarshalBTXML(d *XMLDecoder, start xml.StartElement) error {
	if debug {
		log.Printf("SucceederNode.UnmarshalBTXML start:%v", start)
	}

	if err := s.decoratorNode.unmarshalXML(d, start); err != nil {
		return errors.WithMessagef(err, "SucceederNode %s Unmarshal", XMLTokenToString(start))
	}

	if s.child != nil {
		s.child.SetParent(s)
	}

	return d.Skip()
}

func (r *RepeaterNode) MarshalBTXML(e *XMLEncoder, start xml.StartElement) error {
	if debug {
		log.Printf("RepeaterNode.MarshalBTXML start:%v", start)
	}

	if err := e.EncodeSE(start, func(e *XMLEncoder) error {

		if err := e.EncodeElement(r.limited, xml.StartElement{Name: XMLName(XMLStringLimited)}); err != nil {
			return errors.WithMessage(err, "Marshal limited")
		}

		return r.decoratorNode.marshalXML(e)

	}); err != nil {
		return errors.WithMessagef(err, "RepeaterNode %s Marshal", XMLTokenToString(start))
	}

	return nil
}

func (r *RepeaterNode) UnmarshalBTXML(d *XMLDecoder, start xml.StartElement) error {
	if debug {
		log.Printf("RepeaterNode.UnmarshalBTXML start:%v", start)
	}

	if err := d.DecodeElementAt(&r.limited, XMLName(XMLStringLimited)); err != nil {
		return errors.WithMessagef(err, "RepeaterNode %s Unmarshal limited", XMLTokenToString(start))
	}

	if err := r.decoratorNode.unmarshalXML(d, start); err != nil {
		return errors.WithMessagef(err, "RepeaterNode %s Unmarshal", XMLTokenToString(start))
	}

	if r.child != nil {
		r.child.SetParent(r)
	}

	return d.Skip()
}

func (r *RepeatUntilFailNode) MarshalBTXML(e *XMLEncoder, start xml.StartElement) error {
	if debug {
		log.Printf("RepeatUntilFailNode.MarshalBTXML start:%v", start)
	}

	if err := e.EncodeSE(start, func(e *XMLEncoder) error {

		if err := e.EncodeElement(r.successOnFail, xml.StartElement{Name: XMLName(XMLStringSuccessOnFail)}); err != nil {
			return errors.WithMessage(err, "Marshal successOnFail")
		}

		return r.decoratorNode.marshalXML(e)

	}); err != nil {
		return errors.WithMessagef(err, "RepeatUntilFailNode %s Marshal", XMLTokenToString(start))
	}

	return nil
}

func (r *RepeatUntilFailNode) UnmarshalBTXML(d *XMLDecoder, start xml.StartElement) error {
	if debug {
		log.Printf("RepeatUntilFailNode.UnmarshalBTXML start:%v", start)
	}

	if err := d.DecodeElementAt(&r.successOnFail, XMLName(XMLStringSuccessOnFail)); err != nil {
		return errors.WithMessagef(err, "RepeatUntilFailNode %s Unmarshal successOnFail", XMLTokenToString(start))
	}

	if err := r.decoratorNode.unmarshalXML(d, start); err != nil {
		return errors.WithMessagef(err, "RepeatUntilFailNode %s Unmarshal", XMLTokenToString(start))
	}

	if r.child != nil {
		r.child.SetParent(r)
	}

	return d.Skip()
}

func (c *compositeNode) marshalXML(e *XMLEncoder) error {
	childCount := c.ChildCount()
	if childCount > 0 {
		var err error

		childsStart := xml.StartElement{Name: XMLName(XMLStringChilds)}
		childsStart.Attr = append(childsStart.Attr, xml.Attr{Name: XMLName("count"), Value: strconv.Itoa(childCount)})

		if err = e.EncodeToken(childsStart); err != nil {
			return err
		}

		for i := 0; i < childCount; i++ {
			child := c.Child(i)
			if err = e.EncodeNode(child, xml.StartElement{Name: XMLName(XMLStringChild)}); err != nil {
				return errors.WithMessagef(err, "Marshal No.%d child", i)
			}
		}

		if err = e.EncodeToken(childsStart.End()); err != nil {
			return err
		}

		return nil
	}

	return nil
}

func (c *compositeNode) unmarshalXML(d *XMLDecoder, start xml.StartElement) error {
	if err := d.DecodeAtUntil(XMLName(XMLStringChilds), start.End(), func(d *XMLDecoder, s xml.StartElement) error {
		xmlCountName := XMLName("count")
		var childCount int
		var childs []Node
		for _, attr := range s.Attr {
			if attr.Name == xmlCountName {
				var err error
				if childCount, err = strconv.Atoi(attr.Value); err != nil {
					return errors.WithMessage(err, "Unmarshal child count")
				} else if childCount <= 0 {
					return fmt.Errorf("invalid child count: %d", childCount)
				} else {
					childs = make([]Node, 0, childCount)
					break
				}
			}
		}

		if childCount > 0 {
			xmlChildName := XMLName(XMLStringChild)
			if err := d.DecodeAtUntil(xmlChildName, s.End(), func(d *XMLDecoder, s xml.StartElement) error {
				if len(childs) >= childCount {
					return errors.New("too many children")
				}

				if node, err := d.DecodeNode(s); err != nil {
					return err
				} else {
					childs = append(childs, node)
					return nil
				}

			}); err != nil {
				return errors.WithMessagef(err, "Unmarshal No.%d child", len(childs))
			}

			if len(childs) < childCount {
				return errors.New("too few children")
			}
		}

		c.children = childs

		if err := d.Skip(); err != nil {
			return err
		}

		return ErrXMLDecodeStop

	}); err != nil {
		return err
	}

	return nil
}

func (s *SequenceNode) MarshalBTXML(e *XMLEncoder, start xml.StartElement) error {
	if debug {
		log.Printf("SequenceNode.MarshalBTXML start:%v", start)
	}

	if err := e.EncodeSE(start, func(e *XMLEncoder) error {
		return s.compositeNode.marshalXML(e)
	}); err != nil {
		return errors.WithMessagef(err, "SequenceNode %s Marshal", XMLTokenToString(start))
	}

	return nil
}

func (s *SequenceNode) UnmarshalBTXML(d *XMLDecoder, start xml.StartElement) error {
	if debug {
		log.Printf("SequenceNode.UnmarshalBTXML start:%v", start)
	}

	if err := s.compositeNode.unmarshalXML(d, start); err != nil {
		return errors.WithMessagef(err, "SequenceNode %s Unmarshal", XMLTokenToString(start))
	}

	for _, v := range s.children {
		v.SetParent(s)
	}

	return d.Skip()
}

func (s *SelectorNode) MarshalBTXML(e *XMLEncoder, start xml.StartElement) error {
	if debug {
		log.Printf("SelectorNode.MarshalBTXML start:%v", start)
	}

	if err := e.EncodeSE(start, func(e *XMLEncoder) error {
		return s.compositeNode.marshalXML(e)
	}); err != nil {
		return errors.WithMessagef(err, "SelectorNode %s Marshal", XMLTokenToString(start))
	}

	return nil
}

func (s *SelectorNode) UnmarshalBTXML(d *XMLDecoder, start xml.StartElement) error {
	if debug {
		log.Printf("SelectorNode.UnmarshalBTXML start:%v", start)
	}

	if err := s.compositeNode.unmarshalXML(d, start); err != nil {
		return errors.WithMessagef(err, "SelectorNode %s Unmarshal", XMLTokenToString(start))
	}

	for _, v := range s.children {
		v.SetParent(s)
	}

	return d.Skip()
}

func (r *RandSequenceNode) MarshalBTXML(e *XMLEncoder, start xml.StartElement) error {
	if debug {
		log.Printf("RandSequenceNode.MarshalBTXML start:%v", start)
	}

	if err := e.EncodeSE(start, func(e *XMLEncoder) error {
		return r.compositeNode.marshalXML(e)
	}); err != nil {
		return errors.WithMessagef(err, "RandSequenceNode %s Marshal", XMLTokenToString(start))
	}

	return nil
}

func (r *RandSequenceNode) UnmarshalBTXML(d *XMLDecoder, start xml.StartElement) error {
	if debug {
		log.Printf("RandSequenceNode.UnmarshalBTXML start:%v", start)
	}

	if err := r.compositeNode.unmarshalXML(d, start); err != nil {
		return errors.WithMessagef(err, "RandSequenceNode %s Unmarshal", XMLTokenToString(start))
	}

	for _, v := range r.children {
		v.SetParent(r)
	}

	return d.Skip()
}

func (r *RandSelectorNode) MarshalBTXML(e *XMLEncoder, start xml.StartElement) error {
	if debug {
		log.Printf("RandSelectorNode.MarshalBTXML start:%v", start)
	}

	if err := e.EncodeSE(start, func(e *XMLEncoder) error {
		return r.compositeNode.marshalXML(e)
	}); err != nil {
		return errors.WithMessagef(err, "RandSelectorNode %s Marshal", XMLTokenToString(start))
	}

	return nil
}

func (r *RandSelectorNode) UnmarshalBTXML(d *XMLDecoder, start xml.StartElement) error {
	if debug {
		log.Printf("RandSelectorNode.MarshalBTXML start:%v", start)
	}

	if err := r.compositeNode.unmarshalXML(d, start); err != nil {
		return errors.WithMessagef(err, "RandSelectorNode %s Unmarshal", XMLTokenToString(start))
	}

	for _, v := range r.children {
		v.SetParent(r)
	}

	return d.Skip()
}

func (p *ParallelNode) MarshalBTXML(e *XMLEncoder, start xml.StartElement) error {
	if debug {
		log.Printf("ParallelNode.MarshalBTXML start:%v", start)
	}

	if err := e.EncodeSE(start, func(e *XMLEncoder) error {
		return p.compositeNode.marshalXML(e)
	}); err != nil {
		return errors.WithMessagef(err, "ParallelNode %s Marshal", XMLTokenToString(start))
	}

	return nil
}

func (p *ParallelNode) UnmarshalBTXML(d *XMLDecoder, start xml.StartElement) error {
	if debug {
		log.Printf("ParallelNode.MarshalBTXML start:%v", start)
	}

	if err := p.compositeNode.unmarshalXML(d, start); err != nil {
		return errors.WithMessagef(err, "ParallelNode %s Unmarshal", XMLTokenToString(start))
	}

	for _, v := range p.children {
		v.SetParent(p)
	}

	return d.Skip()
}

func (t BevType) MarshalXMLAttr(name xml.Name) (xml.Attr, error) {
	return xml.Attr{Name: name, Value: t.String()}, nil
}

func (t *BevType) UnmarshalXMLAttr(attr xml.Attr) error {
	if meta, ok := bevName2META[attr.Value]; ok {
		*t = meta.typ
		return nil
	} else {
		return fmt.Errorf("invalid BevType %s", attr.Value)
	}
}

func (b *BevNode) MarshalBTXML(e *XMLEncoder, start xml.StartElement) error {
	if debug {
		log.Printf("BevNode.MarshalBTXML start:%v", start)
	}

	if b.bevParams != nil {
		var err error
		var bevTypeAttr xml.Attr
		if bevTypeAttr, err = b.bevParams.BevType().MarshalXMLAttr(XMLName(XMLStringBevType)); err == nil {
			start.Attr = append(start.Attr, bevTypeAttr)

			switch o := b.bevParams.(type) {
			case XMLMarshaler:
				err = o.MarshalBTXML(e, start)
			default:
				err = e.Encoder.EncodeElement(b.bevParams, start)
			}

		}

		if err != nil {
			return errors.WithMessagef(err, "BevNode %s Marshal", XMLTokenToString(start))
		}
	}

	return nil
}

func (b *BevNode) UnmarshalBTXML(d *XMLDecoder, start xml.StartElement) error {
	if debug {
		log.Printf("BevNode.UnmarshalBTXML start:%v", start)
	}

	var err error
	var bevParams BevParams

	xmlNameBevType := XMLName(XMLStringBevType)
	for _, attr := range start.Attr {
		if attr.Name == xmlNameBevType {
			var bevType BevType
			if err = bevType.UnmarshalXMLAttr(attr); err == nil {
				bevParams = getBevMETAByType(bevType).createParams()
				err = d.DecodeElement(bevParams, start)
			}

			break
		}
	}

	if err != nil {
		return errors.WithMessagef(err, "BevNode %s Unmarshal", start)
	} else if bevParams != nil {
		b.bevParams = bevParams
		return nil
	} else {
		return d.Skip()
	}
}
