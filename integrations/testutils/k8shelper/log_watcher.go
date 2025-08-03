package k8shelper

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"strings"
	"sync"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
)

// ...existing code...

// LogWatcher represents a log watching session
type LogWatcher struct {
	ctx          context.Context
	cancel       context.CancelFunc
	found        chan string
	podName      string
	namespace    string
	clientset    kubernetes.Interface
	mu           sync.RWMutex
	logs         []string
	searchString string
}

// WatchLogsForString starts watching pod logs in a separate goroutine and searches for a specific string
func (k *k8sHelper) WatchLogsForString(searchString string) (*LogWatcher, error) {
	ctx, cancel := context.WithCancel(context.Background())

	watcher := &LogWatcher{
		ctx:          ctx,
		cancel:       cancel,
		found:        make(chan string, 1),
		podName:      k.name,
		namespace:    k.namespace,
		clientset:    k.clientset,
		logs:         make([]string, 0),
		searchString: searchString,
	}

	go func() {
		defer close(watcher.found)

		// Get pod logs with follow option
		req := k.clientset.CoreV1().Pods(k.namespace).GetLogs(k.name, &corev1.PodLogOptions{
			Follow: true,
		})

		podLogs, err := req.Stream(ctx)
		if err != nil {
			fmt.Printf("Error getting pod logs: %v\n", err)
			return
		}
		defer podLogs.Close()

		scanner := bufio.NewScanner(podLogs)
		for scanner.Scan() {
			select {
			case <-ctx.Done():
				return
			default:
				line := scanner.Text()
				fmt.Printf("[%s] %s\n", k.name, line) // Print logs for debugging

				// Store log line
				watcher.mu.Lock()
				watcher.logs = append(watcher.logs, line)
				watcher.mu.Unlock()

				// Check if the line contains the search string
				if strings.Contains(line, searchString) {
					select {
					case watcher.found <- line:
					case <-ctx.Done():
						return
					}
					return
				}
			}
		}

		if err := scanner.Err(); err != nil && err != io.EOF {
			fmt.Printf("Error scanning logs: %v\n", err)
		}
	}()

	return watcher, nil
}

// Wait waits for the search string to be found or context to be cancelled
func (lw *LogWatcher) Wait() (string, error) {
	select {
	case line := <-lw.found:
		return line, nil
	case <-lw.ctx.Done():
		return "", lw.ctx.Err()
	}
}

// Stop stops the log watcher
func (lw *LogWatcher) Stop() {
	lw.cancel()
}

func (lw *LogWatcher) GetSearchString() string {
	return lw.searchString
}

func (lw *LogWatcher) GetPodName() string {
	return lw.podName
}

// GetLogs returns all collected log lines
func (lw *LogWatcher) GetLogs() []string {
	lw.mu.RLock()
	defer lw.mu.RUnlock()
	logs := make([]string, len(lw.logs))
	copy(logs, lw.logs)
	return logs
}
