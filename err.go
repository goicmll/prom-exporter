package prometheus

type PromError struct {
	Msg string
}

func (pe PromError) Error() string {
	return pe.Msg
}
