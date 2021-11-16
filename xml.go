package bevtree

import (
	"bytes"
	"encoding/xml"
	"io"
	"log"

	"github.com/godyy/bevtree/internal/assert"
	"github.com/pkg/errors"
)

type XMLMarshaler interface {
	MarshalBTXML(*XMLEncoder, xml.StartElement) error
}

type XMLUnmarshaler interface {
	UnmarshalBTXML(*XMLDecoder, xml.StartElement) error
}

type BevXMLEncoder interface {
	EncodeXMLBev(*XMLEncoder, BevDefiner, xml.StartElement) error
}

type BevXMLDecoder interface {
	DecodeXMLBev(*XMLDecoder, *BevDefiner, xml.StartElement) error
}

type XMLEncoder struct {
	*xml.Encoder
	bevEnc BevXMLEncoder
}

func NewXMLEncoder(bevEncoder BevXMLEncoder, w io.Writer) *XMLEncoder {
	assert.NotNilArg(bevEncoder, "bevEncoder")

	return &XMLEncoder{
		Encoder: xml.NewEncoder(w),
		bevEnc:  bevEncoder,
	}
}

func (e *XMLEncoder) EncodeElement(v interface{}, start xml.StartElement) error {
	if marshaler, ok := v.(XMLMarshaler); ok {
		return marshaler.MarshalBTXML(e, start)
	} else {
		return e.Encoder.EncodeElement(v, start)
	}
}

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

	return err
}

func (e *XMLEncoder) EncodeTree(tree *BevTree) error {
	if tree == nil {
		return nil
	}

	return e.EncodeStartEnd(tree, xml.StartElement{Name: xmlNameBevTree})
}

func (e *XMLEncoder) encodeNode(n node, xmlName xml.Name) error {
	start := xml.StartElement{
		Name: xmlName,
	}

	if ntAttr, err := n.nodeType().MarshalXMLAttr(xmlNameNodeType); err == nil {
		start.Attr = append(start.Attr, ntAttr)
	} else {
		return err
	}

	return e.EncodeStartEnd(n, start)
}

func (e *XMLEncoder) encodeBev(bd BevDefiner, start xml.StartElement) error {
	return e.bevEnc.EncodeXMLBev(e, bd, start)
}

type XMLDecoder struct {
	*xml.Decoder
	bevDec BevXMLDecoder
}

func NewXMLDecoder(bevDecoder BevXMLDecoder, r io.Reader) *XMLDecoder {
	assert.NotNilArg(bevDecoder, "bevDecoder")

	return &XMLDecoder{
		Decoder: xml.NewDecoder(r),
		bevDec:  bevDecoder,
	}
}

func (d *XMLDecoder) DecodeElement(v interface{}, start xml.StartElement) error {
	if unmarshal, ok := v.(XMLUnmarshaler); ok {
		return unmarshal.UnmarshalBTXML(d, start)
	} else {
		return d.Decoder.DecodeElement(v, &start)
	}
}

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

func (d *XMLDecoder) decodeBev(pbd *BevDefiner, start xml.StartElement) error {
	return d.bevDec.DecodeXMLBev(d, pbd, start)
}

var errXMLValueTypeNotSupported = errors.New("xml: value type not supported")

var errDecodeXMLStop = errors.New("decode XML stop")

type decodeXMLCallback func(d *XMLDecoder, start xml.StartElement) error

func createXMLName(name string) xml.Name {
	return xml.Name{"", name}
}

func (t nodeType) MarshalXMLAttr(name xml.Name) (xml.Attr, error) {
	return xml.Attr{Name: name, Value: t.String()}, nil
}

func (t *nodeType) UnmarshalXMLAttr(attr xml.Attr) error {
	if nt, ok := nodeString2Types[attr.Value]; ok {
		*t = nt
		return nil
	} else {
		return errors.Errorf("invalid nodeType %s", attr.Value)
	}
}

var xmlNameNodeType = createXMLName("nodetype")
var xmlNameBevTree = createXMLName("bevtree")

func marshalXMLBevTree(t *BevTree, bevEncoder BevXMLEncoder) ([]byte, error) {
	var buf = bytes.NewBuffer(nil)
	e := NewXMLEncoder(bevEncoder, buf)
	e.Indent("", "    ")

	if err := e.EncodeTree(t); err != nil {
		return nil, err
	}

	e.Flush()

	return buf.Bytes(), nil
}

func unmarshalXMLBevTree(data []byte, t *BevTree, bevDecoder BevXMLDecoder) error {
	var buf = bytes.NewReader(data)
	d := NewXMLDecoder(bevDecoder, buf)
	return d.DecodeTree(t)
}

var xmlNameNode = createXMLName("node")

func errXMLAttrNotFound(attrName xml.Name) error {
	return errors.Errorf("attr %s:%s not found", attrName.Space, attrName.Local)
}

var xmlNameRoot = createXMLName("root")

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

	if err := d.DecodeEndTo(start.End(), func(d *XMLDecoder, start xml.StartElement) error {
		if start.Name == xmlNameRoot {
			if debug {
				log.Printf("BevTree.UnmarshalBTXML root:%v", start)
			}

			t.root_ = newRoot()
			if err := d.DecodeElement(t.root_, start); err != nil {
				return errors.WithMessage(err, "unmarshal root")
			}
		}

		return nil
	}); err != nil {
		return err
	} else if t.root_ == nil {
		return errors.New("no root")
	} else {
		return nil
	}
}

var xmlNameChild = createXMLName("child")

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
	return d.DecodeEndTo(start.End(), n.unmarshalBTXML)
}

func (n *oneChildNodeBase) unmarshalBTXML(d *XMLDecoder, start xml.StartElement) error {
	if start.Name == xmlNameChild {
		var child node
		if err := d.decodeNode(&child, start); err != nil {
			return errors.WithMessage(err, "unmarshal child")
		}

		n.SetChild(child)
	}

	return nil
}

var xmlNameLimited = createXMLName("limited")

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

	return d.DecodeEndTo(start.End(), r.unmarshalBTXML)
}

func (r *RepeaterNode) unmarshalBTXML(d *XMLDecoder, start xml.StartElement) error {
	if start.Name == xmlNameLimited {
		if debug {
			log.Printf("RepeaterNode.unmarshalBTXML limited")
		}

		if err := d.DecodeElement(&r.limited, start); err != nil {
			return errors.WithMessage(err, "unmarshal limited")
		}

		return nil
	} else {
		return r.decoratorNodeBase.unmarshalBTXML(d, start)
	}
}

var xmlNameSuccessOnFail = createXMLName("successonfail")

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

	return d.DecodeEndTo(start.End(), r.unmarshalBTXML)
}

func (r *RepeatUntilFailNode) unmarshalBTXML(d *XMLDecoder, start xml.StartElement) error {
	if start.Name == xmlNameSuccessOnFail {
		if debug {
			log.Printf("RepeatUntilFailNodee.unmarshalBTXML successOnFail start:%v", start)
		}

		if err := d.DecodeElement(&r.successOnFail, start); err != nil {
			return errors.WithMessage(err, "unmarshal successOnFail")
		}

		return nil
	} else {
		return r.decoratorNodeBase.unmarshalBTXML(d, start)
	}
}

var xmlNameChilds = createXMLName("childs")

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
	return d.DecodeEndTo(start.End(), c.unmarshalBTXML)
}

func (c *compositeNodeBase) unmarshalBTXML(d *XMLDecoder, start xml.StartElement) error {
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

var xmlNameBev = createXMLName("bev")

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
