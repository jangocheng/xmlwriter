package xmlwriter

import (
	"fmt"
	"strings"
)

type Raw string

func (t Raw) kind() NodeKind { return RawNode }

func (t Raw) write(w *Writer) error {
	// Raw is a special case here - it does not call Writer.Next() so
	// you can write raw inside nodes which are Open but not Opened.
	if err := w.writeBeginCur(RawNode); err != nil {
		return err
	}
	w.printer.WriteString(string(t))
	if w.Indenter != nil {
		w.last = Event{StateEnded, RawNode, 0}
	}
	return w.printer.cachedWriteError()
}

type Text string

func (t Text) kind() NodeKind { return TextNode }

func (t Text) write(w *Writer) error {
	s := string(t)
	if w.Indenter != nil {
		s = w.Indenter.Wrap(s)
	}
	if w.Enforce {
		if err := w.checkParent(NoNode, ElemNode); err != nil {
			return err
		}
		// TODO: CharData ::= [^<&]* - ([^<&]* ']]>' [^<&]*)
	}
	if err := w.writeBeginNext(TextNode); err != nil {
		return err
	}
	err := w.printer.EscapeString(s)
	if w.Indenter != nil {
		w.last = Event{StateEnded, TextNode, 0}
	}
	return err
}

type CommentContent string

func (c CommentContent) kind() NodeKind { return CommentContentNode }

func (c CommentContent) write(w *Writer) error {
	s := string(c)
	if w.Indenter != nil {
		s = w.Indenter.Wrap(s)
	}
	if w.Enforce {
		if err := w.checkParent(NoNode, CommentNode); err != nil {
			return err
		}
		// FIXME: we could escape this. should we?
		if strings.Index(s, "--") >= 0 {
			return fmt.Errorf("xmlwriter: comment may not contain '--'")
		}
		if err := CheckChars(s, w.StrictChars); err != nil {
			return err
		}
	}

	if err := w.writeBeginNext(CommentContentNode); err != nil {
		return err
	}
	if _, err := w.printer.WriteString(s); err != nil {
		return err
	}
	if w.Indenter != nil {
		w.last = Event{StateEnded, CommentContentNode, 0}
	}
	return nil
}

type Comment struct {
	Content string
}

func (c Comment) start(w *Writer) error {
	if err := w.pushBegin(CommentNode, []NodeKind{NoNode, DocNode, DTDNode, ElemNode}); err != nil {
		return err
	}
	np := &w.nodes[w.current+1]
	np.clear()
	np.kind = CommentNode
	np.comment = c
	return w.pushEnd()
}

func (c Comment) kind() NodeKind { return CommentNode }

func (c Comment) write(w *Writer) error {
	if err := w.StartComment(c); err != nil {
		return err
	}
	if err := w.EndComment(); err != nil {
		return err
	}
	return nil
}

func (c Comment) open(n *node, w *Writer) error {
	w.printer.WriteString("<!--")
	return w.printer.cachedWriteError()
}

func (c Comment) opened(n *node, w *Writer, prev NodeState) error {
	if n.comment.Content != "" {
		if err := w.WriteCommentContent(n.comment.Content); err != nil {
			return err
		}
		n.comment.Content = ""
	}
	return nil
}

func (c Comment) end(n *node, w *Writer, prev NodeState) error {
	w.printer.WriteString("-->")
	return w.printer.cachedWriteError()
}

type CDataContent string

func (c CDataContent) kind() NodeKind { return CDataContentNode }

func (c CDataContent) write(w *Writer) error {
	s := string(c)
	if w.Enforce {
		if err := w.checkParent(NoNode, CDataNode); err != nil {
			return err
		}
		// FIXME: we could escape this. should we?
		if strings.Index(s, "]]>") >= 0 {
			return fmt.Errorf("xmlwriter: cdata may not contain ']]>'")
		}
		if err := CheckChars(s, w.StrictChars); err != nil {
			return err
		}
	}

	if err := w.writeBeginNext(CDataContentNode); err != nil {
		return err
	}
	if _, err := w.printer.WriteString(s); err != nil {
		return err
	}
	if w.Indenter != nil {
		w.last = Event{StateEnded, CDataContentNode, 0}
	}
	return nil
}

type CData struct {
	Content string
}

func (c CData) start(w *Writer) error {
	if err := w.pushBegin(CDataNode, []NodeKind{NoNode, ElemNode}); err != nil {
		return err
	}
	np := &w.nodes[w.current+1]
	np.clear()
	np.kind = CDataNode
	np.cdata = c
	return w.pushEnd()
}

func (c CData) kind() NodeKind { return CDataNode }

