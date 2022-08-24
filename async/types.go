package async

type AsyncTask func()

type ComputableAsyncTask func() interface{}

type Waitable interface {
	Wait()
	IsOpen() bool
}

type Gettable interface {
	Get() interface{}
}

type WaitGettable interface {
	Waitable
	Gettable
}

type Executor interface {
	Execute(task AsyncTask)
}
