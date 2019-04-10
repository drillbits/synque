// Copyright 2019 drillbits
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

package synque

import (
	"encoding/json"
	"log"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"sync"

	"github.com/rs/xid"
	"google.golang.org/api/drive/v3"
)

type Queue struct {
	s sync.Map
}

func NewQueue() *Queue {
	return &Queue{}
}

func (q *Queue) Store(key string, value *Task) {
	q.s.Store(key, value)
}

func (q *Queue) Delete(key string) {
	q.s.Delete(key)
}

func (q *Queue) Has(key string) bool {
	_, ok := q.s.Load(key)
	return ok
}

func (q Queue) MarshalJSON() ([]byte, error) {
	var tasks []*Task
	q.s.Range(func(k, v interface{}) bool {
		if task, ok := v.(*Task); ok {
			tasks = append(tasks, task)
		}
		return true
	})
	return json.Marshal(&struct {
		Tasks []*Task `json:"tasks"`
	}{
		Tasks: tasks,
	})
}

type Dispatcher struct {
	pool    chan *Worker
	queue   chan *Task
	workers []*Worker
	waiting *Queue
	wg      sync.WaitGroup
	quit    chan struct{}
	client  *http.Client
}

func NewDispatcher(client *http.Client, maxWorkers int, maxQueues int) *Dispatcher {
	d := &Dispatcher{
		pool:    make(chan *Worker, maxWorkers),
		queue:   make(chan *Task, maxQueues),
		waiting: NewQueue(),
		quit:    make(chan struct{}),
		client:  client,
	}

	d.workers = make([]*Worker, cap(d.pool))
	for i := 0; i < cap(d.pool); i++ {
		w := &Worker{
			dispatcher: d,
			task:       make(chan *Task),
			quit:       make(chan struct{}),
		}
		d.workers[i] = w
	}

	return d
}

func (d *Dispatcher) Start() {
	for _, w := range d.workers {
		w.Start()
	}

	for {
		select {
		case t := <-d.queue:
			guid := xid.New()
			id := guid.String()
			t.ID = id
			d.waiting.Store(id, t)
			log.Printf("waiting %s\n", id)
			go func() {
				worker := <-d.pool
				if d.waiting.Has(id) {
					t.Running = true
					worker.task <- t
				}
			}()
		case <-d.quit:
			log.Println("quit dispatcher")
			return
		}
	}
}

func (d *Dispatcher) Wait() {
	d.wg.Wait()
}

func (d *Dispatcher) Enqueue(t *Task) {
	d.wg.Add(1)
	d.queue <- t
}

type Worker struct {
	dispatcher *Dispatcher
	task       chan *Task
	quit       chan struct{}
}

func (w *Worker) Start() {
	go func() {
		for {
			w.dispatcher.pool <- w

			select {
			case t := <-w.task:
				err := t.Do(w.dispatcher.client)
				if err != nil {
					log.Printf("failed to do: %s", err)
				}
				w.dispatcher.waiting.Delete(t.ID)
				w.dispatcher.wg.Done()
			case <-w.quit:
				return
			}
		}
	}()
}

type Task struct {
	ID          string   `json:"id"`
	Filename    string   `json:"filename"`
	Description string   `json:"description"`
	Parents     []string `json:"parents"`
	MimeType    string   `json:"mimeType"`
	Running     bool     `json:"running"`
}

func (t *Task) Do(client *http.Client) error {
	service, err := drive.New(client)
	if err != nil {
		return err
	}

	f, err := os.Open(t.Filename)
	if err != nil {
		return err
	}

	filename := filepath.Base(t.Filename)
	if t.MimeType == "" {
		t.MimeType = mime.TypeByExtension(filepath.Ext(filename))
	}
	dst := &drive.File{
		Name:        filename,
		Description: t.Description,
		Parents:     t.Parents,
		MimeType:    t.MimeType,
	}

	log.Printf("uploading %s\n", filename)
	res, err := service.Files.Create(dst).Media(f).Do()
	if err != nil {
		return err
	}
	log.Printf("uploaded https://drive.google.com/file/d/%s/view\n", res.Id)

	return nil
}
