package eventline

type DateFormat string

const (
	DateFormatAbsolute DateFormat = "absolute"
	DateFormatRelative DateFormat = "relative"
)

var DateFormatValues = []DateFormat{
	DateFormatAbsolute,
	DateFormatRelative,
}
