// Copyright 2013 marpie. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"flag"
	"fmt"
	"github.com/marpie/winfsnotify"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

func parseArguments(parameters []string, event *winfsnotify.Event) []string {
	out_params := make([]string, len(parameters))
	i := 0
	for _, entry := range parameters {
		output := strings.Replace(entry, "[[Cookie]]", fmt.Sprintf("%d", event.Cookie), -1)
		output = strings.Replace(output, "[[Filename]]", event.Name, -1)
		output = strings.Replace(output, "[[Info]]", event.String(), -1)
		out_params[i] = output
		i += 1
	}
	return out_params
}

func main() {
	w, err := winfsnotify.NewWatcher()
	if err != nil {
		return
	}
	defer w.Close()

	event := flag.String("event", "all", "possible Events: all, access, close, create, delete, modify, move.")
	one_shot := flag.Bool("one-shot", false, "only execute once (true/false)")
	pace := flag.Bool("pace", true, "only execute once a second. (true/false)")
	filterRegex := flag.String("filter", ".*", "filter by regex. (default: include all)")
	excludeRegex := flag.String("exclude", "", "exclude by regex. (default: exclude none)")
	//recursive := flag.Bool("recursive", true, "recursively wait for events. (true/false)")
	flag.Parse()

	command := flag.Arg(0)
	if len(command) < 1 {
		fmt.Printf("%s [command] [param1 [param2 [param3]...]])\n\n", os.Args[0])
		flag.Usage()
		fmt.Printf("\nThe supplied command is parsed for the values, that are replace with the received event values:")
		fmt.Printf("  [[Cookie]]=unique cookie of the event\n  [[Name]]=File/Directory name\n  [[Info]]=Debug information.")
		fmt.Printf("\nExamples:\nOpen newly created files/folders in Explorer:\n  notifyexec.exe -event=create explorer.exe [[Filename]]\n")
		return
	}
	parameters := flag.Args()[1:]

	// set-up regex
	var filter, exclude *regexp.Regexp
	if *filterRegex != "" {
		filter, err = regexp.Compile(*filterRegex)
		if err != nil {
			fmt.Printf("Error compiling regex: %v", err)
			return
		}
	}

	if *excludeRegex != "" {
		exclude, err = regexp.Compile(*excludeRegex)
		if err != nil {
			fmt.Printf("Error compiling regex: %v", err)
			return
		}
	}

	var mode uint32 = winfsnotify.FS_ALL_EVENTS // listen to all events
	switch *event {
	case "all":
		mode = winfsnotify.FS_ALL_EVENTS
	case "access":
		mode = winfsnotify.FS_ACCESS
	case "close":
		mode = winfsnotify.FS_CLOSE
	case "create":
		mode = winfsnotify.FS_CREATE
	case "delete":
		mode = winfsnotify.FS_DELETE
	case "modify":
		mode = winfsnotify.FS_MODIFY
	case "move":
		mode = winfsnotify.FS_MOVE
	default:
		println("Unknown mode! Listening to all events!")
	}

	if *one_shot {
		mode |= winfsnotify.FS_ONESHOT
	}

	if err := w.AddWatch(".", mode); err != nil {
		fmt.Printf("Error adding watch: %v\n", err)
		return
	}

	last := time.Now().Unix()
	for event := range w.Event {
		if *pace {
			now := time.Now().Unix()
			if last == now {
				time.Sleep(1 * time.Second)
				last = now
				continue
			}
			last = now
		}

		if *excludeRegex != "" {
			if exclude.MatchString(event.Name) {
				continue
			}
		}

		if *filterRegex != "" {
			if !filter.MatchString(event.Name) {
				continue
			}
		}
		println(event.String())

		output := parseArguments(parameters, event)

		cmd := exec.Command(command, output...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Start(); err != nil {
			fmt.Printf("Error while running command: %v\n", err)
		}

		if *one_shot {
			break
		}
	}
}
