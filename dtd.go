package xmlwriter

import (
	"fmt"
)

type DTD struct {
	Name     string
	PublicID string
	SystemID string
}

func (d DTD) kind() NodeKind { return DTDNode }

func (d DTD) start(w *Writer) error {
	if err := w.pushBegin(DTDNode, []NodeKind{NoNode, DocNode}); err != nil {
		return err
	}
	np := &w.nodes[w.current+1]
	np.clear()
	np.kind = DTDNode
	np.dtd = d
	return w.pushEnd()
}

func (d DTD) open(n *node, w *Writer) error {
	if w.Enforce {
		if len(d.Name) == 0 {
			return fmt.Errorf("xmlwriter: DTD name must not be empty")
		}
		if err := CheckName(d.Name); err != nil {
			return err
		}
	}
	w.printer.WriteString("<!DOCTYPE ")
	w.printer.WriteString(d.Name)
	if d.PublicID != "" || d.SystemID != "" {
		w.printer.WriteByte(' ')
		return w.printer.writeExternalID(d.PublicID, d.SystemID, w.Enforce)
	} else {
		return w.printer.cachedWriteError()
	}
}

func (d DTD) opened(n *node, w *Writer, prev NodeState) error {
	if n.children > 0 {
		w.printer.WriteString(" [")
	}
	return w.printer.cachedWriteError()
}

func (d DTD) end(n *node, w *Writer, prev NodeState) error {
	if n.children > 0 {
		w.printer.WriteString("]>")
	} else {
		w.printer.WriteString(">")
	}
	return w.printer.cachedWriteError()
}

const DTDEmpty = "EMPTY"
const DTDAny = "ANY"

const DTDPCData = "#PCDATA"
const DTDCData = "CDATA"

type DTDElem struct {
	Name string
	Decl string
}

func (d DTDElem) kind() NodeKind { return DTDElemNode }

func (d DTDElem) writable() {}

func (d DTDElem) write(w *Writer) error {
	if w.Enforce {
		if err := w.checkParent(NoNode, DTDNode); err != nil {
			return err
		}
		if len(d.Name) == 0 {
			return fmt.Errorf("xmlwriter: ELEMENT name must not be empty")
		}
		if len(d.Decl) == 0 {
			return fmt.Errorf("xmlwriter: ELEMENT decl must not be empty")
		}
		if err := CheckName(d.Name); err != nil {
			return err
		}
	}

	if err := w.writeBeginNext(DTDElemNode); err != nil {
		return err
	}
	w.printer.WriteString("<!ELEMENT ")
	w.printer.WriteString(d.Name)
	w.printer.WriteByte(' ')
	w.printer.WriteString(d.Decl)
	w.printer.WriteString(">")
	if w.Indenter != nil {
		w.last = Event{StateEnded, DTDElemNode, 0}
	}
	return w.printer.cachedWriteError()
}

type DTDEntity struct {
	Name     string
	Content  string
	IsPE     bool
	PublicID string
	SystemID string
	NDataID  string
}

func (d DTDEntity) kind() NodeKind { return DTDEntityNode }

func (d DTDEntity) writable() {}

func (d DTDEntity) write(w *Writer) error {
	if w.Enforce {
		if len(d.Name) == 0 {
			return fmt.Errorf("xmlwriter: ENTITY name must not be empty")
		}
		if err := CheckName(d.Name); err != nil {
			return err
		}
		if err := w.checkParent(NoNode, DTDNode); err != nil {
			return err
		}
	}

	if err := w.writeBeginNext(DTDEntityNode); err != nil {
		return err
	}
	w.printer.WriteString("<!ENTITY ")
	if d.IsPE {
		w.printer.WriteString("% ")
	}
	w.printer.WriteString(d.Name)

	if d.SystemID != "" || d.PublicID != "" {
		w.printer.WriteByte(' ')

		// external ref
		if w.Enforce && d.Content != "" {
			return fmt.Errorf("xmlwriter: external ID and content cannot both be provided")
		}
		if err := w.printer.writeExternalID(d.PublicID, d.SystemID, w.Enforce); err != nil {
			return err
		}
		if d.NDataID != "" {
			if !d.IsPE {
				w.printer.WriteString(" NDATA ")
				if w.Enforce {
					if err := CheckName(d.NDataID); err != nil {
						return err
					}
				}
				w.printer.WriteString(d.NDataID)
			} else {
				return fmt.Errorf("xmlwriter: IsPE and NDataID both provided")
			}
		}

	} else {
		// explicit content (parental advisory)
		if w.Enforce && d.NDataID != "" {
			return fmt.Errorf("xmlwriter: external ID required for NDataID")
		}

		w.printer.WriteByte(' ')
		if err := w.printer.writeEntityValue(d.Content, w.Enforce); err != nil {
			return err
		}
	}

	w.printer.WriteString(">")
	if w.Indenter != nil {
		w.last = Event{StateEnded, DTDEntityNode, 0}
	}
	return w.printer.cachedWriteError()
}

