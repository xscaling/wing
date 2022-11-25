package context

type ControllerContext struct {
}

func New() (ControllerContext, error) {
	ctx := ControllerContext{}
	return ctx, nil
}
