package utils

import (
	"container/list"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
)

type (
	jQueue struct {
		Jobber
		prio       bool
		resChannel chan error
	}

	dejQueue struct {
		resChannel chan *jQueue
	}

	JPool struct {
		JQprio        *list.List
		jqNormal      *list.List
		qChannel      chan *jQueue
		deqChannel    chan *dejQueue
		chShutdownQ   chan string
		jChannel      chan string
		chShutdownJob chan struct{}
		WGshutdown    sync.WaitGroup
		jqueueds      int32
		actRoutines   int32
		qCapacity     int32
	}
)

type Jobber interface {
	Run(jobRoutine int)
}

// New creates a new JPool.
func New(numberOfRoutines int, qCapacity int32) (jP *JPool) {
	jP = &JPool{
		JQprio:        list.New(),
		jqNormal:      list.New(),
		qChannel:      make(chan *jQueue),
		deqChannel:    make(chan *dejQueue),
		chShutdownQ:   make(chan string),
		jChannel:      make(chan string, qCapacity),
		chShutdownJob: make(chan struct{}),
		jqueueds:      0,
		actRoutines:   0,
		qCapacity:     qCapacity,
	}

	for jobRoutine := 0; jobRoutine < numberOfRoutines; jobRoutine++ {
		jP.WGshutdown.Add(1)

		go jP.jobRoutine(jobRoutine)
	}

	go jP.queueRoutine()

	return jP
}

func (jP *JPool) Shutdown(goRoutine string) (err error) {
	defer catchPanic(&err, goRoutine, "Shutdown")

	fmt.Println(goRoutine, "Shutdown", "Started")
	fmt.Println(goRoutine, "Shutdown", "Queue Routine")

	jP.chShutdownQ <- "Shutdown"
	<-jP.chShutdownQ

	close(jP.chShutdownQ)
	close(jP.qChannel)
	close(jP.deqChannel)

	fmt.Println(goRoutine, "Shutdown", "Shutting Down Job Routines")

	close(jP.chShutdownJob)
	jP.WGshutdown.Wait()

	close(jP.jChannel)

	fmt.Println(goRoutine, "Shutdown Completed")
	return err
}

func (jP *JPool) JQueue(goRoutine string, jober Jobber, prio bool) (err error) {
	defer catchPanic(&err, goRoutine, "jQueue")

	job := jQueue{
		jober,
		prio,
		make(chan error),
	}

	defer close(job.resChannel)

	jP.qChannel <- &job
	err = <-job.resChannel

	return err
}

func (jP *JPool) QueuedJobs() int32 {
	return atomic.AddInt32(&jP.jqueueds, 0)
}

func (jP *JPool) ActRoutines() int32 {
	return atomic.AddInt32(&jP.actRoutines, 0)
}

func catchPanic(err *error, goRoutine string, functionName string) {
	if r := recover(); r != nil {
		// Capture the stack trace.
		buf := make([]byte, 10000)
		runtime.Stack(buf, false)

		fmt.Println(goRoutine, functionName, "PANIC Defered", r, " : Stack Trace ", string(buf))

		if err != nil {
			*err = fmt.Errorf("%v", r)
		}
	}
}

func (jP *JPool) queueRoutine() {
	for {
		select {
		case <-jP.chShutdownQ:
			fmt.Println("shutdown queue Routine")
			jP.chShutdownQ <- "Down"
			return

		case jQueue := <-jP.qChannel:
			jP.Enqueue(jQueue)
			break

		case dejQueue := <-jP.deqChannel:
			jP.Dequeue(dejQueue)
			break
		}
	}
}

func (jP *JPool) Enqueue(jQueue *jQueue) {
	defer catchPanic(nil, "Queue", "Enqueue")

	if atomic.AddInt32(&jP.jqueueds, 0) == jP.qCapacity {
		jQueue.resChannel <- fmt.Errorf("Job Pool At Capacity")
		return
	}

	if jQueue.prio == true {
		jP.JQprio.PushBack(jQueue)
	} else {
		jP.jqNormal.PushBack(jQueue)
	}

	atomic.AddInt32(&jP.jqueueds, 1)

	jQueue.resChannel <- nil

	jP.jChannel <- "Wake Up"
}

func (jP *JPool) Dequeue(dejQueue *dejQueue) {
	defer catchPanic(nil, "Queue", "Dequeue")

	var nextJob *list.Element

	if jP.JQprio.Len() > 0 {
		nextJob = jP.JQprio.Front()
		jP.JQprio.Remove(nextJob)
	} else {
		nextJob = jP.jqNormal.Front()
		jP.jqNormal.Remove(nextJob)
	}

	atomic.AddInt32(&jP.jqueueds, -1)

	job := nextJob.Value.(*jQueue)

	dejQueue.resChannel <- job
}

func (jP *JPool) jobRoutine(jobRoutine int) {
	for {
		select {
		case <-jP.chShutdownJob:
			fmt.Println("shutdown job routine : ", jobRoutine)
			jP.WGshutdown.Done()
			return

		case <-jP.jChannel:
			jP.JobSafety(jobRoutine)
			break
		}
	}
}

func (jP *JPool) dejQueue() (job *jQueue, err error) {
	defer catchPanic(&err, "jobRoutine", "dejQueue")

	reqJob := dejQueue{
		resChannel: make(chan *jQueue), // Result Channel.
	}

	defer close(reqJob.resChannel)

	jP.deqChannel <- &reqJob
	job = <-reqJob.resChannel

	return job, err
}

func (jP *JPool) JobSafety(jobRoutine int) {
	defer catchPanic(nil, "jobRoutine", "JobSafety")
	defer atomic.AddInt32(&jP.actRoutines, -1)

	atomic.AddInt32(&jP.actRoutines, 1)

	jQueue, err := jP.dejQueue()
	if err != nil {
		fmt.Println("doJob safely error :", err)
		return
	}

	jQueue.Run(jobRoutine)
}
