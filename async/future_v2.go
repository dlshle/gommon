package async

// TODO future V2 with generic and more efficient execution
// future.transformOn(prevFuture, onSuccess, executor) -> this basically adds a listener on prevFuture to continue execute logic with onSuccess upon completion using executor,
// which avoids blocking or waiting from the next future.
// also, future.transformOn returns a future which wraps the result of onSuccess, so that the next future can be chained.
