package common

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

type InsnDescription struct {
	Word       uint32
	Mnemonic   string
	Format     *InsnFormat
	OrigFormat *InsnFormat
	Attribs    map[string]string
}

type InsnFormat struct {
	Args []*Arg
}

type Arg struct {
	Kind  ArgKind
	Slots []*Slot
	Post  PostprocessOp
}

type Slot struct {
	Offset uint
	Width  uint
}

type PostprocessOp struct {
	Kind   PostprocessOpKind
	Amount int
}

type PostprocessOpKind int

const (
	PostprocessOpKindNone PostprocessOpKind = 0
	PostprocessOpKindAdd  PostprocessOpKind = 1
	PostprocessOpKindShl  PostprocessOpKind = 2
)

func (k *PostprocessOp) CanonicalRepr() string {
	switch k.Kind {
	case PostprocessOpKindNone:
		return ""
	case PostprocessOpKindAdd:
		return "p" + strconv.Itoa(k.Amount)
	case PostprocessOpKindShl:
		return "s" + strconv.Itoa(k.Amount)
	default:
		panic("unreachable")
	}
}

type ArgKind int

const (
	ArgKindUnknown     ArgKind = 0
	ArgKindIntReg      ArgKind = 1
	ArgKindFPReg       ArgKind = 2
	ArgKindFCCReg      ArgKind = 3
	ArgKindVReg        ArgKind = 4
	ArgKindXReg        ArgKind = 5
	ArgKindSignedImm   ArgKind = 6
	ArgKindUnsignedImm ArgKind = 7
)

func (k ArgKind) Validate() error {
	switch k {
	case ArgKindIntReg,
		ArgKindFPReg,
		ArgKindFCCReg,
		ArgKindVReg,
		ArgKindXReg,
		ArgKindSignedImm,
		ArgKindUnsignedImm:
		return nil
	}

	return fmt.Errorf("unknown arg kind: %d", k)
}

func (k ArgKind) IsImm() bool {
	switch k {
	case ArgKindSignedImm, ArgKindUnsignedImm:
		return true
	default:
		return false
	}
}

func (s *Slot) Validate() error {
	if s.Offset > 31 {
		return fmt.Errorf("slot offset %d > 31", s.Offset)
	}

	if s.Width == 0 {
		return errors.New("slot width is zero")
	}

	if s.MSB() > 31 {
		return fmt.Errorf("slot spans beyond insn word; MSB is %dth bit", s.MSB())
	}

	return nil
}

func (s *Slot) MSB() uint {
	return s.Offset + s.Width - 1
}

func (s *Slot) Bitmask() uint32 {
	// Example when offset = 5, width = 5:
	//
	// 1 << offset = 0b100000
	// (1 << offset) - 1 = 0b11111          <-- A
	//
	// MSB is bit 5 + 5 - 1 = 9
	// 1 << (MSB + 1) = 0b10000000000
	// (1 << (MSB + 1)) - 1 = 0b1111111111  <-- B
	//
	// B - A = 0b1111100000
	a := (uint64(1) << s.Offset) - 1
	b := (uint64(1) << (s.MSB() + 1)) - 1
	return uint32(b - a)
}

func (s *Slot) String() string {
	if s == nil {
		return "<nil Slot>"
	}

	err := s.Validate()
	if err != nil {
		return fmt.Sprintf("<invalid Slot: %#v>", s)
	}

	return fmt.Sprintf("<Slot %s>", s.CanonicalRepr())
}

func (s *Slot) CanonicalRepr() string {
	// returns things like "k16"
	var sb strings.Builder
	sb.WriteByte(offsetCharsLower[s.Offset])
	sb.WriteString(strconv.FormatInt(int64(s.Width), 10))
	return sb.String()
}

func (a *Arg) Validate() error {
	err := a.Kind.Validate()
	if err != nil {
		return err
	}

	if len(a.Slots) == 0 {
		return errors.New("arg has no slots")
	}

	switch a.Kind {
	case ArgKindIntReg, ArgKindFPReg, ArgKindVReg, ArgKindXReg:
		if len(a.Slots) != 1 {
			return errors.New("len(slots) != 1 for a register arg")
		}

		if a.Slots[0].Width != 5 {
			return errors.New("slot width not 5 for a register arg")
		}

	case ArgKindFCCReg:
		if len(a.Slots) != 1 {
			return errors.New("len(slots) != 1 for a FCC register arg")
		}

		if a.Slots[0].Width != 3 {
			return errors.New("slot width not 3 for a FCC register arg")
		}
	}

	var seenSlotsMask uint32
	for _, s := range a.Slots {
		err := s.Validate()
		if err != nil {
			return err
		}

		mask := s.Bitmask()
		if mask&seenSlotsMask != 0 {
			return fmt.Errorf("slot %s overlapped with other slots", s)
		}

		seenSlotsMask |= mask
	}

	return nil
}

