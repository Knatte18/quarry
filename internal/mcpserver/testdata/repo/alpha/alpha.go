package alpha

// Widget is a fixture type.
type Widget struct {
	Name string
}

// Greet is a fixture method on Widget.
func (w Widget) Greet() string {
	return "hi " + w.Name
}

// NewWidget is a fixture function.
func NewWidget(name string) Widget {
	return Widget{Name: name}
}

// MaxWidgets is a fixture const.
const MaxWidgets = 10

// DefaultWidget is a fixture var.
var DefaultWidget = Widget{Name: "default"}
