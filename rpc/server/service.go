package server

import (
	"dotxt/task"
	"net/rpc"
	"sync"
)

type Service any

var services = map[string]func() Service{
<<<<<<< Updated upstream
	"TaskService": func() Service { return &TaskService{} },
||||||| Stash base
=======
	"TaskService": func() Service {
		return &TaskService{}
	},
>>>>>>> Stashed changes
}

func init() {
	for name, svcInit := range services {
		rpc.RegisterName(name, svcInit())
	}
}

type TaskService struct {
	mu sync.RWMutex
}

func (s *TaskService) AddTask(args *ArgTask, reply *ArgErr) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	err := task.AddTaskFromStr(args.Task, args.Path)
	if err != nil {
		reply.Err = err.Error()
	}
	return nil
}
