package resource

// Logger interfaces are used to provide logging to the resources package.
type Logger interface {
	Info(string, ...interface{})
	Error(error, string, ...interface{})
}
