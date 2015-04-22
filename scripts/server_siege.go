package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"runtime"
	"strings"
	"time"
)

var (
	concurreny = flag.Int("c", 10, "Concurrency")
	file       = flag.String("f", "", "Url file")
)

func fetch(url string, c chan bool) {
	status := false

	resp, err := http.Get(url)
	if err != nil {
		fmt.Println(err)
	}
	defer resp.Body.Close()

	_, _ = ioutil.ReadAll(resp.Body)
	if http.StatusOK == resp.StatusCode {
		status = true
	} else {
		fmt.Println(url)
	}

	c <- status
}

func readFile(filename string) (result []string, err error) {
	f, err1 := os.Open(filename)
	defer f.Close()
	if err1 != nil {
		return nil, err1
	}

	r := bufio.NewReader(f)
	for {
		url, err2 := r.ReadString(10)
		if err2 == io.EOF {
			break
		} else if err2 != nil {
			fmt.Println(err2)
			return nil, err2
		} else {
			result = append(result, strings.TrimSpace(url))
		}
	}

	return result, nil
}

func main() {
	flag.Parse()

	var urls []string
	var err error

	filename := *file
	if filename != "" {
		urls, err = readFile(filename)
		if err != nil {
			fmt.Println(err)
			return
		}
	}

	fmt.Println(*concurreny, "concurrent fetchers...")
	runtime.GOMAXPROCS(runtime.NumCPU())

	start := time.Now()

	success, fail := 0, 0

	result := make(chan bool)
	tokenChan := make(chan bool, *concurreny)

	for i := 0; i < *concurreny; i++ {
		tokenChan <- true
	}

	for _, url := range urls {
		go func(u string) {
			<-tokenChan
			defer func() { tokenChan <- true }()

			fetch(u, result)
		}(url)
	}

	for i := 0; i < len(urls); i++ {
		select {
		case status := <-result:

			if status == true {
				success += 1
			} else {
				fail += 1
			}
		}
	}

	total := success + fail
	fmt.Println("Total:", total)
	fmt.Println("Success:", success)
	fmt.Println("Fail:", fail)
	fmt.Printf("Finished in %v\n", time.Since(start))
}
