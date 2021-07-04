package status

type Status struct{}

func (s Status) BadRequest(msg string) [3]string {
	return [3]string{"400", "Bad Request", msg}
}

func (s Status) NotFound(msg string) [3]string {
	return [3]string{"404", "Not Found", msg}
}

func (s Status) InternalError(msg string) [3]string {
	return [3]string{"500", "Internal Server Error", msg}
}
