package schema_flags

type ProxyFlagSpec struct {
	Set   func(rawValue string) error
	Parse func(rawValue string) (value any, err error)
	Usage func() string
}

// doesn't hold a value at all, just proxy to callback
type schemaFlagValueProxy struct {
	schemaFlagValueCommon
	proxy ProxyFlagSpec
}

var _ SchemaFlagValue = (*schemaFlagValueProxy)(nil)

func newSchemaFlagValueProxy(
	desc SchemaFlagValueDesc,
	proxy ProxyFlagSpec,
) *schemaFlagValueProxy {
	return &schemaFlagValueProxy{
		initSchemaFlagValueCommon(desc),
		proxy,
	}
}

func (o *schemaFlagValueProxy) Usage() string {
	if o.proxy.Usage != nil {
		return o.proxy.Usage()
	}
	return o.schemaFlagValueCommon.Usage()
}

func (o *schemaFlagValueProxy) Set(rawValue string) error {
	_ = o.schemaFlagValueCommon.Set(rawValue)
	if rawValue == ValueHelpIsRequired {
		return nil
	}
	if o.proxy.Set != nil {
		return o.proxy.Set(rawValue)
	}
	return nil
}

func (o *schemaFlagValueProxy) Parse() (value any, err error) {
	if o.rawValue == ValueHelpIsRequired {
		return nil, ErrWantHelp
	}
	if o.proxy.Parse != nil {
		return o.proxy.Parse(o.rawValue)
	}
	return o.schemaFlagValueCommon.Parse()
}
