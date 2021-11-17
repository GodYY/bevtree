package bevtree

import (
	"bytes"
	"encoding/xml"
	"io"
	"log"
	"os"

	"github.com/GodYY/gutils/assert"
	"github.com/pkg/errors"
)

// help function to create xml.Name in namespace bevtree.
func createXMLName(name string) xml.Name {
	return xml.Name{Space: "", Local: name}
}

// xml.Name for BevTree.
var xmlNameBevTree = createXMLName("bevtree")

// xml.Name for nodeType.
var xmlNameNodeType = createXMLName("nodetype")

// xml.Name for root node.
var xmlNameRoot = createXMLName("root")

// xml.Name for childs.
var xmlNameChilds = createXMLName("childs")

// xml.Name for child.
var xmlNameChild = createXMLName("child")

// xml.Name for limited.
var xmlNameLimited = createXMLName("limited")

// xml.Name for success on fail.
var xmlNameSuccessOnFail = createXMLName("successonfail")

// xml.Name for Bev.
var xmlNameBev = createXMLName("bev")

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

// BevXMLEncoder is the interface implemented by objects than can encode
// bevtree behavior definer.
type BevXMLEncoder interface {
	EncodeXMLBev(*XMLEncoder, BevDef, xml.StartElement) error
}

// BevXMLDecoder is the interface implemented by objects than can decode
// bevtree behavior definer.
type BevXMLDecoder interface {
	DecodeXMLBev(*XMLDecoder, *BevDef, xml.StartElement) error
}

// An Encoder writes bevtree XML data to an output stream.
type XMLEncoder struct {
	*xml.Encoder
	bevEnc BevXMLEncoder
}

