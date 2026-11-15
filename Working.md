A Job is one piece of work stored as data: an id, a type, a payload, and a state. The states are WAITING, RUNNING, STALLED, COMPLETED, and FAILED. The handler is the function that actually does the work, and that comes later.

The repository is the interface for saving and loading jobs. MemoryRepository is the local one, used for tests, and it protects its map with a mutex. Postgres is the real database. It locks rows in SQL instead of using that mutex.

Something else creates the jobs, and they start in WAITING. PoolFunction is the bridge. It opens a job channel and starts RunWorkers in a goroutine. RunWorkers starts one goroutine per worker. Each worker takes a job, runs the handler, and saves the next state. A WaitGroup waits until those goroutines finish. It does not do the locking.

ClaimJobs reads WAITING and STALLED rows, marks them RUNNING, and returns them. The pool sends those RUNNING jobs on the channel. slots is how many workers are idle. inFlight is how many jobs have been sent and are not finished. When a worker finishes, it signals on free. If no worker is idle, the pool waits in select until that signal arrives or the context is cancelled.

If ClaimJobs returns nothing and a worker is still busy, the pool waits again. If it returns nothing and every worker is idle, the pool closes the job channel. The workers exit, RunWorkers sends back its error, and PoolFunction returns.

Each job gets a child context from context.WithTimeout, using the duration passed into RunWorkers. The handler sees that context. If it returns ErrNetworkTimeout before the work finishes, the worker stores STALLED. ClaimJobs will claim that row again. There is no attempt count yet, so a deadline shorter than the work retries until the process is cancelled. Next session starts there: record the attempt count and how long the try took, and store FAILED once the count hits a max.