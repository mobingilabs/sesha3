package main

import (
	"bufio"
	"fmt"
	"os/exec"
	"strings"
)

func main() {
	messages := make(chan string)

	go func() {
		connect := exec.Command("gotty", "bash")
		stdout, _ := connect.StderrPipe()

		connect.Start()
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			out := scanner.Text()
			if strings.Index(out, "Alternative URL") != -1 {
				out = strings.Split(out, "URL: ")[1]
				messages <- out
			}
		}
		connect.Wait()
	}()

	fmt.Println(<-messages)
}
