package common

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
)

var insnRE = regexp.MustCompile(`^([0-9a-f]{8}) ([a-z][0-9a-z_.]*) +(EMPTY|[0-9DJKACFSUdjkam]+)$`)

func ParseInsnDescriptionLine(line string) (*InsnDescription, error) {
	matches := insnRE.FindStringSubmatch(line)
	if matches == nil {
		return nil, errors.New("malformed insn description line")
	}

	wordStr := matches[1]
	mnemonic := matches[2]
	insnFmtStr := matches[3]

	word64, err := strconv.ParseUint(wordStr, 16, 32)
	if err != nil {
		panic("should never happen")
	}
	word := uint32(word64)

	insnFmt, err := ParseInsnFormat(insnFmtStr)
	if err != nil {
		return nil, err
	}

	return &InsnDescription{
		Word:     word,
		Mnemonic: mnemonic,
		Format:   insnFmt,
	}, nil
}

func ParseInsnFormat(input string) (*InsnFormat, error) {
	// special-case "EMPTY"
	if input == "EMPTY" {
		return &InsnFormat{
			Args: nil,
		}, nil
	}

	inputRunes := make([]rune, 0, len(input))
	for _, ch := range input {
		inputRunes = append(inputRunes, ch)
	}

	lexer := insnFormatLexer{
		input: inputRunes,
	}

	var args []*Arg
	for !lexer.eof() {
		a, err := lexer.consumeArg()
		if err != nil {
			return nil, err
		}

		args = append(args, a)
	}

	return &InsnFormat{
		Args: args,
	}, nil
}

type insnFormatLexer struct {
	input []rune

	curr int
}

func (l *insnFormatLexer) eof() bool {
	return l.curr >= len(l.input)
}

func (l *insnFormatLexer) eat() rune {
	result := l.input[l.curr]
	l.curr++
	return result
}

func (l *insnFormatLexer) peek() (next rune, wouldEOF bool) {
	if l.eof() {
		return 0, true
	}
	return l.input[l.curr], false
}

func (l *insnFormatLexer) consumeArg() (*Arg, error) {
	// EOF is checked outside (in ParseInsnFormat)
	prefixCh := l.eat()

	switch prefixCh {
	case 'D':
		return makeRegArg(0, ArgKindIntReg), nil
	case 'J':
		return makeRegArg(5, ArgKindIntReg), nil
	case 'K':
		return makeRegArg(10, ArgKindIntReg), nil
	case 'A':
		return makeRegArg(15, ArgKindIntReg), nil

	case 'C':
		offsetCh := l.eat()
		offset, err := parseOffsetCh(offsetCh)
		if err != nil {
			return nil, err
		}

		return makeFCCRegArg(offset), nil

	case 'F':
		offsetCh := l.eat()
		offset, err := parseOffsetCh(offsetCh)
		if err != nil {
			return nil, err
		}

		return makeRegArg(offset, ArgKindFPReg), nil

	case 'S', 'U':
		var kind ArgKind
		if prefixCh == 'S' {
			kind = ArgKindSignedImm
		} else {
			kind = ArgKindUnsignedImm
		}

		slots, err := l.consumeAtLeastOneSlot()
		if err != nil {
			return nil, err
		}

		return &Arg{
			Kind:  kind,
			Slots: slots,
		}, nil
	}

	return nil, fmt.Errorf("invalid prefix char %s", strconv.QuoteRune(prefixCh))
}

func (l *insnFormatLexer) consumeAtLeastOneSlot() ([]*Slot, error) {
	var result []*Slot
	for {
		ch, wouldEOF := l.peek()
		if wouldEOF {
			break
		}

		_, err := parseOffsetCh(ch)
		if err != nil {
			break
		}

		s, err := l.consumeSlot()
		if err != nil {
			return nil, err
		}

		result = append(result, s)
	}

	if len(result) == 0 {
		return nil, errors.New("no slot was consumed")
	}

	return result, nil
}

func (l *insnFormatLexer) consumeSlot() (*Slot, error) {
	offsetCh := l.eat()
	offset, err := parseOffsetCh(offsetCh)
	if err != nil {
		return nil, err
	}

	width := l.consumeUint()

	return &Slot{
		Offset: offset,
		Width:  width,
	}, nil
}

func (l *insnFormatLexer) consumeUint() uint {
	firstCh := l.eat()
	result := uint(firstCh - '0')

	for {
		nextCh, wouldEOF := l.peek()
		if wouldEOF {
			break
		}

		if nextCh < '0' || nextCh > '9' {
			break
		}

		_ = l.eat() // must be same as nextCh
		result = 10*result + uint(nextCh-'0')
	}

	return result
}

func parseOffsetCh(ch rune) (uint, error) {
	switch ch {
	case 'd':
		return 0, nil
	case 'j':
		return 5, nil
	case 'k':
		return 10, nil
	case 'a':
		return 15, nil
	case 'm':
		return 16, nil
	}

	return 0, fmt.Errorf("invalid offset char %s", strconv.QuoteRune(ch))
}

func makeRegArg(offset uint, kind ArgKind) *Arg {
	return &Arg{
		Kind: kind,
		Slots: []*Slot{
			{Offset: offset, Width: 5},
		},
	}
}

func makeFCCRegArg(offset uint) *Arg {
	return &Arg{
		Kind: ArgKindFCCReg,
		Slots: []*Slot{
			{Offset: offset, Width: 3},
		},
	}
}
