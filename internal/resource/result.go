package resource

type Result struct {
	err     error
	requeue bool
}

func (r Result) Err() error {
	return r.err
}

func (r Result) Requeue() bool {
	return r.requeue
}

func (r Result) Yield() bool {
	return r.requeue || r.err != nil
}

var (
	Done    = Result{}
	Requeue = Result{requeue: true}
)