func (a *Arg) Bitmask() uint32 {
	var result uint32
	for _, s := range a.Slots {
		result |= s.Bitmask()
	}
	return result
}

func (a *Arg) TotalWidth() uint {
	var result uint
	for _, s := range a.Slots {
		result += s.Width
	}
	return result
}

func (a *Arg) String() string {
	if a == nil {
		return "<nil Arg>"
	}

	err := a.Validate()
	if err != nil {
		return fmt.Sprintf("<invalid Arg: %#v>", a)
	}

	return fmt.Sprintf("<Arg %s>", a.CanonicalRepr())
}

const offsetCharsUpper = "D____J____K____A__________________________"
const offsetCharsLower = "d____j____k____am_n_______________________"

func (a *Arg) CanonicalRepr() string {
	var sb strings.Builder

	switch a.Kind {
	case ArgKindIntReg:
		sb.WriteByte(offsetCharsUpper[a.Slots[0].Offset])

	case ArgKindFPReg:
		sb.WriteRune('F')
		sb.WriteByte(offsetCharsLower[a.Slots[0].Offset])

	case ArgKindFCCReg:
		sb.WriteRune('C')
		sb.WriteByte(offsetCharsLower[a.Slots[0].Offset])

	case ArgKindVReg:
		sb.WriteRune('V')
		sb.WriteByte(offsetCharsLower[a.Slots[0].Offset])

	case ArgKindXReg:
		sb.WriteRune('X')
		sb.WriteByte(offsetCharsLower[a.Slots[0].Offset])

	case ArgKindSignedImm, ArgKindUnsignedImm:
		if a.Kind == ArgKindSignedImm {
			sb.WriteRune('S')
		} else {
			sb.WriteRune('U')
		}

		for _, s := range a.Slots {
			sb.WriteString(s.CanonicalRepr())
		}

	default:
		panic("unreachable")
	}

	if a.Post.Kind != PostprocessOpKindNone {
		sb.WriteRune('p')
		sb.WriteString(a.Post.CanonicalRepr())
	}

	return sb.String()
}

func (f *InsnFormat) Validate() error {
	return f.validate(false)
}

func (f *InsnFormat) ValidateManualSyntax() error {
	return f.validate(true)
}

func (f *InsnFormat) validate(manualSyntax bool) error {
	regsParsingFinished := false
	var seenArgsMask uint32
	for _, a := range f.Args {
		err := a.Validate()
		if err != nil {
			return err
		}

		mask := a.Bitmask()
		if mask&seenArgsMask != 0 {
			return fmt.Errorf("arg %s overlapped with other args", a)
		}

		seenArgsMask |= mask

		// register args must come before immediates for canonicalized syntax
		// skip the check in case we're validating manual syntax repr
		if manualSyntax {
			continue
		}

		isImm := a.Kind.IsImm()
		if !regsParsingFinished {
			if isImm {
				// first time seeing an immediate, mark end of register args
				regsParsingFinished = true
			}
			// still processing registers, and that's okay
		} else {
			if !isImm {
				return fmt.Errorf("register arg %s comes after immediate arg", a)
			}
			// we're all immediates now and all is fine
		}
	}

	return nil
}

func (f *InsnFormat) String() string {
	if f == nil {
		return "<nil InsnFormat>"
	}

	err := f.Validate()
	if err != nil {
		return fmt.Sprintf("<invalid InsnFormat: %#v>", f)
	}

	return fmt.Sprintf("<InsnFormat %s>", f.CanonicalRepr())
}

func (f *InsnFormat) CanonicalRepr() string {
	if len(f.Args) == 0 {
		return "EMPTY"
	}

	var sb strings.Builder
	for _, a := range f.Args {
		sb.WriteString(a.CanonicalRepr())
	}
	return sb.String()
}

func (f *InsnFormat) ArgsBitmask() uint32 {
	var mask uint32
	for _, a := range f.Args {
		mask |= a.Bitmask()
	}
	return mask
}

func (f *InsnFormat) MatchBitmask() uint32 {
	return ^f.ArgsBitmask()
}

func (d *InsnDescription) Validate() error {
	if d.Mnemonic == "" {
		return errors.New("empty mnemonic")
	}

	err := d.Format.Validate()
	if err != nil {
		return err
	}

	if d.Word&d.Format.ArgsBitmask() != 0 {
		return errors.New("insn word has non-zero bit inside arg slots")
	}

	return nil
}
