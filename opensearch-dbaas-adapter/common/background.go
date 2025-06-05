// Copyright 2024-2025 NetCracker Technology Corporation
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package common

import "sync"

// BackgroundExecutor runs tasks in a single goroutine. Tasks are submitted in queue and being executed according
// to the FIFO rule. Always use constructor NewBackgroundExecutor() to create new instance of the BackgroundExecutor.
// BackgroundExecutor can be shutdown by calling Shutdown() function.
type BackgroundExecutor struct {
	// queue contains submitted tasks that need to be run
	queue  chan func()
	active bool
	mutex  sync.Mutex
}

// NewBackgroundExecutor creates new BackgroundExecutor instance and starts goroutine which monitors it's task queue.
func NewBackgroundExecutor() *BackgroundExecutor {
	executor := BackgroundExecutor{
		queue:  make(chan func(), 5),
		active: true,
	}
	executor.start()
	return &executor
}

func (executor *BackgroundExecutor) Submit(work func()) {
	defer executor.mutex.Unlock()
	executor.mutex.Lock()
	if executor.active {
		executor.queue <- work
	}
}

// Shutdown deactivates the BackgroundExecutor so it will no longer accept work for submitting. All the works that have
// been already submitted will be executed and finished as usual.
func (executor *BackgroundExecutor) Shutdown() {
	defer executor.mutex.Unlock()
	executor.mutex.Lock()
	if executor.active {
		executor.active = false
		close(executor.queue)
	}
}

// Starts taking work from queue and running it until the task channel is closed.
func (executor *BackgroundExecutor) start() {
	go func() {
		for {
			work, more := <-executor.queue
			if more {
				work()
			} else {
				return
			}
		}
	}()
}
