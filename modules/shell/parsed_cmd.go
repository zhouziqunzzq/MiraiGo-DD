package shell

import "errors"
import "strings"

const CmdIdentifier = '/'

type ParsedCmd struct {
	Name string
	Args []string
	//KwArgs map[string]string
}

func NewParsedCmd(name string) *ParsedCmd {
	return &ParsedCmd{
		Name: name,
		Args: make([]string, 0),
		//KwArgs: make(map[string]string),
	}
}

func (pc *ParsedCmd) AppendArg(arg string) {
	pc.Args = append(pc.Args, arg)
}

//func (pc *ParsedCmd) AppendKwArg(kw, arg string) error {
//	if _, ok := pc.KwArgs[kw]; ok {
//		return errors.New("Keyword " + kw + " already exist")
//	}
//	pc.KwArgs[kw] = arg
//	return nil
//}

// try parsing cmd from a string
// return
// 	- nil if s is not a cmd, or *ParsedCmd for a successful parsing
// 	- an error indicating the error in the parsing process, if any
func parseCmd(s string) (*ParsedCmd, error) {
	s = strings.TrimSpace(s)
	if len(s) == 0 || s[0] != CmdIdentifier {
		return nil, nil
	}

	s = s[1:]
	ss := strings.Split(s, " ")
	if len(ss) == 0 {
		return nil, errors.New("command name not specified")
	}

	// parse cmd name
	pc := NewParsedCmd(ss[0])
	ss = ss[1:]

	// parse args
	for _, rawStr := range ss {
		// skip spaces
		rawStr = strings.TrimSpace(rawStr)
		if len(rawStr) == 0 {
			continue
		}

		pc.AppendArg(rawStr)
	}

	return pc, nil
}
