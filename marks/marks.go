package marks

type Category struct {
	Name string
}

type Score struct {
	Unknown     bool
	Hidden      bool
	Actual, Max float64
}