func (c CData) write(w *Writer) error {
	if err := w.StartCData(c); err != nil {
		return err
	}
	if err := w.EndCData(); err != nil {
		return err
	}
	return nil
}

func (c CData) open(n *node, w *Writer) error {
	w.printer.WriteString("<![CDATA[")
	return w.printer.cachedWriteError()
}

func (c CData) opened(n *node, w *Writer, prev NodeState) error {
	if c.Content != "" {
		if err := w.WriteCDataContent(c.Content); err != nil {
			return err
		}
		n.cdata.Content = ""
	}
	return nil
}

func (c CData) end(n *node, w *Writer, prev NodeState) error {
	w.printer.WriteString("]]>")
	return w.printer.cachedWriteError()
}

type Doc struct {
	// Do not print 'encoding="..."' into the document opening
	SuppressEncoding bool

	// Override Writer.Encoding with this string if not nil
	ForcedEncoding *string

	// Do not print 'version="..."' into the document opening
	SuppressVersion bool

	// Override Writer.Version with this string if not nil
	ForcedVersion *string

	// If nil, do not print 'standalone="..."'
	Standalone *bool
}

// ForceEncoding is a fluent convenience function for assigning a
// non-pointer string to Doc.ForcedEncoding.
func (d Doc) ForceEncoding(v string) Doc { d.ForcedEncoding = &v; return d }

// ForceVersion is a fluent convenience function for assigning a
// non-pointer string to Doc.ForcedVersion.
func (d Doc) ForceVersion(v string) Doc { d.ForcedVersion = &v; return d }

// WithStandalone is a fluent convenience function for assigning
// a non-pointer bool to Doc.Standalone
func (d Doc) WithStandalone(v bool) Doc { d.Standalone = &v; return d }

func (d Doc) kind() NodeKind { return DocNode }

func (d Doc) start(w *Writer) error {
	if err := w.pushBegin(DocNode, []NodeKind{NoNode}); err != nil {
		return err
	}
	np := &w.nodes[w.current+1]
	np.clear()
	np.kind = DocNode
	np.doc = d
	return w.pushEnd()
}

func (d Doc) open(n *node, w *Writer) error {
	w.printer.WriteString("<?xml")

	space := true
	if !d.SuppressVersion {
		version := ""
		if d.ForcedVersion != nil {
			version = *d.ForcedVersion
		} else {
			version = w.Version
			if version == "" {
				version = "1.0"
			}
		}
		space = false
		if err := w.printer.printAttr("version", version, w.Enforce); err != nil {
			return err
		}
	}

	if !d.SuppressEncoding {
		enc := ""
		if d.ForcedEncoding != nil {
			enc = *d.ForcedEncoding
		} else {
			enc = w.encoding
		}
		if enc != "" {
			if w.Enforce {
				if err := CheckEncoding(enc); err != nil {
					return err
				}
			}
			space = false
			if err := w.printer.printAttr("encoding", enc, w.Enforce); err != nil {
				return err
			}
		}
	}

	if d.Standalone != nil {
		v := "yes"
		if !*d.Standalone {
			v = "no"
		}
		space = false
		if err := w.printer.printAttr("standalone", v, w.Enforce); err != nil {
			return err
		}
	}

	if space {
		w.printer.WriteByte(' ')
	}
	w.printer.WriteString("?>")
	w.printer.WriteString(w.NewlineString)
	return w.printer.cachedWriteError()
}

func (d Doc) opened(n *node, w *Writer, prev NodeState) error {
	return nil
}

func (d Doc) end(n *node, w *Writer, prev NodeState) error {
	return nil
}

type PI struct {
	Target  string
	Content string
}

func (p PI) kind() NodeKind { return PINode }

func (p PI) write(w *Writer) error {
	if w.Enforce {
		if err := w.checkParent(NoNode, DocNode, ElemNode); err != nil {
			return err
		}
		if strings.ToLower(p.Target) == "xml" {
			return fmt.Errorf("xmlwriter: PI target may not be 'xml'")
		}
		if err := CheckName(p.Target); err != nil {
			return err
		}
		if strings.Index(p.Content, "?>") >= 0 {
			return fmt.Errorf("xmlwriter: PI content may not contain '?>'")
		}
	}

	if err := w.writeBeginNext(PINode); err != nil {
		return err
	}
	w.printer.WriteString("<?")
	w.printer.WriteString(p.Target)
	w.printer.WriteByte(' ')
	if err := w.WriteRaw(p.Content); err != nil {
		return err
	}
	w.printer.WriteString("?>")

	err := w.printer.cachedWriteError()
	if w.Indenter != nil {
		w.last = Event{StateEnded, PINode, 0}
	}
	return err
}
