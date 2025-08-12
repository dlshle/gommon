package async

type AsyncTask func()

type TaskFunc func() error

type ComputableAsyncTask func() interface{}

type ComputableAsyncTaskWithError func() (interface{}, error)

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
