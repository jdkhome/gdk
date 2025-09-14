package model

import "github.com/jdkhome/gdk/enums"

// IPVersion IP协议的版本
type IPVersion enums.Member[enums.Value]

var (
	_IPVersionEnumBuilder = enums.NewBuilder[IPVersion]()
	IPVersion_V4          = _IPVersionEnumBuilder.Add("ipv4", enums.NewEnumValue("IPV4"))
	IPVersion_V6          = _IPVersionEnumBuilder.Add("ipv6", enums.NewEnumValue("IPV6"))
	IPVersions            = _IPVersionEnumBuilder.Build()
)
