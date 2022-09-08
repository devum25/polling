package main

import (
	"log"
	"net/http"
	"time"
)


const(
	numPollers = 2       // number of pollers goroutine to launch
	pollInterval = 60*time.Second  // how often to poll each url
	statusInterval = 10*time.Second // how often to log status
	errTimeout = 10*time.Second // back-off timeout on error
)

var urls = []string{
	"http://www.google.com/",
	"http://www.golang.org/",
	"http://blog.golang.org",
}

// state represents last-known state of a URL
type State struct{
	url string
	status string
}

func StateMonitor(updateInterval time.Duration) chan <-State{
	updates := make(chan State)
	urlStatus := make(map[string]string)
	ticker := time.NewTicker(updateInterval)
	go func(){
		for{
			select{
			case <-ticker.C:
				logState(urlStatus)
			case s := <-updates:
				urlStatus[s.url] = s.status
			}
		}
	}()

	return updates
}

func logState(urlState map[string]string){
	log.Println("Current state:")
    for k,v := range urlState{
            log.Printf("%s %s",k,v)
	}
}

// Resource represent http url to be polled
type Resource struct{
	url string
	errCount int
}

// Polls executes an http head request for url
// and returns the http status string or an error string
func (r *Resource) Poll() string{
	resp,err := http.Head(r.url)
	if err != nil{
		log.Println("Error",r.url,err)
		r.errCount++
		return err.Error()
	}

	r.errCount=0
	return resp.Status
}

// sleep sleeps for an appropriate interval of time
// before sending the resouces to done
func (r *Resource) Sleep(done chan<-*Resource){
	time.Sleep(pollInterval+errTimeout*time.Duration(r.errCount))
	done <- r
}

func Poller(in <-chan *Resource,out chan<- *Resource,status chan<-State){
     for r := range in{
		 s := r.Poll()
		 status <- State{r.url,s}
		 out <- r
	 }
}

func main(){
   pending,complete := make(chan *Resource),make(chan *Resource)

   status := StateMonitor(statusInterval)

   for i:=0;i<numPollers;i++{
	   go Poller(pending,complete,status)
   }

   go func(){
	   for _,url := range urls{
		   pending <- &Resource{url: url}
	   }
   }()

   for r := range complete{
	   go r.Sleep(pending)
   }
}