// NewXMLEncoder returns a new encoder that writes to w with bevEncoder.
func NewXMLEncoder(bevEncoder BevXMLEncoder, w io.Writer) *XMLEncoder {
	assert.NotNilArg(bevEncoder, "bevEncoder")

	return &XMLEncoder{
		Encoder: xml.NewEncoder(w),
		bevEnc:  bevEncoder,
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

	return e.EncodeStartEnd(tree, xml.StartElement{Name: xmlNameBevTree})
}

// encodeNode invoke EncodeStartEnd to write the bevtree XML encoding of
// node n to the stream with name as element name:
// <name.Space:name.Local nodetype="nodetype">
// ...
// </name.Space:name.Local>
func (e *XMLEncoder) encodeNode(n node, name xml.Name) error {
	start := xml.StartElement{
		Name: name,
	}

	if ntAttr, err := n.nodeType().MarshalXMLAttr(xmlNameNodeType); err == nil {
		start.Attr = append(start.Attr, ntAttr)
	} else {
		return err
	}

	return e.EncodeStartEnd(n, start)
}

// encodeBev writes the bevtree XML encoding of BevDef bd to the stream.
func (e *XMLEncoder) encodeBev(bd BevDef, start xml.StartElement) error {
	return e.bevEnc.EncodeXMLBev(e, bd, start)
}

// A XMLDecoder represents an bevtree XML parser reading a particular
// input stream.
type XMLDecoder struct {
	*xml.Decoder
	bevDec BevXMLDecoder
}

// NewXMLDecoder creates a new bevtree XML parser reading from r.
// If r does not implement io.ByteReader, NewXMLDecoder will
// do its own buffering.
func NewXMLDecoder(bevDecoder BevXMLDecoder, r io.Reader) *XMLDecoder {
	assert.NotNilArg(bevDecoder, "bevDecoder")

	return &XMLDecoder{
		Decoder: xml.NewDecoder(r),
		bevDec:  bevDecoder,
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

type decodeXMLCallback func(d *XMLDecoder, start xml.StartElement) error

// DecodeEndTo read element end to endTo unless f return error.
// If f return errDecodeXMLStop, DecodeEndTo return nil.
func (d *XMLDecoder) DecodeEndTo(endTo xml.EndElement, f decodeXMLCallback) error {
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

	return d.DecodeEndTo(xml.EndElement{Name: xmlNameBevTree}, func(d *XMLDecoder, start xml.StartElement) error {
		if start.Name != xmlNameBevTree {
			return nil
		}

		if err := d.DecodeElement(t, start); err != nil {
			return err
		}

		return errDecodeXMLStop
	})
}

func errXMLAttrNotFound(attrName xml.Name) error {
	return errors.Errorf("attr %s:%s not found", attrName.Space, attrName.Local)
}

// decodeNode parse nodetype attr from start, then create node
// based on nodetype, finally invoke node.UnmarshalBTXML and
// set *pnode to node.
func (d *XMLDecoder) decodeNode(pnode *node, start xml.StartElement) error {
	var err error

	var nt nodeType
	foundNT := false
	for _, attr := range start.Attr {
		if attr.Name == xmlNameNodeType {
			if err = nt.UnmarshalXMLAttr(attr); err != nil {
				return errors.WithMessage(err, "unmarshal nodetype")
			}

			foundNT = true
			break
		}
	}

	if !foundNT {
		return errXMLAttrNotFound(xmlNameNodeType)
	}

	node := createNode(nt)
	nodeUnmarshaler, ok := node.(XMLUnmarshaler)
	if !ok {
		return errors.New("node not implements XMLUnmarshaler")
	}

	if err = nodeUnmarshaler.UnmarshalBTXML(d, start); err != nil {
		return err
	}

	*pnode = node
	return nil
}

// decodeBev parse element from start to create BevDef.
func (d *XMLDecoder) decodeBev(pbd *BevDef, start xml.StartElement) error {
	return d.bevDec.DecodeXMLBev(d, pbd, start)
}

// marshal t to valid XML attribute.
func (t nodeType) MarshalXMLAttr(name xml.Name) (xml.Attr, error) {
	return xml.Attr{Name: name, Value: t.String()}, nil
}

// unmarshal an XML attribute to t.
func (t *nodeType) UnmarshalXMLAttr(attr xml.Attr) error {
	if nt, ok := nodeString2Types[attr.Value]; ok {
		*t = nt
		return nil
	} else {
		return errors.Errorf("invalid nodeType %s", attr.Value)
	}
}

// MarshalXMLBevTree return an bevtree XML encoding of t.
func MarshalXMLBevTree(t *BevTree, bevEncoder BevXMLEncoder) ([]byte, error) {
	if t == nil {
		return nil, nil
	}

	if bevEncoder == nil {
		return nil, errors.New("bevEncoder nil")
	}

	var buf = bytes.NewBuffer(nil)
	e := NewXMLEncoder(bevEncoder, buf)
	e.Indent("", "    ")

	if err := e.EncodeTree(t); err != nil {
		return nil, err
	}

	e.Flush()

	return buf.Bytes(), nil
}

// UnmarshalXMLBevTree parses the bevtree XML-encoded BevTree
// data and stores the result in the BevTree pointed to by t.
func UnmarshalXMLBevTree(data []byte, t *BevTree, bevDecoder BevXMLDecoder) error {
	if data == nil || t == nil {
		return nil
	}

	if bevDecoder == nil {
		return errors.New("bevDecoder nil")
	}

	var buf = bytes.NewReader(data)
	d := NewXMLDecoder(bevDecoder, buf)
	return d.DecodeTree(t)
}

// EncodeBevTreeXMLFile works like MarshalXMLBevTree but write
// encoded data to file.
func EncodeBevTreeXMLFile(path string, t *BevTree, bevEncoder BevXMLEncoder) error {
	if t == nil {
		return nil
	}

	if bevEncoder == nil {
		return errors.New("bevEncoder nil")
	}

	file, err := os.Create(path)
	if err != nil {
		return err
	}

	enc := NewXMLEncoder(bevEncoder, file)
	if err := enc.EncodeTree(t); err != nil {
		file.Close()
		return errors.WithMessage(err, "encode tree")
	}

	if err := enc.Flush(); err != nil {
		file.Close()
		return errors.WithMessage(err, "flush")
	}

	if err := file.Close(); err != nil {
		return errors.WithMessage(err, "file close")
	}

	return nil
}

// DecodeBevTreeXMLFile works like UnmarshalXMLBevTree but read
// encoded data from file.
func DecodeBevTreeXMLFile(path string, t *BevTree, bevDecoder BevXMLDecoder) error {
	if t == nil {
		return nil
	}

	if bevDecoder == nil {
		return errors.New("bevDecoder nil")
	}

	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	dec := NewXMLDecoder(bevDecoder, file)
	if err := dec.DecodeTree(t); err != nil {
		return errors.WithMessage(err, "decode tree")
	}

	return nil
}

func (t *BevTree) MarshalBTXML(e *XMLEncoder, start xml.StartElement) error {
	if err := e.EncodeStartEnd(t.root_, xml.StartElement{Name: xmlNameRoot}); err != nil {
		return errors.WithMessage(err, "marshal root")
	}

	return nil
}

func (t *BevTree) UnmarshalBTXML(d *XMLDecoder, start xml.StartElement) error {
	if debug {
		log.Printf("BevTree.UnmarshalBTXML start:%v", start)
	}

	rootDecoded := false
	if err := d.DecodeEndTo(start.End(), func(d *XMLDecoder, start xml.StartElement) error {
		if start.Name == xmlNameRoot {
			if debug {
				log.Printf("BevTree.UnmarshalBTXML root:%v", start)
			}

			if t.root_ == nil {
				t.root_ = newRoot()
			}

			if err := d.DecodeElement(t.root_, start); err != nil {
				return errors.WithMessage(err, "unmarshal root")
			}

			rootDecoded = true
		}

		return nil
	}); err != nil {
		return err
	} else if !rootDecoded {
		return errors.New("no root")
	} else {
		return nil
	}
}

func (n *oneChildNodeBase) MarshalBTXML(e *XMLEncoder, start xml.StartElement) error {
	if n.child == nil {
		return nil
	}

	return e.encodeNode(n.child, xmlNameChild)
}

func (n *oneChildNodeBase) UnmarshalBTXML(d *XMLDecoder, start xml.StartElement) error {
	if debug {
		log.Printf("oneChildNodeBase.UnmarshalBTXML start:%v", start)
	}
	return d.DecodeEndTo(start.End(), n.onDecodeXMLElement)
}

func (n *oneChildNodeBase) onDecodeXMLElement(d *XMLDecoder, start xml.StartElement) error {
	if start.Name == xmlNameChild {
		var child node
		if err := d.decodeNode(&child, start); err != nil {
			return errors.WithMessage(err, "unmarshal child")
		}

		n.SetChild(child)
	}

	return nil
}

func (r *RepeaterNode) MarshalBTXML(e *XMLEncoder, start xml.StartElement) error {
	if debug {
		log.Printf("RepeaterNode.MarshalBTXML start:%v", start)
	}

	var err error

	if err = e.EncodeElement(r.limited, xml.StartElement{Name: xmlNameLimited}); err != nil {
		return errors.WithMessage(err, "marshal limited")
	}

	return r.decoratorNodeBase.MarshalBTXML(e, start)
}

func (r *RepeaterNode) UnmarshalBTXML(d *XMLDecoder, start xml.StartElement) error {
	if debug {
		log.Printf("RepeaterNode.UnmarshalBTXML start:%v", start)
	}

	return d.DecodeEndTo(start.End(), r.onDecodeXMLElement)
}

func (r *RepeaterNode) onDecodeXMLElement(d *XMLDecoder, start xml.StartElement) error {
	if start.Name == xmlNameLimited {
		if debug {
			log.Printf("RepeaterNode.unmarshalBTXML limited")
		}

		if err := d.DecodeElement(&r.limited, start); err != nil {
			return errors.WithMessage(err, "unmarshal limited")
		}

		return nil
	} else {
		return r.decoratorNodeBase.onDecodeXMLElement(d, start)
	}
}

func (r *RepeatUntilFailNode) MarshalBTXML(e *XMLEncoder, start xml.StartElement) error {
	if debug {
		log.Printf("RepeatUntilFailNode.MarshalBTXML start:%v", start)
	}

	var err error

	if err = e.EncodeElement(r.successOnFail, xml.StartElement{Name: xmlNameSuccessOnFail}); err != nil {
		return errors.WithMessage(err, "marshal successOnFail")
	}

	return r.decoratorNodeBase.MarshalBTXML(e, start)
}

func (r *RepeatUntilFailNode) UnmarshalBTXML(d *XMLDecoder, start xml.StartElement) error {
	if debug {
		log.Printf("RepeatUntilFailNode.UnmarshalBTXML start:%v", start)
	}

	return d.DecodeEndTo(start.End(), r.onDecodeXMLElement)
}

func (r *RepeatUntilFailNode) onDecodeXMLElement(d *XMLDecoder, start xml.StartElement) error {
	if start.Name == xmlNameSuccessOnFail {
		if debug {
			log.Printf("RepeatUntilFailNodee.unmarshalBTXML successOnFail start:%v", start)
		}

		if err := d.DecodeElement(&r.successOnFail, start); err != nil {
			return errors.WithMessage(err, "unmarshal successOnFail")
		}

		return nil
	} else {
		return r.decoratorNodeBase.onDecodeXMLElement(d, start)
	}
}

func (c *compositeNodeBase) MarshalBTXML(e *XMLEncoder, start xml.StartElement) error {
	if c.childCount > 0 {
		var err error

		childsStart := xml.StartElement{Name: xmlNameChilds}

		if err = e.EncodeToken(childsStart); err != nil {
			return errors.WithMessage(err, "marshal childs")
		}

		for i, node := 0, c.firstChild; node != nil; i, node = i+1, node.NextSibling() {
			if err = e.encodeNode(node, xmlNameChild); err != nil {
				return errors.WithMessagef(err, "marshal %d child", i)
			}
		}

		if err = e.EncodeToken(childsStart.End()); err != nil {
			return errors.WithMessage(err, "marshal childs")
		}

		return nil
	}

	return nil
}

func (c *compositeNodeBase) UnmarshalBTXML(d *XMLDecoder, start xml.StartElement) error {
	return d.DecodeEndTo(start.End(), c.onDecodeXMLElement)
}

func (c *compositeNodeBase) onDecodeXMLElement(d *XMLDecoder, start xml.StartElement) error {
	if start.Name == xmlNameChilds {
		return d.DecodeEndTo(start.End(), func(d *XMLDecoder, start xml.StartElement) error {
			if start.Name == xmlNameChild {
				var node node
				if err := d.decodeNode(&node, start); err != nil {
					return errors.WithMessagef(err, "unmarshal %d child", c.childCount)
				}

				c.AddChild(node)
				return nil
			}

			return nil
		})
	}

	return nil
}

func (b *BevNode) MarshalBTXML(e *XMLEncoder, start xml.StartElement) error {
	if debug {
		log.Printf("BevNode.MarshalBTXML start:%v", start)
	}

	bevStart := xml.StartElement{Name: xmlNameBev}

	if err := e.encodeBev(b.bevDef, bevStart); err != nil {
		return errors.WithMessage(err, "marshal bev")
	}

	return nil
}

func (b *BevNode) UnmarshalBTXML(d *XMLDecoder, start xml.StartElement) error {
	if debug {
		log.Printf("BevNode.UnmarshalBTXML start:%v", start)
	}

	if err := d.decodeBev(&b.bevDef, start); err != nil {
		return errors.WithMessage(err, "unmarshal bev")
	}

	return nil
}
