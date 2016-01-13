package main

import (
	"flag"
	"reflect"
	"strings"
)

type FlagSet struct {
	*flag.FlagSet
	undef []string
}

func NewFlagSet(name string, errorHandling flag.ErrorHandling) *FlagSet {
	return &FlagSet{FlagSet: flag.NewFlagSet(name, errorHandling)}
}

func (fs *FlagSet) ParseDefined(args []string) error {
	def, undef := fs.SplitDefined(args)
	fs.undef = undef
	return fs.Parse(def)
}

func (fs FlagSet) Undefined() []string {
	return fs.undef
}

func (fs FlagSet) SplitDefined(args []string) ([]string, []string) {
	mp := map[string]*flag.Flag{}
	fs.VisitAll(func(f *flag.Flag) {
		mp[f.Name] = f
	})

	def := make([]string, 0, len(args))
	undef := make([]string, 0, len(args))

	for len(args) > 0 {

		isFlag := len(args[0]) > 0 && args[0][0] == '-'

		if f, has := mp[strings.TrimLeft(args[0], "-")]; isFlag && has {

			m := reflect.ValueOf(f.Value).MethodByName("IsBoolFlag")
			isBool := m.IsValid() && m.Call(nil)[0].Bool()

			if isBool || strings.ContainsRune(args[0], '=') {
				def = append(def, args[0])
				args = args[1:]
			} else {
				def = append(def, args[0], args[1])
				args = args[2:]
			}

		} else {
			undef = append(undef, args[0])
			args = args[1:]
		}
	}

	return def, undef
}
