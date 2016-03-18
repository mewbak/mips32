package mips32

import (
	"errors"
	"regexp"
	"strconv"
	"strings"
)

var (
	commentRegexp      = regexp.MustCompile("^(.*?)(#|//|;)(.*)$")
	directiveRegexp    = regexp.MustCompile("^\\.(text|word)\\s+" + constantNumberPattern + "$")
	symbolMarkerRegexp = regexp.MustCompile("^" + symbolNamePattern + ":$")
	instNameRegexp     = regexp.MustCompile("^[A-Z]*$")
)

// A TokenizedLine represents one line of an assembly program, translated into syntactic tokens.
// No more than one of Directive, SymbolDecl, and Instruction will be non-nil.
// The Comment field may be non-nil regardless of the other fields.
type TokenizedLine struct {
	LineNumber int
	Comment    *string

	Directive    *TokenizedDirective
	Instruction  *TokenizedInstruction
	SymbolMarker *string
}

// Equal returns true if this tokenized line is equivalent to another one.
// This is a deep comparison, and all fields (including the comment and line number) are compared.
func (t *TokenizedLine) Equal(t1 *TokenizedLine) bool {
	if t.LineNumber != t1.LineNumber {
		return false
	}
	if (t.Comment == nil) != (t1.Comment == nil) {
		return false
	} else if t.Comment != nil && *t.Comment != *t1.Comment {
		return false
	}
	if (t.Directive == nil) != (t1.Directive == nil) {
		return false
	} else if t.Directive != nil && *t.Directive != *t1.Directive {
		return false
	}
	if (t.Instruction == nil) != (t1.Instruction == nil) {
		return false
	} else if t.Instruction != nil && !t.Instruction.Equal(t1.Instruction) {
		return false
	}
	if (t.SymbolMarker == nil) != (t1.SymbolMarker == nil) {
		return false
	} else if t.SymbolMarker != nil && *t.SymbolMarker != *t1.SymbolMarker {
		return false
	}
	return true
}

// A TokenizedDirective represents a directive like ".text 0x5000" or ".data 0x0".
type TokenizedDirective struct {
	Name     string
	Constant uint32
}

// A TokenizedInstruction represents an instruction call.
type TokenizedInstruction struct {
	Name string
	Args []*ArgToken
}

// Equal returns whether or not two TokenizedInstructions are syntactically equivalent.
//
// Syntactic equivalence is not the same thing as semantic equivalence.
// For example, "J Symbol5" might do the same thing as "J 0x54", but the two expressions are
// syntactically different.
func (t *TokenizedInstruction) Equal(t1 *TokenizedInstruction) bool {
	if t.Name != t1.Name {
		return false
	}
	if len(t.Args) != len(t1.Args) {
		return false
	}
	for i, arg := range t.Args {
		if *arg != *t1.Args[i] {
			return false
		}
	}
	return true
}

// TokenizeSource takes a source file and tokenizes each line.
// It returns an array of tokenized lines, on an error if one occurred.
func TokenizeSource(source string) ([]TokenizedLine, error) {
	splitLines := strings.Split(source, "\n")
	res := make([]TokenizedLine, 0, len(splitLines))
	for lineNum, lineText := range splitLines {
		line, err := tokenizeLine(lineText)
		if err != nil {
			linePreamble := "error on line " + strconv.Itoa(lineNum+1) + ": "
			return nil, errors.New(linePreamble + err.Error())
		} else if (line == TokenizedLine{}) {
			continue
		}
		line.LineNumber = lineNum + 1
		res = append(res, line)
	}
	return res, nil
}

// tokenizeLine tokenizes a single line of assembly code.
func tokenizeLine(lineText string) (line TokenizedLine, err error) {
	trimmed := strings.TrimSpace(lineText)
	if len(trimmed) == 0 {
		return
	}

	commentMatch := commentRegexp.FindStringSubmatch(trimmed)
	if commentMatch != nil {
		commentStr := commentMatch[3]
		beforeComment := commentMatch[1]
		line, err = tokenizeLine(beforeComment)
		line.Comment = &commentStr
		return
	}

	directiveMatch := directiveRegexp.FindStringSubmatch(trimmed)
	if directiveMatch != nil {
		directiveConstant, err := parseConstant(directiveMatch[2])
		if err != nil {
			return line, err
		}
		return TokenizedLine{
			Directive: &TokenizedDirective{
				Name:     directiveMatch[1],
				Constant: directiveConstant,
			},
		}, nil
	}

	symbolMatch := symbolMarkerRegexp.FindStringSubmatch(trimmed)
	if symbolMatch != nil {
		return TokenizedLine{
			SymbolMarker: &symbolMatch[1],
		}, nil
	}

	fields := strings.Fields(trimmed)
	if len(fields) == 0 || !instNameRegexp.MatchString(fields[0]) {
		err = errors.New("invalid/missing instruction name")
		return
	}

	line.Instruction = &TokenizedInstruction{
		Name: fields[0],
		Args: make([]*ArgToken, len(fields)-1),
	}

	for i, field := range fields[1:] {
		if i != len(fields)-2 {
			if !strings.HasSuffix(field, ",") {
				err = errors.New("missing comma after operand " + strconv.Itoa(i+1))
				return
			}
			field = field[:len(field)-1]
		}
		line.Instruction.Args[i], err = ParseArgToken(field)
		if err != nil {
			err = errors.New("operand " + strconv.Itoa(i+1) + ": " + err.Error())
			return
		}
	}

	return
}