type DTDAttList struct {
	Name  string
	Attrs []DTDAttr
}

func (d DTDAttList) start(w *Writer) error {
	if err := w.pushBegin(DTDAttListNode, []NodeKind{NoNode, DTDNode}); err != nil {
		return err
	}
	np := &w.nodes[w.current+1]
	np.clear()
	np.kind = DTDAttListNode
	np.dtdAttList = d
	return w.pushEnd()
}

func (d DTDAttList) kind() NodeKind { return DTDAttListNode }

func (d DTDAttList) write(w *Writer) error {
	if err := w.StartDTDAttList(d); err != nil {
		return err
	}
	if err := w.EndDTDAttList(); err != nil {
		return err
	}
	return nil
}

func (d DTDAttList) open(n *node, w *Writer) error {
	if w.Enforce {
		if len(d.Name) == 0 {
			return fmt.Errorf("xmlwriter: DTD attlist name must not be empty")
		}
		if err := CheckName(d.Name); err != nil {
			return err
		}
	}
	w.printer.WriteString("<!ATTLIST ")
	w.printer.WriteString(d.Name)
	return w.printer.cachedWriteError()
}

func (d DTDAttList) opened(n *node, w *Writer, prev NodeState) error {
	for _, attr := range d.Attrs {
		if err := attr.write(w); err != nil {
			return err
		}
	}
	n.dtdAttList.Attrs = nil
	return nil
}

func (d DTDAttList) end(n *node, w *Writer, prev NodeState) error {
	return w.printer.WriteByte('>')
}

type AttType string

const (
	AttString   AttType = "CDATA"
	AttID       AttType = "ID"
	AttIDRef    AttType = "IDREF"
	AttIDRefs   AttType = "IDREFS"
	AttEntity   AttType = "ENTITY"
	AttEntities AttType = "ENTITIES"
	AttNmtoken  AttType = "NMTOKEN"
	AttNmtokens AttType = "NMTOKENS"
)

type DTDAttr struct {
	Name     string
	Type     AttType
	Required bool
	Decl     string
}

func (d DTDAttr) kind() NodeKind { return DTDAttrNode }

func (d DTDAttr) write(w *Writer) error {
	if w.Enforce {
		if err := w.checkParent(NoNode, DTDAttListNode); err != nil {
			return err
		}
		if len(d.Name) == 0 {
			return fmt.Errorf("xmlwriter: DTD attr name must not be empty")
		}
		if len(d.Type) == 0 {
			return fmt.Errorf("xmlwriter: DTD attr type must not be empty")
		}
		if err := CheckName(d.Name); err != nil {
			return err
		}
	}

	if err := w.writeBeginNext(DTDAttrNode); err != nil {
		return err
	}

	// HACK: if there are no parents and we are writing these outside an
	// attlist, this leading space will always be present.
	w.printer.WriteByte(' ')

	w.printer.WriteString(d.Name)
	w.printer.WriteByte(' ')
	w.printer.WriteString(string(d.Type))
	w.printer.WriteByte(' ')

	if d.Decl != "" {
		if d.Required {
			w.printer.WriteString(`#FIXED `)
		}
		w.printer.WriteString(`"`)
		w.printer.EscapeAttrString(d.Decl)
		w.printer.WriteByte('"')

	} else {
		if d.Required {
			w.printer.WriteString("#REQUIRED")
		} else {
			w.printer.WriteString("#IMPLIED")
		}
	}
	if w.Indenter != nil {
		w.last = Event{StateEnded, DTDAttrNode, 0}
	}
	return w.printer.cachedWriteError()
}

type Notation struct {
	Name     string
	SystemID string
	PublicID string
}

func (n Notation) kind() NodeKind { return NotationNode }

func (n Notation) write(w *Writer) error {
	if w.Enforce {
		if err := w.checkParent(NoNode, DTDNode); err != nil {
			return err
		}
		if len(n.Name) == 0 {
			return fmt.Errorf("xmlwriter: NOTATION name must not be empty")
		}
		if err := CheckName(n.Name); err != nil {
			return err
		}
		if len(n.PublicID) == 0 && len(n.SystemID) == 0 {
			return fmt.Errorf("xmlwriter: NOTATION requires external ID: '<!NOTATION' S Name S (ExternalID | PublicID) S? '>'")
		}
	}

	if err := w.writeBeginNext(NotationNode); err != nil {
		return err
	}
	w.printer.WriteString("<!NOTATION ")
	w.printer.WriteString(n.Name)
	w.printer.WriteByte(' ')
	if n.SystemID != "" {
		if err := w.printer.writeExternalID(n.PublicID, n.SystemID, w.Enforce); err != nil {
			return err
		}
	} else if n.PublicID != "" {
		if err := w.printer.writePublicID(n.PublicID, n.SystemID, w.Enforce); err != nil {
			return err
		}
	}
	w.printer.WriteString(">")
	if w.Indenter != nil {
		w.last = Event{StateEnded, NotationNode, 0}
	}
	return w.printer.cachedWriteError()
}
