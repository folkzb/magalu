package cmd

import "os"

type osArgParser struct {
	allArgs     []string
	mainArgs    []string
	chainedArgs [][]string
}

// Includes all but the first argument, which is the program name
func (o *osArgParser) AllArgs() []string {
	if o.allArgs == nil {
		o.allArgs = os.Args[1:]
	}
	return o.allArgs
}

// Includes all arguments until the first "!" separator. Same as first array in ChainedArgs
func (o *osArgParser) MainArgs() []string {
	if o.mainArgs == nil {
		if len(o.ChainedArgs()) > 0 {
			o.mainArgs = o.ChainedArgs()[0]
		} else {
			o.mainArgs = []string{}
		}
	}
	return o.mainArgs
}

// Includes all arguments, separated by "!". First array is the same as AllArgs
func (o *osArgParser) ChainedArgs() [][]string {
	if o.chainedArgs == nil {
		o.chainedArgs = splitSlice(o.AllArgs(), "!")
	}
	return o.chainedArgs
}
